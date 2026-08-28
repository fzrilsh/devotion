# docs/001-capacity-exchange-marketplace/contracts/README.md

# Peta Kontrak → Requirement

Berkas: `openapi.yaml` (OpenAPI 3.1). Setiap operasi memuat `x-fr` sebagai penunjuk
requirement, memenuhi Gate III konstitusi.

## Cakupan

65 operasi pada 57 path. Seluruh 91 FR spec tercakup, kecuali tiga belas yang memang bukan
kontrak API, dan itu disebut eksplisit di bawah agar `/analyze` tidak melaporkannya
sebagai celah. Skema `ErrorCode` di `openapi.yaml` memuat 31 kode galat mesin yang stabil.

## FR yang tidak diwujudkan sebagai endpoint

| FR | Ditegakkan di mana |
|----|--------------------|
| FR-010 | Perilaku, bukan endpoint: listing tayang tanpa menunggu verifikasi |
| FR-019, FR-078 | Algoritma alokasi di dalam `internal/order` |
| FR-024 | Larangan: skor tidak menerima faktor selain empat kriteria keras |
| FR-037 | Perilaku penjadwal: request tak dibalas melewati batas ditandai kedaluwarsa |
| FR-040 | Larangan: tidak ada endpoint pembayaran, dan itu memang tujuannya |
| FR-055, FR-056 | Antarmuka frontend |
| FR-075 | Subcommand `seed:regions` dan `seed:master-data` |
| FR-076 | Bentuk `ListingRequest`: tidak ada kolom kapasitas per jenis produk |
| FR-079 | Constraint basis data |
| FR-085, FR-086 | Perilaku pengirim notifikasi |

## Peta per User Story

| Story | Endpoint utama |
|-------|----------------|
| US1 Listing kapasitas | `POST /auth/register`, `PUT /profile/me`, `POST /listing/me`, `POST /master/proposals` |
| US2 Pencarian | `GET /search`, `GET /master/products`, `GET /regions/cities`, `GET /profile/{profileId}` |
| US3 Request kuota | `POST /quota-requests`, `POST /candidates/{candidateId}/offers`, `POST /offers/{offerId}/counter`, `POST /offers/{offerId}/accept` |
| US4 Kalender | `GET PUT /listing/me/periods` |
| US5 Pesanan | `GET /work-orders`, `POST /work-orders/{workOrderId}/status`, `/confirm`, `/cancel`, `/payments`, `/disputes` |
| US6 Reputasi | `POST /work-orders/{workOrderId}/reviews`, `GET /profile/{profileId}/reviews` |
| US7 Admin | `/admin/verification`, `/admin/master/items`, `/admin/proposals`, `/admin/reviews`, `/admin/late-orders`, `/admin/disputes`, `/admin/whatsapp`, `/admin/whatsapp/reconnect` |

## Kewajiban pengujian per endpoint

Konstitusi mewajibkan setiap endpoint punya minimal dua pengujian, satu jalur berhasil dan
satu penolakan peran, dan satu penolakan masukan tidak sah bagi endpoint yang menerima
masukan. Dengan 65 operasi, itu sekitar 150 pengujian endpoint, di luar pengujian aturan
yang disebut khusus konstitusi.

## Endpoint yang wajib diuji secara khusus

| Endpoint | Yang diuji |
|----------|------------|
| `GET /search` | Urutan dapat diulang; stabil antar halaman; skor tidak terpengaruh verifikasi dan reputasi; kapasitas terjumlah lintas periode; listing sendiri dikecualikan |
| `POST /offers/{offerId}/accept` | Dua kesepakatan berbarengan atas periode sama; alokasi mengisi minggu terawal; kegagalan sebagian membatalkan seluruhnya |
| `POST /candidates/{candidateId}/offers` | Penolakan saat kapasitas kurang, dengan angka tersisa yang benar |
| `POST /work-orders/{workOrderId}/cancel` | Seluruh alokasi dibalik; tertutup setelah produksi |
| `POST /work-orders/{workOrderId}/status` | Transisi melompat ditolak beserta daftar transisi yang diizinkan |
| `POST /work-orders/{workOrderId}/disputes` | Menghentikan hitungan konfirmasi otomatis |
| `GET /files/{fileId}` | Bukan pemilik dan bukan admin ditolak |
| `POST /quota-requests` | Listing sendiri ditolak, termasuk tanpa melalui pencarian |
| Konfirmasi otomatis | Dengan sumber waktu digantikan, bukan menunggu tujuh hari |

## Catatan penerapan

- Kursor paginasi bersifat opaque bagi klien: teruskan apa adanya, jangan diurai.
- `POST /auth/recover/request` selalu 202 agar tidak membocorkan keberadaan akun.
- `GET /health` dan seluruh operasi ber-`security: []` adalah satu-satunya yang tidak
  memerlukan sesi.
- Respons galat selalu `application/problem+json`, termasuk untuk `/api/*` yang tidak
  dikenali, bukan `index.html`, agar kesalahan penulisan alamat tidak menghasilkan HTML
  yang menyesatkan saat diagnosis.