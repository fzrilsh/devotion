# Implementation Plan: Capacity Exchange — Marketplace Subkontrak Kapasitas Konveksi (MVP)

**Branch**: `001-capacity-exchange-marketplace` | **Date**: 2026-08-21 | **Spec**: `specs/001-capacity-exchange-marketplace/spec.md`

**Input**: Feature specification from `specs/001-capacity-exchange-marketplace/spec.md`

**Constitution**: Devotion Constitution v2.1.0

## Summary

Membangun Devotion, platform web responsif tempat UMKM konveksi mendaftarkan kapasitas produksi yang menganggur dan UMKM yang kelebihan order mencarinya berdasarkan kecocokan keras, mengirim request kuota ke beberapa kandidat, bernegosiasi harga, lalu mengeksekusi pesanan sampai tuntas dengan reputasi yang terbentuk dari transaksi nyata. Masalah yang dijawab: pencarian subkontraktor yang hanya lewat relasi personal sehingga jangkauannya terbatas dan tidak ada mekanisme matching sistematis [1], listing kapasitas statis yang cepat kedaluwarsa sehingga transaksi gagal [1], dan tidak adanya sistem reputasi antar UMKM yang belum saling kenal [1].

Pendekatan teknis: satu biner Go yang menyajikan frontend React hasil build, menghabiskan TLS sendiri dengan Cloudflare Origin Certificate, dan berbicara ke satu PostgreSQL. Dua layanan di `docker-compose.yml`, tidak ada proses ketiga. Modularitas lewat batas paket internal, bukan batas jaringan. Pekerjaan terjadwal ditangani penjadwal di dalam proses yang sama, dengan perhitungan saat baca sebagai lapisan kedua agar tidak ada tenggat yang terlewat bila proses sempat mati.

## Technical Context

**Language/Version**: Go 1.22+ (backend), TypeScript 5.x pada React 18 (frontend)

**Primary Dependencies**:

Backend — `net/http` (router bawaan, pola path + metode sejak 1.22), `jackc/pgx/v5` (driver Postgres), `sqlc` (generator kode dari SQL, perkakas build), `golang-migrate` (migrasi), `golang.org/x/crypto/bcrypt` (hash kata sandi), `go.mau.fi/whatsmeow` (WhatsApp), `getsentry/sentry-go` (pelacak error), `log/slog` + `net/smtp` + `image/jpeg` + `crypto/rand` (standard library, nol dependency tambahan untuk log terstruktur, email, pengolahan gambar, dan token sesi)

Frontend — Vite, React, TanStack Query (pengambilan data server), Zod + React Hook Form (validasi form), Tailwind CSS (gaya), Leaflet + tile OpenStreetMap (peta), `@sentry/react`

**Storage**: PostgreSQL 16 untuk seluruh data termasuk store sesi whatsmeow; berkas unggahan pada volume disk VPS (maksimal 5MB per berkas, total maksimal 500MB), dilayani hanya lewat handler Go yang memeriksa peran

**Testing**: `go test` dengan skema Postgres terpisah pada layanan basis data yang sama (backend), Jest (frontend), pengujian end-to-end manual oleh penguji di luar tim mengikuti dokumen skenario

**Target Platform**: VPS Linux tunggal, 2GB RAM, 50GB storage, fresh install; Docker Compose dua layanan; Cloudflare sebagai DNS dan proxy tepi dengan mode SSL/TLS Full (strict)

**Project Type**: Web application monolith — satu repository, satu deployable, frontend disematkan ke dalam biner backend

**Performance Goals**: Hasil pencarian tampil dalam 3 detik pada koneksi seluler lambat (SC-010); pencarian yang diulang menghasilkan urutan identik termasuk antar halaman (SC-013)

**Constraints**: 2GB RAM dan 50GB disk pada satu server; maksimal 2 layanan runtime; dilarang memproses dana pihak mana pun; dilarang membangun artefak di server; ukuran log kontainer dan total unggahan wajib dibatasi

**Scale/Scope**: 86 functional requirement, 7 user story, 16 entitas, 19 success criteria. Data demo sekitar 50 usaha; SC-003 menargetkan 200 usaha aktif pada bulan ketiga sebagai sasaran bisnis, mengikuti proyeksi sekitar 1.500 UMKM aktif pada tahun ketiga [1]

**Anggaran memori pada 2GB** (perkiraan, wajib diverifikasi saat penyiapan server):

| Komponen | Perkiraan |
|----------|-----------|
| PostgreSQL (`max_connections` diturunkan ke 20) | 300–400MB |
| Go + whatsmeow | 150–250MB |
| Sistem operasi dan Docker | 250–350MB |
| Sisa untuk lonjakan | ~1GB |
| Swap yang disiapkan | 2GB |

**NEEDS CLARIFICATION** — tiga hal yang belum dapat saya pastikan dan akan diselesaikan di `research.md`:

1. Cara paling andal menolak koneksi yang tidak datang dari Cloudflare pada mode Full (strict): pembatasan firewall ke rentang alamat Cloudflare, Authenticated Origin Pulls, atau keduanya. Rentang alamat Cloudflare juga perlu diambil dan dipatok beserta tanggal pengambilannya.
2. Bentuk dan ketersediaan respons wilayah.id saat ini, serta apakah kode wilayahnya sesuai kode resmi BPS/Kemendagri. Saya belum memeriksa endpoint-nya langsung.
3. Angka penyetelan PostgreSQL yang tepat untuk 2GB (`shared_buffers`, `work_mem`, `effective_cache_size`) beserta batas pool koneksi di sisi Go.

Batas dan kuota layanan luar (rate limiting Cloudflare paket gratis, batas ukuran body proxy, kuota harian Mailjet paket gratis) tidak saya hafal dan dapat berubah; semuanya perlu diperiksa langsung di dasbor masing-masing, bukan diasumsikan dari dokumen ini.

## Constitution Check

*GATE: Wajib lolos sebelum Phase 0. Diperiksa ulang setelah Phase 1.*

### Gate I — Monolith-First (NON-NEGOTIABLE)

| Aturan | Rencana | Status |
|--------|---------|--------|
| Backend dan frontend satu repository | `backend/` dan `frontend/` dalam satu repo | LOLOS |
| Frontend disajikan proses backend yang sama | Hasil build Vite disematkan lewat `embed.FS`, dilayani handler Go dengan fallback SPA | LOLOS |
| Maksimal dua layanan runtime | `docker-compose.yml`: `backend`, `postgres` | LOLOS |
| Tidak ada broker, worker, cron, cache, proxy sebagai proses | Tidak ada. TLS oleh Go sendiri dengan Cloudflare Origin Certificate | LOLOS |
| Pekerjaan terjadwal tanpa proses kedua | Penjadwal `time.Ticker` di dalam proses backend, ditambah perhitungan saat baca sebagai lapisan kedua | LOLOS |
| Notifikasi di dalam proses yang sama | Goroutine pengirim membaca antrean dari tabel notifikasi; kegagalan tidak menggagalkan transaksi (FR-086) | LOLOS |
| Antar modul lewat pemanggilan fungsi | Paket `internal/*` saling memanggil langsung; tidak ada HTTP ke diri sendiri | LOLOS |
| Perkakas pengembangan bukan proses runtime | `sqlc`, `golang-migrate` CLI, Jest, `go test` hanya saat build dan uji | LOLOS |
| Perintah sekali jalan lewat subcommand | `devotion serve`, `admin:create`, `seed:master-data`, `seed:wilayah`, `seed:test-data`, `reset:test-data` | LOLOS |
| Pengujian tanpa layanan basis data tambahan | Skema `test_*` pada layanan Postgres yang sama | LOLOS |
| Layanan luar dicatat, tidak dihitung | Cloudflare, Mailjet, Sentry, pemantau uptime, wilayah.id — dicatat di `docs/layanan-luar.md` beserta akibat bila mati | LOLOS |

### Gate II — Demo-Ready Over Complete

| Aturan | Rencana | Status |
|--------|---------|--------|
| Setiap story dapat didemokan lewat antarmuka | Tujuh story punya halaman sendiri; tidak ada Acceptance Scenario yang menuntut akses basis data manual | LOLOS |
| Data contoh untuk keadaan berhasil dan gagal | `seed:test-data` menyiapkan keduanya, termasuk hasil pencarian kosong dan penawaran yang tertolak karena kapasitas | LOLOS |
| Data acuan terisi dengan satu perintah | `seed:wilayah` dan `seed:master-data` | LOLOS |
| Demo tidak bergantung layanan yang bisa mati | Notifikasi di dalam platform (FR-054) menjadi jalur pengamatan; WhatsApp dan email boleh gagal tanpa merusak alur | LOLOS |

Turunan yang mengikat urutan pengerjaan: pengisian daftar baku dan wilayah adalah prasyarat data bagi User Story 1 dan 2, sehingga masuk fase Foundational, terpisah dari antarmuka admin yang tetap di prioritas terakhir.

### Gate III — Traceability to Spec

Setiap task di `tasks.md` akan mencantumkan FR atau user story. Setiap pengujian menyebutkan FR pada namanya. Setiap skenario penguji manual menunjuk Acceptance Scenario. Kontrak API di `contracts/` akan memetakan setiap endpoint ke FR yang dilayaninya. **LOLOS** — dapat ditegakkan, diperiksa saat `/tasks`.

### Gate IV — Minimal Dependencies

Setiap dependency dan pembenarannya. Yang dapat diselesaikan standard library sengaja tidak memakai dependency.

| Dependency | Pembenaran | Alternatif yang ditolak |
|------------|------------|-------------------------|
| `pgx/v5` | Driver Postgres yang matang; mendukung penguncian baris dan transaksi yang dibutuhkan FR-036 | `database/sql` + `lib/pq` — kurang mendukung tipe Postgres dan lebih lambat |
| `sqlc` | Query pencarian dan skor kecocokan harus eksplisit dan deterministik (FR-023 sampai FR-025); generator menjaga SQL tetap terlihat | GORM — menyembunyikan SQL, justru pada bagian paling penting |
| `golang-migrate` | Migrasi berversi yang dapat dijalankan otomatis saat startup | Menulis migrasi sendiri — sekitar 200 baris kode yang tidak perlu |
| `bcrypt` | Hash kata sandi yang memang ditujukan untuk itu | argon2id — lebih baik, tetapi biaya memorinya perlu disetel hati-hati pada 2GB |
| `whatsmeow` | Satu-satunya cara mengirim WhatsApp tanpa API resmi yang butuh verifikasi bisnis; berjalan sebagai library, bukan layanan | API resmi Meta — verifikasi bisnis tidak akan selesai sebelum tenggat |
| `sentry-go`, `@sentry/react` | Mengetahui adanya kerusakan sebelum juri menemukannya | Self-host Sentry — menuntut beberapa layanan, melanggar Gate I |
| TanStack Query | Daftar pesanan, hasil pencarian, dan status per kandidat semuanya perlu disegarkan; menghemat banyak kode keadaan | `useEffect` manual — menulis ulang cache, invalidasi, dan keadaan galat |
| Zod + React Hook Form | Validasi form dengan skema yang sama dipakai memeriksa bentuk respons | Validasi manual — rawan dan berulang pada belasan form |
| Tailwind CSS | FR-055 menuntut mobile-first; menulis CSS sendiri lebih lambat | Component library — menambah bobot; hanya dipilih bila tenggat menekan |
| Leaflet + OpenStreetMap | FR-064 menampilkan titik lokasi; tanpa kunci API dan tanpa penagihan | Google Maps, Mapbox — menuntut kunci dan kartu kredit |

Diselesaikan tanpa dependency: log terstruktur (`log/slog`), pengiriman email (`net/smtp`), pembuangan metadata gambar dan pengubahan ukuran (`image/jpeg`, `image/png` — dekode lalu enkode ulang membuang EXIF), token sesi (`crypto/rand`), pengenal baris (`gen_random_uuid()` bawaan Postgres), perhitungan jarak haversine (aritmetika sendiri, bukan PostGIS), pembatasan laju domain (tabel Postgres), router (`net/http`).

**LOLOS** dengan catatan: sebelas dependency untuk project seukuran ini masih dapat dipertanggungjawabkan, tetapi lima di antaranya milik frontend. Bila tenggat menekan, kandidat pertama yang dipangkas adalah Sentry di sisi frontend.

### Gate V — Deterministic Behavior

| Aturan | Rencana | Status |
|--------|---------|--------|
| Urutan hasil identik pada pengulangan, termasuk antar halaman | Pengurutan penuh oleh SQL dengan pemecah seri berakhir pada pengenal listing (FR-025); paginasi memakai keyset, bukan offset | LOLOS |
| Skor hanya dari kriteria keras | Empat kriteria FR-023 dihitung sebagai empat nilai boolean yang dijumlahkan; tidak ada bobot | LOLOS |
| Setiap keputusan dapat dijelaskan satu kalimat | Respons pencarian menyertakan status per kriteria (FR-026) | LOLOS |
| Satu sumber waktu yang dapat digantikan | Interface `Clock` disuntikkan ke setiap service; `time.Now()` dilarang di dalam logika bisnis | LOLOS |
| Batas minggu Senin, Asia/Jakarta, tipe tanggal | Kolom `date` untuk awal periode; seluruh pergeseran minggu dihitung di WIB; waktu kejadian `timestamptz` | LOLOS |
| Uang bilangan bulat rupiah | Kolom `bigint`, tanpa pecahan | LOLOS |
| Data acuan dari luar diambil sekali dan disalin ke repo | `seed:wilayah` mengambil dari wilayah.id, menyimpan ke Postgres, sekaligus menulis `docs/master-data/wilayah.json` sebagai cadangan | LOLOS |

### Batasan Tambahan

| Batasan | Rencana | Status |
|---------|---------|--------|
| Batas keuangan | Tidak ada payment gateway; hanya Catatan Pembayaran sebagai pernyataan pengguna (FR-040 sampai FR-043) | LOLOS |
| Unggahan tidak lewat path statis | Handler `GET /api/files/{id}` memeriksa peran sebelum mengirim byte | LOLOS |
| Nama berkas dibuat sistem, tipe divalidasi dari isi | UUID sebagai nama; tipe diperiksa dari magic bytes | LOLOS |
| Metadata lokasi gambar dibuang | Dekode dan enkode ulang saat unggah | LOLOS |
| Segmen origin terenkripsi, koneksi non-Cloudflare ditolak | Cloudflare Origin Certificate + Full (strict) + pembatasan firewall | LOLOS, rinciannya menunggu Phase 0 |
| Alamat asal hanya dipercaya dari rentang Cloudflare | Middleware memeriksa `RemoteAddr` terhadap rentang yang dipatok sebelum membaca header | LOLOS |
| Pembatasan laju berbasis data domain | Per akun untuk percobaan masuk, per nomor untuk kode sekali pakai, per pengguna untuk request kuota | LOLOS |
| Kredensial dan nomor layanan tidak di repo | Seluruhnya variabel lingkungan; `.env.example` hanya memuat nama kunci tanpa nilai | LOLOS |
| Membangun artefak tidak di server | GitHub Actions membangun image, VPS hanya menarik dan menjalankan | LOLOS |
| Ukuran log dibatasi | `max-size` dan `max-file` pada kedua layanan compose | LOLOS |
| Cadangan terjadwal, salinan di luar server, jumlah dibatasi | `pg_dump` harian lewat cron host, disalurkan ke gzip, tiga salinan terakhir, disalin keluar VPS | PERLU DICATAT — lihat Complexity Tracking |

### Hasil Gate

Lolos untuk melanjutkan ke Phase 0, dengan dua hal tercatat di Complexity Tracking dan tiga hal yang harus diselesaikan `research.md`.

## Project Structure

### Documentation (this feature)

```text
docs/specs/001-capacity-exchange-marketplace/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md

docs/memory/
└── constitution.md
```

### Source Code (repository root)

```text
devotion/
├── README.md                       # template panitia, struktur tidak diubah
├── LICENSE                         # MIT
├── docker-compose.yml              # tepat 2 layanan: backend, postgres
├── .env.example                    # nama variabel tanpa nilai
├── .github/workflows/ci.yml        # vet, test, build image, deploy
├── backend/
│   ├── cmd/devotion/main.go        # subcommand: serve, admin:create, seed:*, reset:test-data
│   ├── internal/
│   │   ├── platform/               # clock, config, httpx, session, storage, scheduler, ratelimit, cloudflare
│   │   ├── account/                # akun, peran, profil usaha, verifikasi identitas   (US1, US7)
│   │   ├── masterdata/             # daftar baku, wilayah, usulan item                 (US1, US2, US7)
│   │   ├── listing/                # listing kapasitas, periode ketersediaan           (US1, US4)
│   │   ├── search/                 # kriteria keras, skor, pemecah seri, keyset page    (US2)
│   │   ├── quota/                  # request kuota, penawaran, counter-offer            (US3)
│   │   ├── order/                  # pesanan, alokasi kapasitas, pembatalan, pembayaran (US4, US5)
│   │   ├── reputation/             # ulasan, tingkat penyelesaian                       (US6)
│   │   ├── notification/           # antrean, pengirim email/WA/in-app, percobaan ulang (semua)
│   │   └── admin/                  # verifikasi, moderasi, mediasi                      (US7)
│   ├── db/
│   │   ├── migrations/             # golang-migrate, berurutan
│   │   └── queries/                # sumber SQL untuk sqlc
│   ├── webdist/                    # hasil build frontend, disematkan embed.FS
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── pages/                  # satu berkas per layar, dikelompokkan per user story
│   │   ├── components/
│   │   ├── api/                    # klien, tipe dari OpenAPI, hook TanStack Query
│   │   ├── schemas/                # skema Zod
│   │   └── lib/
│   ├── package.json
│   └── vite.config.ts
└── docs/
    ├── setup-vps.md                # dari fresh install sampai aplikasi jalan
    ├── menjalankan.md
    ├── pengujian.md
    ├── skenario-uji-manual.md      # untuk penguji eksternal, beserta akun uji
    ├── temuan-penguji.md
    ├── layanan-luar.md             # Cloudflare, Mailjet, Sentry, wilayah.id, pemantau uptime
    ├── dependencies.md             # alasan setiap dependency
    ├── utang-teknis.md
    └── master-data/                # wilayah.json, jenis-produk.json, jenis-mesin.json
```

**Structure Decision**: Monolith dua bagian dalam satu repository, dengan `backend/` sebagai satu-satunya proses aplikasi. Frontend dibangun di CI, hasilnya disalin ke `backend/webdist/`, lalu disematkan ke dalam biner lewat `embed.FS`. Handler statis melayani berkas yang ada dan mengembalikan `index.html` untuk path yang bukan `/api/*` agar penyegaran halaman dalam tidak menghasilkan 404.

Batas modul mengikuti batas user story sedapat mungkin, sehingga setiap fase `tasks.md` menyentuh paket yang jelas. Tiga paket sengaja dipakai bersama beberapa story: `masterdata` menjadi prasyarat US1 dan US2, `notification` dipakai hampir semua story, dan `platform` memuat `Clock` yang wajib dapat digantikan saat pengujian.

Konsekuensi arah ketergantungan yang perlu dijaga: `order` mengubah kapasitas milik `listing` lewat baris Alokasi Kapasitas (FR-077), sehingga `order` bergantung pada `listing`, bukan sebaliknya. `search` hanya membaca. Tidak ada ketergantungan melingkar.

## Complexity Tracking

Dua penyimpangan dari konstitusi yang perlu dicatat.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| Direktori tingkat atas `specs/` dan `memory/` di luar daftar yang diizinkan Batasan Struktur Repository | Artefak spec-driven development ini adalah bagian dari penilaian kualitas dokumen, dan `specs/NNN-*` adalah konvensi yang dirujuk seluruh artefak termasuk `plan.md` ini | Memindahkannya ke `docs/specs/` dan `docs/memory/` akan memenuhi konstitusi secara harfiah, tetapi memutus konvensi yang sudah dirujuk lintas berkas. **Rekomendasi: pindahkan ke `docs/` bila panitia menilai struktur direktori secara ketat, atau amandemen konstitusi menjadi 2.2.0 untuk memasukkan keduanya ke daftar yang diizinkan.** Keputusan pemilik project |
| Cron di tingkat host untuk `pg_dump` harian | Konstitusi mewajibkan cadangan terjadwal dengan salinan di luar server, sementara Gate I melarang proses terjadwal kedua. Cron host bukan bagian dari aplikasi dan tidak muncul di `docker-compose.yml`, tetapi tetap merupakan pekerjaan terjadwal di server yang sama | Penjadwal di dalam proses backend ditolak karena `pg_dump` harus berjalan meski aplikasi sedang mati atau rusak — justru saat itulah cadangan paling dibutuhkan. Menambah layanan cron ke compose ditolak karena melanggar batas dua layanan |

Dua hal berikut **bukan** pelanggaran tetapi wajib tercatat di `docs/layanan-luar.md` sesuai Gate I: Cloudflare, Mailjet, Sentry, pemantau uptime, dan wilayah.id semuanya berjalan di luar server dan tidak dihitung sebagai proses runtime.

Satu risiko yang tidak melanggar aturan apa pun tetapi perlu terlihat di rencana: whatsmeow memakai protokol WhatsApp Web, bukan API resmi. Nomor yang mengirim kode sekali pakai ke banyak penerima baru berisiko diblokir, dan sesinya dapat lepas kapan saja. FR-002 menjadikan verifikasi nomor HP sebagai gerbang, sehingga blokir nomor berarti tidak ada akun baru yang dapat dibuat saat demo. Mitigasi yang masuk rencana: halaman admin yang menampilkan QR dan status sambungan agar dapat disambungkan ulang tanpa akses SSH, subcommand darurat untuk memverifikasi akun secara manual, dan email yang tetap dipertahankan sebagai kanal kedua. Ini akan dicatat sebagai risiko di `research.md`, bukan sebagai utang teknis, karena keputusannya sadar dan sudah dimitigasi.

## Yang Berikutnya

`plan.md` berhenti di sini sesuai aturan satu artefak per giliran. Urutan berikutnya:

1. **`research.md`** (Phase 0) — menyelesaikan tiga NEEDS CLARIFICATION di Technical Context, ditambah keputusan tercatat untuk: pola penguncian baris pada alokasi kapasitas lintas periode, bentuk keyset pagination yang menjaga SC-013, cara menyematkan frontend beserta fallback SPA, strategi penjadwal dalam proses berpasangan dengan perhitungan saat baca, ketahanan sesi whatsmeow, dan penyiapan SPF/DKIM/DMARC di Cloudflare untuk Mailjet.
2. **`data-model.md`, `contracts/`, `quickstart.md`** (Phase 1) — termasuk runbook VPS dari fresh install: pengguna non-root, firewall yang membatasi ke rentang Cloudflare, swap 2GB, Docker, Cloudflare Origin Certificate, penyetelan Postgres untuk 2GB, batas log, volume unggahan, cron cadangan, lalu urutan seed dan pembuatan admin.
3. **`/tasks`** — task berlabel per user story, dengan task pengujian karena pengujian otomatis diwajibkan konstitusi 2.1.0.

Satu keputusan menunggu pemilik project sebelum Phase 1: penempatan `specs/` dan `memory/` pada tabel Complexity Tracking di atas.