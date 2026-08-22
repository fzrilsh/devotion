# Menjalankan Devotion

Cara menyalakan aplikasi secara lokal dan di server. Isi menyusul bersama task
yang memproduksinya (compose, config, serve).

## Menghitung jumlah layanan

Gate I mewajibkan tepat dua layanan runtime. Perintah pemeriksaannya:

```bash
docker-compose config --services | wc -l   # harus 2
```
