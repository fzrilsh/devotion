<div align="center">

# Devotion
### Marketplace kapasitas produksi untuk UMKM konveksi

[![GitHub Repository](https://img.shields.io/badge/GitHub-Repository-181717?style=for-the-badge&logo=github)](https://github.com/fzrilsh/devotion)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

**Submission ITECHNO CUP 2026, Web Development**

**Tim: [Isi nama tim sebelum submission]**

</div>

---

## Daftar isi

- [Tentang proyek](#tentang-proyek)
- [Fitur unggulan](#fitur-unggulan)
- [Demo dan screenshot](#demo-dan-screenshot)
- [Teknologi](#teknologi)
- [Arsitektur sistem](#arsitektur-sistem)
- [Instalasi dan setup](#instalasi-dan-setup)
- [Penggunaan](#penggunaan)
- [Dokumentasi API](#dokumentasi-api)
- [Testing](#testing)
- [Tim pengembang](#tim-pengembang)
- [Lisensi](#lisensi)

---

## Tentang proyek

### Latar belakang

UMKM konveksi sering mencari mitra produksi melalui relasi pribadi. Cara ini membuat jangkauan terbatas dan menyulitkan pemberi order ketika jumlah pesanan melebihi kapasitas produksinya. Di sisi lain, kapasitas mesin dan tenaga kerja yang sedang kosong tidak mudah ditemukan oleh usaha lain.

Informasi kapasitas juga cepat berubah. Listing yang tidak memiliki kalender ketersediaan dapat menyebabkan calon pemberi order memilih mitra yang sebenarnya sudah penuh.

### Solusi yang ditawarkan

Devotion adalah marketplace B2B yang mempertemukan dua pihak:

- **Subkontraktor**, yaitu UMKM konveksi yang memiliki kapasitas produksi tersedia.
- **Pemberi order**, yaitu UMKM atau brand yang membutuhkan tambahan kapasitas untuk memenuhi pesanan.

Subkontraktor dapat membuat profil usaha, mencantumkan mesin dan produk yang dapat dikerjakan, mengatur kapasitas mingguan, serta memperbarui kalender ketersediaan. Pemberi order dapat mencari kandidat berdasarkan produk, mesin, lokasi, jumlah pesanan, dan tenggat. Satu request kuota dapat dikirim ke beberapa kandidat sekaligus.

Hasil pencarian memakai empat kriteria keras dan skor 0 sampai 4. Kriteria yang digunakan adalah kesesuaian produk, mesin, jeda kesiapan, dan kapasitas sampai tenggat. Urutan hasil tidak ditentukan oleh reputasi atau status verifikasi agar alasan di balik setiap rekomendasi tetap dapat dijelaskan.

### Tujuan proyek

- **Tujuan utama**: Membantu UMKM konveksi menemukan atau menawarkan kapasitas produksi dengan proses yang lebih terstruktur.
- **Target pengguna**: Pemilik konveksi subkontraktor, pemilik brand atau UMKM pemberi order, dan admin operasional platform.
- **Value proposition**: Kapasitas yang sebelumnya sulit ditemukan dapat dipublikasikan, dicari, dan diminta berdasarkan data yang sama.
- **Kaitan tema**: Kalender kapasitas membuat informasi produksi dapat menyesuaikan perubahan ketersediaan. Solusi ini mendukung SDG 8 melalui akses order dan pemanfaatan kapasitas kerja, serta SDG 9 melalui digitalisasi proses matching antar pelaku industri.

### Batasan produk

Devotion tidak menahan atau menyalurkan dana pengguna. Pembayaran dilakukan langsung antar pihak. Platform hanya menyiapkan alur kesepakatan dan pencatatan yang dibutuhkan pada tahap pengembangan berikutnya.

---

## Fitur unggulan

### Fitur utama

| Fitur | Deskripsi | Nilai untuk pengguna |
|---|---|---|
| **Profil dan autentikasi usaha** | Registrasi dengan pilihan peran, login, sesi berbasis cookie, verifikasi email dan nomor HP, pemulihan kata sandi, serta profil usaha. | Setiap pihak memiliki identitas, peran, dan data usaha yang jelas. |
| **Listing kapasitas produksi** | Subkontraktor mengatur kapasitas mingguan, jeda kesiapan, jenis produk, jenis mesin, visibilitas listing, dan periode ketersediaan. | Kapasitas yang kosong dapat ditemukan tanpa mengandalkan relasi pribadi. |
| **Pencarian dan matching** | Pemberi order memfilter produk, mesin, jumlah, tenggat, dan cakupan wilayah. Hasil dihitung berdasarkan kapasitas lintas periode dan skor kriteria keras. | Kandidat dapat dibandingkan dengan alasan kecocokan yang dapat dijelaskan. |
| **Request kuota multi-kandidat** | Satu request dapat dikirim ke beberapa listing. Setiap kandidat memiliki status sendiri dan batas balasan 72 jam. | Pemberi order dapat membandingkan kesanggupan beberapa mitra dari satu alur. |

### Fitur tambahan

- **Master data produk, mesin, dan wilayah** untuk menjaga istilah pencarian tetap konsisten.
- **Usulan item baru** bagi pengguna ketika jenis produk atau mesin belum tersedia di daftar baku.
- **Notifikasi in-app** dengan status sudah dibaca, jumlah belum dibaca, dan preferensi kanal email atau WhatsApp.
- **Rate limiting** untuk login, kode verifikasi, dan request kuota.
- **Pengamanan sesi** dengan token acak yang disimpan sebagai hash di database, cookie `httpOnly`, `Secure`, dan `SameSite=Lax`.
- **Health check** untuk database, koneksi WhatsApp, dan ruang penyimpanan.
- **Error response** yang konsisten dalam format `application/problem+json` dengan pesan berbahasa Indonesia.

### Status implementasi saat ini

Implementasi saat ini berada pada branch `origin/develop/backend` dan `origin/develop/frontend`. README ini berada pada branch `docs/README`. Status yang dapat diverifikasi dari implementasi tersebut:

| Area | Status |
|---|---|
| Landing page dan layout responsif | Tersedia di frontend |
| Halaman login, registrasi, verifikasi, dan pemulihan akun | Tampilan tersedia; penyambungan seluruh form ke API masih berjalan |
| API akun, profil, master data, listing, kalender, pencarian, request kuota, notifikasi, dan health check | Tersedia di backend |
| Dashboard bisnis dan halaman alur matching di frontend | Masih dalam pengembangan |
| Penawaran, pesanan, ulasan, dan panel moderasi admin | Ada di kontrak produk, belum tersedia sebagai endpoint atau service lengkap pada branch develop saat ini |

---

## Demo dan screenshot

### Live demo

Belum ada URL demo publik yang tersimpan di repository ini. Tambahkan URL deployment pada bagian berikut sebelum submission:

**Live demo: [Isi URL demo]**

### Screenshot aplikasi

Screenshot final belum disimpan di repository. Setelah deployment, tambahkan screenshot halaman berikut ke folder `docs/screenshots/` dan perbarui tautannya:

- Landing page
- Halaman login atau registrasi
- Listing kapasitas dan kalender
- Hasil pencarian kandidat
- Request kuota

### Video demo

**Link video demo: [Isi URL video jika sudah tersedia]**

---

## Teknologi

### Tech stack

#### Frontend

```text
Framework       : React 18.3.1
Build tool      : Vite 8.2.0
Language        : TypeScript 5.7.2
Styling         : Tailwind CSS 4.3.3
Server state    : TanStack Query 5.102.1
Form and schema : React Hook Form 7.86.0, Zod 4.4.3
Routing         : React Router 7.18.2
UI utilities    : Motion, React Icons, clsx, tailwind-merge
API client      : Fetch API dengan session cookie
```

#### Backend

```text
Runtime         : Go 1.25.0
HTTP            : net/http
Database        : PostgreSQL 16
Database access : pgx/v5 5.7.5, sqlc generated queries
Migration       : golang-migrate
Authentication  : bcrypt dan session cookie httpOnly
Notifications   : Mailjet dan WhatsApp melalui whatsmeow
Monitoring      : Sentry, opsional melalui environment variable
```

#### DevOps dan tools

```text
Runtime         : Docker Compose dengan dua layanan, backend dan postgres
CI/CD           : GitHub Actions
Container image : GitHub Container Registry
Edge and TLS    : Cloudflare, TLS dikelola oleh binary Go pada production
Testing         : go test, go vet, ESLint, TypeScript build
```

### Alasan pemilihan teknologi

| Teknologi | Alasan pemilihan |
|---|---|
| **React dan Vite** | Cocok untuk SPA responsif. Hasil build dapat disematkan ke binary Go sehingga tidak perlu layanan frontend terpisah. |
| **Go dan net/http** | Menghasilkan satu backend yang ringan dengan router bawaan. Batas layanan tetap sederhana dan sesuai aturan lomba. |
| **PostgreSQL** | Data kapasitas, periode, peran, dan request membutuhkan transaksi, constraint, serta penguncian baris yang konsisten. |
| **OpenAPI dan openapi-typescript** | Kontrak API menjadi sumber tipe frontend sehingga perubahan bentuk respons lebih mudah dilacak. |
| **Tailwind CSS** | Memudahkan pembuatan antarmuka mobile-first tanpa menambah component library kedua. |

### Dependencies utama

```text
Frontend
  react
  react-dom
  vite
  @tanstack/react-query
  react-hook-form
  zod
  react-router-dom
  tailwindcss

Backend
  github.com/jackc/pgx/v5
  github.com/golang-migrate/migrate/v4
  golang.org/x/crypto
  golang.org/x/term
  go.mau.fi/whatsmeow
  github.com/getsentry/sentry-go
```

---

## Arsitektur sistem

### System architecture

Frontend disajikan oleh backend Go yang sama. Pada deployment, hanya ada dua layanan runtime, yaitu `backend` dan `postgres`. Cloudflare, Mailjet, WhatsApp, dan Sentry adalah layanan luar, bukan container tambahan pada aplikasi.

```text
Alur aplikasi

1. Browser pengguna membuka React SPA.
2. React SPA berkomunikasi dengan backend Go melalui HTTP JSON dan session cookie.
3. Backend Go menyimpan data di PostgreSQL 16.
4. Backend Go dapat mengirim notifikasi melalui WhatsApp dan Mailjet.
5. Sentry digunakan sebagai monitoring opsional.

Layanan runtime: backend dan postgres
```

### Database schema

Diagram berikut menunjukkan relasi inti. Skema lengkap, constraint, dan migrasi ada di `docs/001-capacity-exchange-marketplace/data-model.md` serta `backend/db/migrations/`.

```text
Relasi inti database

province
  └── city
       └── business_profile
            └── capacity_listing
                 ├── availability_period
                 ├── listing_product ── catalog_item
                 ├── listing_machine ── catalog_item
                 └── request_candidate ── quota_request

user_account
  ├── business_profile
  ├── session
  ├── quota_request
  └── notification

Skema lengkap ada di data-model.md dan backend/db/migrations/.
```

### Folder structure

```text
devotion/
├── backend/
│   ├── cmd/devotion/           # serve dan subcommand operasional
│   ├── internal/
│   │   ├── account/            # akun, peran, profil, autentikasi
│   │   ├── admin/              # integrasi WhatsApp dan status admin
│   │   ├── listing/             # listing kapasitas dan kalender
│   │   ├── masterdata/          # produk, mesin, dan wilayah
│   │   ├── notification/        # feed dan pengiriman notifikasi
│   │   ├── platform/            # HTTP, sesi, storage, scheduler, security
│   │   ├── quota/               # request kuota
│   │   └── search/              # pencarian dan skor kecocokan
│   ├── db/migrations/           # migrasi PostgreSQL
│   ├── db/queries/              # SQL sumber sqlc
│   └── webdist/                 # hasil build frontend yang di-embed
├── frontend/
│   └── src/
│       ├── api/                 # API client dan tipe kontrak
│       ├── components/          # komponen antarmuka
│       ├── pages/               # halaman aplikasi
│       ├── hooks/               # custom hooks
│       ├── providers/           # provider aplikasi
│       └── routes/              # route guest dan protected
├── docs/
│   └── 001-capacity-exchange-marketplace/
├── docker-compose.yml
├── .env.example
├── LICENSE
└── README.md
```

---

## Instalasi dan setup

### Prasyarat

Pastikan perangkat sudah memiliki:

- **Git**
- **Docker Engine** dan Docker Compose v2
- **Go 1.25.0**
- **Node.js 20 atau lebih baru** dan npm

### Langkah instalasi

#### 1. Clone repository

```bash
git clone https://github.com/fzrilsh/devotion.git
cd devotion
```

#### 2. Siapkan environment variable

```bash
cp .env.example .env
```

Isi nilai lokal berikut di `.env`. Contoh ini hanya untuk pengembangan lokal:

```env
APP_ENV=development
APP_BASE_URL=http://localhost:8080
POSTGRES_USER=devotion
POSTGRES_PASSWORD=GANTI_PASSWORD_LOKAL
POSTGRES_DB=devotion
DATABASE_URL=postgres://devotion:GANTI_PASSWORD_LOKAL@localhost:5432/devotion?sslmode=disable
UPLOAD_PATH=/absolute/path/to/devotion/backend/uploads
UPLOAD_TOTAL_LIMIT_MB=500
UPLOAD_FILE_LIMIT_MB=5
```

Ganti `/absolute/path/to/devotion` dengan lokasi repository di perangkatmu. Jangan memakai password contoh ini untuk production. Konfigurasi TLS, Mailjet, WhatsApp, dan Sentry dijelaskan di `docs/001-capacity-exchange-marketplace/quickstart.md`. Jangan commit file `.env`.

#### 3. Jalankan PostgreSQL

```bash
mkdir -p /absolute/path/to/devotion/backend/uploads
docker compose up -d postgres
```

Migrasi database berjalan otomatis ketika backend mulai.

#### 4. Jalankan backend

Dari root repository, muat environment variable lalu jalankan server:

```bash
set -a
. ./.env
set +a

cd backend
go run ./cmd/devotion serve
```

Backend development tersedia di `http://localhost:8080`.

#### 5. Jalankan frontend

Buka terminal baru dari root repository:

```bash
cd frontend
echo "VITE_API_URL=http://localhost:8080/api" > .env.local
npm ci
npm run dev
```

Vite akan menampilkan alamat development server pada terminal, biasanya `http://localhost:5173`.

#### 6. Isi data acuan

Untuk mengisi data wilayah, produk, mesin, dan admin, ikuti bagian B12 pada `docs/001-capacity-exchange-marketplace/quickstart.md`. Perintah seeding membaca data wilayah dari salinan repository dan tidak bergantung pada layanan luar ketika aplikasi berjalan.

---

## Penggunaan

### Menjalankan aplikasi

```bash
# Backend
cd backend
go run ./cmd/devotion serve

# Frontend, pada terminal lain
cd frontend
npm run dev

# Build frontend
npm run build

# Periksa kualitas kode frontend
npm run lint
```

### Panduan pengguna

#### Untuk subkontraktor

1. Buka halaman registrasi.
2. Pilih peran **Subkontraktor** atau **Keduanya**.
3. Isi data usaha, lokasi, email, nomor HP, dan kata sandi.
4. Selesaikan verifikasi email dan nomor HP.
5. Lengkapi profil usaha.
6. Buat listing dengan kapasitas mingguan, produk, mesin, dan jeda kesiapan.
7. Perbarui kalender ketika kapasitas berubah atau sudah penuh.

#### Untuk pemberi order

1. Masuk sebagai **Pemberi Order** atau **Keduanya**.
2. Isi produk, mesin, jumlah, tenggat, dan cakupan wilayah.
3. Bandingkan kandidat berdasarkan skor kecocokan dan kapasitas tersisa.
4. Kirim satu request kuota ke beberapa kandidat.
5. Pantau status setiap kandidat dan batas waktu balasan.

Alur bisnis di atas sudah tersedia pada kontrak dan service backend. Halaman frontend untuk listing, pencarian, dan request kuota masih dalam pengembangan.

#### Untuk admin

Admin pertama dibuat melalui subcommand backend. Pengelolaan status koneksi WhatsApp tersedia melalui endpoint admin. Panel admin lengkap untuk verifikasi, moderasi, dan mediasi belum menjadi alur frontend pada build saat ini.

---

## Dokumentasi API

### Base URL

```text
Development: http://localhost:8080/api
Production:  ditentukan oleh APP_BASE_URL
Health check: http://localhost:8080/health
```

### Endpoint yang tersedia pada backend

#### Autentikasi dan profil

```http
POST  /api/auth/register
POST  /api/auth/verify-email
POST  /api/auth/verify-phone
POST  /api/auth/resend-code
POST  /api/auth/login
POST  /api/auth/logout
POST  /api/auth/recover/request
POST  /api/auth/recover/confirm
GET   /api/me
PATCH /api/me/roles
GET   /api/profile/me
PUT   /api/profile/me
GET   /api/profile/{profileId}
```

#### Master data dan wilayah

```http
GET  /api/master/products
GET  /api/master/machines
GET  /api/regions/provinces
GET  /api/regions/cities
POST /api/master/proposals
```

#### Listing, pencarian, dan request kuota

```http
GET  /api/listing/me
POST /api/listing/me
PUT  /api/listing/me
PUT  /api/listing/me/visibility
GET  /api/listing/me/periods
PUT  /api/listing/me/periods
GET  /api/search
POST /api/quota-requests
GET  /api/quota-requests
```

#### Notifikasi dan operasional

```http
GET  /api/notifications
POST /api/notifications/{notificationId}/read
GET  /api/notifications/preferences
PUT  /api/notifications/preferences
GET  /api/admin/whatsapp
GET  /health
```

### Contoh request

```bash
curl -i -c cookies.txt \
  -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}'

curl -i -b cookies.txt \
  http://localhost:8080/api/me
```

Email dan kata sandi pada contoh hanya data contoh. Gunakan akun lokal milik sendiri saat menguji.

### Kontrak API lengkap

Kontrak OpenAPI 3.1 tersedia di [docs/001-capacity-exchange-marketplace/contracts/openapi.yaml](docs/001-capacity-exchange-marketplace/contracts/openapi.yaml). Peta endpoint terhadap requirement tersedia di [contracts/README.md](docs/001-capacity-exchange-marketplace/contracts/README.md).

---

## Testing

### Perintah pengujian

```bash
# Backend
cd backend
go vet ./...
go test ./...

# Frontend
cd ../frontend
npm run lint
npm run build
```

Pengujian integrasi backend memakai skema terpisah pada PostgreSQL yang sama. Beberapa pengujian akan dilewati jika `DATABASE_URL_TEST` belum menunjuk ke database yang dapat dijangkau.

Frontend saat ini memiliki lint dan build check. Script `npm test` belum tersedia pada `frontend/package.json`, sehingga coverage frontend belum diklaim di README ini.

### Cakupan yang diuji pada backend

- Autentikasi, peran, sesi, dan rate limiting.
- Validasi profil, listing, dan periode kapasitas.
- Idempotensi horizon kalender.
- Urutan pencarian yang deterministik dan pagination berbasis cursor.
- Skor pencarian dari kriteria keras.
- Request kuota ke beberapa kandidat.
- Notifikasi dan percobaan pengiriman kanal.
- Migrasi dan constraint PostgreSQL.
- Validasi file, batas ukuran, kuota storage, dan penghapusan metadata gambar.

---

## Tim pengembang

Nama tim resmi belum dicantumkan di repository. Isi bagian ini sebelum mengirim submission.

| Nama | Peran | GitHub |
|---|---|---|
| **Fazril Syaveral Hillaby** | Backend developer | [@fzrilsh](https://github.com/fzrilsh) |
| **ChikoID** | Frontend developer | [@ChikoID](https://github.com/ChikoID) |

---

## Lisensi

Proyek ini menggunakan [MIT License](LICENSE).

---

<div align="center">

**Dibuat untuk ITECHNO CUP 2026**

</div>
