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
- `docker-compose.yml` dengan tepat dua layanan runtime (`postgres`,
  `backend`), penyetelan Postgres untuk 2GB dari research.md R-03, batas log
  `max-size 10m`/`max-file 3` di keduanya, volume `pgdata`, bind mount
  `${UPLOAD_PATH}`, dan `TZ: Asia/Jakarta`. Gate I: hitung entri di bawah
  `services:`, harus dua. (T005)
- Kerangka sembilan berkas `docs/*.md` (`menjalankan`, `pengujian`,
  `dependencies`, `utang-teknis`, `layanan-luar`, `temuan-penguji`,
  `cloudflare-ips`, `setup-vps`, `skenario-uji-manual`) plus
  `frontend/CHANGELOG.md`. Diisi sekarang: `layanan-luar.md`, `setup-vps.md`
  (ekstrak quickstart.md A-B), `skenario-uji-manual.md` (penunjuk ke §F),
  `utang-teknis.md` (tiga item Complexity Tracking). `cloudflare-ips.md`
  menyusul di T013. (T006)
- `backend/Dockerfile` multi-stage (build `golang:1.23.4-alpine`, runtime
  `alpine:3.20` non-root) dan `.github/workflows/ci.yml`. Urutan pipeline:
  `go vet` -> `go test` (Postgres sebagai layanan CI, bukan runtime, jadi Gate I
  tetap dua) -> build frontend -> salin `frontend/dist/.` ke `backend/webdist/`
  sebelum docker build -> push GHCR tag `<sha>` dan `latest` -> deploy SSH di
  `main`. (T007)
- `internal/platform/clock.go`: `Clock` interface, `SystemClock` (waktu
  ter-lokalisasi Asia/Jakarta), `TestClock` dengan `Set`/`Advance` ber-mutex,
  dan `WeekStart` (Senin awal minggu WIB) yang dipakai kedua lapisan penjadwal.
  Uji menyisir tree: `time.Now()` dilarang di luar `platform` dan `cmd`. (T008)
- `internal/platform/config`: `Load(getenv)` memvalidasi konfigurasi tanpa
  mengubah state proses. Wajib di semua environment: `APP_ENV`, `APP_BASE_URL`,
  `DATABASE_URL`, `UPLOAD_PATH`; wajib hanya di produksi: TLS, CF client CA,
  Mailjet, `MAIL_FROM`, `WHATSAPP_NUMBER`, `SENTRY_DSN`. Default
  `UPLOAD_TOTAL_LIMIT_MB=500`, `UPLOAD_FILE_LIMIT_MB=5`. `APP_ENV` tak dikenal
  adalah galat. Semua variabel hilang dikumpulkan dalam satu galat yang hanya
  memuat nama, tidak pernah nilai. `IsProduction()` untuk penjaga
  `seed:test-data`/`reset:test-data`. (T009)
- 14 migrasi SQL (`000001_extensions` sampai `000014_rate_limit`, 28 berkas
  up/down) yang memetakan data-model.md §12 satu banding satu, ditambah runner
  `internal/platform/migrate`. Runner memakai `iofs` atas migrasi yang di-embed
  (`db/embed.go`), jalan di bawah `pg_try_advisory_lock` dengan kunci konstanta
  pada satu koneksi yang di-pin, dan mengembalikan nil tanpa galat bila lock
  dipegang proses lain (skip saat rollover deploy). Tanpa `DEFAULT now()` di
  mana pun; kolom waktu diisi aplikasi lewat `Clock`. Down migration kebalikan
  tepat dalam urutan mundur (trigger sebelum fungsi, fungsi sebelum tabel).
  Uji: versi 14 `dirty=false`, idempoten dua kali, down-up kembali ke versi 14,
  tiga fungsi trigger terpasang lewat `pg_trigger`, empat constraint kunci lewat
  `pg_constraint`, sapuan larangan `DEFAULT` waktu, dan kelengkapan 14 pasang
  migrasi. Uji integrasi memakai skema terpisah pada Postgres yang sama dan
  `t.Skip` bila `DATABASE_URL_TEST` tak terjangkau. (T010)
