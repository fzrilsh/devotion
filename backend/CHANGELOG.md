# Changelog Backend

Semua perubahan penting pada backend dicatat di sini. Format mengikuti
Conventional Commits, entri ditulis pada branch kerja yang sama dengan
perubahannya.

## [Belum dirilis]

### Ditambahkan
- Modul Go `github.com/fzrilsh/devotion/backend` dengan toolchain dipatok
  `go 1.23.4`.
- Dispatcher subcommand di `cmd/devotion` dengan delapan perintah terdaftar:
  `serve`, `admin:create`, `seed:regions`, `seed:master-data`,
  `seed:test-data`, `reset:test-data`, `user:verify`, `health:check`. Semua
  masih stub kecuali dispatcher; diisi di branch berikutnya. (T002)
