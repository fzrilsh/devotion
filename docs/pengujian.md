# Pengujian

Cara menjalankan pengujian otomatis dan aturan yang wajib dipatuhi setiap uji
baru. Cakupan berupa kewajiban yang bisa diperiksa, bukan angka persentase.

## Menjalankan uji backend

```bash
cd backend
DATABASE_URL_TEST=postgres://devotion:devotion@127.0.0.1:5434/devotion?sslmode=disable \
  go test ./... -p 1
```

`-p 1` menjalankan paket secara berurutan agar total koneksi tetap di bawah
`max_connections=20` (pool `pgx` 15). Tanpa itu, beberapa paket yang berjalan
paralel menghabiskan koneksi dan gagal dengan SQLSTATE 53300. CI memakai flag
yang sama.

`DATABASE_URL_TEST` harus disebut eksplisit di lokal, bukan diandalkan
bawaannya. Nilai bawaan di `internal/db/testdb` menunjuk port 5432, sedangkan
compose menerbitkan Postgres ke `127.0.0.1:5434`. Uji yang tidak dapat
menjangkau basis data memilih `t.Skip` sambil menyebut nama variabelnya,
sengaja, agar basis data yang mati tidak tampak seperti uji yang lulus. Akibat
praktisnya: menjalankan `go test ./...` tanpa variabel ini melewati seluruh uji
basis data dan hasilnya tetap hijau. Periksa keluarannya, `ok` yang seharusnya
`SKIP` adalah pertanda variabelnya salah.

Di CI, Postgres disediakan sebagai layanan infrastruktur runner, bukan layanan
runtime, jadi Gate I tidak terpengaruh. DSN-nya diset di blok `env` job
`backend` pada `.github/workflows/ci.yml`.

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
