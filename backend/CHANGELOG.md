# Changelog Backend

Semua perubahan penting pada backend dicatat di sini. Format mengikuti
Conventional Commits, entri ditulis pada branch kerja yang sama dengan
perubahannya.

## [Belum dirilis]

### Ditambahkan
- Modul Go `github.com/fzrilsh/devotion/backend` dengan toolchain dipatok
  `go 1.25.0`.
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
- `backend/Dockerfile` multi-stage (build `golang:1.24.1-alpine`, runtime
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
- 15 migrasi SQL (`000001_extensions` sampai `000015_verification_code`, 30 berkas
  up/down) yang memetakan data-model.md §12 satu banding satu, ditambah runner
  `internal/platform/migrate`. Runner memakai `iofs` atas migrasi yang di-embed
  (`db/embed.go`), jalan di bawah `pg_try_advisory_lock` dengan kunci konstanta
  pada satu koneksi yang di-pin, dan mengembalikan nil tanpa galat bila lock
  dipegang proses lain (skip saat rollover deploy). Tanpa `DEFAULT now()` di
  mana pun; kolom waktu diisi aplikasi lewat `Clock`. Down migration kebalikan
  tepat dalam urutan mundur (trigger sebelum fungsi, fungsi sebelum tabel).
  Uji: versi 15 `dirty=false`, idempoten dua kali, down-up kembali ke versi 15,
  tiga fungsi trigger terpasang lewat `pg_trigger`, empat constraint kunci lewat
  `pg_constraint`, sapuan larangan `DEFAULT` waktu, dan kelengkapan 15 pasang
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
- Modul notifikasi `internal/notification`: antrean transaksional plus
  pengiriman kanal. `Enqueue` menerima `pgx.Tx`, bukan pool, sehingga baris
  notifikasi selalu ditulis di dalam transaksi kejadiannya (FR-086);
  notifikasi dalam platform selalu tersimpan meski semua kanal dimatikan
  preferensi, karena feed adalah satu-satunya jalur observasi penguji manual
  (FR-054). `IsTransactional` adalah fungsi atas enum `event_type`, bukan
  kolom yang bisa disalah-set pemanggil: hanya `calendar_stale`,
  `deadline_approaching`, dan `rating_request` yang non-transaksional, sisanya
  transaksional dan tidak bisa dibungkam preferensi (FR-091). Event
  transaksional mengantre email dan WhatsApp tanpa syarat; event
  non-transaksional hanya mengantre kanal yang masih diaktifkan akun. Kanal
  dikirim oleh job penjadwal `Deliver` (didaftarkan lewat `DeliverJob`),
  maksimal tiga percobaan lalu `failed_permanent`, hitungannya di
  `notification_channel` (FR-085); kegagalan kirim tidak pernah menyentuh baris
  notifikasi dalam platform. Email lewat `net/smtp` ke Mailjet tanpa SDK. Empat
  endpoint terjaga peran: daftar feed dengan kursor keyset opaque dan
  `unread_count`, tandai dibaca (idempoten, 404 bila bukan milik pemanggil atau
  id tak sah), GET dan PUT preferensi non-transaksional. Uji menunjuk FR-051,
  FR-054, FR-055, FR-085, FR-086, FR-091. (T023)
- Tautan WhatsApp di `internal/admin`: klien whatsmeow berjalan sebagai
  goroutine di dalam proses `serve` (research.md R-08), bukan layanan kedua,
  jadi Gate I tetap dua. Sesinya disimpan di Postgres yang sama lewat handle
  `database/sql` kedua (driver `pgx` stdlib), dibatasi `SetMaxOpenConns(2)` dan
  dianggarkan dari lima koneksi yang disisakan di luar pool 15. `Manager`
  membuka store, meng-upgrade skemanya, dan mengurus siklus hidup klien:
  `Start` menyalurkan kode QR ke status saat perangkat belum terpasang lalu
  `Connect`, `onEvent` membedakan `Connected` dari `LoggedOut`. `SendText`
  memenuhi `notification.WhatsAppSender` sehingga kanal WhatsApp kini terkirim.
  `GET /api/admin/whatsapp` khusus admin mengembalikan `WhatsAppStatus`
  (`connected`, `qr_code`, `last_error`); nomor layanan tidak pernah muncul di
  respons, log, maupun Sentry (FR-082), ditegakkan secara struktural karena
  tipe status tak punya field untuk membawanya. Subcommand `user:verify
  --email`/`--phone` memverifikasi akun tanpa antarmuka supaya nomor yang
  terblokir sesaat tak menghalangi pembuatan akun. Dependency `go.mau.fi/whatsmeow`
  dicatat di `docs/dependencies.md`. Uji menunjuk FR-082: gate admin (401/403/200),
  null saat kosong, dan QR/galat sampai ke body. (T024a)
- `internal/platform/health` menyajikan `GET /health` publik (`security:[]`)
  yang memeriksa tiga ketergantungan: ping basis data, tautan WhatsApp lewat
  `Manager.Connected()`, dan ruang sisa volume unggahan lewat `Statfs`. Balasan
  503 bila salah satu gagal, dengan status per ketergantungan (`ok`/`down`) di
  body; enum status itu tak punya ruang untuk nomor layanan (FR-082).
  `internal/platform/observability` menyalakan Sentry dengan `BeforeSend`
  berbentuk allowlist: event keluar dibangun ulang dari field aman saja, jadi
  request, cookie, user, Extra, dan Contexts dibuang alih-alih disaring, dan
  field sensitif baru aman secara default. Subcommand `health:check` menyelidik
  `GET /health` lewat HTTP untuk healthcheck kontainer tanpa `curl` di image.
  Dependency `github.com/getsentry/sentry-go` dicatat di `docs/dependencies.md`.
  Uji menunjuk FR-082: 503 saat tiap ketergantungan mati, dan scrub membuang
  kata sandi, token, nomor telepon, serta rujukan dokumen identitas. (T025)
- Kontrak `openapi.yaml` diselaraskan dengan data-model.md dan migrasi untuk
  jalur User Story 1. `RegisterRequest` kini membawa `city_code` dan `roles`
  wajib (profil usaha lahir dalam transaksi register, jadi `GET /profile/me`
  tak pernah 404). `ListingRequest`/`Listing` memakai `weekly_capacity`,
  `readiness_lead_days`, `product_item_ids`, dan `machines` dengan
  `machine_count`, menyamai `listing_product`/`listing_machine`, bukan lagi item
  tunggal. `AvailabilityPeriod`/`PeriodUpdateItem` mendapat `marked_full`.
  `ProfileUpdateRequest` membuang `address`/`province_code` yang tak berkolom;
  `MyProfile`/`PublicProfile` menurunkan `city_name`/`province_*` dari `city`.
  Kode galat `LISTING_ALREADY_EXISTS` (409) ditambahkan ke enum `Problem.code`
  dan `httpx` (`codes.go` kini 30 kode). Kunci respons validasi `'400'` pada
  path US1 diseragamkan ke `'422'` (status nyata `httpx.StatusFor`); sisa `'400'`
  path story lain dicatat sebagai utang di `docs/utang-teknis.md`. (T026-kontrak)
- Profil usaha (`internal/account/profile.go`, `profile_http.go`,
  `db/queries/profile.sql`): `GET /api/profile/me` dan `PUT /api/profile/me` di
  balik autentikasi, `GET /api/profile/{profileId}` publik. Profil kini lahir
  bersama akun dalam satu transaksi `POST /api/auth/register` (`CreateAccount` +
  `CreateProfile` di `db.WithTx`), sehingga `RegisterRequest` mewajibkan
  `city_code`/`business_name` dan `GET /profile/me` tak pernah 404. Kota tak
  dikenal menjadi 422 `FieldError` pada `city_code`, bukan pelanggaran foreign
  key 500. `PUT` memvalidasi nama minimal 3 karakter, koordinat lengkap atau
  kosong sebagai pasangan, dan rentang dalam wilayah Indonesia (menyalin
  `coordinates_within_indonesia`). `MyProfile`/`PublicProfile` menurunkan
  `city_name`/`province_code`/`province_name` dari join `city`+`province`;
  `verification_status` null dan reputasi kosong di US1. Id profil yang cacat
  atau tak dikenal pada path publik dijawab 404 tanpa membedakan keduanya.
  (T026)
- Fondasi listing kapasitas (T027, sedang berjalan): kueri SQL listing dan
  kalender di `db/queries/listing.sql` (`CreateListing`, `GetListingByProfile`,
  `GetListingByID`, `LockListingByProfile` `FOR UPDATE`, `UpdateListing`,
  `SetListingPublished`, `TouchCalendarUpdatedAt`, `RaiseHorizonUntil` dengan
  `GREATEST` agar horizon tak pernah mundur, `InsertPeriodsUpToWeek` idempoten
  lewat `ON CONFLICT`, `ListPeriodsInRange`, `LockPeriodByWeek`, `UpsertPeriod`,
  `PropagateCapacityToFuturePeriods`, `FindFutureAllocatedPeriodOverCapacity`,
  `PeriodHasActiveAllocation`, `CountActiveCatalogItemsOfType`, dan tautan
  `listing_product`/`listing_machine`), hasil `sqlc generate` di
  `internal/db/sqlcgen`, `internal/platform/dateid.go` (`FormatDateID` menghasilkan
  "24 Agustus 2026" ter-lokalisasi Asia/Jakarta untuk periode dan notifikasi),
  serta kerangka paket `internal/listing/listing.go` (`Service{pool, clock}`,
  `New`, `queries()`, `InitialHorizonWeeks = 14` yang memenuhi sekaligus minimal
  13 periode FR-088 dan minimal 3 bulan FR-017, `MaxPeriodBatch = 26`).
- Listing kapasitas subkontraktor (T027): enam rute di `internal/listing/http.go`,
  semuanya di belakang `httpx.RequireRole(auth, RoleSubcontractor)` sehingga tak
  ada yang lolos gerbang `UncoveredAPIRoutes()` dan peran salah ditolak 403.
  `GET /api/listing/me`, `POST /api/listing/me` (FR-010: listing langsung tayang
  tanpa gerbang verifikasi, `published` true sejak insert), `PUT /api/listing/me`,
  `PUT /api/listing/me/visibility` (nonaktif sementara lalu aktifkan kembali,
  FR-015), `GET /api/listing/me/periods`, dan `PUT /api/listing/me/periods`.
  `EnsureHorizon` menjamin setiap periode mingguan sampai minggu target ada lalu
  menaikkan `horizon_until`; idempoten dan aman dipanggil bersamaan tanpa
  advisory lock karena duplikasi dicegah `one_period_per_week`, kemunduran
  horizon dicegah `GREATEST`, dan deadlock dicegah urutan lock tetap (baris
  listing sebelum `availability_period`). Batas minggu dihitung satu tempat lewat
  `platform.WeekStart`, jadi `week_start_is_monday` dan `horizon_is_monday` tak
  bisa dilanggar. Propagasi kapasitas FR-089 di `PUT /api/listing/me`:
  `FindFutureAllocatedPeriodOverCapacity` menolak seluruh permintaan dengan 409
  `CAPACITY_ALREADY_ALLOCATED` bila ada periode mendatang yang pemakaiannya sudah
  melebihi kapasitas baru, `PropagateCapacityToFuturePeriods` menulis kapasitas
  baru hanya ke minggu `>= minggu berjalan` yang belum teralokasi, dan periode
  teralokasi dibiarkan utuh. Pra-pemeriksaan `CountActiveCatalogItemsOfType`
  memvalidasi tipe item sebelum insert, sehingga id mesin yang dikirim sebagai
  produk dijawab 422 menyebut `product_item_ids`, bukan 500 dari
  `trg_reject_wrong_product_item`. `PUT /api/listing/me/periods` memvalidasi
  seluruh batch (1..26 elemen, tiap `week_start` hari Senin dalam rentang minggu
  berjalan sampai 26 minggu ke depan, kapasitas non-negatif, tanpa minggu ganda)
  sebelum menulis apa pun, lalu dalam satu transaksi mengunci listing,
  memperpanjang horizon, dan mengunci tiap periode urut menaik; kapasitas di
  bawah pemakaian jadi 409 `CAPACITY_ALREADY_ALLOCATED` dan tanda penuh saat ada
  alokasi aktif jadi 409 `PERIOD_ALREADY_ALLOCATED`. `TouchCalendarUpdatedAt`
  adalah satu-satunya jalur yang memajukan `calendar_updated_at` (FR-021).
  `platform.ParseDate` mengurai `week_start` sebagai tengah malam Asia/Jakarta
  agar tanggal Senin tetap Senin, bukan bergeser sehari akibat lokalisasi UTC.
  Terpasang di `cmd/devotion/serve.go` lewat `listing.New(pool, clock).Register(router, acc)`
  sebelum gerbang rute. Uji pendamping di `internal/listing/listing_test.go`
  (jalur berhasil, penolakan peran, dan masukan tak sah tiap rute; propagasi
  FR-089; idempotensi dan konsistensi horizon; `FormatDateID` di
  `internal/platform/dateid_test.go`).
- Kalender awal dan horizon (T028): kemampuan FR-017 dan FR-088 sudah terkirim
  utuh di dalam T027 lewat `EnsureHorizon` (`internal/listing/calendar.go`) yang
  membuat periode mingguan minimal 13 minggu ke depan saat listing dibuat,
  memakai kapasitas mingguan sebagai kapasitas total, menyimpan periode terjauh
  di `horizon_until` konsisten dengan `MAX(week_start)`, dan memperpanjang
  horizon secara idempoten serta aman dipanggil bersamaan tanpa membuat baris
  ganda. T035 memanggil fungsi perpanjangan ini sebagai API internal, bukan kode
  yang hanya dipakai saat pembuatan listing. Ditandai selesai di `tasks.md`;
  tanpa perubahan kode baru karena cakupannya sudah dites di
  `internal/listing/listing_test.go` (`TestCreateListing_HorizonAwal*`,
  `TestEnsureHorizon_*`).
- Usulan item daftar baku (T029, FR-061): `POST /api/master/proposals` di
  `internal/masterdata/http.go` menerima usulan item baru dari pemanggil
  bisnis, digerbang ke `RoleSubcontractor` dan `RoleBuyer` (keduanya memilih
  item dari daftar baku yang sama, FR-022). Body divalidasi (`kind` product
  atau machine, `proposed_name` 2..80 karakter, 422 per-field), usulan tersimpan
  berstatus `pending` dengan `created_at` dari `Clock` (Rule 5). Metode domain
  `DecideProposal` (`internal/masterdata/proposal.go`) menerapkan keputusan admin
  dan mengantre notifikasi `item_proposal_decision` ke pengusul di dalam satu
  transaksi (FR-086), memenuhi syarat FR-061 bahwa pengusul diberi tahu saat
  usulannya diputus; permukaan HTTP admin `/admin/proposals` menyusul di T068.
  Kueri `InsertItemProposal`, `GetItemProposalByID`, `DecideItemProposal` di
  `db/queries/masterdata.sql` dan hasil `sqlc generate`. Diuji di
  `internal/masterdata/proposal_test.go` (`TestCreateProposal_Success_FR061`,
  `TestCreateProposal_RejectsRole_FR061`, `TestCreateProposal_RejectsInvalidInput_FR061`,
  `TestDecideProposal_NotifiesProposer_FR061`).
  Dua hal sengaja ditunda ke T068 (permukaan HTTP admin, FR-058): (1)
  `DecideProposal` memakai UPDATE ber-guard `WHERE status = 'pending'`, jadi
  keputusan atas proposal yang sudah diputus membuat `DecideItemProposal`
  mengembalikan `pgx.ErrNoRows` yang naik jadi galat transaksi; T068 harus
  memetakannya ke 409, bukan 500 mentah. (2) `DecideProposal` menerima `itemID`
  dari pemanggil untuk constraint `approved_yields_item`; belum ada pembuat item
  katalog dari proposal yang disetujui, jadi T068 perlu menyambungkan pembuatan
  `catalog_item` sesungguhnya saat approve.
- Test backend US1 (T030): melengkapi cakupan uji US1 di modul `account` dan
  `listing` tanpa menulis ulang yang sudah ada. Audit menemukan `listing` sudah
  memenuhi trio (jalur berhasil, penolakan peran, penolakan masukan) untuk
  seluruh rute-nya plus kasus horizon awal dan perpanjangan idempoten, jadi
  celah terpusat di `account`. Ditambah `internal/account/us1_test.go`:
  `TestPatchRoles_MenambahPeran_Berhasil_FR001`,
  `TestPatchRoles_TanpaSesi_Unauthorized_FR001`,
  `TestPatchRoles_MencabutSemuaPeran_Ditolak_FR001` (trio PATCH /me/roles),
  `TestVerifyPhone_JalurBerhasil_FR002`, `TestVerifyPhone_KodeSalah_Ditolak_FR002`,
  `TestResendCode_ChannelTidakSah_Ditolak_FR002`, `TestResendCode_SelaluDiterima_FR002`
  (gerbang verifikasi dua kanal FR-002), dan
  `TestPublicProfile_IdTidakDikenal_NotFound_FR016`. `handlePatchRoles`
  (`internal/account/handlers.go`) kini menolak permintaan yang mencabut kedua
  peran dengan 422 (FR-001), menghindari agar constraint `has_at_least_one_role`
  muncul sebagai 500. Komentar pada
  `TestCreateListing_TanpaPengajuanVerifikasi_TetapTayang_FR010` diperjelas: spec
  kita sengaja menyimpang dari status "Menunggu Verifikasi" dokumen sumber, dan
  test itu mengunci keputusan tersebut (FR-010).
- Mesin pencarian (T035): modul baca `internal/search` dengan rute tunggal
  `GET /api/search` bergerbang `RoleBuyer`. Kueri `SearchCandidates` di
  `db/queries/search.sql` mengikuti `data-model.md` §10: rentang kapasitas per
  kandidat dari minggu kesiapan (Senin dari tanggal pencarian + `readiness_lead_days`)
  sampai minggu deadline, minggu di luar `horizon_until` dihitung berkapasitas
  penuh (FR-088), empat kriteria keras dijumlahkan menjadi skor 0-4 tanpa
  pembobotan maupun normalisasi (FR-023, FR-024), filter yang tidak diisi
  dihitung terpenuhi dan dilaporkan tidak dievaluasi (FR-026), pemecah seri lima
  tingkat berakhir di `listing_id` (FR-025), keyset pagination opaque (FR-080),
  dan pengecualian listing milik pencari (FR-081). Perluasan wilayah kota →
  provinsi → nasional lewat parameter `region_level`, saran pelonggaran saat
  hasil kosong di tingkat nasional (FR-028), dan perpanjangan horizon kandidat
  lolos di transaksi tersendiri di luar kueri baca (FR-088). Skor tidak
  terpengaruh reputasi, verifikasi, kebaruan kalender, maupun jarak (FR-024).
- Uji determinisme dan rentang kapasitas mesin pencarian (T036) di
  `internal/search`: urutan identik pada pengulangan dan stabil antar halaman
  meski ada listing baru disisipkan di tengah penelusuran (SC-013, FR-025); skor
  tak berubah saat rating, verifikasi, dan kebaruan kalender diubah (FR-024);
  3.000 potong pada 500/minggu lolos di deadline 8 minggu dan gagal di 4 minggu
  (SC-019); jeda kesiapan 14 hari membuang dua minggu pertama sehingga totalnya
  di bawah kandidat jeda nol (SC-020); minggu kesiapan yang melampaui deadline
  menghasilkan kapasitas nol dan kriteria (d) tak terpenuhi (SC-020); kapasitas
  di luar horizon awal tetap dihitung penuh sampai deadline lalu periodenya
  benar-benar dimaterialisasi (SC-021, FR-088); filter mesin kosong membuat
  kriterianya terpenuhi dan dilaporkan tidak dievaluasi (FR-023, FR-026).
- Modul request kuota (T039): paket tulis `internal/quota` dengan dua rute
  `POST /api/quota-requests` dan `GET /api/quota-requests`, keduanya digerbang
  peran pembeli. Satu aksi mengirim satu request ke beberapa listing kandidat
  sekaligus, tiap kandidat membawa statusnya sendiri (FR-029), dan pembeli
  melihat daftar request-nya sendiri dengan keyset pagination kursor opaque
  terbaru dulu (FR-030, FR-080). Jendela balasan 72 jam (`reply_due_at`) dan
  `created_at` keduanya diambil dari `Clock` yang disuntikkan, bukan
  `time.Now()` (FR-082, Aturan 5). Listing tak dikenal atau belum tayang jadi
  422; listing milik pembeli sendiri ditolak 409 `SELF_REQUEST` sebelum ada
  insert apa pun, trigger basis data hanya jaring pengaman (FR-083). Tiap
  kandidat memicu notifikasi `request_received` di dalam transaksi yang sama
  sehingga kegagalan antrean membatalkan seluruh request.

### Diperbaiki
- Keyset pagination mesin pencarian (T036): klausa `WHERE` kursor yang memakai
  satu perbandingan row-value `<` untuk kelima kolom urut diganti rantai OR
  leksikografis eksplisit. Perbandingan row-value tunggal keliru karena urutan
  mencampur arah (skor dan sisa kapasitas menurun, sedangkan jeda, nama, dan
  `listing_id` menaik), sehingga kandidat berskor sama bisa muncul dua kali
  antar halaman. Tiap tingkat kini dibandingkan pada arahnya sendiri setelah
  tingkat di atasnya seri (SC-013, FR-025).
- CI: `GO_VERSION` diselaraskan dengan directive `go` di `backend/go.mod`
  (1.25.0), sehingga runner tak lagi mengunduh toolchain terpisah tiap run.
  `actions/setup-go` diberi `cache-dependency-path: backend/go.sum` supaya cache
  modul menemukan berkas checksum yang ada di subdirektori `backend/`, bukan di
  root repository.


