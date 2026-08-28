# Pengujian

Cara menjalankan pengujian otomatis dan aturan yang wajib dipatuhi setiap uji
baru. Cakupan berupa kewajiban yang bisa diperiksa, bukan angka persentase.

## Menjalankan uji backend

```bash
cd backend
go test ./...              # seluruh paket
go test ./... -p 1         # serial antar paket, wajib bila memakai basis data
```

`-p 1` menjalankan paket secara berurutan agar total koneksi tetap di bawah
`max_connections=20` (pool `pgx` 15). Tanpa itu, beberapa paket yang berjalan
paralel menghabiskan koneksi dan gagal dengan SQLSTATE 53300. CI memakai flag
yang sama.

Uji yang butuh basis data menunjuk `DATABASE_URL_TEST`. Di CI, Postgres
disediakan sebagai layanan infrastruktur runner, bukan layanan runtime, jadi
Gate I tidak terpengaruh.

## Menjalankan uji frontend

Isi menyusul bersama T003. Perintahnya `cd frontend && npm test` (Jest).

## Skema terpisah, bukan basis data terpisah

Uji backend memakai skema terpisah pada layanan Postgres yang sama. Konstitusi
melarang menambah layanan basis data untuk keperluan pengujian.

## Waktu yang digantikan untuk alur bertenggat

Uji yang menyangkut tenggat memakai `Clock` yang digantikan, bukan menunggu
waktu nyata. Konfirmasi otomatis tujuh hari diverifikasi dengan menggeser waktu
uji, bukan dengan menunggu tujuh hari. `time.Now()` tidak boleh muncul di dalam
logika bisnis.

## Penamaan uji menyebut FR

Setiap uji menyebutkan FR yang diverifikasinya pada namanya atau komentar di
atasnya:

```go
func TestPencarian_UrutanDapatDiulang_FR023_FR025_SC013(t *testing.T) { ... }
```

## Minimum per endpoint

1. Satu jalur berhasil.
2. Satu penolakan karena peran pemanggil tidak berwenang.
3. Satu penolakan masukan tidak sah, bila endpoint menerima masukan.

Daftar lengkap aturan yang wajib diuji secara khusus ada di
`docs/001-capacity-exchange-marketplace/contracts/README.md`.
