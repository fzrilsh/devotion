# Utang Teknis

Jalan pintas dan penyimpangan yang dicatat karena tenggat atau karena gerbang
konstitusi saling berbenturan. Setiap entri menyebut akibatnya.

## Cron host untuk `pg_dump` harian

Konstitusi mewajibkan cadangan terjadwal dengan salinan di luar server, sementara
Gate I melarang proses terjadwal kedua di dalam compose. Cadangan dijalankan lewat
cron di tingkat host, bukan di dalam proses backend dan bukan sebagai layanan
compose.

**Alasan:** `pg_dump` harus tetap jalan justru ketika aplikasi mati atau rusak,
saat cadangan paling dibutuhkan. Penjadwal di dalam proses backend gagal memenuhi
ini, dan menambah layanan cron ke compose melanggar batas dua layanan.

**Akibat:** cron host tidak muncul di `docker-compose.yml` sehingga tidak terlihat
dari sana. Keberadaannya harus didokumentasikan agar tidak terlupakan saat setup
server.

## Kredensial akun uji di `docs/skenario-uji-manual.md`

Batasan Keamanan melarang kredensial di dokumentasi, sementara gerbang Pengujian
End-to-End mewajibkan kredensial akun uji tersedia bagi penguji eksternal yang
tidak punya akses basis data.

**Alasan:** tidak ada alternatif yang memenuhi keduanya. Dibatasi tiga syarat:
akun hanya ada pada data `seed:test-data` yang menolak berjalan saat
`APP_ENV=production`; kata sandinya tidak dipakai akun sungguhan mana pun; domain
`.test` tidak dapat diregistrasi sehingga tidak ada email nyata yang terlibat.

**Akibat:** kredensial yang tampak di dokumen publik hanya berlaku pada data uji
non-produksi. Bila `seed:test-data` sampai jalan di produksi, syarat pengaman
pertama runtuh.

## Direktori tingkat atas `.github/`

Konstitusi mewajibkan CI membangun image, dan CI GitHub Actions menuntut
`.github/workflows/`. `CLAUDE.md` melarang menambah direktori tingkat atas dan
tidak menyebut `.github/`.

**Alasan:** pipeline membangun backend dan frontend sekaligus, jadi tidak menjadi
milik salah satu area. `.github/` adalah lokasi yang diwajibkan penyedia CI, bukan
pilihan struktur.

**Akibat:** ada satu direktori tingkat atas di luar daftar `CLAUDE.md`. Terbatas
pada definisi CI saja.

## Kode status `'400'` pada path di luar User Story 1

Backend memetakan `VALIDATION_FAILED` ke 422 lewat `httpx.StatusFor`, sehingga
kunci respons validasi di `openapi.yaml` seharusnya `'422'`, bukan `'400'`. Saat
mengamandemen kontrak untuk US1, hanya path US1 (`/auth/register`, `/profile/me`
PUT, `/listing/me` POST/PUT, `/listing/me/periods` PUT, `/master/proposals`) yang
diseragamkan ke `'422'`.

**Alasan:** path story lain belum diimplementasikan, jadi menyeragamkannya
sekarang berisiko menyentuh kontrak yang masih akan berubah bersama kodenya.
Diseragamkan saat masing-masing story dikerjakan.

**Akibat:** untuk sementara `openapi.yaml` memuat campuran `'400'` dan `'422'` pada
respons validasi. Kunci `'400'` yang tersisa (mis. pada `/auth/login`,
`/auth/recover/*`, path search dan order) tidak cocok dengan status 422 yang
sebenarnya dikembalikan backend sampai path itu digarap.
