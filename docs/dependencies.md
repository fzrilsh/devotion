# Dependencies

Daftar dependency di luar standard library beserta alasannya. Prinsip IV
mewajibkan setiap tambahan dibenarkan di sini. Isi menyusul bersama task yang
menambahkannya.

## Backend

### github.com/golang-migrate/migrate/v4 (v4.18.3)

Runner migrasi. Dipakai lewat sumber `iofs` di atas berkas SQL yang di-embed,
jadi image tidak perlu membawa biner `migrate` terpisah dan migrasi jalan
otomatis saat startup di bawah `pg_try_advisory_lock`. Sudah tercantum di
Primary Dependencies plan.md. (T010)

### github.com/jackc/pgx/v5 (v5.7.5)

Driver dan pool PostgreSQL. Dipakai langsung oleh runner migrasi
(`database/pgx/v5` dan `stdlib`) dan menjadi dasar `sql_package: pgx/v5` untuk
sqlc di T011. Sudah tercantum di Primary Dependencies plan.md. (T010)

Catatan versi Go: menarik `pgx/v5` menggeser direktif `go` di `go.mod` ke
`1.24.0` dengan `toolchain go1.24.1`. Ini di atas `go 1.23.4` yang semula
dipatok, tetapi tetap memenuhi batas bawah CLAUDE.md (Go 1.22+) dan diperlukan
oleh graf modul; dibiarkan apa adanya, bukan dipaksa turun.

### golang.org/x/crypto (v0.37.0)

Semula tarikan tak langsung lewat `pgx/v5`, kini langsung: `internal/account`
memakai `golang.org/x/crypto/bcrypt` untuk hash kata sandi dengan cost 10.
CLAUDE.md mematok bcrypt sebagai satu-satunya jalur hashing kata sandi, dan
`go mod tidy` menaikkannya dari `// indirect` menjadi dependency langsung.
Sudah tercantum di Primary Dependencies plan.md. (T014)
