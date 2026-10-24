package main

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	o, err := parse([]string{"--nw", "--target", "./cmd/server", "--", "--port", "8080"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !o.noWatch || o.target != "./cmd/server" || !reflect.DeepEqual(o.args, []string{"--port", "8080"}) {
		t.Fatalf("unexpected options: %+v", o)
	}
	defaultOptions, err := parse(nil, io.Discard)
	if err != nil || defaultOptions.target != "." || defaultOptions.noWatch {
		t.Fatalf("bad defaults: %+v, %v", defaultOptions, err)
	}
}

func TestParseRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"server"}, {"--unknown"}, {"--target", ""}, {"--target", "-bad"}} {
		if _, err := parse(args, io.Discard); err == nil {
			t.Errorf("accepted %q", args)
		}
	}
}

func TestHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		if _, err := parse([]string{arg}, io.Discard); err != flag.ErrHelp {
			t.Errorf("%s: %v", arg, err)
		}
	}
}

func TestExitCode(t *testing.T) {
	if exitCode(nil) != 0 {
		t.Fatal("successful exit must return zero")
	}
}
