# Panduan agent kontributor

Berlaku untuk source repository godev. Untuk memakai CLI pada proyek lain, lihat `skills/godev/SKILL.md`.

## Struktur

- `main.go`: parsing CLI, build, dan watch loop.
- `snapshot.go`: pemindaian dan hash source.
- `process.go`: lifecycle proses bersama.
- `process_windows.go`, `process_unix.go`: ownership dan cleanup per platform.
- `main_test.go`, `snapshot_test.go`: test unit.

## Kontrak yang dijaga

- Root package tetap `main`, module `github.com/LruSarion/godev`, agar instalasi `go install ...@latest` bekerja.
- Jangan tambahkan konfigurasi, path, atau argumen khusus proyek pengguna.
- `--nw` tidak memindai source atau memicu restart otomatis.
- Aplikasi lama tetap hidup selama build; build gagal tidak menghentikannya.
- Teruskan argumen aplikasi setelah `--` tanpa shell interpolation.
- Cleanup hanya proses milik runner. Pertahankan perlindungan terhadap PID reuse.
- Jangan mengklaim cleanup descendant Unix setara Windows.
- Tidak membaca `.env` atau memasang dependency sistem secara otomatis.

## Cara bekerja

- Periksa perubahan pengguna sebelum mengedit; jangan menimpa pekerjaan lain.
- Pertahankan Go minimum di `go.mod`; hindari dependency baru tanpa alasan jelas.
- Tambahkan test parsing/snapshot ketika mengubah perilakunya; perubahan lifecycle perlu verifikasi runtime platform terkait.
- Format file Go yang diubah dengan `gofmt`.
- Ikuti izin validasi pengguna sebelum menjalankan test, build, atau aplikasi. Laporkan tepat pemeriksaan yang dijalankan dan yang tidak.
- Perbarui README jika CLI atau batasan berubah.
- Jangan commit, amend, push, membuat release, atau mengubah konfigurasi global tanpa permintaan.

## Pemeriksaan yang relevan

`go test ./...` dan `go build .` untuk pemeriksaan dasar. Verifikasi proses mencakup build sukses/gagal, edit selama build, aplikasi keluar sendiri, Ctrl+C saat idle/build/run, path dengan spasi, dan cleanup descendant. Test unit tidak membuktikan semua perilaku runtime tersebut.
