# CLAUDE.md, Devotion

Panduan wajib untuk agent yang menulis kode di repository ini. Baca ini sebelum
menyentuh berkas apa pun.

**Devotion** (artinya kesetiaan) adalah platform Capacity Exchange: marketplace
subkontrak B2B yang mempertemukan UMKM konveksi berkapasitas produksi menganggur
dengan UMKM yang order-nya melebihi kapasitas sendiri. Masalah yang dijawab:
pencarian subkontraktor yang selama ini hanya lewat relasi personal sehingga
jangkauannya terbatas dan tidak ada mekanisme matching sistematis [1].

**Konteks**: submission lomba, bukan sistem produksi. Dinilai dari demo yang
berjalan dan kualitas dokumen, di bawah tenggat ketat.

---

## Sumber Kebenaran

Jangan menebak. Setiap keputusan sudah tertulis di salah satu berkas ini:

| Butuh apa | Baca |
|-----------|------|
| Prinsip dan gerbang mutu | `docs/memory/constitution.md` |
| Perilaku produk, 91 requirement | `docs/001-capacity-exchange-marketplace/spec.md` |
| Endpoint, skema, kode galat | `docs/001-capacity-exchange-marketplace/contracts/openapi.yaml` |
| Tabel, constraint, indeks | `docs/001-capacity-exchange-marketplace/data-model.md` |
| Alasan keputusan teknis | `docs/001-capacity-exchange-marketplace/research.md` |
| Struktur folder, dependency | `docs/001-capacity-exchange-marketplace/plan.md` |
| Cara menjalankan, skenario uji | `docs/001-capacity-exchange-marketplace/quickstart.md` |
| Daftar pekerjaan | `docs/001-capacity-exchange-marketplace/tasks.md` |

Bila kode bertentangan dengan spec, **spec yang benar**. Ubah spec lebih dulu,
baru kode. Jangan pernah sebaliknya.

Bila spec bertentangan dengan constitution: constitution menang pada hal teknis,
spec menang pada hal perilaku produk.

---

## Delapan Aturan yang Tidak Boleh Dilanggar

Melanggar salah satu berarti pekerjaan belum selesai, apa pun alasannya.

### 1. Maksimal dua layanan runtime

`docker-compose.yml` hanya boleh memuat `backend` dan `postgres`. Tidak ada
yang lain. Tidak ada nginx, Caddy, Redis, worker, cron container, message
broker, maupun layanan frontend.

Frontend disajikan oleh biner Go yang sama lewat `embed.FS`. TLS dihabiskan Go
sendiri memakai Cloudflare Origin Certificate.

Ini aturan dari panitia lomba. Pelanggarannya menggugurkan submission, bukan
sekadar menambah utang teknis.

**Cara memeriksa**: hitung entri di bawah `services:`. Lebih dari dua berarti
pelanggaran.

### 2. Jangan pakai Next.js API routes

Frontend adalah Vite + React SPA. Menaruh endpoint di frontend menciptakan
backend kedua di samping Go, dan itu pelanggaran aturan 1 yang paling mudah
terjadi tanpa disadari.

### 3. Uang selalu bilangan bulat rupiah

`bigint` di Postgres, `int64` di Go, `integer` di TypeScript. Tidak ada
`float`, `decimal`, maupun `numeric` untuk uang. Rupiah tidak punya pecahan
dalam praktik B2B ini.

### 4. Minggu dimulai Senin, Asia/Jakarta

Periode kapasitas disimpan sebagai kolom `date` berisi tanggal Senin awal
minggu, bukan `timestamptz`. Semua pergeseran batas minggu dihitung di WIB.

Waktu kejadian (perubahan status, pencatatan pembayaran) tetap `timestamptz`,
dikonversi ke WIB hanya saat ditampilkan.

Mencampur keduanya adalah sumber bug yang mahal: pesanan bisa jatuh ke minggu
yang salah dan kapasitas berkurang dari periode yang keliru.

### 5. Dilarang `time.Now()` di dalam logika bisnis

```go
type Clock interface{ Now() time.Time }
```

`Clock` disuntikkan ke setiap service. Tanpa ini, konfirmasi otomatis tujuh hari
hanya dapat diuji dengan menunggu tujuh hari.

### 6. Dependency baru wajib dibenarkan

Default-nya standard library. Daftar dependency yang sudah disetujui ada di
`plan.md` bagian Primary Dependencies. **Jangan menambah di luar daftar itu
tanpa bertanya lebih dulu.**

Yang sengaja diselesaikan tanpa dependency: log terstruktur (`log/slog`), email
(`net/smtp`), pembuangan EXIF (`image/jpeg`), token acak (`crypto/rand`), UUID
(`gen_random_uuid()` Postgres), jarak haversine (aritmetika sendiri, bukan
PostGIS), router (`net/http` bawaan Go 1.22+).

Satu kemampuan, satu dependency. Tidak ada dua library untuk urusan sama.

### 7. Seluruh antarmuka bahasa Indonesia

Label, pesan galat, judul halaman, notifikasi, semuanya. Mobile-first, dan
alur inti harus bisa diselesaikan dengan keyboard serta terbaca pembaca layar.

Dokumen sumber menempatkan dukungan multi-bahasa sebagai non-goal karena fokus
pasar domestik [1], jadi jangan buat lapisan i18n.

### 8. Tanpa AI slop

Tulisan di kode, komentar, dokumen, pesan commit, dan seluruh keluaran harus
terbaca seperti ditulis manusia.

- **Dilarang em-dash (`—`).** Pakai koma, titik, atau tanda kurung. Ini penanda
  AI slop yang paling kentara dan paling sering lolos tanpa sengaja.
- Hindari frasa klise pembuka dan penutup: "it's worth noting", "in today's
  fast-paced world", "delve into", "in conclusion", "furthermore", "moreover",
  dan sejenisnya, termasuk padanan bahasa Indonesianya.
- Jangan menebalkan kata secara berlebihan, jangan bikin daftar berpoin kalau
  kalimat biasa sudah cukup, jangan menutup dengan ringkasan yang tidak diminta.
- Langsung ke inti. Tulis sepadat yang tersisa maknanya.

Kalau ragu sebuah kalimat terdengar seperti mesin, tulis ulang.

---

## Struktur Repository

```text
devotion/
├── CLAUDE.md              # berkas ini
├── README.md              # template panitia, JANGAN ubah strukturnya
├── LICENSE                # MIT
├── docker-compose.yml     # tepat 2 layanan
├── .env.example           # nama variabel tanpa nilai
├── backend/
│   ├── cmd/devotion/      # subcommand: serve, admin:create, seed:*, reset:*
│   ├── internal/
│   │   ├── platform/      # clock, config, httpx, session, storage, scheduler,
│   │   │                  # ratelimit, cloudflare
│   │   ├── account/       # akun, peran, profil, verifikasi identitas
│   │   ├── masterdata/    # daftar baku, wilayah, usulan item
│   │   ├── listing/       # listing kapasitas, periode ketersediaan
│   │   ├── search/        # kriteria keras, skor, pemecah seri, keyset
│   │   ├── quota/         # request kuota, penawaran, counter-offer
│   │   ├── order/         # pesanan, alokasi kapasitas, pembatalan, pembayaran
│   │   ├── reputation/    # ulasan, tingkat penyelesaian
│   │   ├── notification/  # antrean, pengirim, percobaan ulang
│   │   └── admin/         # verifikasi, moderasi, mediasi
│   ├── db/migrations/     # golang-migrate, berurutan
│   ├── db/queries/        # sumber SQL untuk sqlc
│   └── webdist/           # hasil build frontend, disematkan embed.FS
├── frontend/src/
│   ├── pages/             # satu berkas per layar, dikelompokkan per user story
│   ├── components/
│   ├── api/               # klien, tipe hasil generate, hook TanStack Query
│   ├── schemas/           # skema Zod
│   └── lib/
└── docs/
```

Jangan menambah direktori tingkat atas. Berkas pengujian berada di dalam
`backend/` dan `frontend/`, bukan di direktori `tests/` tingkat atas.

Arah ketergantungan antar paket backend yang harus dijaga: `order` mengubah
kapasitas milik `listing` lewat baris alokasi, jadi `order` bergantung pada
`listing`, bukan sebaliknya. `search` hanya membaca. Tidak boleh ada
ketergantungan melingkar.

---

## Strategi Branch, Commit, dan Push

Repository ini monolith tapi dikerjakan per domain. Alur branch dipisah antara
frontend dan backend supaya keduanya bisa maju paralel tanpa saling menabrak.

### Aturan

- **Setiap unit kerja punya branch sendiri.** Satu `feat`, `fix`, `chore`, dan
  sejenisnya tidak pernah dikerjakan langsung di branch integrasi. Buat branch
  kerja lebih dulu, selesaikan di sana, baru gabungkan.
- **Penamaan branch kerja**: `<area>/<tipe>/<short-desc>`.
  - Frontend: `frontend/feat/login-form`, `frontend/fix/cursor-paginasi`.
  - Backend: `backend/feat/alokasi-kapasitas`, `backend/chore/sqlc-regen`.
  - `<tipe>` mengikuti Conventional Commits: `feat`, `fix`, `chore`, `docs`,
    `refactor`, `test`, dan seterusnya. `<short-desc>` singkat dan `kebab-case`.
- **Branch integrasi per area**: branch kerja digabung ke `develop/<area>`.
  - Frontend: `frontend/feat/*` -> `develop/frontend`.
  - Backend: `backend/feat/*` -> `develop/backend`.
- **Branch staging menggabungkan semuanya.** `develop/frontend` dan
  `develop/backend` digabung ke `staging` untuk pengujian terintegrasi. Semua
  uji end-to-end lintas domain terjadi di sini.
- **`main` adalah rilis.** Hanya menerima gabungan yang sudah lolos di
  `staging`. Jangan push kerja harian langsung ke `main`.

### Alur ringkas

```text
frontend/feat/short-desc ─┐
                          ├─► develop/frontend ─┐
frontend/fix/short-desc ──┘                     │
                                                ├─► staging ─► main
backend/feat/short-desc ──┐                     │
                          ├─► develop/backend ──┘
backend/chore/short-desc ─┘
```

Arah gabung selalu naik: branch kerja -> `develop/<area>` -> `staging` ->
`main`. Jangan pernah melompati tingkat, dan jangan menggabungkan mundur tanpa
alasan yang dicatat.

### CHANGELOG per area

Setiap perubahan, entah fitur baru, perbaikan, penghapusan, atau perubahan
perilaku, dicatat di `CHANGELOG.md` milik area yang dikerjakan, bukan di satu
berkas gabungan.

- Kerja di backend dicatat di `backend/CHANGELOG.md`.
- Kerja di frontend dicatat di `frontend/CHANGELOG.md`.

Catat entri di CHANGELOG pada branch kerja yang sama dengan perubahannya, dalam
commit yang sama bila memungkinkan, supaya riwayat dan catatan tidak terpisah.
Perubahan tanpa entri CHANGELOG dianggap belum selesai.

---

## Backend (Go)

**Stack**: Go 1.22+, `net/http` (router bawaan), `pgx/v5` + `sqlc`,
`golang-migrate`, `bcrypt`, `whatsmeow`, `sentry-go`, PostgreSQL 16.

### Yang harus benar

- **SQL ditulis eksplisit** di `db/queries/`, lalu `sqlc` menghasilkan kode Go.
  Jangan pakai ORM. Query pencarian dan skor kecocokan adalah bagian paling
  penting di project ini dan harus terlihat.
- **Setiap endpoint memeriksa peran pemanggil.** Endpoint tanpa pemeriksaan
  peran dianggap cacat, bukan sekadar belum lengkap.
- **Alokasi kapasitas dalam satu transaksi** dengan `SELECT ... FOR UPDATE`
  terurut menaik menurut `minggu_mulai`. Urutan itu pencegah deadlock, bukan
  kerapian.
- **Sesi**: cookie `httpOnly`, `Secure`, `SameSite=Lax`. Yang disimpan di
  database adalah **hash** token, bukan token mentah.
- **Migrasi** jalan otomatis saat startup dengan `pg_try_advisory_lock`.
- **Pekerjaan terjadwal** dua lapisan: dihitung saat data dibaca, plus
  `time.Ticker` di dalam proses yang sama. Bukan proses kedua. Setiap pekerjaan
  penjadwal dibungkus advisory lock.
- **Kegagalan notifikasi tidak boleh menggagalkan transaksi.** Baris notifikasi
  ditulis di dalam transaksi kejadiannya; pengiriman ke email dan WhatsApp
  berjalan setelahnya, maksimal 3 percobaan.
- **Galat** memakai `application/problem+json` dengan `code` mesin yang stabil
  dan `detail` bahasa Indonesia yang bisa dikutip penguji. Daftar 28 kode ada di
  `openapi.yaml`.
- **Log** memakai `log/slog` format JSON dengan request ID di setiap baris.
- **Berkas unggahan** tidak pernah dilayani lewat path statis. Selalu lewat
  handler yang memeriksa peran. Nama berkas dibuat sistem (UUID), tipe divalidasi
  dari magic bytes bukan dari header, metadata lokasi gambar dibuang.

### Yang mudah salah

- Perhitungan tenggat ditulis **satu kali** di satu fungsi domain, dipakai
  kedua lapisan penjadwal. Kalau diduplikasi, pesanan yang sama akan tampak
  berbeda status di halaman berbeda.
- `/api/*` yang tidak dikenali mengembalikan **404 JSON**, bukan `index.html`.
  Kalau HTML, kesalahan penulisan alamat endpoint jadi menyesatkan saat diagnosis.
- Header alamat asal dari proxy tepi **hanya dipercaya** bila koneksinya memang
  datang dari rentang alamat Cloudflare yang sudah dipatok.

---

## Frontend (React)

**Stack**: Vite, React 18, TypeScript, TanStack Query, Zod + React Hook Form,
Tailwind CSS, Leaflet + OpenStreetMap, Jest.

### Yang harus benar

- **Generate tipe dari `openapi.yaml`**, jangan tulis tangan. Tipe yang ditulis
  tangan akan menyimpang dari kontrak tanpa ada yang tahu.
- **`credentials: 'include'`** pada semua permintaan. Token **tidak pernah**
  disimpan di `localStorage` maupun `sessionStorage`. Satu celah XSS akan
  langsung berarti pengambilalihan akun, dan aplikasi ini memuat dokumen
  identitas.
- **Jangan duplikasi mesin keadaan pesanan.** `PesananDetail` sudah mengirim
  `transisi_diizinkan` dan `boleh_dibatalkan_sendiri`. Render tombol dari array
  itu. Kalau logikanya ditulis ulang di React, dua tempat akan berbeda pada
  suatu titik.
- **Kursor paginasi bersifat opaque.** Teruskan `kursor_berikutnya` apa adanya.
  Jangan diurai, jangan diubah jadi `?page=2`. Itu langsung melanggar jaminan
  urutan stabil antar halaman.
- **Tampilkan penjelasan kriteria** pada hasil pencarian. Respons mengirim
  `kriteria` per kandidat; pengguna harus bisa melihat kriteria mana yang tidak
  terpenuhi.
- **Tingkat penyelesaian** hanya ditampilkan sebagai persentase bila
  `cukup_data: true`. Kalau tidak, tampilkan keterangannya.
- **Peta** memakai Leaflet dengan tile OpenStreetMap. Tanpa kunci API. Jarak
  bersifat informatif saja.
- **Validasi di frontend tidak menggantikan validasi di backend.** Keduanya ada.

### Yang mudah salah

- State selain data server cukup Context bawaan React. **Jangan tambah Redux
  atau Zustand**. Tidak ada state global yang rumit di aplikasi ini.
- Satu component library saja, atau Tailwind saja. Jangan dua-duanya.
- Kalau memilih component library, pilih yang komponennya sudah benar secara
  aksesibilitas.

---

## Pengujian

Pengujian otomatis **diwajibkan**. Cakupannya berupa kewajiban yang bisa
diperiksa, bukan angka persentase.

Minimum per endpoint:

1. Satu jalur berhasil.
2. Satu penolakan karena peran pemanggil tidak berwenang.
3. Satu penolakan masukan tidak sah, bila endpoint menerima masukan.

Setiap pengujian **menyebutkan FR yang diverifikasinya** pada namanya atau
komentar di atasnya:

```go
func TestPencarian_UrutanDapatDiulang_FR023_FR025_SC013(t *testing.T) { … }
```

Pengujian backend memakai **skema terpisah pada layanan Postgres yang sama**.
Jangan menambah layanan basis data untuk pengujian.

Pengujian bertenggat memakai `Clock` yang digantikan, bukan menunggu waktu nyata.

Aturan yang wajib diuji secara khusus, daftar lengkapnya di
`contracts/README.md`, tetapi ini yang paling mudah rusak diam-diam:

- Urutan hasil pencarian dapat diulang, termasuk antar halaman
- Skor tidak terpengaruh reputasi, verifikasi, kebaruan kalender, maupun jarak
- Kapasitas terjumlah lintas periode sampai deadline
- Dua kesepakatan berbarengan atas periode yang sama: hanya satu berhasil
- Pembatalan pra-produksi membalik seluruh baris alokasi
- Larangan request kuota ke listing milik sendiri
- Konfirmasi otomatis tujuh hari, dan penghentiannya oleh sengketa
- Tingkat penyelesaian membebani hanya pihak yang membatalkan
- Dokumen identitas tidak dapat diakses selain pemilik dan admin

Pengujian end-to-end dijalankan **manual oleh penguji di luar tim** mengikuti
`quickstart.md` bagian F. Karena itu, label dan pesan galat harus jelas dan bisa
dikutip dalam laporan.

---

## Gerbang Sebelum Sebuah Story Dinyatakan Selesai

- [ ] Seluruh Acceptance Scenario story itu bisa dijalankan lewat antarmuka,
      tanpa menyentuh database manual dan tanpa penjelasan lisan
- [ ] Pengujian otomatis ada, menunjuk FR, dan lulus
- [ ] Seluruh pengujian sebelumnya masih lulus
- [ ] Skenario uji manual untuk story itu sudah tertulis beserta datanya
- [ ] Jumlah layanan di `docker-compose.yml` masih dua
- [ ] Dependency baru, bila ada, sudah dicatat alasannya
- [ ] Endpoint baru memeriksa peran, dan ada pengujian yang membuktikan penolakan

---

## Keamanan

- Kredensial hanya di variabel lingkungan. **Jangan pernah** menulis nilai
  kredensial, kunci API, atau nomor telepon layanan di dalam kode, dokumentasi,
  maupun artefak perencanaan. Repository ini publik.
- `.env` tidak pernah di-commit. `.env.example` hanya memuat nama kunci.
- Kata sandi dengan bcrypt.
- Pembatasan laju berbasis data domain ditegakkan di aplikasi, bukan diserahkan
  ke proxy tepi: percobaan masuk per akun, kode sekali pakai per nomor dan per
  alamat asal, request kuota per pengguna.
- Data yang dikirim ke Sentry dibersihkan dari kata sandi, token, nomor telepon,
  dan apa pun yang menyangkut dokumen identitas.
- Data uji tidak memuat data pribadi orang sungguhan.

---

## Batas Keuangan

Platform **tidak** menahan, menyalurkan, maupun memproses dana pihak mana pun.

Dilarang memasang payment gateway, escrow, maupun dompet internal. Pembayaran
terjadi langsung antar pihak; platform hanya mencatat pernyataan mereka, tanpa
kolom jumlah uang.

Melewati batas ini menuntut perubahan spec lebih dulu.

---

## Batas Sumber Daya

VPS 2GB RAM, 50GB disk.

- **Jangan pernah build di server.** Build Vite pada mesin 2GB sambil Postgres
  hidup akan kehabisan memori, dan yang dibunuh kernel biasanya Postgres. CI
  membangun image, server hanya menarik dan menjalankan.
- Ukuran log kontainer dibatasi. Log yang tumbuh tanpa batas akan mengisi disk,
  lalu Postgres berhenti menulis dan aplikasi mati total.
- Total unggahan dibatasi 500MB, 5MB per berkas.
- `max_connections` Postgres 20, pool di Go 15. Lima disisakan untuk `pg_dump`,
  `psql`, dan migrasi.

---

## Bila Menemukan Pertentangan

Jangan menebak dan jangan diam-diam menyimpang. Lakukan salah satu:

1. Bila spec keliru → sebutkan FR mana, usulkan perbaikannya, tunggu keputusan.
2. Bila constitution menghalangi hal yang benar-benar dibutuhkan → catat di
   tabel Complexity Tracking `plan.md` beserta alasan dan alternatif yang
   ditolak.
3. Bila tenggat memaksa jalan pintas → catat di `docs/utang-teknis.md` beserta
   akibatnya.

Pelanggaran yang tidak tercatat berarti pekerjaan itu belum selesai.