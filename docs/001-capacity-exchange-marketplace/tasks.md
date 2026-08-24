# Tasks: Capacity Exchange, Devotion

**Input**: `docs/001-capacity-exchange-marketplace/`
**Last Revised**: 2026-08-22
**Prerequisites**: `spec.md` (91 FR), `plan.md`, `research.md`, `data-model.md`, `contracts/openapi.yaml` (operasi lihat `contracts/README.md`), `quickstart.md`, `docs/memory/constitution.md` v2.1.0

**Tests**: DIWAJIBKAN. Konstitusi v2.1.0 menetapkan pengujian otomatis sebagai gerbang mutu, bukan pilihan.

**Organisasi**: per user story, agar setiap story dapat diimplementasikan, diuji, dan didemokan sebagai tambahan nilai yang berdiri sendiri.

## Format

```text
- [ ] T0XX [P?] [Story?] [Lane] Judul singkat
  **Modul**: direktori tempat pekerjaan ini berada
  **FR**: requirement yang dilayani
  **Kemampuan**: apa yang harus bisa dilakukan setelah task ini selesai
  **Dependency**: paket baru yang dipakai dan task prasyarat (T0XX), atau "tidak ada"
  **Selesai bila**: kriteria yang bisa diperiksa, bukan dinilai
  **Saran**: usulan pemecahan, boleh diabaikan bila ada cara lebih baik
  **Hati-hati**: hal yang mudah salah dan mahal diperbaiki belakangan
```

- **[P]** = boleh paralel: modulnya berbeda dan tidak saling menunggu
- **[Story]** = US1–US7, hanya pada fase story
- **[Lane]** = jalur kerja yang menentukan branch: `[BE]` backend (`backend/*`), `[FE]` frontend (`frontend/*`), `[OPS]` setup, infrastruktur, dan dokumen (branch setup tersendiri). Urutan tag selalu `[P] [Story] [Lane]`.
- Pemecahan file di dalam modul diserahkan pelaksana, **kecuali** empat path yang dipatok

## Empat Path yang Dipatok

| Path | Alasan |
|------|--------|
| `backend/internal/platform/clock.go` | Disuntikkan ke seluruh service; Prinsip V |
| `backend/db/migrations/` | Urutan sudah ditetapkan `data-model.md` §12 |
| `backend/webdist/` | Target build frontend, dirujuk `embed.FS` |
| `docker-compose.yml` | Gate I dihitung dari jumlah entri `services:` |

## Istilah yang Dipakai Lintas Task

Diambil dari `spec.md` bagian Istilah yang Mengikat. Salah memahami keduanya berarti salah mengimplementasikan pencarian dan alokasi:

- **Minggu kesiapan mulai** = periode mingguan yang memuat tanggal acuan + `jeda_kesiapan_hari`. Tanggal acuan: tanggal kesepakatan pada pesanan, tanggal pencarian pada perhitungan kandidat. Ini periode paling awal yang boleh dihitung maupun dialokasikan.
- **Rentang kapasitas** = periode mingguan dari minggu kesiapan mulai sampai periode yang memuat deadline, inklusif. Seluruh penjumlahan dan alokasi memakai rentang ini, **bukan** dari minggu berjalan.

---

## Phase 1: Setup

**Tujuan**: repository dapat dibangun dan dijalankan, meski belum ada fitur.

- [ ] T001 [OPS] Struktur repository dan berkas tingkat atas
  **Modul**: root
  **Kemampuan**: `README.md` (template panitia, struktur tidak diubah), `LICENSE` MIT, `CLAUDE.md`, `.gitignore`, `.env.example` berisi nama variabel tanpa nilai, direktori `backend/`, `frontend/`, `docs/`
  **Dependency**: tidak ada
  **Selesai bila**: tidak ada direktori tingkat atas di luar daftar konstitusi; `.env` masuk `.gitignore`
  **Hati-hati**: `.env.example` tidak boleh memuat satu pun nilai sungguhan. Repository ini publik. `CLAUDE.md` di root, bukan di `docs/`, agar terbaca agent.

- [x] T002 [P] [BE] Inisialisasi modul Go dan subcommand
  **Modul**: `backend/cmd/devotion/`
  **Kemampuan**: `go.mod` dengan Go 1.22+, dispatcher subcommand: `serve`, `admin:create`, `seed:regions`, `seed:master-data`, `seed:test-data`, `reset:test-data`, `user:verify`, `health:check`
  **Dependency**: tidak ada, `flag.NewFlagSet` dari standard library; prasyarat T001
  **Selesai bila**: `go run ./cmd/devotion` menampilkan daftar subcommand; `go vet ./...` bersih
  **Saran**: satu berkas per subcommand, dispatcher tipis di `main.go`. Subcommand adalah proses sekali jalan, bukan proses runtime, jadi tidak melanggar Gate I.

- [ ] T003 [P] [FE] Inisialisasi frontend
  **Modul**: `frontend/`
  **Kemampuan**: Vite + React 18 + TypeScript + Tailwind, Jest, struktur `src/{pages,components,api,schemas,lib}`
  **Dependency**: sesuai `plan.md` Primary Dependencies. Jangan tambah di luar daftar itu; prasyarat T001
  **Selesai bila**: `npm run build` menghasilkan `dist/`; `npm test` jalan meski belum ada test
  **Hati-hati**: Vite, bukan Next.js. Next.js akan menggoda menaruh API route di frontend, dan itu backend kedua, pelanggaran Gate I yang paling mudah terjadi tanpa disadari.

- [ ] T004 [P] [FE] Generator tipe dari OpenAPI
  **Modul**: `frontend/src/api/`
  **Kemampuan**: skrip npm menghasilkan tipe TypeScript dari `docs/001-capacity-exchange-marketplace/contracts/openapi.yaml`
  **Dependency**: `openapi-typescript` (devDependency); prasyarat T003
  **Selesai bila**: tipe ter-generate dan dapat diimpor; skrip terdokumentasi di `docs/menjalankan.md`
  **Hati-hati**: jangan pernah menulis tipe respons dengan tangan. Yang ditulis tangan akan menyimpang dari kontrak tanpa ada yang tahu.

- [ ] T005 [P] [OPS] Compose dua layanan
  **Modul**: `docker-compose.yml` (path dipatok)
  **Kemampuan**: tepat dua layanan `backend` dan `postgres`, penyetelan Postgres untuk 2GB sesuai `research.md` R-03, batas log `max-size 10m` `max-file 3` pada keduanya, volume `pgdata` dan bind `/opt/devotion/unggahan`
  **Dependency**: tidak ada; prasyarat T001
  **Selesai bila**: `docker compose config` valid; jumlah entri di bawah `services:` tepat dua
  **Hati-hati**: batas log bukan kebersihan. Log tanpa batas mengisi 50GB, lalu Postgres berhenti menulis dan aplikasi mati total.

- [ ] T006 [P] [OPS] Kerangka dokumentasi dan changelog
  **Modul**: `docs/` + `backend/` + `frontend/`
  **Kemampuan**: di `docs/`, terdiri dari `menjalankan.md`, `pengujian.md`, `dependencies.md`, `utang-teknis.md`, `layanan-luar.md`, `temuan-penguji.md`, `cloudflare-ips.md`, `setup-vps.md`, `skenario-uji-manual.md`, dan direktori `master-data/`. Changelog **terpisah per bagian**: `backend/CHANGELOG.md` dan `frontend/CHANGELOG.md`
  **Dependency**: tidak ada; prasyarat T001
  **Selesai bila**: seluruh berkas ada dengan judul dan kerangka bagian; tidak ada `docs/changelog.md`
  **Saran**: kedua changelog diisi setiap kali sebuah story ditutup di checkpoint, bukan direkonstruksi di akhir. `layanan-luar.md` mencatat Cloudflare, Mailjet, Sentry, pemantau uptime, dan wilayah.id beserta akibat bila masing-masing mati.

- [ ] T007 [OPS] CI pipeline
  **Modul**: `.github/workflows/`
  **Kemampuan**: `go vet`, `go test`, `npm test`, `npm run build`, salin `dist` → `backend/webdist/`, build image multi-stage, push ke GHCR, deploy via SSH
  **Dependency**: prasyarat T002, T003, T004 (menjalankan vet dan test Go, test dan build frontend, lalu menyalin `dist` ke `webdist`)
  **Selesai bila**: pipeline hijau pada commit kosong; image terbit dengan tag `<sha>` dan `latest`
  **Hati-hati**: membangun artefak di server dilarang konstitusi. Build Vite pada 2GB sambil Postgres hidup akan kehabisan memori, dan yang dibunuh kernel biasanya Postgres.

**Checkpoint**: repository dapat dibangun, image terbit, compose valid.

---

## Phase 2: Foundational (Blocking)

**⚠️ Tidak ada pekerjaan user story yang boleh dimulai sebelum fase ini selesai.**

- [x] T008 [BE] Clock yang dapat digantikan
  **Modul**: `backend/internal/platform/clock.go` (path dipatok)
  **FR**: Prinsip V
  **Kemampuan**: interface `Clock` dengan `Now()`, implementasi nyata, implementasi uji yang waktunya dapat disetel dan digeser
  **Dependency**: tidak ada; prasyarat T007
  **Selesai bila**: ada test yang membuktikan waktu dapat digeser
  **Hati-hati**: dikerjakan **sekarang**, bukan menyusul. Menambahkannya setelah service jadi berarti menyentuh seluruh service. `time.Now()` dilarang muncul di dalam logika bisnis mana pun. `data-model.md` juga melarang `DEFAULT now()` pada seluruh tabel, setiap `INSERT` mengirim waktunya dari `Clock`.

- [x] T009 [BE] Konfigurasi dan bootstrap
  **Modul**: `backend/internal/platform/config/`
  **Kemampuan**: memuat variabel lingkungan, memvalidasi yang wajib saat startup, membedakan `APP_ENV=development` dan `production`
  **Dependency**: tidak ada, `os.Getenv` cukup; prasyarat T007
  **Selesai bila**: variabel wajib yang hilang menghentikan startup dengan pesan yang menyebut nama variabelnya
  **Saran**: pada `development`, backend melayani HTTP biasa tanpa TLS dan tanpa pemeriksaan sertifikat klien Cloudflare.

- [x] T010 [BE] Migrasi basis data
  **Modul**: `backend/db/migrations/` (path dipatok)
  **FR**: seluruh entitas
  **Kemampuan**: migrasi berurutan sesuai `data-model.md` §12, dijalankan otomatis saat startup dengan `pg_try_advisory_lock`
  **Dependency**: `golang-migrate` (versi dipatok); prasyarat T007, T009
  **Selesai bila**: `docker compose up` menjalankan migrasi sampai versi terakhir; `schema_migrations.dirty = false`; menjalankan dua kali tidak menimbulkan galat
  **Hati-hati**: seluruh constraint, indeks, dan **tiga trigger** wajib ikut: `used_capacity_within_total`, `week_start_is_monday`, `readiness_not_past_deadline`, `city_belongs_to_province`, trigger jenis item, trigger cegah request ke diri sendiri, trigger cegah alokasi sebelum kesiapan. Constraint itu bukan hiasan: ia yang menahan kerusakan data ketika logika aplikasi keliru. **Tidak ada `DEFAULT now()` di satu pun tabel.**

- [x] T011 [BE] Lapisan akses data
  **Modul**: `backend/db/queries/` + konfigurasi `sqlc`
  **Kemampuan**: `sqlc.yaml`, pool `pgx` dengan `MaxConns=15`, helper transaksi
  **Dependency**: `pgx/v5`, `sqlc` (perkakas build); prasyarat T007, T009, T010
  **Selesai bila**: `sqlc generate` berhasil; pool tersambung; helper transaksi punya test
  **Hati-hati**: pool 15 dari `max_connections` 20, lima disisakan untuk `pg_dump`, `psql`, dan migrasi. Tanpa sisa itu, cadangan harian gagal justru saat trafik tinggi.

- [x] T012 [P] [BE] Lapisan HTTP dan format galat
  **Modul**: `backend/internal/platform/httpx/`
  **FR**: seluruh endpoint
  **Kemampuan**: router `net/http`, middleware request ID, pemulihan panic, `application/problem+json` dengan kode galat dari `openapi.yaml` (jumlah di `contracts/README.md`), log `slog` JSON dengan request ID di setiap baris
  **Dependency**: tidak ada, `net/http` dan `log/slog` standard library; prasyarat T007, T009
  **Selesai bila**: galat validasi mengembalikan bentuk `ProblemValidasi` beserta daftar field; setiap baris log memuat request ID
  **Hati-hati**: `/api/*` yang tidak dikenali mengembalikan 404 JSON, **bukan** `index.html`. Kalau HTML, kesalahan penulisan alamat endpoint jadi menyesatkan saat diagnosis.

- [x] T013 [P] [BE] Kepercayaan alamat asal
  **Modul**: `backend/internal/platform/cloudflare/`
  **Kemampuan**: rentang alamat Cloudflare dipatok sebagai konstanta beserta tanggal pengambilan, fungsi `RealIP` yang hanya mempercayai header bila koneksi datang dari rentang itu
  **Dependency**: tidak ada; prasyarat T007
  **Selesai bila**: test membuktikan header diabaikan pada koneksi di luar rentang; `docs/cloudflare-ips.md` memuat daftar beserta tanggal
  **Hati-hati**: daftar rentang di `research.md` R-01 ditulis dari ingatan dan **wajib dicocokkan** ke `cloudflare.com/ips-v4` dan `ips-v6` sebelum dipatok. Jangan mengambilnya lewat jaringan saat startup, satu kegagalan HTTP akan membuat aplikasi gagal menyala.

- [x] T014 [BE] Sesi, autentikasi, dan akun
  **Modul**: `backend/internal/platform/session/` + `backend/internal/account/`
  **FR**: FR-001, FR-002, FR-003, FR-005
  **Kemampuan**: registrasi, verifikasi kode enam digit untuk email dan nomor, masuk, keluar, pemulihan kata sandi, cookie `httpOnly Secure SameSite=Lax`, hash token di basis data. **Ditambah**: `GET /me` (akun yang sedang masuk dalam bentuk `MyAccount`: `roles{subcontractor,buyer}`, `profile_id`, `is_admin`) dan `PATCH /me/roles` (menambah peran usaha, menolak pencabutan bila masih ada pesanan aktif)
  **Dependency**: `bcrypt` cost 10; prasyarat T010, T011, T012, T016
  **Selesai bila**: seluruh endpoint `/auth/*`, `/me`, dan `/me/roles` sesuai `openapi.yaml`; keluar akun benar-benar mengakhiri sesi; test membuktikan yang disimpan adalah hash, bukan token mentah
  **Hati-hati**: `POST /auth/recover/request` selalu 202, agar tidak membocorkan apakah sebuah email terdaftar. Dua endpoint `/me` sebelumnya tidak punya task pemilik, itu celah yang ditemukan `/analyze`.

- [x] T015 [BE] Middleware peran
  **Modul**: `backend/internal/platform/httpx/`
  **FR**: FR-005
  **Kemampuan**: pemeriksaan peran per endpoint; satu akun boleh memegang dua peran usaha; admin terpisah dari peran usaha
  **Dependency**: prasyarat T012, T014
  **Selesai bila**: test membuktikan penolakan untuk setiap kombinasi peran yang tidak berwenang
  **Hati-hati**: endpoint tanpa pemeriksaan peran dianggap **cacat**, bukan belum lengkap. Ini gerbang yang diperiksa di setiap story.

- [x] T016 [P] [BE] Pembatasan laju berbasis data domain
  **Modul**: `backend/internal/platform/ratelimit/`
  **Kemampuan**: batas per akun untuk percobaan masuk (5/15 menit), per nomor untuk kode sekali pakai (3/jam), per alamat asal untuk kode (10 nomor/jam), per pengguna untuk request kuota (20/jam)
  **Dependency**: tidak ada, tabel Postgres; prasyarat T011, T013
  **Selesai bila**: keempat batas punya test; respons 429 menyertakan `Retry-After`
  **Hati-hati**: tabel, bukan penyimpanan dalam memori. Kalau di memori, penerapan versi baru jadi cara termudah melewatinya. Batas per alamat asal yang menutup pemutaran nomor, itu yang melindungi nomor WhatsApp dari pemblokiran.

- [x] T017 [P] [BE] Penyimpanan berkas
  **Modul**: `backend/internal/platform/storage/`
  **FR**: FR-006, FR-009
  **Kemampuan**: unggah maksimal 5MB, nama berkas UUID dibuat sistem, tipe divalidasi dari magic bytes, metadata lokasi gambar dibuang, kuota total 500MB, akses hanya lewat handler yang memeriksa peran
  **Dependency**: tidak ada, `image/jpeg` dan `image/png` dekode-enkode ulang membuang EXIF; prasyarat T011, T015
  **Selesai bila**: bukan pemilik dan bukan admin ditolak (test wajib); berkas dengan ekstensi menipu ditolak; kuota penuh mengembalikan pesan yang jelas
  **Hati-hati**: **jangan pernah** melayani berkas lewat path statis. Foto lokasi usaha dari ponsel membawa koordinat GPS, dan banyak konveksi rumahan berarti itu alamat rumah orang. Dokumen sumber memitigasi risiko kebocoran data identitas dengan enkripsi AES-256, akses RBAC, dan penetration test kuartalan [1]; versi ini hanya menegakkan kontrol akses, sehingga pengujiannya wajib.

- [x] T018 [BE] Penjadwal dua lapisan
  **Modul**: `backend/internal/platform/scheduler/`
  **FR**: FR-021, FR-037, FR-045, FR-068, FR-069
  **Kemampuan**: `time.Ticker` 5 menit di dalam proses yang sama, setiap pekerjaan dibungkus advisory lock
  **Dependency**: tidak ada; prasyarat T008, T011
  **Selesai bila**: penjadwal menyala saat startup dan tercatat di log; test membuktikan pekerjaan tidak berjalan ganda
  **Hati-hati**: perhitungan tenggat ditulis **satu kali** di satu fungsi domain, dipakai bersama lapisan hitung-saat-baca. Kalau diduplikasi, pesanan yang sama akan tampak berbeda status di halaman berbeda. Bukan proses kedua, Gate I.

- [x] T019 [BE] Data acuan: wilayah dan daftar baku
  **Modul**: `backend/internal/masterdata/` + subcommand seed
  **FR**: FR-058, FR-062, FR-075
  **Kemampuan**: `seed:regions` mengambil provinsi dan kabupaten/kota dari wilayah.id dengan flag `--refresh`, default membaca salinan `docs/master-data/regions.json`; `seed:master-data` mengisi jenis produk dan mesin; keduanya idempoten memakai kode sebagai identitas. **Ditambah**: endpoint baca `GET /master/products`, `GET /master/machines`, `GET /regions/provinces`, `GET /regions/cities?province=`; `Normalisasi kode kabupaten/kota dengan membuang titik (32.73 → 3273) sebelum menyimpan.`
  **Dependency**: tidak ada, `net/http` dan `encoding/json`; prasyarat T010, T011, T012
  **Selesai bila**: kedua perintah jalan dua kali tanpa menduplikasi; hitungan provinsi, kota, produk, mesin semuanya lebih dari nol; keempat endpoint sesuai kontrak
  **Hati-hati**: Bentuk respons sudah diverifikasi; lihat docs/master-data/README.md. Normalisasi kode adalah langkah yang paling mudah terlupakan dan gagalnya senyap sampai constraint menolak seluruh baris. Jalankan kueri verifikasi di README itu setelah seed.

- [x] T020 [BE] Admin pertama
  **Modul**: `backend/cmd/devotion/`
  **Kemampuan**: `admin:create` meminta kata sandi lewat prompt tanpa menampilkan ketikan, idempoten
  **Dependency**: prasyarat T014
  **Selesai bila**: admin dapat masuk; menjalankan dua kali tidak membuat admin ganda
  **Hati-hati**: kata sandi lewat prompt, bukan argumen, karena argumen tersimpan di riwayat shell.

- [ ] T021 [FE] Frontend: shell aplikasi
  **Modul**: `frontend/src/`
  **FR**: FR-055, FR-056
  **Kemampuan**: layout mobile-first bahasa Indonesia, routing, klien API dengan `credentials: 'include'`, TanStack Query, penanganan galat yang menampilkan `detail` dari respons, halaman masuk dan daftar
  **Dependency**: prasyarat T004, T014
  **Selesai bila**: dapat masuk dan keluar lewat antarmuka; alur inti dapat diselesaikan dengan keyboard
  **Hati-hati**: token **tidak pernah** disimpan di `localStorage` maupun `sessionStorage`. Satu celah XSS akan langsung berarti pengambilalihan akun, dan aplikasi ini memuat dokumen identitas.

- [x] T022 [BE] Penyajian frontend oleh backend
  **Modul**: `backend/internal/platform/httpx/` + `backend/webdist/` (path dipatok)
  **Kemampuan**: `embed.FS` menyematkan hasil build, berkas statis ber-hash dengan cache panjang, fallback `index.html` untuk path non-API, TLS dengan Cloudflare Origin Certificate dan verifikasi sertifikat klien
  **Dependency**: prasyarat T012, T021
  **Selesai bila**: penyegaran pada halaman dalam tidak menghasilkan 404; `/api/*` tak dikenal mengembalikan JSON
  **Hati-hati**: ini yang membuat Gate I terpenuhi tanpa layanan frontend maupun proxy. Pastikan Cloudflare tidak meng-cache `/api/*`, hasil pencarian ter-cache menampilkan kapasitas basi, persis masalah informasi tidak aktual yang platform ini dibangun untuk menyelesaikan.

- [x] T023 [BE] Notifikasi
  **Modul**: `backend/internal/notification/`
  **FR**: FR-051 sampai FR-054, FR-074, FR-085, FR-086, FR-091
  **Kemampuan**: baris notifikasi ditulis di dalam transaksi kejadiannya beserta kolom `transactional`; goroutine pengirim ke email dan WhatsApp maksimal 3 percobaan; notifikasi di dalam platform selalu tampil; preferensi kanal hanya berlaku bagi kejadian non-transaksional
  **Dependency**: `net/smtp` (standard library, Mailjet lewat SMTP), `whatsmeow`; prasyarat T010, T011, T018
  **Selesai bila**: test membuktikan kegagalan kirim tidak menggagalkan transaksi pemicunya; setelah 3 percobaan ditandai gagal permanen; endpoint `/notifications` dan `/notifications/preferences` sesuai kontrak
  **Hati-hati**: penggolongan transaksional versus non-transaksional ada di `data-model.md` §9, hanya `calendar_stale`, `deadline_approaching`, dan `rating_request` yang non-transaksional. Perhatikan bahwa `confirmation_due_approaching` **transaksional** karena berujung pada penutupan pesanan otomatis, jadi tidak boleh dapat dimatikan. Notifikasi di dalam platform adalah satu-satunya jalur pengamatan bagi penguji manual.

- [x] T024a [BE] Sambungan WhatsApp dan endpoint QR
  **Modul**: `backend/internal/admin/`
  **FR**: FR-002, FR-052
  **Kemampuan**: kelola sesi `whatsmeow`, endpoint yang menyajikan QR dan status sambungan, penyambungan ulang tanpa akses server; `user:verify --phone` sebagai jalan darurat; status WhatsApp masuk ke endpoint health
  **Dependency**: `whatsmeow`; prasyarat T015, T023
  **Selesai bila**: endpoint QR dan status hanya dapat diakses admin; endpoint health menyertakan status WhatsApp; `user:verify --phone` memverifikasi nomor tanpa antarmuka
  **Hati-hati**: FR-002 menjadikan verifikasi nomor sebagai gerbang pendaftaran, jadi sesi yang lepas berarti tidak ada akun baru yang bisa dibuat. Nomor layanan hanya dari variabel lingkungan.

- [ ] T024b [FE] Halaman admin WhatsApp
  **Modul**: `frontend/src/pages/admin/`
  **FR**: FR-002, FR-052
  **Dependency**: T024a
  **Kemampuan**: menampilkan QR dan status sambungan dari endpoint T024a, tombol sambung ulang tanpa akses server
  **Selesai bila**: QR dapat dipindai lewat antarmuka; status tersambung terlihat
  **Hati-hati**: halaman ini yang mencegah kehilangan demo saat sesi WhatsApp lepas. Render dari endpoint, jangan menyimpan status di sisi klien.

- [x] T025 [P] [BE] Health check dan Sentry
  **Modul**: `backend/internal/platform/`
  **Kemampuan**: `GET /health` memeriksa basis data, WhatsApp, dan kuota penyimpanan; Sentry dengan pembersihan data sensitif
  **Dependency**: `sentry-go`; prasyarat T011, T012, T024a
  **Selesai bila**: health mengembalikan 503 bila ada ketergantungan gagal; test membuktikan kata sandi, token, nomor telepon, dan hal terkait dokumen identitas tidak terkirim ke Sentry

**Checkpoint**: fondasi siap. Seluruh user story boleh dimulai, dan bila ada dua pelaksana, boleh paralel.

---

## Phase 3: User Story 1, Listing Kapasitas (P1) 🎯 MVP

**Goal**: subkontraktor dapat mendaftarkan kapasitas produksinya dan listing itu langsung dapat ditemukan pihak lain.

**Independent Test**: daftar sebagai subkontraktor, buat listing lengkap, buka halaman publiknya sebagai pengunjung lain, seluruh atribut kapasitas tampil benar.

- [x] T026 [P] [US1] [BE] Profil usaha
  **Modul**: `backend/internal/account/`
  **FR**: FR-004, FR-057
  **Kemampuan**: baca dan ubah profil sendiri, profil publik, kota dari data wilayah, koordinat opsional
  **Dependency**: prasyarat T014, T017, T019
  **Selesai bila**: `/profile/me` dan `/profile/{profileId}` sesuai kontrak; koordinat di luar Indonesia ditolak; lintang tanpa bujur ditolak

- [x] T027 [US1] [BE] Listing kapasitas
  **Modul**: `backend/internal/listing/`
  **FR**: FR-012, FR-013, FR-014, FR-015, FR-076
  **Kemampuan**: buat, ubah, nonaktifkan, aktifkan kembali; satu angka kapasitas mingguan untuk seluruh jenis produk; jeda kesiapan mulai dalam hari
  **Dependency**: prasyarat T015, T019, T026
  **Selesai bila**: `/listing/me` sesuai kontrak; atribut wajib kosong ditolak dengan menyebut kolomnya
  **Saran**: pisahkan service dan handler; validasi di service supaya dapat dites tanpa HTTP
  **Hati-hati**: **jangan** buat kolom kapasitas per jenis produk (FR-076). Mesin dan tenaga kerjanya berbagi, sehingga angka terpisah akan mengizinkan penyanggupan ganda pada minggu yang sama. **Jeda kesiapan mulai bukan durasi menyelesaikan pekerjaan**, melainkan jeda sebelum produksi dapat dimulai.

- [x] T028 [US1] [BE] Kalender awal dan horizon
  **Modul**: `backend/internal/listing/`
  **FR**: FR-017, FR-088
  **Kemampuan**: periode mingguan dibuat otomatis untuk minimal 3 bulan ke depan saat listing dibuat, memakai kapasitas mingguan sebagai kapasitas total; kolom `horizon_until` menyimpan periode terjauh yang sudah dibuat; fungsi memperpanjang horizon sampai minggu tertentu, idempoten dan dapat dipanggil ulang
  **Dependency**: prasyarat T008, T027
  **Selesai bila**: setiap `week_start` adalah hari Senin; horizon awal minimal 13 periode; `horizon_until` konsisten dengan `MAX(week_start)`; memperpanjang ke minggu yang sudah tercakup tidak membuat baris ganda
  **Hati-hati**: horizon **bukan** nilai tetap. T035 akan memanggil fungsi perpanjangan ini ketika ada pencarian berdeadline lebih jauh (FR-088), jadi rancang sebagai operasi yang aman dipanggil berulang dan aman dipanggil bersamaan. Batas minggu dihitung di Asia/Jakarta, disimpan sebagai `date`; constraint `week_start_is_monday` akan menolak bila perhitungannya keliru.

- [x] T029 [P] [US1] [BE] Usulan item daftar baku
  **Modul**: `backend/internal/masterdata/`
  **FR**: FR-061
  **Kemampuan**: pengguna mengusulkan item baru, listing tetap dapat disimpan dengan item yang tersedia
  **Dependency**: prasyarat T019, T023
  **Selesai bila**: `POST /master/proposals` sesuai kontrak; pengusul menerima notifikasi saat diputuskan

- [x] T030 [P] [US1] [BE] Test backend US1
  **Modul**: `backend/internal/{account,listing}/`
  **Kemampuan**: jalur berhasil, penolakan peran, penolakan masukan tidak sah untuk setiap endpoint; listing tayang tanpa verifikasi; horizon awal terbentuk benar dan perpanjangan idempoten
  **Dependency**: prasyarat T026, T027, T028, T029
  **Selesai bila**: setiap nama test menyebut FR yang diuji; seluruhnya lulus
  **Hati-hati**: uji juga bahwa listing tetap tayang tanpa pengajuan verifikasi. Dokumen sumber justru menempatkan status "Menunggu Verifikasi" pada alur listing [1]; spec kita sengaja menyimpang, dan test inilah yang mengunci keputusan itu.

- [ ] T031 [P] [US1] [FE] Frontend: pendaftaran dan verifikasi
  **Modul**: `frontend/src/pages/auth/`
  **FR**: FR-001, FR-002
  **Kemampuan**: form daftar dengan pilihan peran, pemilih kota, halaman kode verifikasi, tombol kirim ulang dengan jeda membesar
  **Dependency**: prasyarat T014, T021
  **Selesai bila**: pesan yang muncul saat mencoba membuat listing sebelum verifikasi menjelaskan apa yang harus dilakukan

- [ ] T032 [US1] [FE] Frontend: form listing
  **Modul**: `frontend/src/pages/listing/`
  **FR**: FR-012, FR-013, FR-076
  **Kemampuan**: pemilih jenis produk dan mesin dari daftar baku, satu kolom kapasitas mingguan, kolom jeda kesiapan mulai dengan penjelasan artinya, tautan mengusulkan item baru
  **Dependency**: Zod + React Hook Form; prasyarat T021, T027
  **Selesai bila**: tidak ada kolom teks bebas untuk jenis produk dan mesin; tidak ada kolom kapasitas per produk; galat per kolom tampil dari respons backend
  **Saran**: beri label yang jelas pada jeda kesiapan mulai, misalnya "berapa hari setelah kesepakatan Anda bisa mulai produksi", karena pengguna mudah salah mengira ini lama pengerjaan.

- [ ] T033 [US1] [FE] Frontend: profil publik
  **Modul**: `frontend/src/pages/profile/`
  **FR**: FR-016, FR-064
  **Kemampuan**: atribut listing, ketersediaan terkini, peta lokasi, ringkasan reputasi, lencana verifikasi
  **Dependency**: Leaflet + tile OpenStreetMap; prasyarat T021, T026
  **Selesai bila**: dapat dibuka tanpa masuk; peta tampil tanpa kunci API

- [ ] T034 [US1] [OPS] Skenario uji manual US1
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 10 langkah dari `quickstart.md` bagian F US1, dengan kolom temuan
  **Dependency**: prasyarat T007, T030, T031, T032, T033
  **Selesai bila**: setiap langkah menyebut akun yang dipakai; salah tulis "Bu... maksudnya Pak Budi" pada langkah 1.8 sudah diperbaiki

**Checkpoint**: US1 berfungsi dan dapat didemokan sendiri. Isi `backend/CHANGELOG.md` dan `frontend/CHANGELOG.md`.

---

## Phase 4: User Story 2, Pencarian (P2)

**Goal**: pemberi order menemukan subkontraktor yang cocok, dengan kapasitas dijumlahkan lintas periode di dalam rentang kapasitas.

**Independent Test**: dengan listing yang sudah tayang, cari 3.000 potong dengan deadline delapan minggu; kandidat berkapasitas 500 per minggu berjeda 0 hari ikut muncul dan ditandai memenuhi kriteria kapasitas.

- [x] T035 [US2] [BE] Mesin pencarian
  **Modul**: `backend/internal/search/`
  **FR**: FR-022 sampai FR-028, FR-063, FR-080, FR-081, FR-087, FR-088
  **Kemampuan**:
  - Hitung **minggu kesiapan mulai** per kandidat = Senin dari (tanggal pencarian + `jeda_kesiapan_hari` listing). Ini batas awal **rentang kapasitas**; batas akhirnya periode yang memuat deadline.
  - Jumlahkan kapasitas tersisa hanya di dalam rentang itu. Kandidat yang minggu kesiapannya melampaui minggu deadline memiliki rentang kosong sehingga kapasitasnya nol.
  - Perpanjang horizon: bila `horizon_until < minggu_deadline`, hitung minggu yang belum dibuat sebagai berkapasitas penuh, lalu panggil fungsi perpanjangan T028 untuk kandidat yang lolos, di dalam transaksi tersendiri, **di luar** kueri pencarian.
  - Empat kriteria keras sebagai empat nilai boolean yang dijumlahkan; **kriteria yang filternya tidak diisi dihitung terpenuhi** dan responsnya menyebutkan kriteria mana yang tidak dievaluasi.
  - Pemecah seri lima tingkat, keyset pagination, perluasan wilayah kota → provinsi → nasional, saran pelonggaran saat kosong, pengecualian listing sendiri.
  **Selesai bila**: `GET /search` sesuai kontrak; bentuk kueri mengikuti `data-model.md` §10; skor tetap bernilai 0–4
  **Dependency**: prasyarat T027, T028
  **Hati-hati**: **tidak ada pembobotan dan tidak ada normalisasi skor.** Rating, tingkat penyelesaian, verifikasi, kebaruan kalender, jarak, dan tanggal pendaftaran tidak boleh mempengaruhi urutan (FR-024), termasuk kebaruan kalender, meski dokumen sumber justru menyarankan penalti penurunan skor pencarian bagi yang tidak update kalender [1]. `listing_id` sebagai pemecah seri terakhir wajib ada; tanpanya urutan bisa bertukar antar permintaan. Pencarian tetap operasi baca: perpanjangan horizon jangan diletakkan di dalam kueri, karena itu akan memicu penulisan pada setiap permintaan.

- [x] T036 [P] [US2] [BE] Test determinisme dan rentang kapasitas
  **Modul**: `backend/internal/search/`
  **FR**: FR-023, FR-024, FR-025, FR-080, FR-087, FR-088, SC-013, SC-019, SC-020, SC-021
  **Kemampuan**:
  - Urutan identik pada pengulangan; stabil antar halaman meski ada listing baru disisipkan (SC-013).
  - Skor tidak berubah saat rating, verifikasi, atau kebaruan kalender diubah (FR-024).
  - Skenario 3.000 potong pada kapasitas 500/minggu jeda 0 hari dengan deadline delapan minggu **lolos**; dengan deadline empat minggu **tidak lolos** (SC-019).
  - Kandidat berjeda kesiapan 14 hari: dua minggu pertama **tidak** ikut dijumlahkan, sehingga total kapasitasnya lebih kecil dari kandidat berjeda 0 hari pada deadline yang sama (SC-020).
  - Kandidat berjeda kesiapan yang minggu kesiapannya melampaui deadline: kapasitas nol, kriteria (d) tidak terpenuhi.
  - Pencarian 3.000 potong pada kapasitas 200/minggu dengan deadline lima bulan (di luar horizon awal) tetap dinilai berdasarkan kapasitas penuh sampai deadline, dan periode yang kurang benar-benar terbentuk setelahnya (SC-021).
  - Filter mesin dikosongkan: kriteria mesin terpenuhi bagi semua kandidat, dan respons menyebutkan kriteria itu tidak dievaluasi.
  - Listing sendiri dikecualikan (FR-081).
  **Selesai bila**: seluruh test lulus dan namanya menyebut FR atau SC; test bertenggat memakai `Clock` yang digeser
  **Dependency**: prasyarat T035
  **Hati-hati**: ini kelompok test terpenting di seluruh project. Empat di antaranya, yaitu SC-020, SC-021, dan dua kasus jeda kesiapan, menutup bug yang **tidak akan** tertangkap pengujian manual karena angka totalnya tetap tampak masuk akal.

- [ ] T037 [P] [US2] [FE] Frontend: halaman pencarian
  **Modul**: `frontend/src/pages/search/`
  **FR**: FR-022, FR-026, FR-027, FR-028, FR-063, FR-080
  **Kemampuan**: filter produk, mesin, wilayah, jumlah, deadline, jeda maksimal; kartu hasil menampilkan seluruh atribut keputusan termasuk minggu kesiapan mulai dan total kapasitas sampai deadline; penjelasan kriteria yang tidak terpenuhi dan yang tidak dievaluasi; tombol perluas yang menyebut tingkat berikutnya; keadaan kosong beserta saran
  **Selesai bila**: kursor diteruskan apa adanya; tidak ada kandidat ganda saat berpindah halaman
  **Dependency**: prasyarat T021, T035
  **Hati-hati**: kursor bersifat opaque. Jangan diurai, jangan diubah jadi `?page=2`, karena itu langsung melanggar jaminan urutan stabil. Tampilkan juga kriteria yang tidak dievaluasi, jangan hanya yang gagal, supaya pengguna paham kenapa banyak kandidat berskor sama.

- [ ] T038 [US2] [OPS] Skenario uji manual US2
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 13 langkah dari `quickstart.md` bagian F US2, termasuk langkah membandingkan kandidat berjeda 0 hari dengan berjeda 21 hari
  **Dependency**: prasyarat T007, T036, T037

**Checkpoint**: matching berfungsi dan dapat dijelaskan ke pengguna. Isi kedua changelog.

---

## Phase 5: User Story 3, Request Kuota dan Negosiasi (P3)

**Goal**: pemberi order mengirim satu request ke beberapa kandidat, membandingkan penawaran, dan menyepakati satu.

**Independent Test**: kirim ke tiga kandidat, balas dari dua dengan harga berbeda, bandingkan, terima satu.

- [x] T039 [US3] [BE] Request kuota
  **Modul**: `backend/internal/quota/`
  **FR**: FR-029, FR-030, FR-082, FR-083
  **Kemampuan**: kirim ke beberapa kandidat dalam satu tindakan; status per kandidat; batas balasan 72 jam ditetapkan sistem dari `Clock`; penolakan request ke listing sendiri
  **Selesai bila**: `/quota-requests` sesuai kontrak; tidak ada kolom untuk mengatur batas waktu sendiri
  **Dependency**: prasyarat T008, T015, T023, T027
  **Hati-hati**: aplikasi mengirim **kedua** nilai `dibuat_pada` dan `batas_balasan_pada` dari `Clock`; basis data tidak punya `DEFAULT now()` dan constraint-nya hanya menjaga urutan. Angka 72 jam ditegakkan aplikasi dan diuji. FR-083 menyebut jalur "tanpa melalui hasil pencarian" secara eksplisit; trigger basis data adalah jaring pengamannya.

- [x] T040 [US3] [BE] Penawaran dan negosiasi
  **Modul**: `backend/internal/quota/`
  **FR**: FR-031, FR-032, FR-033, FR-035, FR-090
  **Kemampuan**: balas dengan harga dan jeda kesiapan, tolak beralasan, counter-offer berantai dengan riwayat lengkap, perbandingan berdampingan; penolakan bila jumlah melampaui total kapasitas di dalam rentang kapasitas; penolakan bila minggu kesiapan mulai kandidat jatuh setelah periode deadline
  **Selesai bila**: penolakan kapasitas menyebutkan **angka** total kapasitas tersisa pada rentang tersebut; penolakan kesiapan menjelaskan bahwa produksi tidak dapat dimulai sebelum deadline terlampaui
  **Dependency**: prasyarat T035, T039
  **Hati-hati**: harga `int64` rupiah bulat. Setiap counter-offer adalah baris baru, bukan pembaruan baris lama. Rentang kapasitas di sini dihitung dari tanggal penawaran, bukan dari tanggal request, pastikan konsisten dengan T035.

- [ ] T041 [US3] [BE] Pembentukan kesepakatan dan alokasi kapasitas
  **Modul**: `backend/internal/order/`
  **FR**: FR-034, FR-036, FR-018, FR-077, FR-078, FR-084, FR-087
  **Kemampuan**:
  - Hitung dan simpan `minggu_kesiapan_mulai` pesanan = Senin dari (tanggal kesepakatan + `jeda_kesiapan_hari` listing saat itu). Disimpan, bukan dihitung ulang, karena jeda pada listing dapat berubah kemudian sementara alokasi tidak boleh bergeser.
  - Satu transaksi mencakup pembentukan pesanan dan seluruh baris alokasi; penguncian baris periode terurut menaik menurut `week_start`.
  - Alokasi mengisi periode paling awal **di dalam rentang kapasitas** lebih dulu, melewati yang penuh atau habis; kandidat lain ditutup dengan notifikasi.
  **Selesai bila**: pola transaksi mengikuti `research.md` R-04; kegagalan pada salah satu periode membatalkan seluruh pembentukan; tidak ada baris alokasi pada periode sebelum `minggu_kesiapan_mulai`
  **Dependency**: prasyarat T008, T027, T040
  **Hati-hati**: alokasi yang naif akan mulai dari minggu berjalan, dan itu berarti menjadwalkan pekerjaan pada minggu yang menurut pernyataan subkontraktor sendiri belum dapat dipakai. Trigger `cegah_alokasi_sebelum_kesiapan` akan menolaknya, tetapi jangan bergantung pada trigger untuk logika normal; ia jaring pengaman. Pengurutan penguncian adalah pencegah deadlock, bukan kerapian.

- [ ] T042 [P] [US3] [BE] Test balapan alokasi
  **Modul**: `backend/internal/order/`
  **FR**: FR-036, FR-079, FR-084, SC-018
  **Kemampuan**: dua kesepakatan berbarengan atas periode yang sama, hanya satu berhasil, yang gagal menerima alasan; constraint basis data menolak kapasitas terpakai melebihi total meski logika aplikasi dibuat keliru dengan sengaja
  **Selesai bila**: kedua test lulus; test constraint benar-benar membuktikan penolakan di tingkat penyimpanan data
  **Dependency**: prasyarat T041

- [ ] T043 [P] [US3] [BE] Test request kuota
  **Modul**: `backend/internal/quota/`
  **FR**: FR-029, FR-035, FR-082, FR-083, FR-090
  **Kemampuan**: jalur berhasil, penolakan peran, masukan tidak sah, request ke diri sendiri, kapasitas kurang beserta angkanya, kesiapan melampaui deadline, request kedaluwarsa dengan `Clock` digantikan
  **Dependency**: prasyarat T039, T040
  **Hati-hati**: test kedaluwarsa 72 jam wajib memakai `Clock` yang digeser. Karena tidak ada `DEFAULT now()`, baris uji dapat dibuat dengan waktu apa pun secara konsisten.

- [ ] T044 [US3] [FE] Frontend: request dan perbandingan
  **Modul**: `frontend/src/pages/request/`
  **FR**: FR-029, FR-030, FR-032, FR-033
  **Kemampuan**: pilih kandidat dari hasil pencarian, form request, daftar request terkirim dengan status per kandidat, perbandingan penawaran berdampingan, aksi counter-offer dan terima
  **Selesai bila**: batas 72 jam terlihat sebagai informasi, bukan kolom masukan
  **Dependency**: prasyarat T037, T039, T041

- [ ] T045 [US3] [FE] Frontend: request masuk untuk subkontraktor
  **Modul**: `frontend/src/pages/request/`
  **FR**: FR-031, FR-035, FR-090
  **Kemampuan**: daftar request masuk, penanda `dapat_menyanggupi` beserta total kapasitas di dalam rentang, form penawaran, form penolakan beralasan
  **Selesai bila**: bila kapasitas tidak cukup atau kesiapan melampaui deadline, penjelasannya tampil sebelum pengguna menekan kirim
  **Dependency**: prasyarat T021, T040

- [ ] T046 [US3] [OPS] Skenario uji manual US3
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 11 langkah dari `quickstart.md` bagian F US3
  **Dependency**: prasyarat T007, T042, T043, T044, T045

**Checkpoint**: transaksi dapat terbentuk. Isi kedua changelog.

---

## Phase 6: User Story 4, Kalender Aktual (P4)

**Goal**: kalender ketersediaan berkurang otomatis saat pesanan dikonfirmasi dan kembali saat dibatalkan.

**Independent Test**: konfirmasi pesanan besar, pastikan kapasitas berkurang dari minggu terawal di dalam rentang; batalkan sebelum produksi, pastikan seluruhnya kembali.

- [ ] T047 [US4] [BE] Pengelolaan kalender
  **Modul**: `backend/internal/listing/`
  **FR**: FR-017, FR-019, FR-021, FR-089
  **Kemampuan**: baca dan perbarui beberapa periode sekaligus, tandai penuh, penanda kalender basi lebih dari 7 hari; **propagasi perubahan kapasitas mingguan**, yakni ketika `weekly_capacity` listing diubah, perbarui `total_capacity` seluruh periode mendatang yang **belum memiliki alokasi aktif**, dan biarkan periode yang sudah memiliki alokasi tetap seperti semula
  **Selesai bila**: `/listing/me/periods` sesuai kontrak; penanda basi tidak mengubah urutan pencarian; mengubah kapasitas listing benar-benar mengubah periode tanpa alokasi dan tidak menyentuh yang punya alokasi
  **Dependency**: prasyarat T028, T041
  **Hati-hati**: `kalender_diperbarui_pada` terpisah dari `diperbarui_pada`, mengubah listing tidak boleh menghapus penanda basi. Untuk FR-089, **saring periode berdasarkan ada tidaknya baris alokasi aktif lebih dulu**, jangan mencoba memperbarui semuanya lalu menangkap galat constraint; galat itu tidak dapat dijelaskan ke pengguna.

- [ ] T048 [US4] [BE] Pembalikan alokasi
  **Modul**: `backend/internal/order/`
  **FR**: FR-020
  **Kemampuan**: membalik seluruh baris alokasi sebuah pesanan dalam satu transaksi, dengan pola penguncian yang sama seperti pembentukan
  **Selesai bila**: kapasitas setiap periode kembali ke angka sebelum pesanan terbentuk; baris alokasi ditandai `dibalik_pada`, tidak dihapus
  **Dependency**: prasyarat T041

- [ ] T049 [US4] [BE] Penolakan yang bertabrakan dengan alokasi berjalan
  **Modul**: `backend/internal/listing/`
  **FR**: FR-017, FR-089
  **Kemampuan**: menolak penurunan kapasitas periode di bawah yang sudah terpakai dan penandaan penuh atas periode yang sudah teralokasi, dengan pesan yang menyebut minggu mana beserta jumlahnya
  **Selesai bila**: kedua penolakan mengembalikan `KAPASITAS_SUDAH_TERALOKASI` atau `PERIODE_SUDAH_TERALOKASI`, bukan galat basis data mentah
  **Dependency**: prasyarat T041, T047
  **Hati-hati**: constraint akan menolak dengan sendirinya. Tugas task ini menerjemahkannya menjadi pesan yang dapat dibaca pengguna sebelum constraint tersentuh.

- [ ] T050 [P] [US4] [BE] Test alokasi, pembalikan, dan minggu kesiapan
  **Modul**: `backend/internal/{listing,order}/`
  **FR**: FR-018, FR-020, FR-078, FR-087, FR-089, SC-020
  **Kemampuan**:
  - Alokasi 1.200 potong pada kapasitas 500/minggu mengisi 500, 500, 200 pada tiga minggu pertama **di dalam rentang**, dan minggu berikutnya tetap utuh.
  - Jeda kesiapan 14 hari: alokasi dimulai dari minggu ketiga, dan dua minggu sebelumnya **tidak tersentuh sama sekali** (SC-020).
  - Periode yang ditandai penuh dilewati, alokasi berpindah ke minggu berikutnya di dalam rentang.
  - Pembalikan memulihkan seluruh periode ke angka semula.
  - Mengubah kapasitas listing memperbarui periode tanpa alokasi dan tidak mengubah periode yang punya alokasi (FR-089).
  - Trigger menolak upaya menyisipkan alokasi pada periode sebelum `minggu_kesiapan_mulai`.
  **Selesai bila**: seluruh test lulus dan namanya menyebut FR atau SC
  **Dependency**: prasyarat T047, T048
- [ ] T051 [US4] [FE] Frontend: kalender
  **Modul**: `frontend/src/pages/listing/`
  **FR**: FR-017, FR-021
  **Kemampuan**: tampilan kalender mingguan, penyuntingan beberapa periode sekaligus, penanda penuh, penanda basi, rincian alokasi per periode
  **Selesai bila**: setiap periode jelas dimulai hari Senin; alokasi per minggu terlihat beserta pesanan yang memakainya
  **Dependency**: prasyarat T032, T047

- [ ] T052 [US4] [OPS] Skenario uji manual US4
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 11 langkah dari `quickstart.md` bagian F US4, termasuk langkah memverifikasi alokasi berjeda kesiapan 14 hari
  **Dependency**: prasyarat T007, T050, T051

**Checkpoint**: data kapasitas tetap aktual tanpa tindakan manual. Isi kedua changelog.

---

## Phase 7: User Story 5, Pesanan Sampai Tuntas (P5)

**Goal**: kedua pihak memantau pesanan dari diterima sampai dikonfirmasi, dengan pembatalan pra-produksi dan penutupan otomatis.

**Independent Test**: jalankan seluruh transisi status; pada pesanan kedua, biarkan status Dikirim melewati tujuh hari dan pastikan tertutup otomatis.

- [ ] T053 [US5] [BE] Mesin keadaan pesanan
  **Modul**: `backend/internal/order/`
  **FR**: FR-038, FR-039, FR-044
  **Kemampuan**: transisi sesuai diagram `data-model.md` §7; riwayat status dengan waktu dan pelaku; penolakan transisi melompat beserta daftar transisi yang diizinkan
  **Dependency**: prasyarat T041
  **Selesai bila**: `WorkOrderDetail` mengirim `allowed_transitions` dan `self_cancellable`; galat `INVALID_STATUS_TRANSITION` menyebut urutan yang benar
  **Hati-hati**: perubahan oleh penjadwal ditandai `oleh_sistem`, bukan dibiarkan tanpa identitas.

- [ ] T054 [US5] [BE] Pembatalan
  **Modul**: `backend/internal/order/`
  **FR**: FR-065, FR-066, FR-072
  **Kemampuan**: pembatalan oleh kedua pihak selama status masih diterima, wajib beralasan, membalik seluruh alokasi; setelah produksi diarahkan ke sengketa
  **Dependency**: prasyarat T048, T053
  **Selesai bila**: `dibatalkan_oleh_id` tercatat, itu dasar perhitungan tingkat penyelesaian; galat `PEMBATALAN_SETELAH_PRODUKSI` menyebutkan jalur alternatifnya

- [ ] T055 [US5] [BE] Konfirmasi otomatis tujuh hari
  **Modul**: `backend/internal/order/` + scheduler
  **FR**: FR-068, FR-069, FR-070
  **Kemampuan**: dua lapisan, yaitu dihitung saat pesanan dibaca, dan penjadwal untuk pemberitahuan tenggat mendekat serta penulisan status final; dihentikan oleh sengketa
  **Dependency**: prasyarat T018, T053
  **Selesai bila**: satu fungsi domain dipakai kedua lapisan; `dikonfirmasi_otomatis` menandai yang mana
  **Hati-hati**: kalau perhitungan tenggat diduplikasi di beberapa handler, keduanya akan berbeda pada suatu titik dan pesanan yang sama tampak beda status di halaman berbeda.

- [ ] T056 [P] [US5] [BE] Catatan pembayaran
  **Modul**: `backend/internal/order/`
  **FR**: FR-040 sampai FR-043
  **Kemampuan**: catat pernyataan terkirim dan diterima, tanpa kolom jumlah uang; penanda perbedaan pernyataan antar pihak; keterangan bahwa platform tidak menjamin
  **Dependency**: prasyarat T053
  **Selesai bila**: tidak ada satu pun kolom jumlah uang maupun integrasi pembayaran
  **Hati-hati**: Batas Keuangan konstitusi. Dokumen sumber menempatkan escrow wajib sebagai mitigasi gagal bayar dan penipuan [1], sekaligus sebagai alat tawar dalam sengketa kualitas produk [1]. Keduanya sengaja tidak dibangun di versi ini, dan konsekuensinya tercatat di Assumptions spec. Jangan menambahkannya kembali tanpa mengubah spec lebih dulu.

- [ ] T057 [P] [US5] [BE] Pelaporan sengketa
  **Modul**: `backend/internal/order/`
  **FR**: FR-046, FR-070
  **Kemampuan**: laporkan sengketa, hentikan hitungan konfirmasi otomatis, satu sengketa terbuka per pesanan
  **Dependency**: prasyarat T053, T055
  **Selesai bila**: pelaporan berulang ditolak; hitungan otomatis benar-benar berhenti

- [ ] T058 [P] [US5] [BE] Test pesanan
  **Modul**: `backend/internal/order/`
  **FR**: FR-044, FR-065, FR-066, FR-068, FR-070
  **Kemampuan**: transisi melompat ditolak; pembatalan pra-produksi membalik alokasi; pembatalan pasca-produksi ditolak; konfirmasi otomatis dengan `Clock` digantikan; sengketa menghentikan hitungan
  **Dependency**: prasyarat T054, T055, T057
  **Hati-hati**: seluruh test tenggat memakai `Clock` yang digeser, bukan menunggu waktu nyata.

- [ ] T059 [US5] [FE] Frontend: dashboard pesanan
  **Modul**: `frontend/src/pages/work-orders/`
  **FR**: FR-038, FR-039, FR-041, FR-044
  **Kemampuan**: daftar aktif dan riwayat, detail dengan riwayat status, rincian alokasi per minggu, tombol transisi, form pembatalan, catatan pembayaran, tombol laporkan sengketa
  **Dependency**: prasyarat T021, T053, T054, T056, T057
  **Selesai bila**: tombol dirender dari `allowed_transitions` yang dikirim backend
  **Hati-hati**: **jangan** duplikasi mesin keadaan pesanan di React. Kalau logikanya ditulis ulang, dua tempat akan berbeda pada suatu titik.

- [ ] T060 [US5] [FE] Frontend: tenggat konfirmasi
  **Modul**: `frontend/src/pages/work-orders/`
  **FR**: FR-068, FR-069
  **Kemampuan**: tanggal pesanan akan dianggap diterima ditampilkan jelas pada pesanan berstatus Dikirim; penanda bahwa penutupan terjadi otomatis
  **Dependency**: prasyarat T055, T059

- [ ] T061 [US5] [OPS] Skenario uji manual US5
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 13 langkah dari `quickstart.md` bagian F US5
  **Dependency**: prasyarat T007, T058, T059, T060

**Checkpoint**: transaksi dapat diselesaikan. Isi kedua changelog.

---

## Phase 8: User Story 6, Reputasi (P6)

**Goal**: reputasi terbentuk dari transaksi nyata, dan pembatalan hanya membebani pihak yang membatalkan.

**Independent Test**: selesaikan pesanan, isi rating dari kedua sisi; batalkan pesanan lain dari satu sisi, pastikan hanya tingkat penyelesaian pihak itu yang turun.

- [ ] T062 [US6] [BE] Ulasan
  **Modul**: `backend/internal/reputation/`
  **FR**: FR-047, FR-049, FR-050
  **Kemampuan**: rating 1–5 dan teks hanya atas pesanan yang sudah dikonfirmasi, satu kali per pesanan per pihak, tidak anonim
  **Dependency**: prasyarat T053, T055
  **Selesai bila**: ulasan atas pesanan belum selesai ditolak; ulasan atas usaha yang belum pernah bertransaksi ditolak
  **Hati-hati**: pemeriksaan status pesanan tidak dapat ditegakkan `CHECK` karena merujuk tabel lain. Wajib di aplikasi.

- [ ] T063 [US6] [BE] Nilai turunan reputasi
  **Modul**: `backend/internal/reputation/`
  **FR**: FR-048, FR-071, FR-072, FR-073
  **Kemampuan**: rating rata-rata mengecualikan ulasan yang disembunyikan; tingkat penyelesaian dihitung saat dibaca; pembatalan masuk pembagi hanya bagi pihak yang membatalkan; persentase ditahan sampai 3 pesanan disepakati
  **Dependency**: prasyarat T054, T062
  **Selesai bila**: kueri mengikuti `data-model.md` §8; kedua angka penyusun selalu dikirim
  **Hati-hati**: dihitung saat dibaca, **bukan** disimpan sebagai kolom. Kolom yang harus diperbarui setiap kali ulasan disembunyikan atau pesanan dibatalkan adalah sumber ketidaksesuaian yang paling sering muncul.

- [ ] T064 [P] [US6] [BE] Test reputasi
  **Modul**: `backend/internal/reputation/`
  **FR**: FR-047, FR-050, FR-071, FR-072, FR-073
  **Kemampuan**: pembatalan menurunkan tingkat penyelesaian pihak yang membatalkan dan **tidak** mempengaruhi pihak lain; ulasan disembunyikan keluar dari rata-rata; ambang 3 pesanan
  **Dependency**: prasyarat T062, T063
  **Hati-hati**: test FR-072 adalah yang membedakan aturan ini dari perhitungan biasa. Wajib ada.

- [ ] T065 [US6] [FE] Frontend: ulasan dan reputasi
  **Modul**: `frontend/src/pages/{pesanan,profil}/`
  **FR**: FR-047, FR-048, FR-049, FR-073
  **Kemampuan**: form rating setelah pesanan dikonfirmasi, daftar ulasan pada profil dengan nama pemberi dan tanggal transaksi, ringkasan reputasi
  **Dependency**: prasyarat T021, T062, T063
  **Selesai bila**: bila `enough_data: false`, yang tampil adalah keterangannya, bukan persentase

- [ ] T066 [US6] [OPS] Skenario uji manual US6
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 9 langkah dari `quickstart.md` bagian F US6
  **Dependency**: prasyarat T007, T064, T065

**Checkpoint**: trust antar pihak yang belum saling kenal punya dasar yang dapat diperiksa. Isi kedua changelog.

---

## Phase 9: User Story 7, Admin (P7)

**Goal**: admin mengelola daftar baku, memberi lencana verifikasi, dan menengahi sengketa.

**Independent Test**: tambah jenis produk dan pastikan langsung dapat dipilih; setujui dan tolak satu pengajuan verifikasi, pastikan listing tetap tayang pada kedua kasus.

- [ ] T067 [P] [US7] [BE] Verifikasi identitas
  **Modul**: `backend/internal/{account,admin}/`
  **FR**: FR-006 sampai FR-011
  **Kemampuan**: pengajuan dengan dua berkas, antrean admin, setujui atau tolak beralasan, lencana, pengajuan ulang setelah penolakan
  **Dependency**: prasyarat T017, T020, T026
  **Selesai bila**: satu pengajuan menunggu per profil; penolakan wajib beralasan; listing tetap tayang apa pun keputusannya
  **Hati-hati**: verifikasi **bukan gerbang**. Ini menyimpang dari kriteria penerimaan dokumen sumber yang menempatkan status "Menunggu Verifikasi" pada alur listing [1]; penyimpangannya sudah tercatat di Assumptions spec.

- [ ] T068 [P] [US7] [BE] Pengelolaan daftar baku
  **Modul**: `backend/internal/{masterdata,admin}/`
  **FR**: FR-059, FR-060, FR-061
  **Kemampuan**: tambah, ubah nama, nonaktifkan item; keputusan atas usulan pengguna
  **Dependency**: prasyarat T019, T020
  **Selesai bila**: item nonaktif tidak dapat dipilih untuk listing baru, sementara listing yang sudah memakainya tetap utuh dan tetap dapat ditemukan

- [ ] T069 [P] [US7] [BE] Moderasi ulasan
  **Modul**: `backend/internal/admin/`
  **FR**: FR-050
  **Kemampuan**: sembunyikan ulasan dengan alasan, tercatat beserta identitas admin dan waktunya
  **Dependency**: prasyarat T020, T062
  **Selesai bila**: ulasan hilang dari profil publik dan rata-rata rating berubah; `alasan_penyembunyian` wajib terisi

- [ ] T070 [P] [US7] [BE] Pemantauan pesanan telat
  **Modul**: `backend/internal/admin/` + scheduler
  **FR**: FR-045
  **Kemampuan**: daftar pesanan melewati deadline, notifikasi ke kedua pihak
  **Dependency**: prasyarat T018, T020, T053
  **Selesai bila**: pesanan berstatus Produksi yang melewati deadline muncul di daftar admin

- [ ] T071 [US7] [BE] Mediasi sengketa
  **Modul**: `backend/internal/admin/`
  **FR**: FR-046, FR-067
  **Kemampuan**: tandai Dalam Mediasi, baca riwayat lengkap termasuk alokasi kapasitas, catatan pembayaran, dan perbedaan pernyataan; tutup mediasi
  **Dependency**: prasyarat T020, T048, T056, T057
  **Selesai bila**: penutupan sebagai dibatalkan **wajib** menyertakan keputusan pengembalian alokasi, pihak penanggung, dan catatan admin; tanpa ketiganya ditolak
  **Hati-hati**: constraint `penyelesaian_lengkap` menegakkan ini di basis data. Antarmuka harus meminta ketiganya secara eksplisit, bukan memberi nilai bawaan. Mediasi admin adalah jalur yang dipilih untuk fase awal karena penanganan sengketa legal formal menuntut tim hukum dan asuransi, dan tanpa escrow, ia kehilangan salah satu daya paksanya [1].

- [ ] T072 [P] [US7] [BE] Test admin
  **Modul**: `backend/internal/admin/`
  **FR**: FR-007, FR-050, FR-060, FR-067
  **Kemampuan**: non-admin ditolak pada setiap endpoint admin; item nonaktif tidak merusak listing; mediasi tanpa keputusan lengkap ditolak; berkas identitas milik usaha lain tidak dapat diakses
  **Dependency**: prasyarat T067, T068, T069, T070, T071

- [ ] T073 [US7] [FE] Frontend: panel admin
  **Modul**: `frontend/src/pages/admin/`
  **FR**: FR-007, FR-050, FR-059, FR-061, FR-045, FR-046, FR-067
  **Kemampuan**: enam layar, yaitu antrean verifikasi dengan pratayang berkas, kelola daftar baku, antrean usulan, moderasi ulasan, pesanan telat, mediasi sengketa
  **Dependency**: prasyarat T021, T067, T068, T069, T070, T071
  **Selesai bila**: setiap layar dapat diselesaikan tanpa menyentuh basis data
  **Hati-hati**: pengisian awal daftar baku dan wilayah lewat perintah seed, bukan lewat antarmuka, itu yang membuat enam layar ini tetap layak di prioritas terakhir.

- [ ] T074 [US7] [OPS] Skenario uji manual US7
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 17 langkah dari `quickstart.md` bagian F US7
  **Dependency**: prasyarat T007, T072, T073

**Checkpoint**: seluruh tujuh story berfungsi. Isi kedua changelog.

---

## Phase 10: Polish dan Penyiapan Demo

- [ ] T075 [BE] Data uji
  **Modul**: `backend/cmd/devotion/`
  **Kemampuan**: `seed:test-data` menyiapkan 50 usaha, kandidat 500 potong/minggu jeda 0 hari untuk skenario 3.000 potong, kandidat berjeda 14 dan 21 hari untuk memverifikasi minggu kesiapan, kandidat berkapasitas 200/minggu untuk skenario deadline lima bulan, kalender basi 8 hari, request kedaluwarsa, pesanan Dikirim 6 hari dan 8 hari lalu, pesanan telat, dua pengajuan verifikasi, usaha dengan hanya 2 pesanan; `reset:test-data` memulihkan
  **Dependency**: prasyarat T019, T041, T055, T067
  **Selesai bila**: perintah **menolak berjalan** saat `APP_ENV=production`; akun uji memakai domain `.test`
  **Hati-hati**: alur bertenggat disiapkan sebagai data yang sudah berada pada keadaan itu, bukan lewat kendali geser waktu, data yang di-seed tidak berisiko terbawa ke lingkungan sungguhan.

- [ ] T076 [P] [OPS] Dokumen skenario uji manual final
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: seluruh 83 langkah tergabung, bagian "di luar cakupan pengujian Anda" yang menyatakan WhatsApp dan email bukan temuan, kredensial akun uji, cara melaporkan temuan
  **Dependency**: prasyarat T034, T038, T046, T052, T061, T066, T074
  **Selesai bila**: dapat diikuti orang yang belum pernah melihat sistem ini; setiap langkah menyebut akun yang dipakai

- [ ] T077 [P] [OPS] Runbook VPS
  **Modul**: `docs/setup-vps.md`
  **Kemampuan**: 16 langkah dari `quickstart.md` bagian B
  **Dependency**: prasyarat T005
  **Selesai bila**: langkah verifikasi firewall disertakan sebagai gerbang, bukan saran

- [ ] T078 [P] [FE] Aksesibilitas dan bahasa
  **Modul**: `frontend/src/`
  **FR**: FR-055, FR-056
  **Kemampuan**: seluruh alur inti dapat diselesaikan dengan keyboard, label jelas, terbaca pembaca layar, mobile-first
  **Dependency**: prasyarat T059, T065, T073
  **Selesai bila**: tidak ada teks bahasa Inggris yang tampil ke pengguna

- [ ] T079 [P] [OPS] Dokumentasi penutup
  **Modul**: `docs/` + `backend/CHANGELOG.md` + `frontend/CHANGELOG.md`
  **Kemampuan**: `dependencies.md` lengkap dengan alasan setiap paket, `utang-teknis.md` memuat empat risiko yang diterima sadar dari `plan.md`, `layanan-luar.md` beserta akibat bila mati, kedua changelog final, `menjalankan.md`, `pengujian.md`
  **Dependency**: prasyarat T074
  **Selesai bila**: seluruh dependency terdaftar dengan alasannya, dan tidak ada yang di luar `plan.md`

- [ ] T080 [OPS] Penerapan ke VPS dan cadangan
  **Modul**: server
  **Kemampuan**: jalankan runbook sampai selesai, seed wilayah dan daftar baku, buat admin, sambungkan WhatsApp, pasang cron `pg_dump`, salin cadangan keluar VPS, daftarkan pemantau uptime
  **Dependency**: prasyarat T074, T075, T077
  **Selesai bila**: `/api/health` sehat; `curl` ke IP VPS langsung **gagal**; cadangan manual berhasil dan salinannya ada di luar server

- [ ] T081 [OPS] Snapshot dan checklist penjurian
  **Modul**: server + `docs/`
  **Kemampuan**: jalankan checklist `quickstart.md` bagian H, ambil snapshot VPS setelah semuanya terverifikasi
  **Dependency**: prasyarat T076, T080
  **Selesai bila**: seluruh butir checklist tercentang; snapshot ada
  **Hati-hati**: ini yang paling penting dari seluruh fase. Satu server tanpa cadangan berarti satu kesalahan dapat menghapus seluruh submission.

---

## Dependensi

### Antar fase

- Setup: tanpa dependensi
- Foundational: setelah Setup, **memblokir seluruh story**
- Story: seluruhnya setelah Foundational, lalu berurutan P1 → P7, atau paralel bila ada beberapa pelaksana
- Polish: setelah story yang diinginkan selesai

### Ketergantungan baru dari revisi 2026-08-22

| Task | Bergantung pada | Alasan |
|------|-----------------|--------|
| T035 | T028 | Memanggil fungsi perpanjangan horizon (FR-088) |
| T041 | T027 | Membaca `jeda_kesiapan_hari` untuk menghitung `minggu_kesiapan_mulai` |
| T040 | T035 | Rentang kapasitas harus dihitung dengan cara yang sama |
| T047 | T041 | Propagasi FR-089 harus tahu periode mana punya alokasi aktif |
| T050 | T041, T047 | Menguji alokasi dan propagasi bersama |

Konsekuensinya: T028 tidak lagi task tertutup di US1, ia menyediakan fungsi yang dipakai US2. Kerjakan fungsi perpanjangannya sebagai API internal yang jelas, bukan sebagai kode yang hanya dipanggil saat pembuatan listing.

### Antar story

| Story | Dapat dimulai setelah | Catatan |
|-------|----------------------|---------|
| US1 | Foundational | Tanpa dependensi story lain |
| US2 | Foundational, T028 | Butuh fungsi perpanjangan horizon |
| US3 | Foundational, T035 | Rentang kapasitas dipakai bersama |
| US4 | Foundational, T041 | T047 dan T050 menyentuh alokasi dari T041 |
| US5 | Foundational, T041 | Butuh pesanan |
| US6 | Foundational, T055 | Butuh pesanan yang dikonfirmasi |
| US7 | Foundational | Paling independen |

### Di dalam satu story

Query dan model → service → handler → test. Frontend setelah endpoint tersedia, **atau** paralel dengan mock server yang dihasilkan dari `openapi.yaml` (Prism atau MSW).

### Peluang paralel

- Seluruh task Setup bertanda `[P]` kecuali T001 dan T007
- Di Foundational: T012 dan T013 dapat paralel setelah T009; T016 setelah T011 dan T013; T017 setelah T015; T025 setelah T024a
- Setelah Foundational: tujuh story dapat paralel bila ada beberapa pelaksana, dengan memperhatikan tabel ketergantungan baru di atas
- Task test bertanda `[P]` karena modulnya berbeda dari implementasi

### Memotong per pelaksana

Saring berdasarkan prefiks **Modul**: sesi backend mengambil task ber-`backend/`, sesi frontend mengambil `frontend/`. Label story tetap sama di keduanya, jadi keterlacakan ke FR tidak hilang.

---

## Strategi Pelaksanaan

**MVP lebih dulu**: Setup → Foundational → US1 → berhenti dan buktikan US1 berjalan sendiri → demo.

**Bertahap**: setiap story menambah nilai tanpa merusak yang sebelumnya. Berhenti di setiap checkpoint, jalankan skenario uji manual story itu, isi `backend/CHANGELOG.md` dan `frontend/CHANGELOG.md`.

**Bila tenggat menekan**, urutan pemangkasan yang paling sedikit merusak:

1. Sentry di frontend (T025 sebagian), monitoring tidak menambah nilai penjurian
2. US7 layar moderasi ulasan dan pesanan telat (T073 sebagian), sisakan verifikasi dan mediasi, keduanya ada di Acceptance Scenario
3. US6 seluruhnya, reputasi baru bermakna setelah ada transaksi

Yang **tidak boleh** dipangkas: US1 sampai US3, karena tanpa ketiganya tidak ada alur yang dapat didemokan; dan seluruh test pada T036, T042, T050, T058, T064, karena itu aturan yang paling mudah rusak diam-diam. Empat test baru di T036 dan T050, yaitu SC-020, SC-021, dan dua kasus jeda kesiapan, khususnya wajib, karena bug yang ditutupnya menghasilkan angka yang tetap tampak masuk akal.

Setiap pemangkasan dicatat di `docs/utang-teknis.md`. Pengecualian kewajiban pengujian hanya boleh per story, dengan catatan di Complexity Tracking `plan.md`.

---

## Catatan

- Satu task = satu modul + satu kelompok FR. Pemecahan file di dalam modul diserahkan pelaksana kecuali empat path yang dipatok.
- `[P]` bermakna di tingkat modul: dua task `[P]` tidak akan menulis modul yang sama.
- Bila sebuah task terasa perlu menyentuh modul di luar yang tertulis, itu tanda batas modulnya keliru, angkat lebih dulu, jangan diam-diam menyimpang.
- Setiap test menyebut FR atau SC yang diuji pada namanya.
- Commit per task atau per kelompok yang logis.
- Berhenti di setiap checkpoint dan buktikan story itu berdiri sendiri.
- Hindari: task kabur, dua task menulis modul yang sama, dan ketergantungan lintas story yang merusak kemandirian.