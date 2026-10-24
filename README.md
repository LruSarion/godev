# godev

Runner development untuk proyek Go. Build, jalankan, dan restart aplikasi saat source berubah—atau gunakan `--nw` untuk testing tanpa rebuild otomatis.

Native Go, tanpa script PowerShell dan tanpa dependency modul eksternal. Windows-first; backend Linux/macOS/BSD masih eksperimental.

## Instalasi

Butuh **Go 1.22+** di `PATH`. Gunakan versi Go yang masih mendapat pembaruan keamanan dan memenuhi kebutuhan proyek kamu.

```sh
go install github.com/LruSarion/godev@latest
```

Tambahkan `GOBIN` ke `PATH`, atau direktori `bin` di dalam `GOPATH` jika `GOBIN` kosong. Periksa lokasinya dengan:

```sh
go env GOBIN GOPATH
```

Pada Windows, lokasi default biasanya `%USERPROFILE%\go\bin`. Buka ulang terminal setelah mengubah `PATH`.

Untuk update, jalankan ulang perintah instalasi.

## Mulai cepat

Jalankan dari root **proyek kamu**, bukan folder source godev.

```sh
# Proyek dengan package main di root
godev

# Proyek dengan entrypoint terpisah
godev --target ./cmd/server

# Build sekali, jalankan tanpa watcher/restart otomatis
godev --nw --target ./cmd/server

# Teruskan argumen ke aplikasi setelah --
godev --target ./cmd/server -- --port 8080
```

`--port` pada contoh merupakan argumen aplikasi, bukan fitur godev. Gunakan argumen yang didukung aplikasimu. Hentikan runner dengan **Ctrl+C**.

## Opsi

| Opsi | Fungsi |
| --- | --- |
| `--nw` | Build sekali lalu jalankan, tanpa memindai perubahan file. |
| `--target <package>` | Package `main` yang dibuild; default `.`. |
| `--help`, `-h` | Tampilkan bantuan. |
| `-- <args>` | Teruskan semua argumen berikutnya ke aplikasi. |

Pilih satu package `main`, bukan pola seperti `./...`. Working directory dan environment aplikasi mengikuti godev. Stdin diteruskan ke aplikasi; stdout/stderr build dan aplikasi tampil di terminal.

Pada mode `--nw`, perubahan Go baru berlaku setelah runner dihentikan dan dijalankan ulang. Exit code aplikasi diteruskan jika tersedia; kegagalan build/start menghasilkan `1`, argumen tidak valid `2`, dan Ctrl+C `130`.

## Cara rebuild bekerja

1. Godev memantau isi `*.go`, `go.mod`, dan `go.sum` di bawah working directory.
2. Perubahan digabung dengan debounce sebelum build. Hanya satu build aktif.
3. Build memakai executable temporary baru; **aplikasi lama tetap berjalan selama build**.
4. Build gagal: error tampil, aplikasi lama tetap berjalan.
5. Build sukses: aplikasi lama dihentikan, lalu executable baru dijalankan.

Perubahan yang terdeteksi saat build membatalkan build tersebut dan memicu build berikutnya. Jika aplikasi keluar sendiri, watcher menunggu edit berikutnya, bukan restart terus-menerus.

**Restart bukan zero downtime.** Ada jeda pergantian proses, tanpa graceful shutdown atau rollback jika aplikasi baru gagal start. Godev bukan process manager production.

## HTML, CSS, JavaScript, dan embed

- Jika aplikasi membaca frontend dari disk, perubahan bisa terlihat saat refresh browser tanpa rebuild, termasuk dengan `--nw`.
- HTML/CSS/JS dan aset `//go:embed` tidak dipantau. Untuk memperbarui aset embedded, jalankan ulang godev atau ubah source Go agar rebuild terpicu.
- Godev tidak menjalankan bundler dan tidak menyediakan browser auto-reload.

## AI coding agent dan MCP

Agent yang memiliki akses terminal bisa memakai CLI langsung. **Godev bukan MCP server** dan tidak memerlukan MCP khusus. Jika lingkungan agent menyediakan terminal melalui MCP, gunakan perintah CLI yang sama melalui terminal tersebut.

Contoh instruksi untuk agent:

> Jalankan server development dari root proyek dengan `godev --nw --target ./cmd/server`. Pertahankan proses selama sesi testing. Jangan jalankan runner kedua pada port yang sama. Hentikan dengan Ctrl+C saat selesai. Jika source Go berubah, minta izin restart sebelum memuat perubahan.

Panduan penggunaan siap pakai tersedia di [`skills/godev/SKILL.md`](skills/godev/SKILL.md). Salin folder skill ke lokasi skill yang didukung agent kamu; lokasi dan cara aktivasi berbeda antar-agent. Instalasi CLI tidak otomatis memasang skill atau mengubah konfigurasi agent.

Untuk agent yang mengembangkan source godev sendiri, lihat [`AGENTS.md`](AGENTS.md).

## Platform dan batasan

- **Windows:** cleanup memakai Windows Job Object dan `taskkill.exe` untuk proses milik runner. Host yang membatasi job assignment dapat menolak startup.
- **Linux/macOS/BSD: eksperimental.** Cleanup hanya menangani direct child; descendant aplikasi/compiler dapat bertahan.
- Platform lain belum didukung. Gunakan Ctrl+C; SIGTERM belum ditangani khusus. Penghentian paksa runner tidak menjamin cleanup.
- Watch memakai polling hash sekitar 150 ms dan debounce 300 ms. Proyek besar atau network filesystem dapat menambah beban I/O.
- Direktori `.git`, `.local`, `.idea`, `.vscode`, `vendor`, `node_modules`, `dist`, `build`, `tmp`, dan `coverage` dilewati. Pengecualian case-sensitive dan belum dapat dikonfigurasi; symlink tidak diikuti.
- Dependency di luar working directory, `replace` eksternal, `go.work`, file C, environment, dan konfigurasi tidak dipantau.
- Snapshot tidak atomik: edit lalu revert di antara pemindaian atau perubahan tepat setelah pemeriksaan masih dapat lolos.
- Executable temporary harus berada di luar working directory. Jika ditolak, sesuaikan `TEMP`/`TMPDIR`. File temporary dapat tertinggal setelah penghentian paksa.
- `go build` mengikuti konfigurasi Go lokal, termasuk download module/toolchain jika diizinkan. Godev tidak memasang dependency sistem otomatis.

## Troubleshooting

**`godev` tidak ditemukan:** periksa direktori binary Go di `PATH`, lalu buka terminal baru.

**Package bukan executable:** pilih lokasi package `main` lewat `--target`.

**Edit tidak memicu rebuild:** periksa apakah memakai `--nw`, file berada dalam cakupan watch, dan ekstensi didukung.

**Port sudah dipakai:** hentikan instance aplikasi/runner sebelumnya. Jangan mematikan proses lain tanpa memastikan pemiliknya.

## Kontribusi

Source CLI berada di root module. Test unit mencakup parsing argumen dan snapshot filesystem; pengujian lifecycle proses lintas platform memerlukan pemeriksaan runtime terpisah.

```sh
go test ./...
go build .
```
