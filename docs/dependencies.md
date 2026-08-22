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

### golang.org/x/crypto (v0.54.0)

Semula tarikan tak langsung lewat `pgx/v5`, kini langsung: `internal/account`
memakai `golang.org/x/crypto/bcrypt` untuk hash kata sandi dengan cost 10.
CLAUDE.md mematok bcrypt sebagai satu-satunya jalur hashing kata sandi, dan
`go mod tidy` menaikkannya dari `// indirect` menjadi dependency langsung.
Sudah tercantum di Primary Dependencies plan.md. (T014) Versi dinaikkan ke
v0.54.0 saat whatsmeow ditambahkan di T024a.

### golang.org/x/term (v0.45.0)

Membaca kata sandi admin dari terminal tanpa echo di subcommand `admin:create`
(`term.ReadPassword`). Kata sandi tidak boleh lewat flag karena flag tersimpan
di riwayat shell, dan stdlib tidak punya cara portabel mematikan echo terminal.
Ikut transitif bersama `golang.org/x/crypto`, jadi tidak menambah pohon modul
baru; `go mod tidy` menaikkannya dari `// indirect` menjadi dependency langsung.
Di luar Primary Dependencies plan.md, dicatat di sini sesuai Prinsip IV. (T020)

### go.mau.fi/whatsmeow (v0.0.0-20260720135917-a2381054887e)

Klien WhatsApp yang menyalurkan kode verifikasi dan notifikasi. Sudah tercantum
di Primary Dependencies plan.md. Jalan sebagai goroutine di dalam proses `serve`,
bukan layanan kedua (R-08), jadi Gate I tetap dua layanan.

whatsmeow menyimpan sesinya lewat `store/sqlstore`, yang menuntut handle
`database/sql` sendiri, bukan pool `pgx` yang dipakai aplikasi. Handle kedua
dibuka dengan driver `stdlib` pgx (`sql.Open("pgx", DATABASE_URL)`) ke database
yang sama, dan dibatasi `SetMaxOpenConns(2)`. Ini bersaing dengan pool utama
melawan `max_connections=20`: pool Go memakai 15, jadi handle ini dianggarkan
dari 5 koneksi yang disisakan untuk `pg_dump`, `psql`, dan migrasi. Batas 2
menyisakan ruang bagi ketiganya.

Nomor layanan tidak pernah muncul di respons, log, maupun Sentry (FR-082):
`Status` tidak punya field untuk membawanya, dan galat whatsmeow tidak
menyematkannya.

Catatan versi Go: menarik whatsmeow menggeser direktif `go` di `go.mod` ke
`1.25.0`, di atas `1.24.0` yang digeser pgx sebelumnya. Tetap memenuhi batas
bawah CLAUDE.md (Go 1.22+); dibiarkan apa adanya karena dituntut graf modul.
whatsmeow juga menaikkan `golang.org/x/crypto` ke v0.54.0, `golang.org/x/term`
ke v0.45.0, dan menarik `google.golang.org/protobuf` (v1.36.11) sebagai
dependency langsung untuk membangun payload pesan (`waE2E.Message`). (T024a)

### github.com/getsentry/sentry-go (v0.35.3)

Pelapor galat. Satu-satunya penampung eksternal untuk panic dan galat tak
terduga, dipakai lewat `internal/platform/observability` sebagai satu titik
sentuh. `BeforeSend` diterapkan sebagai allowlist: event yang keluar dibangun
ulang dari field aman saja (pesan, exception, level, tag `request_id`),
sehingga `Request`, `User`, `Extra`, dan `Contexts` dibuang alih-alih disaring.
Denylist gagal diam-diam saat SDK atau kode menambah field baru, dan mode
kegagalannya adalah data dokumen identitas atau nomor layanan (FR-082) bocor ke
pihak ketiga, jadi default amannya harus "buang", bukan "teruskan". Sudah
tercantum di Primary Dependencies plan.md. Jalan dalam proses `serve`, bukan
layanan kedua, jadi Gate I tetap dua. (T025)


