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
- Lapisan akses data: `sqlc.yaml` (engine postgresql, `schema: db/migrations`
  supaya tipe tak menyimpang dari database yang dimigrasi, `queries: db/queries`,
  `sql_package: pgx/v5`, `emit_json_tags: false`), `db/queries/health.sql`, dan
  hasil `sqlc generate` di `internal/db/sqlcgen` (di-commit). `internal/db/pool.go`
  menyetel pool ke angka R-03: `MaxConns 15`, `MinConns 2`, lifetime 30m, idle 5m,
  health check 1m, dengan `Ping` saat buka. `tx.go` `WithTx` rollback pada galat
  maupun panic lalu re-panic. Harness `internal/db/testdb`: `New(t, name)` membuat
  skema `test_<name>`, memigrasinya, mengembalikan pool
  `search_path=test_<name>,public`, TRUNCATE (bukan DROP) saat cleanup.
  Kunci advisory migrasi dipindah ke bentuk dua-int
  `pg_try_advisory_lock(class, hashtext(current_schema()))` sehingga skema uji
  yang berbeda tak saling memblokir; ekstensi `citext`/`pgcrypto` dipasang
  `WITH SCHEMA public` dan down 000001 dikosongkan karena ekstensi milik seluruh
  database, bukan per skema. (T011)
- `internal/platform/cloudflare`: rentang IP Cloudflare resmi dipatok sebagai
  konstanta, di-parse sekali di `init` menjadi `[]*net.IPNet` dan panic pada
  entri rusak supaya typo gagal saat startup. `RealIP` memisah `RemoteAddr`,
  mengembalikan kosong bila tak terurai, host mentah bila koneksi di luar rentang
  Cloudflare (tanpa menyentuh header), dan baru mempercayai `CF-Connecting-IP`
  bila koneksi dari rentang tersebut. Konstanta `RetrievedAt` dan `docs/cloudflare-ips.md`
  dijaga sinkron oleh uji. Daftar diverifikasi ke sumber resmi, bukan dari
  research.md R-01. (T013)
- `internal/platform/httpx`: lapisan HTTP dasar. `codes.go` mentranskripsi 29
  kode galat dari openapi.yaml sebagai `type Code string` dan memegang peta
  `Code -> {Status, Title}` sehingga status HTTP diturunkan dari kode dan tidak
  bisa berbeda antar handler. `problem.go` menulis `application/problem+json`
  (RFC 9457) lewat `WriteProblem`, `WriteValidation` (membawa `errors[]` bentuk
  `ProblemValidation`), dan `WriteInternal` (500 generik). `logger.go`:
  `contextHandler` yang menarik `request_id` ke tiap record slog sehingga tak ada
  call site yang bisa lupa. `middleware.go`: rantai dari luar `RequestID` ->
  `Recover` (panic jadi 500 problem+json, stack ke slog bukan ke klien) ->
  `Logger` (JSON: method, path, status, duration_ms) -> `RealIP`. `router.go`:
  `http.ServeMux` pola method+path Go 1.22 dengan catch-all `/api/` yang
  mengembalikan 404 `Problem`, bukan HTML. (T012)
- `internal/platform/ratelimit`: empat jendela R-10 di `map[Target]window`
  sebagai satu-satunya sumber angka (login 5/15m per akun, OTP 3/jam per nomor,
  OTP 10 nomor berbeda/jam per alamat asal, request kuota 20/jam per pengguna).
  State di tabel `rate_limit`, bukan memori, sehingga redeploy bukan jalan
  pintas. `Check` menaikkan penghitung dan membandingkan dalam satu transaksi;
  `INSERT ... ON CONFLICT DO UPDATE` mengunci baris sehingga dua pemanggil
  berbarengan tak bisa sama-sama membaca hitungan sama lalu lolos. `CheckAddress`
  menghitung **nomor berbeda**, bukan percobaan: kirim ulang ke nomor yang sudah
  dihitung tak memakan kuota alamat, hanya nomor baru; sebuah
  `pg_advisory_xact_lock` per alamat men-serialisasi hitung-lalu-catat. Semua
  timestamp dari `Clock` sehingga uji kedaluwarsa jendela menggeser waktu, bukan
  tidur; 429 membawa `Retry-After` sampai jendela bergulir. Kueri di
  `db/queries/ratelimit.sql`, hasil `sqlc generate` di-commit. (T016)
- `internal/platform/session` dan `internal/account`: sesi dan seluruh
  permukaan auth. Token 32 byte `crypto/rand` di-encode base64url pada cookie
  `devotion_session` (`httpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, TTL 7
  hari dengan perpanjangan bergulir); yang tersimpan `token_hash` SHA-256, bukan
  token mentah, dan logout menghapus baris. Sepuluh endpoint sesuai
  `openapi.yaml`: `register`, `verify-email`, `verify-phone`, `resend-code`,
  `login`, `logout`, `recover/request`, `recover/confirm`, `GET /me`,
  `PATCH /me/roles`. `GET /me` mengembalikan bentuk `MyAccount`. bcrypt cost 10;
  login menjalankan rate limit T016 **sebelum** perbandingan bcrypt. Kode enam
  digit R-09 untuk email dan telepon, tersimpan sebagai hash SHA-256, sekali
  pakai lewat `consumed_at`, kedaluwarsa 15 menit dari `Clock`. `POST
  /auth/recover/request` selalu 202 dengan waktu respons dikonstankan lewat
  `platform.ConstantTimeFloor` (wall-clock, bukan `Clock`, karena kebocorannya
  sinyal waktu nyata) agar tak membocorkan keberadaan akun; `recover/confirm`
  mengakhiri semua sesi lain. `golang.org/x/crypto` naik dari indirect menjadi
  dependency langsung, dicatat di `docs/dependencies.md`. (T014)
- Middleware peran di `internal/platform/httpx/auth.go`: peran sebagai bitmask
  `Role` (`RoleSubcontractor`, `RoleBuyer`, `RoleAdmin`; bit admin terpisah,
  tak pernah tersirat oleh peran usaha, satu akun boleh memegang dua peran
  usaha). `RequireAuth` mengizinkan tiap pemanggil terautentikasi dan menaruh
  `Principal` di konteks; `RequireRole` mengizinkan yang memegang salah satu
  peran yang diminta. Kegagalan auth 401, peran salah 403, galat resolusi 500,
  sehingga hiccup basis data tak disalahartikan sebagai sesi absen. httpx tak
  mengimpor `account`: interface `Authenticator` disuntikkan, dan `account`
  mengimplementasinya (`Authenticate` memvalidasi cookie, memuat akun segar,
  melipat tiga flag boolean jadi bitmask). Router mencatat tiap pola sebagai
  publik eksplisit atau ter-gate; `UncoveredAPIRoutes` melaporkan pola `/api/*`
  non-publik yang tak ter-gate, dan uji menuntutnya kosong, sehingga endpoint
  tak bisa terbit tanpa keputusan peran. `logout` dipindah dari Public ke Gated
  (kontrak menuntut 401), `GET /me` dan `PATCH /me/roles` lewat `RequireAuth`
  plus adapter `fromPrincipal`. Uji: matriks penolakan tiap kombinasi peran tak
  berwenang, auth mendahului cek peran, dan rute akun nol tak tercakup. (T015)
- Penyimpanan berkas unggahan di `internal/platform/storage`, satu-satunya
  tempat byte klien menyentuh disk. `Save` menjalankan urutan yang mengikat:
  `io.LimitReader` ke batas per-berkas lebih dulu agar decode bomb tak menguras
  memori, lalu `http.DetectContentType` atas magic bytes (nama dan
  `Content-Type` dari klien tak pernah dipercaya), lalu decode dan re-encode
  gambar lewat `image/jpeg`/`image/png` untuk membuang EXIF (foto lokasi dari
  ponsel membawa koordinat GPS), PDF divalidasi magic bytes lalu disimpan apa
  adanya, lalu cek total 500MB, baru tulis dengan nama acak `crypto/rand`
  berekstensi dari tipe terverifikasi. `Open` menyelesaikan berkas lewat id dan
  menegakkan pemilik-atau-admin (FR-009); tak ada path statis sama sekali. Query
  `db/queries/uploaded_file.sql` (Create/Get/SumBytes), hasil `sqlc generate`
  di-commit. Uji menunjuk FR-006/FR-009: orang asing ditolak `ErrForbidden`,
  berkas berekstensi menipu ditolak `ErrUnsupportedType`, kuota penuh
  `ErrQuotaFull`, kelebihan ukuran `ErrTooLarge`, dan penanda EXIF hilang setelah
  re-encode. (T017)
- Aritmetika tenggat di `internal/order/deadline.go`, satu-satunya tempat waktu
  dihitung untuk kedua lapisan penjadwal (research.md R-07), sehingga sebuah
  pesanan tak pernah tampak beda status di halaman berbeda. Setiap fungsi
  menerima instan sekarang sebagai parameter, tak ada `time.Now()`:
  `ReadinessDeadline`, `AutoConfirmAt`, `IsAutoConfirmDue`,
  `IsAutoConfirmApproaching` (FR-068/FR-069), `IsCalendarStale` (FR-021),
  `IsRequestExpired` (FR-037/FR-082). Batas inklusif diuji per nanodetik.
- Penjadwal lapisan 2 di `internal/platform/scheduler`: satu `time.Ticker` lima
  menit dalam goroutine yang dinyalakan `serve`, bukan proses/cron/container
  kedua, jadi Gate I tetap dua layanan. Tiap job dibungkus
  `pg_try_advisory_lock(class, key)` pada koneksi pool yang di-pin, dilepas lewat
  `defer` pada koneksi sama dengan `context.Background()`, sehingga saat rollover
  deploy container lama melewatkan job alih-alih mengantre. `LockKey` konstanta
  literal dalam satu blok; pendaftaran job kosong (diisi T023). Uji menunjuk
  R-07: dua penjadwal pada database sama menaikkan counter tepat sekali, lock
  terlepas diperiksa lewat `pg_locks`. (T018)
- Modul `internal/masterdata` plus subcommand `seed:regions` dan
  `seed:master-data`, dan empat endpoint baca publik (`security: []`):
  `GET /api/master/products`, `/api/master/machines`, `/api/regions/provinces`,
  `/api/regions/cities` (filter opsional `?province=`). `NormalizeCityCode`
  membuang titik pada kode kota wilayah.id (`32.73` jadi `3273`) sebelum
  disimpan, karena `city_code_format` dan `city_belongs_to_province` menolak
  bentuk lain. `seed:regions` default membaca salinan `docs/master-data/regions.json`;
  `--refresh` mengambil dari wilayah.id, menormalkan, menulis ulang salinan,
  lalu mengisi database. Seeder idempoten pada kode/nama: sisip bila absen,
  perbarui nama bila ada, tak pernah menghapus karena `business_profile`
  merujuknya. Handler memetakan kolom DB (`id`/`type`) ke nama kontrak
  (`item_id`/`kind`). Uji: `NormalizeCityCode` langsung, seed dua kali idempoten,
  nol baris kota dengan `left(code,2) <> province_code`, dan keempat endpoint
  baca. (T019)
- Subcommand `admin:create` di `cmd/devotion`: membuat admin pertama atau
  mereset kata sandinya bila email sudah ada (idempoten lewat
  `INSERT ... ON CONFLICT (email) DO UPDATE`, query `UpsertAdmin`). Kata sandi
  dibaca dari prompt tanpa echo (`golang.org/x/term`, dikonfirmasi dua kali),
  tidak pernah lewat flag karena flag masuk riwayat shell; `--email` dan
  `--phone` dari flag. `account.CreateAdmin` memakai satu jalur bcrypt yang sama
  (`hashPassword`, cost 10) sehingga hashing kata sandi cuma satu tempat; baris
  admin punya `role_admin` true dan kedua peran usaha false, diterima
  `has_at_least_one_role` dan `admin_has_no_business_role`. `golang.org/x/term`
  dinaikkan dari indirect ke dependency langsung, dicatat di
  `docs/dependencies.md`. Uji: dua kali jalan tidak menduplikasi admin, panggilan
  kedua mengganti kata sandi. (T020)
- Penyajian SPA tersemat dan TLS produksi, plus `serve` yang sesungguhnya.
  `embed.go` (`package web`) menyematkan `webdist/` lewat `//go:embed
  all:webdist`; awalan `all:` wajib atau chunk Vite berawalan `_` tersaring
  diam-diam. `webdist/index.html` placeholder di-commit supaya direktif embed
  ter-compile; CI menimpanya dengan hasil build. `httpx.NewStatic` menegakkan
  urutan R-06: `/api/*` ke handler API (termasuk 404 `Problem` untuk path API
  tak dikenal, bukan `index.html`), berkas nyata di `webdist` dengan
  `Cache-Control: public, max-age=31536000, immutable` untuk aset ber-hash,
  sisanya jatuh ke `index.html` dengan `Cache-Control: no-cache`. `tlsconf.Load`
  membangun `tls.Config` dengan `ClientAuth: RequireAndVerifyClientCert`,
  `ClientCAs` dari CA klien Cloudflare, dan `MinVersion: TLS12`, sehingga
  Authenticated Origin Pulls menolak koneksi yang melewati edge (R-01). `serve`
  memuat config, menyambung pool, menjalankan migrasi, membangun router, mendaftar
  `account` dan `masterdata`, menyalakan penjadwal sebagai goroutine, lalu listen
  dengan `ReadHeaderTimeout: 10s` dan shutdown anggun pada SIGINT/SIGTERM: TLS di
  `:443` saat produksi, HTTP polos di `:8080` di pengembangan. Cookie `Secure`
  mati hanya bila `APP_BASE_URL` memakai `http://`, dan pengecualian itu nyaring
  di log. (T022)

