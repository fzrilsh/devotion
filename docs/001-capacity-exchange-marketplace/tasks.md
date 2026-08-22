# Tasks: Capacity Exchange — Devotion

**Input**: `docs/specs/001-capacity-exchange-marketplace/`
**Prerequisites**: `spec.md` (86 FR), `plan.md`, `research.md`, `data-model.md`, `contracts/openapi.yaml`, `quickstart.md`, `docs/memory/constitution.md` v2.1.0

**Tests**: DIWAJIBKAN. Konstitusi v2.1.0 menetapkan pengujian otomatis sebagai gerbang mutu, bukan pilihan.

**Organisasi**: per user story, agar setiap story dapat diimplementasikan, diuji, dan didemokan sebagai tambahan nilai yang berdiri sendiri.

## Format

```text
- [ ] T0XX [P?] [Story?] Judul singkat
  **Modul**: direktori tempat pekerjaan ini berada
  **FR**: requirement yang dilayani
  **Kemampuan**: apa yang harus bisa dilakukan setelah task ini selesai
  **Dependency**: paket baru yang dipakai, atau "tidak ada"
  **Selesai bila**: kriteria yang bisa diperiksa, bukan dinilai
  **Saran**: usulan pemecahan, boleh diabaikan bila ada cara lebih baik
  **Hati-hati**: hal yang mudah salah dan mahal diperbaiki belakangan
```

- **[P]** = boleh paralel: modulnya berbeda dan tidak saling menunggu
- **[Story]** = US1–US7, hanya pada fase story
- Pemecahan file di dalam modul diserahkan pelaksana, **kecuali** empat path yang dipatok di bawah

## Empat Path yang Dipatok

Berubah tempatnya berarti artefak lain rusak:

| Path | Alasan |
|------|--------|
| `backend/internal/platform/clock.go` | Disuntikkan ke seluruh service; Prinsip V |
| `backend/db/migrations/` | Urutan sudah ditetapkan `data-model.md` §12 |
| `backend/webdist/` | Target build frontend, dirujuk `embed.FS` |
| `docker-compose.yml` | Gate I dihitung dari jumlah entri `services:` |

---

## Phase 1: Setup

**Tujuan**: repository dapat dibangun dan dijalankan, meski belum ada fitur.

- [ ] T001 Struktur repository dan berkas tingkat atas
  **Modul**: root
  **Kemampuan**: `README.md` (template panitia, struktur tidak diubah), `LICENSE` MIT, `.gitignore`, `.env.example` berisi nama variabel tanpa nilai, direktori `backend/`, `frontend/`, `docs/`
  **Selesai bila**: tidak ada direktori tingkat atas di luar daftar konstitusi; `.env` masuk `.gitignore`
  **Hati-hati**: `.env.example` tidak boleh memuat satu pun nilai sungguhan. Repository ini publik.

- [ ] T002 [P] Inisialisasi modul Go dan subcommand
  **Modul**: `backend/cmd/devotion/`
  **Kemampuan**: `go.mod` dengan Go 1.22+, dispatcher subcommand: `serve`, `admin:create`, `seed:wilayah`, `seed:master-data`, `seed:test-data`, `reset:test-data`, `user:verify`, `health:check`
  **Dependency**: tidak ada — cukup `flag.NewFlagSet` dari standard library
  **Selesai bila**: `go run ./cmd/devotion` menampilkan daftar subcommand; `go vet ./...` bersih
  **Saran**: satu berkas per subcommand, dispatcher tipis di `main.go`. Subcommand adalah proses sekali jalan, bukan proses runtime — tidak melanggar Gate I.

- [ ] T003 [P] Inisialisasi frontend
  **Modul**: `frontend/`
  **Kemampuan**: Vite + React 18 + TypeScript + Tailwind, Jest, struktur `src/{pages,components,api,schemas,lib}`
  **Dependency**: sesuai `plan.md` Primary Dependencies. Jangan tambah di luar daftar itu.
  **Selesai bila**: `npm run build` menghasilkan `dist/`; `npm test` jalan meski belum ada test
  **Hati-hati**: Vite, bukan Next.js. Next.js akan menggoda menaruh API route di frontend, dan itu backend kedua — pelanggaran Gate I yang paling mudah terjadi tanpa disadari.

- [ ] T004 [P] Generator tipe dari OpenAPI
  **Modul**: `frontend/src/api/`
  **Kemampuan**: skrip npm menghasilkan tipe TypeScript dari `contracts/openapi.yaml`
  **Dependency**: `openapi-typescript` (devDependency)
  **Selesai bila**: tipe ter-generate dan dapat diimpor; skrip terdokumentasi di `docs/menjalankan.md`
  **Hati-hati**: jangan pernah menulis tipe respons dengan tangan. Yang ditulis tangan akan menyimpang dari kontrak tanpa ada yang tahu.

- [ ] T005 [P] Compose dua layanan
  **Modul**: `docker-compose.yml` (path dipatok)
  **Kemampuan**: tepat dua layanan `backend` dan `postgres`, penyetelan Postgres untuk 2GB sesuai `research.md` R-03, batas log `max-size 10m` `max-file 3` pada keduanya, volume `pgdata` dan bind `/opt/devotion/unggahan`
  **Selesai bila**: `docker compose config` valid; jumlah entri di bawah `services:` tepat dua
  **Hati-hati**: batas log bukan kebersihan. Log tanpa batas mengisi 50GB, lalu Postgres berhenti menulis dan aplikasi mati total.

- [ ] T006 [P] Kerangka dokumentasi
  **Modul**: `docs/`
  **Kemampuan**: `menjalankan.md`, `pengujian.md`, `dependencies.md`, `utang-teknis.md`, `layanan-luar.md`, `temuan-penguji.md`, `changelog.md`, `cloudflare-ips.md`
  **Selesai bila**: seluruh berkas ada dengan judul dan kerangka bagian
  **Saran**: `changelog.md` diisi setiap kali sebuah story ditutup, bukan direkonstruksi di akhir. `layanan-luar.md` mencatat Cloudflare, Mailjet, Sentry, pemantau uptime, wilayah.id beserta akibat bila mati.

- [ ] T007 CI pipeline
  **Modul**: `.github/workflows/`
  **Kemampuan**: `go vet`, `go test`, `npm test`, `npm run build`, salin `dist` → `backend/webdist/`, build image multi-stage, push ke GHCR, deploy via SSH
  **Selesai bila**: pipeline hijau pada commit kosong; image terbit dengan tag `<sha>` dan `latest`
  **Hati-hati**: membangun artefak di server dilarang konstitusi. Build Vite pada 2GB sambil Postgres hidup akan kehabisan memori, dan yang dibunuh kernel biasanya Postgres.

**Checkpoint**: repository dapat dibangun, image terbit, compose valid.

---

## Phase 2: Foundational (Blocking)

**⚠️ Tidak ada pekerjaan user story yang boleh dimulai sebelum fase ini selesai.**

- [ ] T008 Clock yang dapat digantikan
  **Modul**: `backend/internal/platform/clock.go` (path dipatok)
  **FR**: Prinsip V
  **Kemampuan**: interface `Clock` dengan `Now()`, implementasi nyata, implementasi uji yang waktunya dapat disetel dan digeser
  **Dependency**: tidak ada
  **Selesai bila**: ada test yang membuktikan waktu dapat digeser
  **Hati-hati**: dikerjakan **sekarang**, bukan menyusul. Menambahkannya setelah service jadi berarti menyentuh seluruh service. `time.Now()` dilarang muncul di dalam logika bisnis mana pun.

- [ ] T009 Konfigurasi dan bootstrap
  **Modul**: `backend/internal/platform/config/`
  **Kemampuan**: memuat variabel lingkungan, memvalidasi yang wajib saat startup, membedakan `APP_ENV=development` dan `production`
  **Dependency**: tidak ada — `os.Getenv` cukup
  **Selesai bila**: variabel wajib yang hilang menghentikan startup dengan pesan yang menyebut nama variabelnya
  **Saran**: pada `development`, backend melayani HTTP biasa tanpa TLS dan tanpa pemeriksaan sertifikat klien Cloudflare.

- [ ] T010 Migrasi basis data
  **Modul**: `backend/db/migrations/` (path dipatok)
  **FR**: seluruh entitas
  **Kemampuan**: 14 migrasi berurutan sesuai `data-model.md` §12, dijalankan otomatis saat startup dengan `pg_try_advisory_lock`
  **Dependency**: `golang-migrate` (versi dipatok)
  **Selesai bila**: `docker compose up` menjalankan migrasi sampai versi terakhir; `schema_migrations.dirty = false`; menjalankan dua kali tidak menimbulkan galat
  **Hati-hati**: seluruh constraint dan indeks di `data-model.md` wajib ikut — terutama `kapasitas_terpakai_tidak_melebihi_total`, `minggu_mulai_hari_senin`, `kota_milik_provinsinya`, dan tiga trigger. Constraint itu bukan hiasan: ia yang menahan kerusakan data ketika logika aplikasi keliru.

- [ ] T011 Lapisan akses data
  **Modul**: `backend/db/queries/` + konfigurasi `sqlc`
  **Kemampuan**: `sqlc.yaml`, pool `pgx` dengan `MaxConns=15`, helper transaksi
  **Dependency**: `pgx/v5`, `sqlc` (perkakas build)
  **Selesai bila**: `sqlc generate` berhasil; pool tersambung; helper transaksi punya test
  **Hati-hati**: pool 15 dari `max_connections` 20 — lima disisakan untuk `pg_dump`, `psql`, dan migrasi. Tanpa sisa itu, cadangan harian gagal justru saat trafik tinggi.

- [ ] T012 [P] Lapisan HTTP dan format galat
  **Modul**: `backend/internal/platform/httpx/`
  **FR**: seluruh endpoint
  **Kemampuan**: router `net/http`, middleware request ID, pemulihan panic, `application/problem+json` dengan 30 kode galat dari `openapi.yaml`, log `slog` JSON dengan request ID di setiap baris
  **Dependency**: tidak ada — `net/http` dan `log/slog` standard library
  **Selesai bila**: galat validasi mengembalikan bentuk `ProblemValidasi` beserta daftar field; setiap baris log memuat request ID
  **Hati-hati**: `/api/*` yang tidak dikenali mengembalikan 404 JSON, **bukan** `index.html`. Kalau HTML, kesalahan penulisan alamat endpoint jadi menyesatkan saat diagnosis.

- [ ] T013 [P] Kepercayaan alamat asal
  **Modul**: `backend/internal/platform/cloudflare/`
  **Kemampuan**: rentang alamat Cloudflare dipatok sebagai konstanta beserta tanggal pengambilan, fungsi `RealIP` yang hanya mempercayai header bila koneksi datang dari rentang itu
  **Dependency**: tidak ada
  **Selesai bila**: test membuktikan header diabaikan pada koneksi di luar rentang
  **Hati-hati**: daftar rentang di `research.md` R-01 ditulis dari ingatan dan **wajib dicocokkan** ke `cloudflare.com/ips-v4` dan `ips-v6` lebih dulu. Jangan mengambilnya lewat jaringan saat startup — satu kegagalan HTTP akan membuat aplikasi gagal menyala.

- [ ] T014 Sesi dan autentikasi
  **Modul**: `backend/internal/platform/session/` + `backend/internal/account/`
  **FR**: FR-002, FR-003
  **Kemampuan**: registrasi, verifikasi kode enam digit untuk email dan nomor, masuk, keluar, pemulihan kata sandi, cookie `httpOnly Secure SameSite=Lax`, hash token di basis data
  **Dependency**: `bcrypt` cost 10
  **Selesai bila**: seluruh endpoint `/auth/*` sesuai `openapi.yaml`; keluar akun benar-benar mengakhiri sesi; test membuktikan yang disimpan adalah hash, bukan token mentah
  **Hati-hati**: `POST /auth/pulihkan/permintaan` selalu 202, agar tidak membocorkan apakah sebuah email terdaftar.

- [ ] T015 Middleware peran
  **Modul**: `backend/internal/platform/httpx/`
  **FR**: FR-005
  **Kemampuan**: pemeriksaan peran per endpoint; satu akun boleh memegang dua peran usaha; admin terpisah dari peran usaha
  **Selesai bila**: test membuktikan penolakan untuk setiap kombinasi peran yang tidak berwenang
  **Hati-hati**: endpoint tanpa pemeriksaan peran dianggap **cacat**, bukan belum lengkap. Ini gerbang yang diperiksa di setiap story.

- [ ] T016 [P] Pembatasan laju berbasis data domain
  **Modul**: `backend/internal/platform/ratelimit/`
  **Kemampuan**: batas per akun untuk percobaan masuk (5/15 menit), per nomor untuk kode sekali pakai (3/jam), per alamat asal untuk kode (10 nomor/jam), per pengguna untuk request kuota (20/jam)
  **Dependency**: tidak ada — tabel Postgres
  **Selesai bila**: keempat batas punya test; respons 429 menyertakan `Retry-After`
  **Hati-hati**: tabel, bukan penyimpanan dalam memori. Kalau di memori, penerapan versi baru jadi cara termudah melewatinya. Batas per alamat asal yang menutup pemutaran nomor — itu yang melindungi nomor WhatsApp dari pemblokiran.

- [ ] T017 [P] Penyimpanan berkas
  **Modul**: `backend/internal/platform/storage/`
  **FR**: FR-006, FR-009
  **Kemampuan**: unggah maksimal 5MB, nama berkas UUID dibuat sistem, tipe divalidasi dari magic bytes, metadata lokasi gambar dibuang, kuota total 500MB, akses hanya lewat handler yang memeriksa peran
  **Dependency**: tidak ada — `image/jpeg` dan `image/png` dekode-enkode ulang membuang EXIF
  **Selesai bila**: bukan pemilik dan bukan admin ditolak (test wajib); berkas dengan ekstensi menipu ditolak; kuota penuh mengembalikan pesan yang jelas
  **Hati-hati**: **jangan pernah** melayani berkas lewat path statis. Foto lokasi usaha dari ponsel membawa koordinat GPS, dan banyak konveksi rumahan berarti itu alamat rumah orang.

- [ ] T018 Penjadwal dua lapisan
  **Modul**: `backend/internal/platform/scheduler/`
  **FR**: FR-021, FR-037, FR-045, FR-068, FR-069
  **Kemampuan**: `time.Ticker` 5 menit di dalam proses yang sama, setiap pekerjaan dibungkus advisory lock
  **Dependency**: tidak ada
  **Selesai bila**: penjadwal menyala saat startup dan tercatat di log; test membuktikan pekerjaan tidak berjalan ganda
  **Hati-hati**: perhitungan tenggat ditulis **satu kali** di satu fungsi domain, dipakai bersama lapisan hitung-saat-baca. Kalau diduplikasi, pesanan yang sama akan tampak berbeda status di halaman berbeda. Bukan proses kedua — Gate I.

- [ ] T019 Data acuan: wilayah dan daftar baku
  **Modul**: `backend/internal/masterdata/` + subcommand seed
  **FR**: FR-058, FR-062, FR-075
  **Kemampuan**: `seed:wilayah` mengambil provinsi dan kabupaten/kota dari wilayah.id dengan flag `--refresh`, default membaca salinan `docs/master-data/wilayah.json`; `seed:master-data` mengisi jenis produk dan mesin; keduanya idempoten memakai kode sebagai identitas
  **Dependency**: tidak ada — `net/http` dan `encoding/json`
  **Selesai bila**: kedua perintah jalan dua kali tanpa menduplikasi; hitungan provinsi, kota, produk, mesin semuanya lebih dari nol
  **Hati-hati**: bentuk respons wilayah.id **belum diperiksa** (`research.md` R-02). Langkah pertama: panggil dua endpoint itu, catat bentuk aslinya di `docs/master-data/README.md`. Kecamatan dan desa jangan diambil — puluhan ribu baris tanpa requirement yang memakainya. Jangan pernah memanggil layanan itu saat melayani permintaan pengguna.

- [ ] T020 Admin pertama
  **Modul**: `backend/cmd/devotion/`
  **Kemampuan**: `admin:create` meminta kata sandi lewat prompt tanpa menampilkan ketikan, idempoten
  **Selesai bila**: admin dapat masuk; menjalankan dua kali tidak membuat admin ganda
  **Hati-hati**: kata sandi lewat prompt, bukan argumen — argumen tersimpan di riwayat shell.

- [ ] T021 Frontend: shell aplikasi
  **Modul**: `frontend/src/`
  **FR**: FR-055, FR-056
  **Kemampuan**: layout mobile-first bahasa Indonesia, routing, klien API dengan `credentials: 'include'`, TanStack Query, penanganan galat yang menampilkan `detail` dari respons, halaman masuk dan daftar
  **Selesai bila**: dapat masuk dan keluar lewat antarmuka; alur inti dapat diselesaikan dengan keyboard
  **Hati-hati**: token **tidak pernah** disimpan di `localStorage` maupun `sessionStorage`. Satu celah XSS akan langsung berarti pengambilalihan akun, dan aplikasi ini memuat dokumen identitas.

- [ ] T022 Penyajian frontend oleh backend
  **Modul**: `backend/internal/platform/httpx/` + `backend/webdist/` (path dipatok)
  **Kemampuan**: `embed.FS` menyematkan hasil build, berkas statis ber-hash dengan cache panjang, fallback `index.html` untuk path non-API, TLS dengan Cloudflare Origin Certificate dan verifikasi sertifikat klien
  **Selesai bila**: penyegaran pada halaman dalam tidak menghasilkan 404; `/api/*` tak dikenal mengembalikan JSON
  **Hati-hati**: ini yang membuat Gate I terpenuhi tanpa layanan frontend maupun proxy. Pastikan Cloudflare tidak meng-cache `/api/*` — hasil pencarian ter-cache menampilkan kapasitas basi, persis masalah informasi tidak aktual yang platform ini dibangun untuk menyelesaikan [1].

- [ ] T023 Notifikasi
  **Modul**: `backend/internal/notification/`
  **FR**: FR-051 sampai FR-054, FR-074, FR-085, FR-086
  **Kemampuan**: baris notifikasi ditulis di dalam transaksi kejadiannya; goroutine pengirim ke email dan WhatsApp maksimal 3 percobaan; notifikasi di dalam platform selalu tampil; preferensi kanal untuk notifikasi non-transaksional
  **Dependency**: `net/smtp` (standard library, Mailjet lewat SMTP), `whatsmeow`
  **Selesai bila**: test membuktikan kegagalan kirim tidak menggagalkan transaksi pemicunya; setelah 3 percobaan ditandai gagal permanen; endpoint `/notifikasi` sesuai kontrak
  **Hati-hati**: notifikasi di dalam platform adalah **satu-satunya** jalur pengamatan bagi penguji manual — mereka tidak punya nomor dan alamat yang dipakai sistem.

- [ ] T024 Halaman admin WhatsApp
  **Modul**: `backend/internal/admin/` + `frontend/src/pages/admin/`
  **FR**: FR-002, FR-052
  **Kemampuan**: menampilkan QR dan status sambungan, penyambungan ulang tanpa akses server; `user:verify --phone` sebagai jalan darurat
  **Selesai bila**: QR dapat dipindai lewat antarmuka; status tersambung terlihat; endpoint health menyertakan status WhatsApp
  **Hati-hati**: FR-002 menjadikan verifikasi nomor sebagai gerbang pendaftaran, jadi sesi yang lepas berarti tidak ada akun baru yang bisa dibuat. Halaman ini yang mencegah kehilangan demo. Nomor layanan hanya dari variabel lingkungan — tidak pernah di kode maupun dokumentasi.

- [ ] T025 [P] Health check dan Sentry
  **Modul**: `backend/internal/platform/`
  **Kemampuan**: `GET /health` memeriksa basis data, WhatsApp, dan kuota penyimpanan; Sentry dengan pembersihan data sensitif
  **Dependency**: `sentry-go`
  **Selesai bila**: health mengembalikan 503 bila ada ketergantungan gagal; test membuktikan kata sandi, token, nomor telepon, dan hal terkait dokumen identitas tidak terkirim ke Sentry

**Checkpoint**: fondasi siap. Seluruh user story boleh dimulai, dan bila ada dua pelaksana, boleh paralel.

---

## Phase 3: User Story 1 — Listing Kapasitas (P1) 🎯 MVP

**Goal**: subkontraktor dapat mendaftarkan kapasitas produksinya dan listing itu langsung dapat ditemukan pihak lain.

**Independent Test**: daftar sebagai subkontraktor, buat listing lengkap, buka halaman publiknya sebagai pengunjung lain, seluruh atribut kapasitas tampil benar.

- [ ] T026 [P] [US1] Profil usaha
  **Modul**: `backend/internal/account/`
  **FR**: FR-004, FR-057
  **Kemampuan**: baca dan ubah profil sendiri, profil publik, kota dari data wilayah, koordinat opsional
  **Selesai bila**: `/profil/saya` dan `/profil/{id}` sesuai kontrak; koordinat di luar Indonesia ditolak; lintang tanpa bujur ditolak

- [ ] T027 [US1] Listing kapasitas
  **Modul**: `backend/internal/listing/`
  **FR**: FR-012, FR-013, FR-014, FR-015, FR-076
  **Kemampuan**: buat, ubah, nonaktifkan, aktifkan kembali; satu angka kapasitas mingguan untuk seluruh jenis produk; jeda kesiapan mulai dalam hari
  **Selesai bila**: `/listing/saya` sesuai kontrak; atribut wajib kosong ditolak dengan menyebut kolomnya
  **Saran**: pisahkan service dan handler; validasi di service supaya dapat dites tanpa HTTP
  **Hati-hati**: **jangan** buat kolom kapasitas per jenis produk. Mesin dan tenaga kerjanya berbagi, sehingga angka terpisah akan mengizinkan penyanggupan ganda pada minggu yang sama. Jeda kesiapan mulai bukan durasi menyelesaikan pekerjaan.

- [ ] T028 [US1] Kalender awal
  **Modul**: `backend/internal/listing/`
  **FR**: FR-017
  **Kemampuan**: periode mingguan dibuat otomatis untuk minimal 3 bulan ke depan saat listing dibuat, memakai kapasitas mingguan sebagai kapasitas total
  **Selesai bila**: setiap `minggu_mulai` adalah hari Senin; horizon minimal 13 periode
  **Hati-hati**: batas minggu dihitung di Asia/Jakarta, disimpan sebagai `date`. Constraint `minggu_mulai_hari_senin` akan menolak bila perhitungannya keliru — dan itu memang gunanya.

- [ ] T029 [P] [US1] Usulan item daftar baku
  **Modul**: `backend/internal/masterdata/`
  **FR**: FR-061
  **Kemampuan**: pengguna mengusulkan item baru, listing tetap dapat disimpan dengan item yang tersedia
  **Selesai bila**: `POST /master/usulan` sesuai kontrak; pengusul menerima notifikasi saat diputuskan

- [ ] T030 [P] [US1] Test backend US1
  **Modul**: `backend/internal/{account,listing}/`
  **Kemampuan**: jalur berhasil, penolakan peran, penolakan masukan tidak sah untuk setiap endpoint; listing tayang tanpa verifikasi
  **Selesai bila**: setiap nama test menyebut FR yang diuji; seluruhnya lulus

- [ ] T031 [P] [US1] Frontend: pendaftaran dan verifikasi
  **Modul**: `frontend/src/pages/auth/`
  **FR**: FR-001, FR-002
  **Kemampuan**: form daftar dengan pilihan peran, pemilih kota, halaman kode verifikasi, tombol kirim ulang dengan jeda membesar
  **Selesai bila**: pesan yang muncul saat mencoba membuat listing sebelum verifikasi menjelaskan apa yang harus dilakukan

- [ ] T032 [US1] Frontend: form listing
  **Modul**: `frontend/src/pages/listing/`
  **FR**: FR-012, FR-013, FR-076
  **Kemampuan**: pemilih jenis produk dan mesin dari daftar baku, satu kolom kapasitas mingguan, kolom jeda kesiapan mulai, tautan mengusulkan item baru
  **Dependency**: Zod + React Hook Form
  **Selesai bila**: tidak ada kolom teks bebas untuk jenis produk dan mesin; tidak ada kolom kapasitas per produk; galat per kolom tampil dari respons backend

- [ ] T033 [US1] Frontend: profil publik
  **Modul**: `frontend/src/pages/profil/`
  **FR**: FR-016, FR-064
  **Kemampuan**: atribut listing, ketersediaan terkini, peta lokasi, ringkasan reputasi, lencana verifikasi
  **Dependency**: Leaflet + tile OpenStreetMap
  **Selesai bila**: dapat dibuka tanpa masuk; peta tampil tanpa kunci API

- [ ] T034 [US1] Skenario uji manual US1
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 10 langkah dari `quickstart.md` bagian F US1, dengan kolom temuan
  **Selesai bila**: setiap langkah menyebut akun yang dipakai; salah tulis "Bu... maksudnya Pak Budi" pada langkah 1.8 sudah diperbaiki

**Checkpoint**: US1 berfungsi dan dapat didemokan sendiri. Isi `changelog.md`.

---

## Phase 4: User Story 2 — Pencarian (P2)

**Goal**: pemberi order menemukan subkontraktor yang cocok, dengan kapasitas dijumlahkan lintas periode sampai deadline.

**Independent Test**: dengan listing yang sudah tayang, cari 3.000 potong dengan deadline delapan minggu; kandidat berkapasitas 500 per minggu ikut muncul dan ditandai memenuhi kriteria kapasitas.

- [ ] T035 [US2] Mesin pencarian
  **Modul**: `backend/internal/search/`
  **FR**: FR-022 sampai FR-028, FR-063, FR-080, FR-081
  **Kemampuan**: empat kriteria keras sebagai empat nilai boolean yang dijumlahkan; kapasitas dijumlahkan dari minggu berjalan sampai minggu deadline; pemecah seri lima tingkat; keyset pagination; perluasan wilayah kota → provinsi → nasional; penjelasan per kriteria; saran pelonggaran saat kosong
  **Selesai bila**: `GET /pencarian` sesuai kontrak; bentuk kueri mengikuti `data-model.md` §10
  **Hati-hati**: **tidak ada pembobotan**. Rating, tingkat penyelesaian, verifikasi, kebaruan kalender, jarak, dan tanggal pendaftaran tidak boleh mempengaruhi urutan. `listing_id` sebagai pemecah seri terakhir wajib ada — tanpanya urutan bisa bertukar antar permintaan.

- [ ] T036 [P] [US2] Test determinisme pencarian
  **Modul**: `backend/internal/search/`
  **FR**: FR-023, FR-024, FR-025, FR-080, SC-013, SC-019
  **Kemampuan**: urutan identik pada pengulangan; stabil antar halaman meski ada listing baru disisipkan; skor tidak berubah saat rating atau verifikasi diubah; skenario 3.000 potong pada kapasitas 500 dengan deadline delapan minggu lolos, dengan deadline empat minggu tidak lolos; listing sendiri dikecualikan
  **Selesai bila**: seluruh test lulus dan namanya menyebut FR
  **Hati-hati**: ini kelompok test terpenting di seluruh project. Aturan urutan mudah rusak diam-diam dan tidak akan tertangkap pengujian manual.

- [ ] T037 [P] [US2] Frontend: halaman pencarian
  **Modul**: `frontend/src/pages/pencarian/`
  **FR**: FR-022, FR-026, FR-027, FR-028, FR-063, FR-080
  **Kemampuan**: filter produk, mesin, wilayah, jumlah, deadline, jeda maksimal; kartu hasil menampilkan seluruh atribut keputusan; penjelasan kriteria yang tidak terpenuhi; tombol perluas yang menyebut tingkat berikutnya; keadaan kosong beserta saran
  **Selesai bila**: kursor diteruskan apa adanya; tidak ada kandidat ganda saat berpindah halaman
  **Hati-hati**: kursor bersifat opaque. Jangan diurai, jangan diubah jadi `?page=2` — itu langsung melanggar jaminan urutan stabil.

- [ ] T038 [US2] Skenario uji manual US2
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 12 langkah dari `quickstart.md` bagian F US2

**Checkpoint**: matching berfungsi dan dapat dijelaskan ke pengguna. Isi `changelog.md`.

---

## Phase 5: User Story 3 — Request Kuota dan Negosiasi (P3)

**Goal**: pemberi order mengirim satu request ke beberapa kandidat, membandingkan penawaran, dan menyepakati satu.

**Independent Test**: kirim ke tiga kandidat, balas dari dua dengan harga berbeda, bandingkan, terima satu.

- [ ] T039 [US3] Request kuota
  **Modul**: `backend/internal/quota/`
  **FR**: FR-029, FR-030, FR-082, FR-083
  **Kemampuan**: kirim ke beberapa kandidat dalam satu tindakan; status per kandidat; batas balasan 72 jam ditetapkan sistem; penolakan request ke listing sendiri
  **Selesai bila**: `/request-kuota` sesuai kontrak; tidak ada kolom untuk mengatur batas waktu sendiri
  **Hati-hati**: FR-083 menyebut jalur "tanpa melalui hasil pencarian" secara eksplisit. Aplikasi menolak dengan pesan yang dapat dibaca pengguna; trigger basis data adalah jaring pengamannya.

- [ ] T040 [US3] Penawaran dan negosiasi
  **Modul**: `backend/internal/quota/`
  **FR**: FR-031, FR-032, FR-033, FR-035
  **Kemampuan**: balas dengan harga dan jeda kesiapan, tolak beralasan, counter-offer berantai dengan riwayat lengkap, perbandingan berdampingan
  **Selesai bila**: penolakan karena kapasitas kurang menyebutkan **angka** total kapasitas tersisa sampai deadline
  **Hati-hati**: harga `int64` rupiah bulat. Setiap counter-offer adalah baris baru, bukan pembaruan baris lama.

- [ ] T041 [US3] Pembentukan kesepakatan dan alokasi kapasitas
  **Modul**: `backend/internal/order/`
  **FR**: FR-034, FR-036, FR-018, FR-077, FR-078, FR-084
  **Kemampuan**: satu transaksi mencakup pembentukan pesanan dan seluruh baris alokasi; penguncian baris periode terurut menaik menurut `minggu_mulai`; alokasi mengisi periode paling awal lebih dulu, melewati yang penuh atau habis; kandidat lain ditutup dengan notifikasi
  **Selesai bila**: pola transaksi mengikuti `research.md` R-04; kegagalan pada salah satu periode membatalkan seluruh pembentukan
  **Hati-hati**: pengurutan penguncian adalah pencegah deadlock, bukan kerapian. Tanpa penguncian, kode ini akan lolos seluruh pengujian manual dan baru terlihat sebagai kapasitas minus.

- [ ] T042 [P] [US3] Test balapan alokasi
  **Modul**: `backend/internal/order/`
  **FR**: FR-036, FR-079, FR-084, SC-018
  **Kemampuan**: dua kesepakatan berbarengan atas periode yang sama — hanya satu berhasil, yang gagal menerima alasan; constraint basis data menolak kapasitas terpakai melebihi total meski logika aplikasi dibuat keliru dengan sengaja
  **Selesai bila**: kedua test lulus; test constraint benar-benar membuktikan penolakan di tingkat penyimpanan data

- [ ] T043 [P] [US3] Test request kuota
  **Modul**: `backend/internal/quota/`
  **FR**: FR-029, FR-035, FR-082, FR-083
  **Kemampuan**: jalur berhasil, penolakan peran, masukan tidak sah, request ke diri sendiri, kapasitas kurang, request kedaluwarsa dengan `Clock` digantikan

- [ ] T044 [US3] Frontend: request dan perbandingan
  **Modul**: `frontend/src/pages/request/`
  **FR**: FR-029, FR-030, FR-032, FR-033
  **Kemampuan**: pilih kandidat dari hasil pencarian, form request, daftar request terkirim dengan status per kandidat, perbandingan penawaran berdampingan, aksi counter-offer dan terima
  **Selesai bila**: batas 72 jam terlihat sebagai informasi, bukan kolom masukan

- [ ] T045 [US3] Frontend: request masuk untuk subkontraktor
  **Modul**: `frontend/src/pages/request/`
  **FR**: FR-031, FR-035
  **Kemampuan**: daftar request masuk, penanda apakah kapasitas sampai deadline mencukupi, form penawaran, form penolakan beralasan

- [ ] T046 [US3] Skenario uji manual US3
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 11 langkah dari `quickstart.md` bagian F US3

**Checkpoint**: transaksi dapat terbentuk. Isi `changelog.md`.

---

## Phase 6: User Story 4 — Kalender Aktual (P4)

**Goal**: kalender ketersediaan berkurang otomatis saat pesanan dikonfirmasi dan kembali saat dibatalkan.

**Independent Test**: konfirmasi pesanan besar, pastikan kapasitas berkurang dari minggu terawal; batalkan sebelum produksi, pastikan seluruhnya kembali.

- [ ] T047 [US4] Pengelolaan kalender
  **Modul**: `backend/internal/listing/`
  **FR**: FR-017, FR-019, FR-021
  **Kemampuan**: baca dan perbarui beberapa periode sekaligus, tandai penuh, penanda kalender basi lebih dari 7 hari
  **Selesai bila**: `/listing/saya/periode` sesuai kontrak; penanda basi tidak mengubah urutan pencarian
  **Hati-hati**: `kalender_diperbarui_pada` terpisah dari `diperbarui_pada` — mengubah listing tidak boleh menghapus penanda basi.

- [ ] T048 [US4] Pembalikan alokasi
  **Modul**: `backend/internal/order/`
  **FR**: FR-020
  **Kemampuan**: membalik seluruh baris alokasi sebuah pesanan dalam satu transaksi, dengan pola penguncian yang sama seperti pembentukan
  **Selesai bila**: kapasitas setiap periode kembali ke angka sebelum pesanan terbentuk; baris alokasi ditandai dibalik, tidak dihapus

- [ ] T049 [US4] Penolakan yang bertabrakan dengan alokasi berjalan
  **Modul**: `backend/internal/listing/`
  **FR**: FR-017
  **Kemampuan**: menolak penurunan kapasitas di bawah yang sudah terpakai dan penandaan penuh atas periode yang sudah teralokasi, dengan pesan yang menyebut minggu mana beserta jumlahnya
  **Selesai bila**: kedua penolakan mengembalikan `KAPASITAS_SUDAH_TERALOKASI` atau `PERIODE_SUDAH_TERALOKASI`, bukan galat basis data mentah
  **Hati-hati**: constraint akan menolak dengan sendirinya. Tugas task ini menerjemahkannya menjadi pesan yang dapat dibaca pengguna.

- [ ] T050 [P] [US4] Test alokasi dan pembalikan
  **Modul**: `backend/internal/{listing,order}/`
  **FR**: FR-018, FR-020, FR-078
  **Kemampuan**: alokasi mengisi minggu terawal lebih dulu dan meluber dengan benar; melewati periode penuh; pembalikan memulihkan seluruh periode; penolakan yang bertabrakan dengan alokasi

- [ ] T051 [US4] Frontend: kalender
  **Modul**: `frontend/src/pages/listing/`
  **FR**: FR-017, FR-021
  **Kemampuan**: tampilan kalender mingguan, penyuntingan beberapa periode sekaligus, penanda penuh, penanda basi, rincian alokasi per periode
  **Selesai bila**: setiap periode jelas dimulai hari Senin; alokasi per minggu terlihat beserta pesanan yang memakainya

- [ ] T052 [US4] Skenario uji manual US4
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 11 langkah dari `quickstart.md` bagian F US4

**Checkpoint**: data kapasitas tetap aktual tanpa tindakan manual. Isi `changelog.md`.

---

## Phase 7: User Story 5 — Pesanan Sampai Tuntas (P5)

**Goal**: kedua pihak memantau pesanan dari diterima sampai dikonfirmasi, dengan pembatalan pra-produksi dan penutupan otomatis.

**Independent Test**: jalankan seluruh transisi status; pada pesanan kedua, biarkan status Dikirim melewati tujuh hari dan pastikan tertutup otomatis.

- [ ] T053 [US5] Mesin keadaan pesanan
  **Modul**: `backend/internal/order/`
  **FR**: FR-038, FR-039, FR-044
  **Kemampuan**: transisi sesuai diagram `data-model.md` §7; riwayat status dengan waktu dan pelaku; penolakan transisi melompat beserta daftar transisi yang diizinkan
  **Selesai bila**: `PesananDetail` mengirim `transisi_diizinkan` dan `boleh_dibatalkan_sendiri`; galat `TRANSISI_STATUS_TIDAK_SAH` menyebut urutan yang benar
  **Hati-hati**: perubahan oleh penjadwal ditandai `oleh_sistem`, bukan dibiarkan tanpa identitas.

- [ ] T054 [US5] Pembatalan
  **Modul**: `backend/internal/order/`
  **FR**: FR-065, FR-066, FR-072
  **Kemampuan**: pembatalan oleh kedua pihak selama status masih diterima, wajib beralasan, membalik seluruh alokasi; setelah produksi diarahkan ke sengketa
  **Selesai bila**: `dibatalkan_oleh_id` tercatat — itu dasar perhitungan tingkat penyelesaian; galat `PEMBATALAN_SETELAH_PRODUKSI` menyebutkan jalur alternatifnya

- [ ] T055 [US5] Konfirmasi otomatis tujuh hari
  **Modul**: `backend/internal/order/` + scheduler
  **FR**: FR-068, FR-069, FR-070
  **Kemampuan**: dua lapisan — dihitung saat pesanan dibaca, dan penjadwal untuk pemberitahuan tenggat mendekat serta penulisan status final; dihentikan oleh sengketa
  **Selesai bila**: satu fungsi domain dipakai kedua lapisan; `dikonfirmasi_otomatis` menandai yang mana
  **Hati-hati**: kalau perhitungan tenggat diduplikasi di beberapa handler, keduanya akan berbeda pada suatu titik dan pesanan yang sama tampak beda status di halaman berbeda.

- [ ] T056 [P] [US5] Catatan pembayaran
  **Modul**: `backend/internal/order/`
  **FR**: FR-040 sampai FR-043
  **Kemampuan**: catat pernyataan terkirim dan diterima, tanpa kolom jumlah uang; penanda perbedaan pernyataan antar pihak; keterangan bahwa platform tidak menjamin
  **Selesai bila**: tidak ada satu pun kolom jumlah uang maupun integrasi pembayaran
  **Hati-hati**: Batas Keuangan konstitusi. Mitigasi gagal bayar berupa escrow wajib pada dokumen sumber tidak berlaku di versi ini, dan itu keputusan yang sudah tercatat di Assumptions spec.

- [ ] T057 [P] [US5] Pelaporan sengketa
  **Modul**: `backend/internal/order/`
  **FR**: FR-046, FR-070
  **Kemampuan**: laporkan sengketa, hentikan hitungan konfirmasi otomatis, satu sengketa terbuka per pesanan
  **Selesai bila**: pelaporan berulang ditolak; hitungan otomatis benar-benar berhenti

- [ ] T058 [P] [US5] Test pesanan
  **Modul**: `backend/internal/order/`
  **FR**: FR-044, FR-065, FR-066, FR-068, FR-070
  **Kemampuan**: transisi melompat ditolak; pembatalan pra-produksi membalik alokasi; pembatalan pasca-produksi ditolak; konfirmasi otomatis dengan `Clock` digantikan; sengketa menghentikan hitungan
  **Hati-hati**: seluruh test tenggat memakai `Clock` yang digeser, bukan menunggu waktu nyata.

- [ ] T059 [US5] Frontend: dashboard pesanan
  **Modul**: `frontend/src/pages/pesanan/`
  **FR**: FR-038, FR-039, FR-041, FR-044
  **Kemampuan**: daftar aktif dan riwayat, detail dengan riwayat status, rincian alokasi per minggu, tombol transisi, form pembatalan, catatan pembayaran, tombol laporkan sengketa
  **Selesai bila**: tombol dirender dari `transisi_diizinkan` yang dikirim backend
  **Hati-hati**: **jangan** duplikasi mesin keadaan pesanan di React. Kalau logikanya ditulis ulang, dua tempat akan berbeda pada suatu titik.

- [ ] T060 [US5] Frontend: tenggat konfirmasi
  **Modul**: `frontend/src/pages/pesanan/`
  **FR**: FR-068, FR-069
  **Kemampuan**: tanggal pesanan akan dianggap diterima ditampilkan jelas pada pesanan berstatus Dikirim; penanda bahwa penutupan terjadi otomatis

- [ ] T061 [US5] Skenario uji manual US5
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 13 langkah dari `quickstart.md` bagian F US5

**Checkpoint**: transaksi dapat diselesaikan. Isi `changelog.md`.

---

## Phase 8: User Story 6 — Reputasi (P6)

**Goal**: reputasi terbentuk dari transaksi nyata, dan pembatalan hanya membebani pihak yang membatalkan.

**Independent Test**: selesaikan pesanan, isi rating dari kedua sisi; batalkan pesanan lain dari satu sisi, pastikan hanya tingkat penyelesaian pihak itu yang turun.

- [ ] T062 [US6] Ulasan
  **Modul**: `backend/internal/reputation/`
  **FR**: FR-047, FR-049, FR-050
  **Kemampuan**: rating 1–5 dan teks hanya atas pesanan yang sudah dikonfirmasi, satu kali per pesanan per pihak, tidak anonim
  **Selesai bila**: ulasan atas pesanan belum selesai ditolak; ulasan atas usaha yang belum pernah bertransaksi ditolak
  **Hati-hati**: pemeriksaan status pesanan tidak dapat ditegakkan `CHECK` karena merujuk tabel lain. Wajib di aplikasi.

- [ ] T063 [US6] Nilai turunan reputasi
  **Modul**: `backend/internal/reputation/`
  **FR**: FR-048, FR-071, FR-072, FR-073
  **Kemampuan**: rating rata-rata mengecualikan ulasan yang disembunyikan; tingkat penyelesaian dihitung saat dibaca; pembatalan masuk pembagi hanya bagi pihak yang membatalkan; persentase ditahan sampai 3 pesanan disepakati
  **Selesai bila**: kueri mengikuti `data-model.md` §8; kedua angka penyusun selalu dikirim
  **Hati-hati**: dihitung saat dibaca, **bukan** disimpan sebagai kolom. Kolom yang harus diperbarui setiap kali ulasan disembunyikan atau pesanan dibatalkan adalah sumber ketidaksesuaian yang paling sering muncul.

- [ ] T064 [P] [US6] Test reputasi
  **Modul**: `backend/internal/reputation/`
  **FR**: FR-047, FR-050, FR-071, FR-072, FR-073
  **Kemampuan**: pembatalan menurunkan tingkat penyelesaian pihak yang membatalkan dan **tidak** mempengaruhi pihak lain; ulasan disembunyikan keluar dari rata-rata; ambang 3 pesanan
  **Hati-hati**: test FR-072 adalah yang membedakan aturan ini dari perhitungan biasa. Wajib ada.

- [ ] T065 [US6] Frontend: ulasan dan reputasi
  **Modul**: `frontend/src/pages/{pesanan,profil}/`
  **FR**: FR-047, FR-048, FR-049, FR-073
  **Kemampuan**: form rating setelah pesanan dikonfirmasi, daftar ulasan pada profil dengan nama pemberi dan tanggal transaksi, ringkasan reputasi
  **Selesai bila**: bila `cukup_data: false`, yang tampil adalah keterangannya, bukan persentase

- [ ] T066 [US6] Skenario uji manual US6
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 9 langkah dari `quickstart.md` bagian F US6

**Checkpoint**: trust antar pihak yang belum saling kenal punya dasar yang dapat diperiksa [1]. Isi `changelog.md`.

---

## Phase 9: User Story 7 — Admin (P7)

**Goal**: admin mengelola daftar baku, memberi lencana verifikasi, dan menengahi sengketa.

**Independent Test**: tambah jenis produk dan pastikan langsung dapat dipilih; setujui dan tolak satu pengajuan verifikasi, pastikan listing tetap tayang pada kedua kasus.

- [ ] T067 [P] [US7] Verifikasi identitas
  **Modul**: `backend/internal/{account,admin}/`
  **FR**: FR-006 sampai FR-011
  **Kemampuan**: pengajuan dengan dua berkas, antrean admin, setujui atau tolak beralasan, lencana, pengajuan ulang setelah penolakan
  **Selesai bila**: satu pengajuan menunggu per profil; penolakan wajib beralasan; listing tetap tayang apa pun keputusannya
  **Hati-hati**: verifikasi **bukan gerbang**. Ini menyimpang dari kriteria penerimaan dokumen sumber yang menempatkan status "Menunggu Verifikasi" pada alur listing [1]; penyimpangannya sudah tercatat di Assumptions spec.

- [ ] T068 [P] [US7] Pengelolaan daftar baku
  **Modul**: `backend/internal/{masterdata,admin}/`
  **FR**: FR-059, FR-060, FR-061
  **Kemampuan**: tambah, ubah nama, nonaktifkan item; keputusan atas usulan pengguna
  **Selesai bila**: item nonaktif tidak dapat dipilih untuk listing baru, sementara listing yang sudah memakainya tetap utuh dan tetap dapat ditemukan

- [ ] T069 [P] [US7] Moderasi ulasan
  **Modul**: `backend/internal/admin/`
  **FR**: FR-050
  **Kemampuan**: sembunyikan ulasan dengan alasan, tercatat
  **Selesai bila**: ulasan hilang dari profil publik dan rata-rata rating berubah

- [ ] T070 [P] [US7] Pemantauan pesanan telat
  **Modul**: `backend/internal/admin/` + scheduler
  **FR**: FR-045
  **Kemampuan**: daftar pesanan melewati deadline, notifikasi ke kedua pihak
  **Selesai bila**: pesanan berstatus Produksi yang melewati deadline muncul di daftar admin [1]

- [ ] T071 [US7] Mediasi sengketa
  **Modul**: `backend/internal/admin/`
  **FR**: FR-046, FR-067
  **Kemampuan**: tandai Dalam Mediasi, baca riwayat lengkap termasuk catatan pembayaran dan perbedaan pernyataan, tutup mediasi
  **Selesai bila**: penutupan sebagai dibatalkan **wajib** menyertakan keputusan pengembalian alokasi dan pihak penanggung; tanpa keduanya ditolak
  **Hati-hati**: constraint `penyelesaian_lengkap` menegakkan ini di basis data. Antarmuka harus meminta keduanya secara eksplisit, bukan memberi nilai bawaan.

- [ ] T072 [P] [US7] Test admin
  **Modul**: `backend/internal/admin/`
  **FR**: FR-007, FR-050, FR-060, FR-067
  **Kemampuan**: non-admin ditolak pada setiap endpoint admin; item nonaktif tidak merusak listing; mediasi tanpa keputusan lengkap ditolak; berkas identitas milik usaha lain tidak dapat diakses

- [ ] T073 [US7] Frontend: panel admin
  **Modul**: `frontend/src/pages/admin/`
  **FR**: FR-007, FR-050, FR-059, FR-061, FR-045, FR-046, FR-067
  **Kemampuan**: enam layar — antrean verifikasi dengan pratayang berkas, kelola daftar baku, antrean usulan, moderasi ulasan, pesanan telat, mediasi sengketa
  **Selesai bila**: setiap layar dapat diselesaikan tanpa menyentuh basis data
  **Hati-hati**: pengisian awal daftar baku dan wilayah lewat perintah seed, bukan lewat antarmuka — itu yang membuat enam layar ini tetap layak di prioritas terakhir.

- [ ] T074 [US7] Skenario uji manual US7
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: 17 langkah dari `quickstart.md` bagian F US7

**Checkpoint**: seluruh tujuh story berfungsi. Isi `changelog.md`.

---

## Phase 10: Polish dan Penyiapan Demo

- [ ] T075 Data uji
  **Modul**: `backend/cmd/devotion/`
  **Kemampuan**: `seed:test-data` menyiapkan 50 usaha, kandidat 500 potong per minggu untuk skenario 3.000 potong, kalender basi 8 hari, request kedaluwarsa, pesanan Dikirim 6 hari dan 8 hari lalu, pesanan telat, dua pengajuan verifikasi, usaha dengan hanya 2 pesanan; `reset:test-data` memulihkan
  **Selesai bila**: perintah **menolak berjalan** saat `APP_ENV=production`; akun uji memakai domain `.test`
  **Hati-hati**: alur bertenggat disiapkan sebagai data yang sudah berada pada keadaan itu, bukan lewat kendali geser waktu — data yang di-seed tidak berisiko terbawa ke lingkungan sungguhan.

- [ ] T076 [P] Dokumen skenario uji manual final
  **Modul**: `docs/skenario-uji-manual.md`
  **Kemampuan**: seluruh 73 langkah tergabung, bagian "di luar cakupan pengujian Anda" yang menyatakan WhatsApp dan email bukan temuan, kredensial akun uji, cara melaporkan temuan
  **Selesai bila**: dapat diikuti orang yang belum pernah melihat sistem ini; setiap langkah menyebut akun yang dipakai

- [ ] T077 [P] Runbook VPS
  **Modul**: `docs/setup-vps.md`
  **Kemampuan**: 16 langkah dari `quickstart.md` bagian B
  **Selesai bila**: langkah verifikasi firewall disertakan sebagai gerbang, bukan saran

- [ ] T078 [P] Aksesibilitas dan bahasa
  **Modul**: `frontend/src/`
  **FR**: FR-055, FR-056
  **Kemampuan**: seluruh alur inti dapat diselesaikan dengan keyboard, label jelas, terbaca pembaca layar, mobile-first
  **Selesai bila**: tidak ada teks bahasa Inggris yang tampil ke pengguna

- [ ] T079 [P] Dokumentasi penutup
  **Modul**: `docs/`
  **Kemampuan**: `dependencies.md` lengkap dengan alasan setiap paket, `utang-teknis.md`, `layanan-luar.md` beserta akibat bila mati, `changelog.md` final, `menjalankan.md`, `pengujian.md`
  **Selesai bila**: seluruh dependency terdaftar dengan alasannya, dan tidak ada yang di luar `plan.md`

- [ ] T080 Penerapan ke VPS dan cadangan
  **Modul**: server
  **Kemampuan**: jalankan runbook sampai selesai, seed wilayah dan daftar baku, buat admin, sambungkan WhatsApp, pasang cron `pg_dump`, salin cadangan keluar VPS, daftarkan pemantau uptime
  **Selesai bila**: `/api/health` sehat; `curl` ke IP VPS langsung **gagal**; cadangan manual berhasil dan salinannya ada di luar server

- [ ] T081 Snapshot dan checklist penjurian
  **Modul**: server + `docs/`
  **Kemampuan**: jalankan checklist `quickstart.md` bagian H, ambil snapshot VPS setelah semuanya terverifikasi
  **Selesai bila**: seluruh butir checklist tercentang; snapshot ada
  **Hati-hati**: ini yang paling penting dari seluruh fase. Satu server tanpa cadangan berarti satu kesalahan dapat menghapus seluruh submission.

---

## Dependensi

### Antar fase

- Setup: tanpa dependensi
- Foundational: setelah Setup — **memblokir seluruh story**
- Story: seluruhnya setelah Foundational, lalu berurutan P1 → P7, atau paralel bila ada beberapa pelaksana
- Polish: setelah story yang diinginkan selesai

### Antar story

| Story | Dapat dimulai setelah | Catatan |
|-------|----------------------|---------|
| US1 | Foundational | Tanpa dependensi story lain |
| US2 | Foundational | Butuh listing untuk data uji, dapat dipakai data seed |
| US3 | Foundational | Butuh listing dan pencarian untuk alur penuh |
| US4 | Foundational | T048 dan T049 menyentuh alokasi dari T041 |
| US5 | Foundational | Butuh pesanan dari T041 |
| US6 | Foundational | Butuh pesanan yang dikonfirmasi dari T055 |
| US7 | Foundational | Paling independen; lapisan kontrol di atas alur yang sudah jalan |

### Di dalam satu story

Query dan model → service → handler → test. Frontend setelah endpoint tersedia, **atau** paralel dengan mock server yang dihasilkan dari `openapi.yaml` (Prism atau MSW) — ini keuntungan nyata dari membekukan kontrak lebih dulu.

### Peluang paralel

- Seluruh task Setup bertanda `[P]` kecuali T001 dan T007
- Di Foundational: T012, T013, T016, T017, T025 dapat paralel setelah T008–T011
- Setelah Foundational: tujuh story dapat paralel bila ada beberapa pelaksana
- Backend dan frontend di dalam satu story dapat paralel dengan mock server
- Task test bertanda `[P]` karena modulnya berbeda dari implementasi

### Memotong per pelaksana

Saring berdasarkan prefiks **Modul**: sesi backend mengambil task ber-`backend/`, sesi frontend mengambil `frontend/`. Label story tetap sama di keduanya, jadi keterlacakan ke FR tidak hilang.

---

## Strategi Pelaksanaan

**MVP lebih dulu**: Setup → Foundational → US1 → berhenti dan buktikan US1 berjalan sendiri → demo.

**Bertahap**: setiap story menambah nilai tanpa merusak yang sebelumnya. Berhenti di setiap checkpoint, jalankan skenario uji manual story itu, isi `changelog.md`.

**Bila tenggat menekan**, urutan pemangkasan yang paling sedikit merusak:

1. Sentry di frontend (T025 sebagian) — monitoring tidak menambah nilai penjurian
2. US7 layar moderasi ulasan dan pesanan telat (T073 sebagian) — sisakan verifikasi dan mediasi, keduanya ada di Acceptance Scenario
3. US6 seluruhnya — reputasi baru bermakna setelah ada transaksi

Yang **tidak boleh** dipangkas: US1 sampai US3, karena tanpa ketiganya tidak ada alur yang dapat didemokan; dan seluruh test pada T036, T042, T050, T058, T064, karena itu aturan yang paling mudah rusak diam-diam.

Setiap pemangkasan dicatat di `docs/utang-teknis.md`. Pengecualian kewajiban pengujian hanya boleh per story, dengan catatan di Complexity Tracking `plan.md` yang menyebut story mana dan risiko apa yang ditanggung. Pengecualian menyeluruh tidak diizinkan.

---

## Catatan

- Satu task = satu modul + satu kelompok FR. Pemecahan file di dalam modul diserahkan pelaksana kecuali empat path yang dipatok.
- `[P]` bermakna di tingkat modul: dua task `[P]` tidak akan menulis modul yang sama.
- Bila sebuah task terasa perlu menyentuh modul di luar yang tertulis, itu tanda batas modulnya keliru — angkat lebih dulu, jangan diam-diam menyimpang.
- Setiap test menyebut FR yang diuji pada namanya.
- Commit per task atau per kelompok yang logis.
- Berhenti di setiap checkpoint dan buktikan story itu berdiri sendiri.
- Hindari: task kabur, dua task menulis modul yang sama, dan ketergantungan lintas story yang merusak kemandirian.