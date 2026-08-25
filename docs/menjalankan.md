# Menjalankan Devotion

Cara menyalakan aplikasi secara lokal dan di server. Isi menyusul bersama task
yang memproduksinya (compose, config, serve).

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

