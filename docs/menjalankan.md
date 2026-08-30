# Menjalankan Devotion

Cara menyalakan aplikasi secara lokal dan di server. Penerapan server lengkap
dari fresh install ada di `docs/setup-vps.md`; dokumen ini fokus pada alur
pengembang di mesin lokal.

## Alur lokal

Prasyarat: Go 1.25+, Docker, Node 20+ (untuk frontend), dan salinan `.env` dari
`.env.example` yang sudah diisi. Untuk pengembangan set `APP_ENV=development`.

1. **Nyalakan Postgres saja lewat compose.** Backend dijalankan dari host, bukan
   di dalam kontainer, agar iterasi cepat tanpa membangun ulang image.

   ```bash
   docker-compose up -d postgres
   ```

   Perintah compose ditulis `docker-compose` di alur lokal dan `docker compose`
   di `docs/setup-vps.md`, dan itu memang beda. Homebrew memasang Compose sebagai
   biner standalone `docker-compose`, sedangkan pemasangan Docker di VPS lewat
   skrip resmi menyediakannya sebagai plugin `docker compose`. Pakai bentuk yang
   ada di mesin masing-masing; keduanya Compose v2 dan menerima berkas yang sama.

   Postgres dipublikasikan ke `127.0.0.1:5434`, jadi `DATABASE_URL` di `.env`
   menunjuk `postgres://...@127.0.0.1:5434/devotion`. Port 5434 dipilih agar tidak
   bertabrakan dengan Postgres lain di mesin pengembang, dan ikatannya ke loopback
   menjaga agar tidak pernah terekspos keluar mesin.
2. **Ekspor `.env` ke lingkungan shell.** Biner Go membaca konfigurasi dari
   variabel lingkungan lewat `os.Getenv`, bukan dari berkas `.env`; yang membaca
   `.env` hanyalah Docker Compose. Tanpa langkah ini `serve` berhenti dengan
   `variabel lingkungan wajib belum diisi: APP_ENV`.

   ```bash
   set -a; . ./.env; set +a
   ```

   Wajib terisi di setiap lingkungan: `APP_ENV`, `APP_BASE_URL`, `DATABASE_URL`,
   `UPLOAD_PATH`. Sisanya (TLS, Mailjet, WhatsApp, Sentry) hanya diwajibkan saat
   `APP_ENV=production`, jadi di lokal boleh dibiarkan kosong. Konsekuensinya
   email dan WhatsApp tidak terkirim di lokal, dan kode verifikasi diambil dari
   log atau lewat `user:verify`.
3. **Jalankan backend.** Migrasi jalan otomatis saat startup di bawah advisory
   lock, jadi tidak ada langkah migrasi manual.

   ```bash
   cd backend
   go run ./cmd/devotion serve
   ```

   Server mendengarkan di `:8080` saat pengembangan. Swagger UI ada di `/docs`
   (lihat bagian bawah).
4. **Isi data acuan dan buat akun.** Sekali saja per basis data baru.

   ```bash
   go run ./cmd/devotion seed:regions
   go run ./cmd/devotion seed:master-data
   go run ./cmd/devotion admin:create
   go run ./cmd/devotion seed:test-data      # hanya non-produksi
   ```

   `seed:test-data` menolak berjalan saat `APP_ENV=production`. Kredensial akun uji
   ada di `docs/skenario-uji-manual.md`.
5. **Jalankan frontend.** Lihat bagian Frontend di bawah.
6. **Uji.** Lihat `docs/pengujian.md`. Ringkasnya:

   ```bash
   cd backend
   DATABASE_URL_TEST=postgres://devotion:devotion@127.0.0.1:5434/devotion?sslmode=disable \
     go test ./... -p 1
   ```

   `DATABASE_URL_TEST` harus disebut eksplisit. Nilai bawaan di dalam kode
   menunjuk port 5432, sedangkan compose menerbitkan 5434, dan uji yang tidak
   dapat menjangkau basis data memilih `t.Skip` daripada gagal. Tanpa variabel
   ini seluruh uji basis data dilewati diam-diam dan hasilnya tampak hijau.

## Frontend

Frontend adalah Vite + React SPA yang di produksi **tidak menjadi layanan
terpisah**: hasil buildnya disalin CI ke `backend/webdist/` lalu disematkan ke
dalam biner Go lewat `embed.FS`. Itulah yang membuat Gate I tetap dua layanan.
Di lokal, ia berjalan sebagai proses pengembangan di host, bukan entri di
`docker-compose.yml`.

**Keadaan sekarang: `frontend/` baru memuat `.gitkeep` dan `CHANGELOG.md`.**
Belum ada `package.json`, jadi belum ada yang bisa dijalankan. Ini bukan
kelalaian dokumen: frontend dikerjakan di jalur terpisah (`develop/frontend`)
dan mulai pada T003. Selama `frontend/package.json` belum ada, job `image` dan
`deploy` di CI dilewati dan image GHCR belum terbit; keduanya menyala sendiri
begitu berkas itu mendarat, tanpa menyentuh workflow.

Begitu T003 selesai, alur lokalnya:

```bash
cd frontend
npm ci
npm run dev
```

Konfigurasi yang harus benar di `vite.config.ts`:

- **Proxy `/api` ke backend.** Dev server berjalan di port sendiri (bawaan Vite
  `5173`), backend di `:8080`. Semua permintaan `/api` diproksikan ke
  `http://127.0.0.1:8080` agar cookie sesi dianggap same-origin. Tanpa proxy,
  cookie `SameSite=Lax` tidak terkirim dan setiap permintaan tampak belum masuk.
- **`credentials: 'include'` pada seluruh permintaan.** Token sesi tidak pernah
  disimpan di `localStorage` maupun `sessionStorage`.
- **Tipe di-generate dari `openapi.yaml`**, tidak ditulis tangan. Sumbernya
  `docs/001-capacity-exchange-marketplace/contracts/openapi.yaml`, yang sama
  dengan yang disajikan di `/docs`.

Uji frontend: `cd frontend && npm test` (Jest).

Selama frontend belum ada, `backend/webdist/` hanya memuat `index.html`
placeholder. Backend tetap menyala normal dan seluruh `/api` bisa diuji lewat
`/docs` atau `curl`.

## Alur server

Ringkasannya: CI membangun image dan mendorongnya ke GHCR, server hanya menarik
dan menjalankan. **Jangan pernah build di server** (VPS 2GB, Vite + Postgres
bersamaan akan kehabisan memori). Langkah lengkap dari fresh install, termasuk
firewall sebagai gerbang, TLS Cloudflare, seed, cadangan, dan snapshot, ada di
`docs/setup-vps.md`.

## Menghitung jumlah layanan

Gate I mewajibkan tepat dua layanan runtime. Perintah pemeriksaannya:

```bash
docker-compose config --services | wc -l   # harus 2
```

Bentuk plugin (`docker compose config --services | wc -l`) setara; pakai yang
tersedia di mesin. Vite dev server tidak dihitung karena ia tidak pernah masuk
`docker-compose.yml`.

## Kontrak API di /docs (hanya pengembangan)

Saat `APP_ENV=development`, `serve` menyajikan Swagger UI di `/docs`. Buka
`http://localhost:8080/docs` untuk membaca kontrak tanpa membuka YAML mentah;
spec-nya juga tersedia di `/docs/openapi.yaml`. Halaman ini membaca salinan
`backend/apidocs/openapi.yaml` yang disematkan, yang disegel byte-identik dengan
`docs/001-capacity-exchange-marketplace/contracts/openapi.yaml`. Setelah kontrak
sumber berubah, jalankan `./backend/apidocs-sync.sh` lalu commit.

Rute `/docs` didaftarkan hanya di pengembangan. Di produksi ia absen dan
alamatnya jatuh ke 404 yang sama seperti path tak dikenal lain, jadi UI ini tidak
pernah terekspos ke publik.
