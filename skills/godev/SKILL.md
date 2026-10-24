---
name: godev
description: Jalankan proyek Go menggunakan godev, pilih target main, gunakan mode tanpa watcher untuk testing stabil, dan kelola restart serta cleanup melalui terminal.
---

# Menggunakan godev

Gunakan saat pengguna meminta menjalankan server Go dengan godev, auto-rebuild, atau testing tanpa restart otomatis. Bukan MCP server; butuh akses terminal.

## Persiapan

1. Temukan root proyek dan package `main` yang benar. Jangan mengasumsikan semua proyek memakai `./cmd/server`.
2. Pastikan Go dan godev tersedia di PATH. Jika belum, tawarkan `go install github.com/LruSarion/godev@latest`; jangan instal atau mengubah PATH global tanpa izin.
3. Periksa instruksi proyek dan izin menjalankan aplikasi. Startup dapat mengakses DB, menjalankan migrasi, atau menghubungi layanan eksternal.
4. Pastikan tidak ada runner aktif untuk aplikasi/port yang sama. Jangan menghentikan proses yang bukan milik sesi.

## Memilih mode

```sh
# Package main di root, auto-rebuild
godev

# Entry point tertentu
godev --target ./cmd/server

# Testing stabil: build awal sekali, tanpa watcher
godev --nw --target ./cmd/server

# Argumen aplikasi setelah pemisah
godev --nw --target ./cmd/server -- --port 8080
```

`--port` hanya contoh; periksa argumen aplikasi. Pilih satu package `main`, bukan `./...`.

## Mengelola sesi

- Gunakan sesi terminal/background yang bisa dikelola; simpan handle sesi agar proses dapat dihentikan.
- Jangan menganggap pesan aplikasi mulai sebagai bukti HTTP siap. Periksa readiness hanya jika pengguna mengizinkan pemeriksaan itu.
- Mode watch mempertahankan aplikasi lama saat build gagal. Halaman yang masih dapat dibuka belum tentu memakai source terbaru; baca output build.
- Dengan `--nw`, perubahan Go membutuhkan stop dan start ulang. Koordinasikan restart agar tidak mengganggu testing pengguna.
- Refresh frontend tanpa build hanya berlaku jika aplikasi membaca file dari disk. Aset embedded tidak dipantau.
- Hentikan runner memakai Ctrl+C setelah selesai kecuali pengguna ingin mempertahankannya. Penghentian paksa tidak menjamin cleanup; Unix hanya menangani direct child.
- Laporkan command, status proses, error, dan batas verifikasi secara faktual. Jangan mengklaim test lulus hanya karena server berjalan.

## Batas scope

Godev memantau `.go`, `go.mod`, dan `go.sum` di bawah working directory dengan pengecualian direktori. Tidak memantau dependency eksternal, `go.work`, atau frontend; tidak menyediakan zero-downtime, bundler, maupun browser auto-reload. Jangan menambahkan argumen khusus OSEE atau proyek lain kecuali aplikasi target memang mendukungnya.
