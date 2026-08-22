# docs/001-capacity-exchange-marketplace/contracts/README.md

# Peta Kontrak → Requirement

Berkas: `openapi.yaml` (OpenAPI 3.1). Setiap operasi memuat `x-fr` sebagai penunjuk
requirement, memenuhi Gate III konstitusi.

## Cakupan

63 operasi pada 56 path. Seluruh 91 FR spec tercakup, kecuali tiga belas yang memang bukan
kontrak API, dan itu disebut eksplisit di bawah agar `/analyze` tidak melaporkannya
sebagai celah.

## FR yang tidak diwujudkan sebagai endpoint

| FR | Ditegakkan di mana |
|----|--------------------|
| FR-010 | Perilaku, bukan endpoint: listing tayang tanpa menunggu verifikasi |
| FR-019, FR-078 | Algoritma alokasi di dalam `internal/order` |
| FR-024 | Larangan: skor tidak menerima faktor selain empat kriteria keras |
| FR-037 | Perilaku penjadwal: request tak dibalas melewati batas ditandai kedaluwarsa |
| FR-040 | Larangan: tidak ada endpoint pembayaran, dan itu memang tujuannya |
| FR-055, FR-056 | Antarmuka frontend |
| FR-075 | Subcommand `seed:wilayah` dan `seed:master-data` |
| FR-076 | Bentuk `ListingRequest`: tidak ada kolom kapasitas per jenis produk |
| FR-079 | Constraint basis data |
| FR-085, FR-086 | Perilaku pengirim notifikasi |

## Peta per User Story

| Story | Endpoint utama |
|-------|----------------|
| US1 Listing kapasitas | `POST /auth/registrasi`, `PUT /profil/saya`, `POST /listing/saya`, `POST /master/usulan` |
| US2 Pencarian | `GET /pencarian`, `GET /master/produk`, `GET /wilayah/kota`, `GET /profil/{id}` |
| US3 Request kuota | `POST /request-kuota`, `POST /kandidat/{id}/penawaran`, `POST /penawaran/{id}/counter`, `POST /penawaran/{id}/terima` |
| US4 Kalender | `GET PUT /listing/saya/periode` |
| US5 Pesanan | `GET /pesanan`, `POST /pesanan/{id}/status`, `/konfirmasi`, `/batalkan`, `/pembayaran`, `/sengketa` |
| US6 Reputasi | `POST /pesanan/{id}/ulasan`, `GET /profil/{id}/ulasan` |
| US7 Admin | `/admin/verifikasi`, `/admin/master/item`, `/admin/usulan`, `/admin/ulasan`, `/admin/pesanan-telat`, `/admin/sengketa`, `/admin/whatsapp` |

## Kewajiban pengujian per endpoint

Konstitusi mewajibkan setiap endpoint punya minimal dua pengujian, satu jalur berhasil dan
satu penolakan peran, dan satu penolakan masukan tidak sah bagi endpoint yang menerima
masukan. Dengan 63 operasi, itu sekitar 150 pengujian endpoint, di luar pengujian aturan
yang disebut khusus konstitusi.

## Endpoint yang wajib diuji secara khusus

| Endpoint | Yang diuji |
|----------|------------|
| `GET /pencarian` | Urutan dapat diulang; stabil antar halaman; skor tidak terpengaruh verifikasi dan reputasi; kapasitas terjumlah lintas periode; listing sendiri dikecualikan |
| `POST /penawaran/{id}/terima` | Dua kesepakatan berbarengan atas periode sama; alokasi mengisi minggu terawal; kegagalan sebagian membatalkan seluruhnya |
| `POST /kandidat/{id}/penawaran` | Penolakan saat kapasitas kurang, dengan angka tersisa yang benar |
| `POST /pesanan/{id}/batalkan` | Seluruh alokasi dibalik; tertutup setelah produksi |
| `POST /pesanan/{id}/status` | Transisi melompat ditolak beserta daftar transisi yang diizinkan |
| `POST /pesanan/{id}/sengketa` | Menghentikan hitungan konfirmasi otomatis |
| `GET /berkas/{id}` | Bukan pemilik dan bukan admin ditolak |
| `POST /request-kuota` | Listing sendiri ditolak, termasuk tanpa melalui pencarian |
| Konfirmasi otomatis | Dengan sumber waktu digantikan, bukan menunggu tujuh hari |

## Catatan penerapan

- Kursor paginasi bersifat opaque bagi klien: teruskan apa adanya, jangan diurai.
- `POST /auth/pulihkan/permintaan` selalu 202 agar tidak membocorkan keberadaan akun.
- `GET /health` dan seluruh operasi ber-`security: []` adalah satu-satunya yang tidak
  memerlukan sesi.
- Respons galat selalu `application/problem+json`, termasuk untuk `/api/*` yang tidak
  dikenali, bukan `index.html`, agar kesalahan penulisan alamat tidak menghasilkan HTML
  yang menyesatkan saat diagnosis.