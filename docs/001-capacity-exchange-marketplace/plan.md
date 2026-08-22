# Implementation Plan: Capacity Exchange, Marketplace Subkontrak Kapasitas Konveksi (MVP)

**Branch**: `001-capacity-exchange-marketplace` | **Date**: 2026-08-21 | **Last Revised**: 2026-08-22

**Spec**: `docs/001-capacity-exchange-marketplace/spec.md` (91 FR)

**Constitution**: `docs/memory/constitution.md` v2.1.0

**Status**: Phase 0 dan Phase 1 selesai. Constitution Check pasca-desain di bawah.

## Summary

Membangun Devotion, platform web responsif tempat UMKM konveksi mendaftarkan kapasitas produksi yang menganggur dan UMKM yang kelebihan order mencarinya berdasarkan kecocokan keras, mengirim request kuota ke beberapa kandidat, bernegosiasi harga, lalu mengeksekusi pesanan sampai tuntas dengan reputasi yang terbentuk dari transaksi nyata.

Empat masalah yang dijawab, seluruhnya dari analisis dokumen sumber: pencarian subkontraktor yang hanya lewat relasi personal sehingga jangkauannya terbatas dan tidak ada mekanisme matching sistematis [1]; listing kapasitas yang statis dan cepat kedaluwarsa sehingga informasinya tidak aktual dan transaksi gagal [1]; tidak adanya sistem reputasi antar UMKM yang belum saling kenal sehingga trust rendah dan transaksi antar pihak asing terhambat [1]; dan pemberi order yang memaksakan kapasitas sendiri sehingga terlambat, kualitas turun, dan reputasinya rusak karena alternatif subkontrak tidak terstruktur [1].

Pendekatan teknis: satu biner Go yang menyajikan frontend React hasil build, menghabiskan TLS sendiri dengan Cloudflare Origin Certificate, dan berbicara ke satu PostgreSQL. Dua layanan di `docker-compose.yml`, tidak ada proses ketiga. Modularitas lewat batas paket internal, bukan batas jaringan. Pekerjaan terjadwal ditangani penjadwal di dalam proses yang sama, dengan perhitungan saat baca sebagai lapisan kedua agar tidak ada tenggat yang terlewat bila proses sempat mati.

## Technical Context

**Language/Version**: Go 1.23.4 (backend; router `net/http` menuntut minimal 1.22, toolchain dipatok tepat di `go.mod`), TypeScript 5.7.2 pada React 18.3.1 (frontend). Versi patok ini menjadi acuan `go.mod` dan `package.json` saat kode terbit; sesuai konstitusi Prinsip IV, tidak ada rentang terbuka.

**Primary Dependencies**:

Backend: `net/http` (router bawaan, pola path + metode sejak 1.22), `jackc/pgx/v5`, `sqlc` (generator, perkakas build), `golang-migrate`, `golang.org/x/crypto/bcrypt`, `go.mau.fi/whatsmeow`, `getsentry/sentry-go`. Dari standard library tanpa dependency tambahan: `log/slog` (log terstruktur), `net/smtp` (email lewat Mailjet), `image/jpeg` dan `image/png` (pembuangan metadata lokasi), `crypto/rand` (token sesi).

Frontend: Vite, React 18, TanStack Query, Zod + React Hook Form, Tailwind CSS, Leaflet + tile OpenStreetMap, `@sentry/react`, `openapi-typescript` (devDependency).

**Storage**: PostgreSQL 16 untuk seluruh data termasuk store sesi whatsmeow dan tabel pembatasan laju; berkas unggahan pada volume disk VPS (maksimal 5MB per berkas, total maksimal 500MB), dilayani hanya lewat handler Go yang memeriksa peran.

**Testing**: `go test` dengan skema Postgres terpisah pada layanan basis data yang sama (backend), Jest (frontend), pengujian end-to-end **manual** oleh penguji di luar tim mengikuti `quickstart.md` bagian F.

**Target Platform**: VPS Linux tunggal, 2GB RAM, 50GB storage; Docker Compose dua layanan; Cloudflare sebagai DNS dan proxy tepi dengan mode SSL/TLS Full (strict) dan Authenticated Origin Pulls.

**Project Type**: Web application monolith, satu repository, satu deployable, frontend disematkan ke dalam biner backend.

**Performance Goals**: Hasil pencarian tampil dalam 3 detik pada koneksi seluler lambat (SC-010); pencarian yang diulang menghasilkan urutan identik termasuk antar halaman (SC-013).

**Constraints**: 2GB RAM dan 50GB disk pada satu server; maksimal 2 layanan runtime; dilarang memproses dana pihak mana pun; dilarang membangun artefak di server; ukuran log kontainer dan total unggahan wajib dibatasi.

**Scale/Scope**: 91 functional requirement, 7 user story, 16 entitas domain pada 25 tabel, 21 success criteria, 63 operasi API, 81 task. Data demo sekitar 50 usaha; SC-003 menargetkan 200 usaha aktif pada bulan ketiga sebagai sasaran bisnis, mengikuti proyeksi sekitar 1.500 UMKM aktif dengan transaksi rata-rata Rp 64 juta per UMKM per tahun pada tahun ketiga [1].

**Anggaran memori pada 2GB** (perkiraan, wajib diverifikasi dengan `docker stats` setelah penyiapan):

| Komponen | Perkiraan |
|----------|-----------|
| PostgreSQL (`max_connections` 20, `shared_buffers` 256MB) | 300–400MB |
| Go + whatsmeow | 150–250MB |
| Sistem operasi dan Docker | 250–350MB |
| Sisa untuk lonjakan | ~1GB |
| Swap yang disiapkan | 2GB, `vm.swappiness=10` |

### Keputusan yang Sudah Tertutup

Empat pertentangan antar artefak yang ditemukan `/analyze` dan diselesaikan pada revisi 2026-08-22. Dicatat di sini karena keempatnya membentuk arsitektur pencarian dan alokasi:

| Isu | Keputusan | Dampak |
|-----|-----------|--------|
| Jeda kesiapan mulai tidak dipakai dalam alokasi maupun penjumlahan kapasitas | Alokasi dan penjumlahan dimulai dari **minggu kesiapan mulai** = minggu yang memuat tanggal acuan + `jeda_kesiapan_hari`. Istilah **rentang kapasitas** dibakukan | FR-087, FR-090, SC-020; kolom `pesanan.minggu_kesiapan_mulai` + trigger; kueri pencarian |
| Horizon kalender 3 bulan lebih pendek dari deadline yang mungkin diminta | Periode dibuat otomatis sampai minggu deadline, **dipicu saat pencarian**, bukan penjadwal bergulir | FR-088, SC-021; kolom `capacity_listing.horizon_until` |
| Constraint `batas_balasan_72_jam` selalu gagal dan melewati `Clock` | `DEFAULT now()` dihapus dari **seluruh** tabel; aplikasi mengirim setiap waktu dari `Clock`; constraint tinggal menjaga urutan | Seluruh 25 tabel; menegakkan Prinsip V pada tingkat data |
| Kriteria mesin tidak terdefinisi ketika filternya dikosongkan | Kriteria yang filternya tidak diisi dihitung **terpenuhi**; respons menyebut kriteria mana yang dievaluasi. Skor tetap 0–4, tanpa normalisasi | FR-023, FR-026 |

Dua temuan WARNING yang juga sudah ditutup: FR-089 (propagasi perubahan kapasitas mingguan ke periode mendatang tanpa alokasi) dan FR-091 (penggolongan notifikasi transaksional versus non-transaksional, dengan kolom `notifikasi.transaksional`).

### Yang Masih Terbuka

Satu keputusan bersyarat dan tiga hal yang wajib diverifikasi sebelum task terkait dimulai:

- **Rentang alamat Cloudflare** di `research.md` R-01 ditulis dari ingatan dan **wajib dicocokkan** ke sumber resmi sebelum dipatok ke konstanta Go.
- **Angka penyetelan PostgreSQL** di R-03 sudah konkret dan siap pakai, tetapi wajib diverifikasi dengan pengukuran nyata pada 2GB.
- **Batas dan kuota layanan luar** (aturan rate limiting Cloudflare paket gratis, batas ukuran body proxy, kuota harian Mailjet) tidak saya hafal dan berubah dari waktu ke waktu. Periksa langsung di dasbor masing-masing, jangan diasumsikan dari dokumen ini.

## Constitution Check (Pasca-Phase 1)

*Diperiksa ulang setelah `research.md`, `data-model.md`, `contracts/`, dan `quickstart.md` selesai.*

### Gate I: Monolith-First (NON-NEGOTIABLE)

| Aturan | Bukti pada artefak desain | Status |
|--------|---------------------------|--------|
| Backend dan frontend satu repository | Struktur di bawah | LOLOS |
| Frontend disajikan proses backend yang sama | `research.md` R-06: `embed.FS` + fallback SPA; T022 | LOLOS |
| Maksimal dua layanan runtime | `quickstart.md` B10: `backend`, `postgres` | LOLOS |
| Tidak ada broker, worker, cron, cache, proxy sebagai proses | TLS oleh Go dengan Origin Certificate (R-01); Cloudflare Tunnel ditolak justru karena `cloudflared` adalah daemon | LOLOS |
| Pekerjaan terjadwal tanpa proses kedua | R-07: perhitungan saat baca + `time.Ticker` dalam proses, masing-masing dibungkus advisory lock | LOLOS |
| Notifikasi di dalam proses yang sama | `data-model.md` §9: baris `notifikasi` di dalam transaksi kejadian, `notifikasi_kanal` diproses goroutine setelahnya (FR-086) | LOLOS |
| Antar modul lewat pemanggilan fungsi | Paket `internal/*`; tidak ada HTTP ke diri sendiri | LOLOS |
| Perkakas pengembangan bukan proses runtime | `sqlc`, `golang-migrate` CLI, Jest, `go test` hanya saat build dan uji | LOLOS |
| Perintah sekali jalan lewat subcommand | Delapan subcommand pada satu biner (T002) | LOLOS |
| Pengujian tanpa layanan basis data tambahan | Skema `test_*` pada layanan Postgres yang sama | LOLOS |
| Layanan luar dicatat, tidak dihitung | `docs/layanan-luar.md`: Cloudflare, Mailjet, Sentry, pemantau uptime, wilayah.id | LOLOS |

**Cara memeriksa**: hitung entri di bawah `services:` pada `docker-compose.yml`. Lebih dari dua, atau ada layanan di luar backend dan basis data, berarti pelanggaran.

### Gate II: Demo-Ready Over Complete

| Aturan | Bukti | Status |
|--------|-------|--------|
| Setiap story dapat didemokan lewat antarmuka | `quickstart.md` §F: 83 langkah verifikasi manual, tujuh story | LOLOS |
| Data contoh keadaan berhasil dan gagal | T075: hasil pencarian kosong, penawaran tertolak karena kapasitas, kalender basi, request kedaluwarsa | LOLOS |
| Data acuan terisi dengan satu perintah | `seed:regions`, `seed:master-data` (T019) | LOLOS |
| Demo tidak bergantung layanan yang bisa mati | Notifikasi di dalam platform sebagai jalur pengamatan; WhatsApp dan email boleh gagal tanpa merusak alur (FR-054, FR-086) | LOLOS |

Turunan yang mengikat urutan: pengisian daftar baku dan wilayah adalah prasyarat data bagi US1 dan US2, sehingga masuk fase Foundational (T019), terpisah dari antarmuka admin yang tetap di prioritas terakhir (T073). Tanpa itu, US7 harus naik ke awal padahal prioritasnya P7.

### Gate III: Traceability to Spec

| Aturan | Bukti | Status |
|--------|-------|--------|
| Setiap task menunjuk FR atau user story | `tasks.md`: 66 dari 81 task punya baris **FR** | LOLOS bersyarat |
| Setiap pengujian menyebut FR | Pola nama disepakati di `CLAUDE.md` | LOLOS |
| Setiap endpoint dipetakan ke FR | `contracts/README.md`: peta 63 operasi + 10 FR yang memang bukan endpoint | LOLOS |
| Skenario penguji menunjuk Acceptance Scenario | `quickstart.md` §F: setiap blok menyebut nomor scenario | LOLOS |

**Bersyarat**: 15 task pada fase Setup dan Polish tidak menunjuk FR karena melayani gerbang konstitusi, bukan requirement produk: T001, T003–T007, T009, T011, T020, T075–T081. Ini dicatat sebagai pengecualian di Complexity Tracking, bukan diabaikan.

### Gate IV: Minimal Dependencies

Sebelas dependency runtime, seluruhnya dengan pembenaran di `plan.md` versi ini dan `docs/dependencies.md`. Yang diselesaikan tanpa dependency: log terstruktur, email, pembuangan EXIF, token acak, UUID, jarak haversine, pembatasan laju, router.

| Dependency | Pembenaran | Alternatif yang ditolak |
|------------|------------|-------------------------|
| `pgx/v5` | Mendukung penguncian baris dan transaksi yang dibutuhkan alokasi kapasitas | `database/sql` + `lib/pq`, dukungan tipe Postgres lebih lemah |
| `sqlc` | Kueri pencarian dan skor harus eksplisit dan deterministik; generator menjaga SQL terlihat | GORM, menyembunyikan SQL justru pada bagian terpenting |
| `golang-migrate` | Migrasi berversi, dijalankan otomatis saat startup | Menulis sendiri, ~200 baris tanpa nilai tambah |
| `bcrypt` | Hash kata sandi yang memang untuk itu | argon2id, lebih baik, tetapi biaya memorinya perlu penyetelan hati-hati pada 2GB |
| `whatsmeow` | Satu-satunya cara mengirim WhatsApp tanpa verifikasi bisnis; library, bukan layanan | API resmi Meta, verifikasi tidak selesai sebelum tenggat |
| `sentry-go`, `@sentry/react` | Mengetahui kerusakan sebelum juri menemukannya | Self-host Sentry, beberapa layanan, melanggar Gate I |
| TanStack Query | Hasil pencarian, daftar pesanan, status kandidat semuanya perlu disegarkan | `useEffect` manual, menulis ulang cache dan invalidasi |
| Zod + React Hook Form | Skema yang sama memvalidasi form dan bentuk respons | Validasi manual, rawan pada belasan form |
| Tailwind CSS | FR-055 menuntut mobile-first | Component library, hanya bila tenggat menekan |
| Leaflet + OpenStreetMap | FR-064 menampilkan titik lokasi tanpa kunci API | Google Maps, Mapbox, menuntut kunci dan penagihan |
| `openapi-typescript` | Tipe frontend dari kontrak, bukan ditulis tangan | Tulis tangan, akan menyimpang tanpa terdeteksi |

**LOLOS**. Bila tenggat menekan, kandidat pertama yang dipangkas adalah Sentry di sisi frontend.

### Gate V: Deterministic Behavior

| Aturan | Bukti | Status |
|--------|-------|--------|
| Urutan identik pada pengulangan, termasuk antar halaman | R-05: keyset tuple lima kolom berakhir pada `listing_id`; `data-model.md` §10 | LOLOS |
| Skor hanya dari kriteria keras | Empat boolean dijumlahkan; FR-024 melarang faktor lain secara eksplisit | LOLOS |
| Setiap keputusan dapat dijelaskan satu kalimat | Respons pencarian mengirim nilai per kriteria (FR-026) | LOLOS |
| Satu sumber waktu yang dapat digantikan | `Clock` disuntikkan; **tidak ada `DEFAULT now()` pada tabel mana pun** | LOLOS |
| Batas minggu Senin, WIB, tipe tanggal | `CHECK EXTRACT(ISODOW) = 1` pada tiga tabel | LOLOS |
| Uang bilangan bulat rupiah | `bigint`, tanpa tipe pecahan | LOLOS |
| Data acuan dari luar diambil sekali, disalin ke repo | T019: `--refresh` hanya di lokal, salinan di `docs/master-data/` | LOLOS |

### Batasan Tambahan

| Batasan | Bukti | Status |
|---------|-------|--------|
| Batas keuangan | Tidak ada payment gateway; `catatan_pembayaran` tanpa kolom jumlah uang. Escrow yang menahan dana dan merilisnya saat pesanan dikonfirmasi selesai [1] sengaja tidak dibangun | LOLOS |
| Unggahan tidak lewat path statis | `GET /api/files/{fileId}` memeriksa peran sebelum mengirim byte | LOLOS |
| Nama berkas dibuat sistem, tipe dari isi | UUID sebagai `path_penyimpanan`; magic bytes | LOLOS |
| Metadata lokasi gambar dibuang | Dekode–enkode ulang saat unggah | LOLOS |
| Segmen origin terenkripsi, koneksi non-Cloudflare ditolak | R-01: tiga lapisan (firewall, Origin Certificate + Full (strict), Authenticated Origin Pulls) | LOLOS |
| Alamat asal hanya dipercaya dari rentang Cloudflare | `RealIP` memeriksa `RemoteAddr` sebelum membaca header | LOLOS |
| Pembatasan laju berbasis data domain | Empat batas di tabel `batas_laju`, bukan di memori | LOLOS |
| Kredensial dan nomor layanan tidak di repo | Variabel lingkungan; `.env.example` hanya nama kunci | LOLOS bersyarat |
| Membangun artefak tidak di server | CI membangun image; VPS hanya `pull` dan `up` | LOLOS |
| Ukuran log dibatasi | `max-size 10m`, `max-file 3` pada kedua layanan | LOLOS |
| Total unggahan dibatasi | 500MB total, 5MB per berkas, `CHECK` + aplikasi | LOLOS |
| Koneksi basis data disesuaikan memori | `max_connections` 20, pool 15, sisa 5 untuk `pg_dump`, `psql`, migrasi | LOLOS |
| Cadangan terjadwal, salinan di luar server, jumlah dibatasi | `pg_dump` harian, gzip, tiga salinan, `rsync` keluar | LOLOS dengan catatan |

**Bersyarat pada kredensial**: `quickstart.md` §E memuat kredensial akun uji karena konstitusi juga mewajibkan kredensial akun uji tersedia bagi penguji eksternal. Dua kewajiban itu bertabrakan secara harfiah, dan penyelesaiannya ada di Complexity Tracking.

### Hasil Gate

Lolos untuk implementasi. Dua pengecualian tercatat di Complexity Tracking, dan empat hal terbuka di Technical Context wajib diselesaikan sebelum task terkait dimulai.

## Project Structure

### Documentation (this feature)

```text
docs/
├── memory/
│   └── constitution.md                     # v2.1.0
└── 001-capacity-exchange-marketplace/
    ├── spec.md                             # 91 FR, revisi 2026-08-22
    ├── plan.md                             # berkas ini
    ├── research.md                         # R-01 sampai R-10
    ├── data-model.md                       # 25 tabel, revisi 2026-08-22
    ├── quickstart.md                       # runbook VPS + 83 langkah uji manual
    ├── tasks.md                            # 81 task, tingkat modul
    ├── contracts/
    │   ├── openapi.yaml                    # 63 operasi, 28 kode galat
    │   └── README.md                       # peta endpoint → FR
    └── checklists/
        └── requirements.md                 # 16 butir
```

Dokumen operasional yang diturunkan dari `quickstart.md` berada langsung di bawah `docs/`: `setup-vps.md`, `menjalankan.md`, `pengujian.md`, `skenario-uji-manual.md`, `temuan-penguji.md`, `layanan-luar.md`, `dependencies.md`, `utang-teknis.md`, `cloudflare-ips.md`, dan data acuan pada `docs/master-data/`.

Changelog **tidak** berada di `docs/`. Ia per bagian: `backend/CHANGELOG.md` dan `frontend/CHANGELOG.md`, diisi setiap kali sebuah story ditutup di checkpoint.

### Source Code (repository root)

```text
devotion/
├── README.md                       # template panitia, struktur tidak diubah
├── LICENSE                         # MIT
├── CLAUDE.md                       # panduan agent, di root agar terbaca
├── docker-compose.yml              # tepat 2 layanan: backend, postgres
├── .env.example                    # nama variabel tanpa nilai
├── .github/workflows/ci.yml
├── backend/
│   ├── CHANGELOG.md
│   ├── cmd/devotion/               # serve, admin:create, seed:*, reset:*, user:verify, health:check
│   ├── internal/
│   │   ├── platform/               # clock, config, httpx, session, storage, scheduler,
│   │   │                           # ratelimit, cloudflare
│   │   ├── account/                # akun, peran, profil usaha, verifikasi identitas
│   │   ├── masterdata/             # daftar baku, wilayah, usulan item
│   │   ├── listing/                # listing kapasitas, periode ketersediaan, horizon
│   │   ├── search/                 # kriteria keras, skor, pemecah seri, keyset
│   │   ├── quota/                  # request kuota, penawaran, counter-offer
│   │   ├── order/                  # pesanan, alokasi kapasitas, pembatalan, pembayaran
│   │   ├── reputation/             # ulasan, tingkat penyelesaian
│   │   ├── notification/           # antrean, pengirim, percobaan ulang
│   │   └── admin/                  # verifikasi, moderasi, mediasi
│   ├── db/migrations/              # 14 migrasi berurutan
│   ├── db/queries/                 # sumber SQL untuk sqlc
│   ├── webdist/                    # hasil build frontend, disematkan embed.FS
│   └── go.mod
├── frontend/
│   ├── CHANGELOG.md
│   ├── src/
│   │   ├── pages/                  # satu berkas per layar, dikelompokkan per user story
│   │   ├── components/
│   │   ├── api/                    # klien, tipe ter-generate, hook TanStack Query
│   │   ├── schemas/                # skema Zod
│   │   └── lib/
│   ├── package.json
│   └── vite.config.ts
└── docs/                           # lihat bagian sebelumnya
```

**Structure Decision**: Monolith dua bagian dalam satu repository, dengan `backend/` sebagai satu-satunya proses aplikasi. Frontend dibangun di CI, hasilnya disalin ke `backend/webdist/`, lalu disematkan lewat `embed.FS`. Handler statis melayani berkas yang ada dan mengembalikan `index.html` untuk path non-API; `/api/*` yang tidak dikenali mengembalikan 404 JSON, bukan HTML, agar kesalahan penulisan alamat tidak menyesatkan saat diagnosis.

Batas modul mengikuti batas user story sedapat mungkin, sehingga setiap fase `tasks.md` menyentuh paket yang jelas dan `[P]` bermakna di tingkat modul. Tiga paket sengaja dipakai bersama: `masterdata` prasyarat US1 dan US2, `notification` dipakai hampir semua story, `platform` memuat `Clock` yang wajib dapat digantikan.

**Arah ketergantungan yang harus dijaga**: `order` mengubah kapasitas milik `listing` lewat baris alokasi (FR-077), sehingga `order` bergantung pada `listing`, bukan sebaliknya. `search` hanya membaca. Tidak ada ketergantungan melingkar.

**Empat path yang dipatok**, berubah tempatnya berarti artefak lain rusak: `backend/internal/platform/clock.go`, `backend/db/migrations/`, `backend/webdist/`, `docker-compose.yml`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| Cron di tingkat host untuk `pg_dump` harian | Konstitusi mewajibkan cadangan terjadwal dengan salinan di luar server, sementara Gate I melarang proses terjadwal kedua. Cron host tidak muncul di `docker-compose.yml` tetapi tetap merupakan pekerjaan terjadwal di server yang sama | Penjadwal di dalam proses backend ditolak karena `pg_dump` harus tetap berjalan ketika aplikasi sedang mati atau rusak, justru saat itulah cadangan paling dibutuhkan. Menambah layanan cron ke compose melanggar batas dua layanan |
| Kredensial akun uji tertulis di `docs/skenario-uji-manual.md` | Batasan Keamanan melarang kredensial di dokumentasi, sementara gerbang Pengujian End-to-End mewajibkan kredensial akun uji tersedia di `docs/` bagi penguji eksternal yang tidak punya akses basis data | Tidak ada alternatif yang memenuhi keduanya. Dibatasi tiga syarat: akun hanya ada pada data `seed:test-data` yang **menolak berjalan** saat `APP_ENV=production`; kata sandinya tidak dipakai akun sungguhan mana pun; domain `.test` tidak dapat diregistrasi sehingga tidak ada email nyata yang terlibat |
| Lima belas task tanpa rujukan FR (T001, T003–T007, T009, T011, T020, T075–T081) | Gate III mewajibkan setiap task menunjuk FR atau user story. Kelima belas task ini melayani gerbang konstitusi (struktur repository, CI, penyiapan server, dokumentasi, cadangan), bukan requirement produk | Memaksakan rujukan FR pada scaffolding akan menghasilkan rujukan yang menyesatkan. Sebagai gantinya, setiap task tersebut menyebut gerbang konstitusi yang dilayaninya pada baris **Kemampuan** atau **Hati-hati** |
| Direktori tingkat atas `.github/` di luar struktur yang didaftar `CLAUDE.md` | Konstitusi mewajibkan CI membangun image, dan CI berbasis GitHub Actions menuntut `.github/workflows/`. `CLAUDE.md` melarang menambah direktori tingkat atas dan tidak menyebut `.github/` | Menaruh definisi CI di dalam `backend/` atau `frontend/` ditolak karena pipeline membangun kedua area sekaligus dan bukan milik salah satunya; `.github/` adalah lokasi yang diwajibkan penyedia CI, bukan pilihan struktur |

Dua hal berikut **bukan** pelanggaran tetapi wajib tercatat di `docs/layanan-luar.md` sesuai Gate I, beserta akibat bila mati: Cloudflare, Mailjet, Sentry, pemantau uptime, dan wilayah.id semuanya berjalan di luar server dan tidak dihitung sebagai proses runtime.

### Risiko yang Diterima Sadar

Empat hal yang bukan pelanggaran aturan tetapi perlu terlihat di rencana, karena semuanya menyimpang dari mitigasi yang dokumen sumber tetapkan:

**whatsmeow memakai protokol WhatsApp Web, bukan API resmi.** FR-002 menjadikan verifikasi nomor HP sebagai gerbang, sehingga nomor yang terblokir berarti tidak ada akun baru yang dapat dibuat saat demo. Mitigasi: halaman admin QR dan status sambungan agar penyambungan ulang tidak memerlukan SSH (T024), subcommand `user:verify` sebagai jalan darurat, email dipertahankan sebagai kanal kedua, dan pembatasan laju per nomor serta per alamat asal untuk mengurangi pola yang terdeteksi sebagai spam.

**Escrow tidak dibangun.** Dokumen sumber menempatkan penahanan dana yang dirilis saat pesanan dikonfirmasi selesai [1] sebagai bagian modul transaksi, dan menjadikannya mitigasi utama risiko gagal bayar sekaligus alat tawar dalam sengketa kualitas. Versi ini menggantinya dengan pencatatan pernyataan pembayaran, sehingga mediasi admin kehilangan salah satu daya paksanya.

**Verifikasi identitas bukan gerbang.** Hasil pencarian dapat memuat usaha yang belum diperiksa, dan lencana menjadi satu-satunya pembeda.

**Skor kecocokan tidak memuat faktor perilaku.** Penalti peringkat bagi subkontraktor yang tidak memperbarui kalender tidak dipakai; penegakannya hanya lewat pengingat dan penanda "Data Belum Diperbarui".

Keempatnya tercatat lengkap beserta konsekuensinya di bagian Assumptions `spec.md`, dan akan dicatat ulang di `docs/utang-teknis.md` saat implementasi berjalan.

## Phase 0 dan Phase 1: Selesai

**Phase 0 → `research.md`**: sepuluh keputusan. R-01 penolakan koneksi non-Cloudflare (tiga lapisan), R-02 sumber data wilayah (bersyarat), R-03 penyetelan PostgreSQL untuk 2GB (angka konkret), R-04 penguncian baris alokasi lintas periode, R-05 keyset pagination, R-06 penyematan frontend dan fallback SPA, R-07 penjadwal dua lapisan, R-08 ketahanan sesi whatsmeow, R-09 email lewat Mailjet dan penyiapan DNS, R-10 sesi, kata sandi, dan pembatasan laju.

**Phase 1 → `data-model.md`, `contracts/`, `quickstart.md`**: 16 entitas pada 25 tabel dengan seluruh constraint dan indeks, tiga trigger untuk aturan yang melintasi tabel, 63 operasi API dengan 28 kode galat berbahasa Indonesia, dan runbook VPS 16 langkah plus 83 langkah verifikasi manual.

**Phase 2 → `tasks.md`**: 81 task tingkat modul, dihasilkan `/tasks`.

## Langkah Berikutnya

Empat hal terbuka di Technical Context diselesaikan lebih dulu: bentuk respons wilayah.id sebelum T019, rentang alamat Cloudflare sebelum T013, angka Postgres saat penyiapan server, batas layanan luar saat pendaftaran akun.

Lalu implementasi menurut `tasks.md`: Setup → Foundational → US1 → berhenti dan buktikan US1 berdiri sendiri → demo. MVP yang disarankan adalah US1 saja; bila tenggat longgar, US1 sampai US3 bersama membentuk alur utuh dari mendaftarkan kapasitas sampai menyepakati pesanan.

Dua artefak yang perlu terbit ulang setelah revisi ini: `tasks.md` (T035 dan T041 memuat rentang kapasitas dan perpanjangan horizon; T028 harus tahu horizon dapat diperpanjang; T036 dan T050 mendapat test dari SC-020 dan SC-021; T047 mendapat FR-089; T006 dan catatan checkpoint menyesuaikan lokasi changelog) dan `checklists/requirements.md` (butir "no implementation details" perlu dinilai ulang setelah FR-036 dan FR-079 dibersihkan).