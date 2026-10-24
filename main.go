package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"
)

type options struct {
	noWatch bool
	target  string
	args    []string
}

func parse(args []string, out io.Writer) (options, error) {
	o := options{}
	fs := flag.NewFlagSet("godev", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&o.noWatch, "nw", false, "build and run once, without scanning")
	fs.StringVar(&o.target, "target", ".", "single main package to build")
	fs.Usage = func() {
		fmt.Fprintln(out, "Usage: godev [--nw] [--target .|./cmd/server] [-- application arguments]")
		fs.PrintDefaults()
	}
	// Require an explicit separator for application arguments.
	for i, arg := range args {
		if arg == "--" {
			o.args = append([]string(nil), args[i+1:]...)
			args = args[:i]
			break
		}
	}
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() != 0 {
		return o, errors.New("application arguments must follow --")
	}
	if o.target == "" || o.target[0] == '-' {
		return o, errors.New("target must name a main package")
	}
	return o, nil
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	o, err := parse(args, os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, "godev: Go tool required in PATH")
		return 1
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	tmp, err := os.MkdirTemp("", "godev-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(tmp)
	// Refuse unusual TMP configurations that put generated files in watch scope.
	if rel, e := filepath.Rel(root, tmp); e == nil && rel != ".." && !isOutside(rel) {
		fmt.Fprintln(os.Stderr, "godev: temporary directory must be outside working directory; change TEMP/TMPDIR")
		return 1
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if o.noWatch {
		path := filepath.Join(tmp, "app"+executableSuffix)
		if err := build(ctx, goTool, o.target, path); err != nil {
			if ctx.Err() != nil {
				return 130
			}
			fmt.Fprintln(os.Stderr, "godev: build:", err)
			return 1
		}
		p, err := start(path, o.args, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer p.stop()
		select {
		case <-ctx.Done():
			return 130
		case err := <-p.done:
			return exitCode(err)
		}
	}
	return watch(ctx, root, tmp, goTool, o)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var e *exec.ExitError
	if errors.As(err, &e) && e.ExitCode() > 0 {
		return e.ExitCode()
	}
	return 1
}

func build(ctx context.Context, goTool, target, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := start(goTool, []string{"build", "-o", path, target}, false)
	if err != nil {
		return err
	}
	defer p.stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-p.done:
		return err
	}
}

type buildResult struct {
	path   string
	source snapshot
	err    error
}

func watch(ctx context.Context, root, tmp, goTool string, o options) int {
	current, err := scan(ctx, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	results := make(chan buildResult, 1)
	var app *process
	var appDone <-chan error
	var building bool
	var buildCancel context.CancelFunc
	defer func() {
		if building {
			buildCancel()
			<-results
		}
		if app != nil {
			app.stop()
		}
	}()
	pending := true
	changed := time.Now().Add(-time.Second)
	sequence := 0
	startBuild := func() {
		sequence++
		path := filepath.Join(tmp, fmt.Sprintf("app-%d%s", sequence, executableSuffix))
		source := current
		buildCtx, cancel := context.WithCancel(ctx)
		buildCancel = cancel
		building, pending = true, false
		fmt.Fprintln(os.Stderr, "godev: building", o.target)
		go func() { results <- buildResult{path, source, build(buildCtx, goTool, o.target, path)} }()
	}
	observe := func(next snapshot) {
		if !same(current, next) {
			current, pending, changed = next, true, time.Now()
			if building {
				buildCancel()
			}
		}
	}
	for {
		if ctx.Err() != nil {
			return 130
		}
		if pending && !building && time.Since(changed) >= 300*time.Millisecond {
			startBuild()
		}
		select {
		case <-ctx.Done():
			return 130
		case <-ticker.C:
			next, err := scan(ctx, root)
			if err != nil {
				fmt.Fprintln(os.Stderr, "godev: scan:", err)
				if building {
					buildCancel()
				}
				pending, changed = true, time.Now()
				continue
			}
			observe(next)
		case result := <-results:
			building = false
			buildCancel()
			// Check contents again, not just timer observations, before replacement.
			next, scanErr := scan(ctx, root)
			if scanErr == nil {
				observe(next)
			}
			if scanErr != nil || pending || !same(result.source, current) || ctx.Err() != nil {
				os.Remove(result.path)
				pending, changed = true, time.Now()
				continue
			}
			if result.err != nil {
				fmt.Fprintln(os.Stderr, "godev: build failed; previous application retained:", result.err)
				os.Remove(result.path)
				continue
			}
			if app != nil {
				app.stop()
				os.Remove(app.cmd.Path)
				app = nil
				appDone = nil
			}
			app, err = start(result.path, o.args, true)
			if err != nil {
				fmt.Fprintln(os.Stderr, "godev: start:", err)
				os.Remove(result.path)
				continue
			}
			appDone = app.done
			fmt.Fprintln(os.Stderr, "godev: application started")
		case err := <-appDone:
			fmt.Fprintf(os.Stderr, "godev: application exited (%d); waiting for edits\n", exitCode(err))
			app.stop()
			os.Remove(app.cmd.Path)
			app, appDone = nil, nil
		}
	}
}
