# Menjalankan Devotion

Cara menyalakan aplikasi secara lokal dan di server. Penerapan server lengkap
dari fresh install ada di `docs/setup-vps.md`; dokumen ini fokus pada alur
pengembang di mesin lokal.

## Alur lokal

Prasyarat: Go 1.25+, Docker, dan salinan `.env` dari `.env.example` yang sudah
diisi. Untuk pengembangan set `APP_ENV=development`.

1. **Nyalakan Postgres saja lewat compose.** Backend dijalankan dari host, bukan
   di dalam kontainer, agar iterasi cepat tanpa membangun ulang image.

   ```bash
   docker-compose up -d postgres
   ```

   Postgres dipublikasikan ke `127.0.0.1:5434`, jadi `DATABASE_URL` di `.env`
   menunjuk `postgres://...@127.0.0.1:5434/devotion`. Port 5434 dipilih agar tidak
   bertabrakan dengan Postgres lain di mesin pengembang, dan ikatannya ke loopback
   menjaga agar tidak pernah terekspos keluar mesin.
2. **Jalankan backend.** Migrasi jalan otomatis saat startup di bawah advisory
   lock, jadi tidak ada langkah migrasi manual.

   ```bash
   cd backend
   go run ./cmd/devotion serve
   ```

   Server mendengarkan di `:8080` saat pengembangan. Swagger UI ada di `/docs`
   (lihat bagian bawah).
3. **Isi data acuan dan buat akun.** Sekali saja per basis data baru.

   ```bash
   go run ./cmd/devotion seed:regions
   go run ./cmd/devotion seed:master-data
   go run ./cmd/devotion admin:create
   go run ./cmd/devotion seed:test-data      # hanya non-produksi
   ```

   `seed:test-data` menolak berjalan saat `APP_ENV=production`. Kredensial akun uji
   ada di `docs/skenario-uji-manual.md`.
4. **Uji.** Lihat `docs/pengujian.md`. Ringkasnya `cd backend && go test ./... -p 1`.

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

