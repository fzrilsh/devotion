# Sumber Data Wilayah

**Sumber**: `https://wilayah.id/api/`
**Diperiksa**: 2026-08-22
**Data per**: 2025-07-04 (dari `meta.updated_at`)

Hanya dua tingkat yang diambil. Kecamatan dan desa tidak dipakai karena tidak ada
requirement yang membutuhkannya, dan jumlahnya puluhan ribu baris.

## Endpoint yang dipakai

```
GET /api/provinces.json          → 38 provinsi
GET /api/regencies/{kode}.json   → kabupaten/kota per provinsi
```

## Bentuk respons

```json
{"data":[{"code":"32","name":"Jawa Barat"}],
 "meta":{"administrative_area_level":1,"updated_at":"2025-07-04"}}

{"data":[{"code":"32.73","name":"Kota Bandung"}],
 "meta":{"administrative_area_level":2,"updated_at":"2025-07-04"}}
```

Respons dibungkus objek `data`, bukan array langsung. Field berbahasa Inggris:
`code` dan `name`.

## Normalisasi wajib

Kode kabupaten/kota dari sumber memakai titik (`32.73`). Seeder **wajib**
membuang titiknya sebelum menyimpan, sehingga menjadi `3273`.

Tanpa normalisasi, dua constraint gagal: `kota_kode_format` menuntut empat digit
tanpa pemisah, dan `kota_milik_provinsinya` membandingkan dua karakter pertama
kode kota dengan kode provinsinya. Pattern `^[0-9]{4}$` pada `openapi.yaml` juga
mengasumsikan bentuk tanpa titik.

Konversinya reversibel bila sewaktu-waktu dibutuhkan: selalu tepat dua digit
setelah titik.

## Verifikasi setelah seed

```sql
SELECT count(*) FROM wilayah_provinsi;   -- harus 38
SELECT count(*) FROM wilayah_kota
 WHERE provinsi_kode = '32';             -- harus 27
SELECT count(*) FROM wilayah_kota
 WHERE left(kode, 2) <> provinsi_kode;   -- harus 0
```

Baris ketiga adalah yang paling penting: bila lebih dari nol, normalisasi tidak
berjalan dan constraint akan menolak.

## Cadangan

`seed:wilayah --refresh` menulis hasil normalisasi ke `wilayah.json` di direktori
ini. Pemakaian berikutnya membaca berkas itu, bukan memanggil jaringan. Prinsip V
melarang bergantung pada sumber luar saat melayani permintaan pengguna, dan
salinan ini juga menyelamatkan penyiapan demo bila layanannya sedang mati.