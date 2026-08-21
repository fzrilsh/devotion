# Phase 0 Research: Capacity Exchange — Devotion

**Feature**: `docs/specs/001-capacity-exchange-marketplace/`
**Date**: 2026-08-21
**Input**: `plan.md` Technical Context, `docs/memory/constitution.md` v2.1.0

Dokumen ini menyelesaikan tiga NEEDS CLARIFICATION di `plan.md` dan mencatat enam keputusan teknis lain yang mengikat Phase 1. Format setiap butir: Decision, Rationale, Alternatives considered.

Batas kejujuran dokumen ini: butir R-02 tidak dapat saya selesaikan dengan angka pasti karena saya tidak memanggil endpoint wilayah.id, dan daftar rentang alamat pada R-01 saya reproduksi dari ingatan sehingga wajib dicocokkan ke sumber resmi sebelum dipatok ke kode. Keduanya ditandai eksplisit. Batas dan kuota layanan luar (aturan rate limiting Cloudflare paket gratis, batas ukuran body proxy, kuota harian Mailjet) tidak saya hafal dan berubah dari waktu ke waktu; semuanya diperiksa langsung di dasbor masing-masing saat penyiapan.

---

## R-01. Menolak koneksi yang tidak datang dari Cloudflare

**Decision**: Pakai **tiga lapisan sekaligus**, karena masing-masing menutup kegagalan yang berbeda.

1. **Firewall host** hanya mengizinkan TCP 443 dari rentang alamat Cloudflare. Port 80 ditutup sepenuhnya dari internet — Cloudflare selalu menghubungi origin di 443 pada mode Full (strict), dan pengalihan HTTP ke HTTPS sudah ditangani di tepi.
2. **Cloudflare Origin Certificate** dipasang di Go, mode SSL/TLS di dasbor diset **Full (strict)**. Ini yang mengenkripsi segmen edge–origin.
3. **Authenticated Origin Pulls** diaktifkan, dan Go memverifikasi sertifikat klien Cloudflare lewat `tls.Config.ClientCAs` + `ClientAuth: tls.RequireAndVerifyClientCert`.

Lapisan 1 gugur bila rentang alamat berubah dan tidak diperbarui. Lapisan 3 tetap menahan meski firewall salah konfigurasi. Karena itu keduanya dipasang, bukan salah satu.

Konfigurasi TLS di Go:

```go
// Sertifikat origin dari Cloudflare, berlaku 15 tahun, hanya dipercaya Cloudflare.
cert, err := tls.LoadX509KeyPair(cfg.OriginCertPath, cfg.OriginKeyPath)

// CA milik Cloudflare untuk memverifikasi Authenticated Origin Pulls.
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(cfg.CloudflareClientCA)

srv := &http.Server{
    Addr: ":443",
    TLSConfig: &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientCAs:    pool,
        ClientAuth:   tls.RequireAndVerifyClientCert,
        MinVersion:   tls.VersionTLS12,
    },
    ReadHeaderTimeout: 10 * time.Second,
}
```

**Rentang alamat Cloudflare untuk firewall dan pemeriksaan `CF-Connecting-IP`.**

PERINGATAN: daftar di bawah saya reproduksi dari ingatan dan **wajib dicocokkan** ke `https://www.cloudflare.com/ips-v4` dan `/ips-v6` sebelum dipatok. Konstitusi Prinsip V mewajibkan data acuan dari layanan luar diambil sekali lalu disimpan sendiri, jadi patok sebagai konstanta Go beserta tanggal pengambilan — jangan mengambilnya lewat jaringan saat startup, karena satu kegagalan HTTP akan membuat aplikasi gagal naik.

IPv4 (15 blok):

```text
173.245.48.0/20     103.21.244.0/22     103.22.200.0/22
103.31.4.0/22       141.101.64.0/18     108.162.192.0/18
190.93.240.0/20     188.114.96.0/20     197.234.240.0/22
198.41.128.0/17     162.158.0.0/15      104.16.0.0/13
104.24.0.0/14       172.64.0.0/13       131.0.72.0/22
```

IPv6 (7 blok):

```text
2400:cb00::/32      2606:4700::/32      2803:f800::/32
2405:b500::/32      2405:8100::/32      2a06:98c0::/29
2c0f:f248::/32
```

**Membaca alamat asal** (Batasan Keamanan konstitusi: hanya dipercaya bila koneksinya memang dari rentang tersebut):

```go
func RealIP(r *http.Request) string {
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    ip := net.ParseIP(host)
    if ip == nil {
        return ""
    }
    if !inCloudflareRange(ip) {
        return host // koneksi langsung: jangan percaya header apa pun
    }
    if cf := net.ParseIP(r.Header.Get("CF-Connecting-IP")); cf != nil {
        return cf.String()
    }
    return host
}
```

Alamat asli ini dipakai untuk dua hal saja: pembatasan laju per-IP pada pengiriman kode sekali pakai, dan log audit keputusan admin. Tidak dipakai untuk otorisasi.

**Rationale**: Mode Flexible adalah kegagalan paling umum dan paling sulit didiagnosis — Cloudflare bicara HTTP ke origin, aplikasi mengira koneksi tidak aman, cookie `Secure` tidak terkirim, dan login gagal tanpa pesan galat yang jelas. Menutup port 80 dan mewajibkan sertifikat klien membuat kesalahan itu mustahil terjadi diam-diam: yang salah akan gagal keras, bukan gagal senyap. Tanpa lapisan firewall atau Authenticated Origin Pulls, siapa pun yang mengetahui alamat VPS bisa melewati seluruh lapisan tepi termasuk rate limiting-nya.

**Alternatives considered**:

- **Cloudflare Tunnel** — origin tidak membuka port apa pun, paling aman. Ditolak karena `cloudflared` adalah daemon, yaitu proses runtime kedua; melanggar Gate I secara langsung.
- **Let's Encrypt lewat ACME di Go** — ditolak karena tantangan HTTP tidak dapat diselesaikan ketika proxy Cloudflare aktif, dan tantangan DNS menuntut kredensial API Cloudflare di server plus penanganan perpanjangan. Origin Certificate berlaku 15 tahun tanpa perpanjangan sama sekali.
- **Hanya firewall, tanpa Authenticated Origin Pulls** — ditolak karena satu kesalahan aturan firewall langsung membuka origin ke internet tanpa gejala apa pun.
- **Sertifikat mandiri (self-signed)** dengan Full (strict) — ditolak; Cloudflare tidak mempercayainya, dan Origin Certificate memberi hal yang sama tanpa masalah itu.

---

## R-02. Sumber data wilayah

**Decision**: `devotion seed:wilayah` mengambil **dua tingkat saja** — provinsi dan kabupaten/kota — dari wilayah.id, menyimpannya ke Postgres, dan **sekaligus menulis salinan** ke `docs/master-data/wilayah.json`. Bila berkas salinan sudah ada, perintah default membaca berkas itu; pengambilan dari jaringan hanya terjadi dengan flag eksplisit `--refresh`. Kecamatan dan desa tidak diambil.

```bash
# Pengambilan pertama, sekali saja, saat menyiapkan project:
devotion seed:wilayah --refresh   # ambil dari wilayah.id, tulis JSON, isi DB

# Semua pemakaian berikutnya, termasuk di VPS dan CI:
devotion seed:wilayah             # baca docs/master-data/wilayah.json, isi DB
```

Idempoten memakai kode wilayah sebagai identitas: sisipkan bila kode belum ada, perbarui nama bila sudah ada, **jangan pernah menghapus** karena Profil Usaha menunjuk ke baris itu.

**PERINGATAN — yang tidak dapat saya pastikan**: saya tidak memanggil endpoint wilayah.id, sehingga saya **tidak mengetahui** bentuk JSON responsnya, nama field-nya, apakah kodenya sesuai kode resmi BPS/Kemendagri, dan apakah layanannya masih aktif hari ini. Yang saya ketahui hanya pola URL yang kamu berikan:

```text
/api/provinces.json
/api/regencies/[PROVINCE_CODE].json
/api/districts/[REGENCY_CODE].json     # tidak dipakai
/api/villages/[DISTRICT_CODE].json     # tidak dipakai
```

Langkah pertama implementasi seeder adalah memanggil dua endpoint pertama, memeriksa bentuk aslinya, lalu menuliskan bentuk itu ke `docs/master-data/README.md`. Saya tidak mengarang struktur field di sini karena akan salah.

Bentuk sasaran setelah normalisasi, apa pun bentuk aslinya:

```json
{
  "diambil_pada": "2026-08-21",
  "sumber": "https://wilayah.id/api/",
  "provinsi": [
    {
      "kode": "32",
      "nama": "Jawa Barat",
      "kota_kabupaten": [
        { "kode": "3273", "nama": "Kota Bandung" },
        { "kode": "3204", "nama": "Kabupaten Bandung" }
      ]
    }
  ]
}
```

Bila wilayah.id tidak dapat dijangkau atau bentuknya tidak sesuai harapan, jalur mundurnya: isi manual hanya provinsi dan kabupaten/kota untuk lima kota target dokumen sumber [1] — cukup untuk demo, dan dapat dilengkapi kemudian tanpa mengubah skema.

**Rationale**: Memanggil layanan luar saat melayani permintaan pengguna dilarang Prinsip V, dan alasannya praktis: satu gangguan jaringan akan mematikan FR-022 dan FR-063 di depan juri. Salinan di repository membuat seed tetap berjalan meski layanan sumber sedang mati saat penyiapan demo — dan penyiapan demo adalah saat paling buruk untuk menemukan bahwa sumber data tidak dapat dijangkau. Kecamatan dan desa tidak diambil karena tidak ada satu pun requirement yang memakainya, sementara jumlahnya puluhan ribu baris pada disk 50GB.

**Alternatives considered**:

- **Memanggil wilayah.id saat pengguna memilih lokasi** — ditolak; melanggar Prinsip V dan menjadikan pendaftaran bergantung pada layanan pihak ketiga.
- **Berkas JSON manual sepenuhnya tanpa API** — ditolak karena mengetik 38 provinsi dan sekitar 500 kabupaten/kota rawan galat dan memakan waktu yang lebih baik dipakai membangun fitur.
- **Mengambil keempat tingkat termasuk kecamatan dan desa** — ditolak; puluhan ribu baris tanpa requirement yang memakainya.
- **Pengelompokan "wilayah sekitar" buatan sendiri** (Bandung Raya, Jabodetabek) — sudah ditolak pada revisi spec karena Jabodetabek melintasi tiga provinsi sehingga perluasan ke provinsi justru akan menyempitkan hasil. FR-062 kini memakai pembagian administratif resmi, dengan konsekuensi yang tercatat di Assumptions.

---

## R-03. Penyetelan PostgreSQL dan pool koneksi untuk 2GB RAM

**Decision**: Angka konkret di bawah, dipasang lewat `command` pada layanan Postgres di `docker-compose.yml`.

| Parameter | Nilai | Alasan |
|-----------|-------|--------|
| `max_connections` | `20` | Default 100 mengasumsikan mesin jauh lebih besar; setiap koneksi menyita memori |
| `shared_buffers` | `256MB` | Sekitar 12,5% dari 2GB. Anjuran umum 25% tidak dipakai karena Go, whatsmeow, dan sistem operasi harus berbagi mesin yang sama |
| `effective_cache_size` | `768MB` | Bukan alokasi, hanya petunjuk bagi perencana query bahwa ada cache sistem operasi sebesar ini |
| `work_mem` | `4MB` | Berlaku **per operasi pengurutan**, bukan per koneksi. Satu query bisa punya beberapa node pengurutan, jadi nilai besar berbahaya pada 20 koneksi |
| `maintenance_work_mem` | `64MB` | Hanya dipakai saat `VACUUM` dan pembuatan indeks, tidak bersamaan dengan trafik puncak |
| `wal_buffers` | `8MB` | Memadai untuk volume tulis sekecil ini |
| `min_wal_size` / `max_wal_size` | `128MB` / `512MB` | Menahan pertumbuhan WAL agar disk tidak terisi |
| `checkpoint_completion_target` | `0.9` | Menyebar tulisan checkpoint agar tidak ada lonjakan I/O |
| `random_page_cost` | `1.1` | VPS memakai SSD; default 4.0 mengasumsikan cakram berputar dan membuat perencana enggan memakai indeks |
| `effective_io_concurrency` | `200` | Sesuai karakteristik SSD |
| `log_min_duration_statement` | `500ms` | Mencatat query lambat sebagai bahan diagnosis SC-010 tanpa membanjiri log |
| `timezone` | `Asia/Jakarta` | Menyelaraskan dengan Prinsip V; batas minggu tetap dihitung eksplisit di kode, tidak diserahkan ke pengaturan ini |

Pool di sisi Go, **15**, sengaja di bawah `max_connections`:

```go
cfg, _ := pgxpool.ParseConfig(dsn)
cfg.MaxConns        = 15               // sisakan 5 untuk psql, pg_dump, dan migrasi
cfg.MinConns        = 2
cfg.MaxConnLifetime = 30 * time.Minute
cfg.MaxConnIdleTime = 5 * time.Minute
cfg.HealthCheckPeriod = 1 * time.Minute
```

Sisa 5 koneksi bukan kelebihan: `pg_dump` harian, sesi `psql` untuk diagnosis, dan migrasi saat penerapan versi baru semuanya memerlukan koneksi. Bila pool memakai seluruh 20, cadangan harian akan gagal tepat saat trafik sedang tinggi.

**Swap 2GB** dipasang dengan `vm.swappiness=10`. Bukan untuk dipakai rutin, melainkan agar lonjakan sesaat tidak langsung berakhir dengan proses dibunuh kernel — dan yang biasanya dibunuh adalah Postgres, bukan penyebab lonjakannya.

**Batas log kontainer** pada kedua layanan compose. Ini pencegahan kegagalan total, bukan kebersihan: log yang tumbuh tanpa batas akan mengisi 50GB, lalu Postgres berhenti menerima tulisan dan aplikasi mati.

```yaml
logging:
  driver: json-file
  options:
    max-size: "10m"
    max-file: "3"
```

**Rationale**: Anjuran penyetelan Postgres yang beredar umumnya menganggap mesin hanya menjalankan Postgres. Di sini satu mesin 2GB menanggung Postgres, Go, whatsmeow, dan sistem operasi, sehingga `shared_buffers` sengaja ditekan di bawah anjuran 25%. `work_mem` ditahan kecil karena sifatnya per operasi pengurutan — kesalahan paling umum adalah menaikkannya sampai perkalian dengan jumlah koneksi melampaui memori yang ada.

Verifikasi setelah dipasang, jangan dipercaya begitu saja: jalankan `free -h` dan `docker stats` saat aplikasi menangani pencarian, dan bandingkan dengan anggaran memori di `plan.md`. Bila `shared_buffers` 256MB terbukti terlalu kecil untuk memenuhi SC-010, naikkan bertahap ke 384MB sambil mengawasi pemakaian swap.

**Alternatives considered**:

- **Default Postgres tanpa penyetelan** — ditolak; `max_connections=100` saja sudah berisiko kehabisan memori pada 2GB.
- **`shared_buffers` 512MB (25% dari 2GB)** — ditolak untuk nilai awal karena menyisakan terlalu sedikit bagi Go dan sistem operasi. Dapat dipertimbangkan hanya setelah pengukuran nyata.
- **Postgres di luar Docker, langsung di host** — sedikit lebih hemat memori. Ditolak karena membuat penyiapan dan pemulihan lebih rumit, sementara `docker-compose.yml` adalah cara memeriksa Gate I.
- **PgBouncer untuk pooling** — ditolak; proses runtime tambahan, melanggar Gate I, dan pooling di sisi pgx sudah memadai untuk skala ini.

---

## R-04. Penguncian baris pada alokasi kapasitas lintas periode

**Decision**: Satu transaksi tunggal yang mencakup pembentukan pesanan dan seluruh baris Alokasi Kapasitas (FR-084), dengan **penguncian baris berurutan menaik menurut tanggal awal periode**, ditambah **constraint tabel** sebagai jaring pengaman.

Urutan di dalam transaksi:

1. `BEGIN`
2. `SELECT ... FROM periode_ketersediaan WHERE listing_id = $1 AND minggu_mulai BETWEEN $2 AND $3 ORDER BY minggu_mulai FOR UPDATE` — mengunci seluruh periode kandidat sekaligus, terurut.
3. Jumlahkan kapasitas tersisa dari periode yang tidak ditandai penuh. Bila kurang dari jumlah pesanan, `ROLLBACK` dan tolak dengan menyebutkan total yang sebenarnya tersisa (FR-035).
4. Sisipkan pesanan.
5. Isi periode paling awal lebih dulu sampai jumlah terpenuhi (FR-018), lewati periode penuh atau habis (FR-078), sisipkan satu baris Alokasi Kapasitas per periode yang terpakai (FR-077), naikkan `kapasitas_terpakai`.
6. `COMMIT`

Constraint yang menegakkan FR-079 pada tingkat penyimpanan data:

```sql
ALTER TABLE periode_ketersediaan
  ADD CONSTRAINT kapasitas_terpakai_tidak_melebihi_total
  CHECK (kapasitas_terpakai >= 0 AND kapasitas_terpakai <= kapasitas_total);
```

Pengurutan `ORDER BY minggu_mulai` pada langkah 2 adalah pencegah deadlock, bukan kerapian: dua transaksi yang mengunci periode yang sama dalam urutan berbeda akan saling menunggu. Dengan urutan yang selalu sama, yang kedua hanya menunggu lalu melihat kapasitas yang sudah berkurang.

Pembatalan (FR-020) membalik seluruh baris alokasi di dalam satu transaksi, dengan pola penguncian yang sama.

**Rationale**: FR-036 mewajibkan kapasitas satu periode hanya jatuh ke satu kesepakatan, dan SC-018 menjadikannya kriteria yang harus dibuktikan lewat pengujian dua kesepakatan berbarengan. Membaca lalu menulis tanpa penguncian akan lolos pada seluruh pengujian manual dan baru terlihat sebagai kapasitas minus di produksi. Constraint tabel dipasang karena logika aplikasi bisa keliru: bila itu terjadi, basis data menolak dan transaksi gagal keras, bukan data yang rusak diam-diam.

**Alternatives considered**:

- **Penguncian optimistis dengan kolom versi** — ditolak; satu pesanan menyentuh beberapa periode, sehingga jumlah percobaan ulang dan penanganan kegagalan sebagiannya lebih rumit daripada penguncian pesimistis di sini.
- **`SELECT FOR UPDATE` per periode satu per satu** — ditolak; membuka celah deadlock kecuali urutannya dijaga ketat, dan itu justru yang dicapai dengan satu perintah terurut.
- **`SERIALIZABLE` isolation** — benar secara teori, ditolak karena menuntut penanganan percobaan ulang di setiap jalur tulis untuk masalah yang cukup diselesaikan penguncian baris.
- **Hanya constraint tabel tanpa penguncian** — ditolak; akan menolak dengan galat basis data yang tidak dapat dijelaskan ke pengguna, sementara FR-035 menuntut pesan yang menyebutkan total kapasitas sebenarnya.

---

## R-05. Keyset pagination yang menjaga SC-013

**Decision**: Paginasi berbasis kursor tuple, bukan `OFFSET`. Kursor memuat kelima kolom pengurutan FR-025 dan diteruskan sebagai string ter-encode.

Urutan penuh dan deterministik:

```sql
ORDER BY skor_kecocokan DESC,
         kapasitas_tersisa_sampai_deadline DESC,
         jeda_kesiapan_mulai ASC,
         nama_usaha ASC,
         listing_id ASC
```

`listing_id` di posisi terakhir adalah pemecah seri final yang menjamin urutan tidak pernah ambigu — tanpa itu, dua listing yang identik pada empat kolom pertama bisa bertukar posisi antar permintaan dan SC-013 gagal.

Halaman berikutnya membandingkan tuple terhadap nilai baris terakhir halaman sebelumnya:

```sql
WHERE (skor, kapasitas_tersisa, -jeda, nama, id) < (:skor, :kapasitas, -:jeda, :nama, :id)
```

Skor kecocokan dihitung sebagai jumlah empat nilai boolean, sesuai FR-023, sehingga nilainya 0 sampai 4 dan dapat diurutkan langsung di SQL:

```sql
(produk_cocok::int + mesin_cocok::int + jeda_cocok::int + kapasitas_cukup::int) AS skor_kecocokan
```

**Rationale**: `OFFSET` melanggar SC-013 secara struktural. Ketika ada listing baru tayang di antara dua permintaan halaman, seluruh baris bergeser: satu kandidat muncul dua kali dan satu lainnya terlewat. Itu persis yang dilarang Acceptance Scenario 5 pada User Story 2. Kursor tuple tidak terpengaruh penyisipan baris baru karena posisinya ditentukan nilai, bukan hitungan. Kebetulan juga lebih cepat pada halaman jauh, meski pada 50 usaha demo perbedaannya tidak akan terasa — yang penting di sini kebenarannya, bukan kecepatannya.

**Alternatives considered**:

- **`LIMIT`/`OFFSET`** — ditolak, melanggar SC-013 seperti diuraikan di atas.
- **Memuat seluruh hasil lalu memaginasi di frontend** — memadai untuk 50 usaha, ditolak karena akan gagal pada target 200 usaha SC-003 dan menuntut penulisan ulang justru ketika platform mulai dipakai.
- **Kursor berupa nomor baris hasil** — ditolak; nomor baris berubah begitu data berubah, jadi tidak lebih baik dari `OFFSET`.

---

## R-06. Menyematkan frontend dan fallback SPA

**Decision**: CI membangun frontend dengan Vite, menyalin hasilnya ke `backend/webdist/`, lalu `embed.FS` menyematkannya ke dalam biner. Satu biner memuat frontend dan backend sekaligus.

```go
//go:embed all:webdist
var webFS embed.FS
```

Aturan perutean, berurutan:

1. `/api/*` → handler API.
2. Berkas statis yang benar-benar ada di `webdist` → dilayani dengan header cache panjang untuk aset ber-hash nama.
3. Semua path lain → `index.html` dengan `Cache-Control: no-cache`, agar penyegaran pada halaman dalam tidak menghasilkan 404.
4. `/api/*` yang tidak dikenali → 404 JSON, **bukan** `index.html`. Tanpa aturan ini, kesalahan penulisan alamat endpoint akan mengembalikan HTML dan menghasilkan pesan galat yang menyesatkan saat diagnosis.

Cloudflare disetel agar **tidak** meng-cache `/api/*`. Aset statis boleh dan sebaiknya di-cache. Hasil pencarian yang ter-cache akan menampilkan kapasitas tersisa yang basi — dan data kapasitas yang tidak aktual adalah persis masalah yang platform ini dibangun untuk menyelesaikan [1].

**Rationale**: Gate I versi 2.1.0 mewajibkan frontend disajikan proses backend yang sama. Penyematan memberi tiga hal sekaligus: satu artefak penerapan, satu origin sehingga CORS tidak perlu ada dan cookie `SameSite=Lax` bekerja normal, dan tidak ada kemungkinan frontend dan backend berbeda versi karena keduanya satu biner.

**Alternatives considered**:

- **Melayani dari direktori di disk** dengan volume Docker — ditolak; membuka kemungkinan versi frontend dan backend tidak sinkron, dan menambah satu volume yang harus diurus.
- **Nginx atau Caddy menyajikan statis** — ditolak; proses runtime kedua, melanggar Gate I.
- **Frontend di Vercel, backend di VPS** — ditolak; dua origin berarti CORS, preflight yang menambah latensi terhadap SC-010, dan cookie lintas domain yang diblokir sebagian browser.

---

## R-07. Pekerjaan terjadwal: penjadwal dalam proses berpasangan dengan perhitungan saat baca

**Decision**: **Dua lapisan**, karena masing-masing menutup kelemahan yang lain.

**Lapisan 1 — perhitungan saat baca.** Keadaan yang bergantung tenggat dihitung dari data, bukan dari kolom yang harus diperbarui:

| Requirement | Cara dihitung |
|-------------|---------------|
| FR-021 kalender basi | `diperbarui_pada < now() - 7 hari` dievaluasi saat pencarian |
| FR-037 request kedaluwarsa | `dikirim_pada + 72 jam < now()` dievaluasi saat request dibaca |
| FR-045 pesanan lewat deadline | `deadline < now()` dievaluasi saat pesanan dibaca |
| FR-068 konfirmasi otomatis | Pesanan berstatus Dikirim dengan `dikirim_pada + 7 hari < now()` diperlakukan sebagai dikonfirmasi |

**Lapisan 2 — penjadwal dalam proses** (`time.Ticker`, setiap 5 menit) untuk hal yang tidak dapat dihitung saat baca karena harus terjadi tanpa dipicu siapa pun: pengiriman notifikasi tenggat mendekat (FR-069), pengingat kalender basi (FR-021), pemberitahuan pesanan lewat deadline (FR-045), dan penulisan status final agar riwayat mencatat waktu yang benar.

```go
type Clock interface{ Now() time.Time }
```

`Clock` disuntikkan ke setiap service. Memanggil `time.Now()` di dalam logika bisnis dilarang, karena Prinsip V mewajibkan seluruh hitungan bertenggat dapat diuji dengan waktu yang digantikan — dan tanpa itu, FR-068 hanya dapat diverifikasi dengan menunggu tujuh hari.

Setiap pekerjaan penjadwal dibungkus `pg_try_advisory_lock`. Satu proses seharusnya tidak berlomba dengan dirinya sendiri, tetapi saat penerapan versi baru ada jeda ketika kontainer lama dan baru sempat hidup bersamaan, dan tanpa penguncian itu notifikasi dapat terkirim dua kali.

**Rationale**: Penjadwal saja tidak cukup — bila proses mati beberapa jam, pesanan yang tenggatnya lewat pada rentang itu tidak akan pernah diperiksa, dan FR-068 gagal senyap. Perhitungan saat baca saja juga tidak cukup, karena FR-069 mewajibkan pemberitahuan terkirim sebelum tenggat jatuh, tanpa menunggu pengguna membuka aplikasi. Keduanya bersama membuat konsistensi tidak bergantung pada keandalan proses.

Syarat yang harus dijaga: perhitungan tenggat ditulis satu kali di satu fungsi domain dan dipakai kedua lapisan. Bila kondisinya diduplikasi di beberapa handler, keduanya akan berbeda pada suatu titik dan pesanan yang sama akan tampak berbeda status di halaman berbeda.

**Alternatives considered**:

- **Cron eksternal memanggil satu endpoint** — ditolak; menambah layanan luar untuk hal yang dapat diselesaikan di dalam proses, dan endpoint pemicu adalah permukaan serangan baru yang harus diamankan.
- **Hanya perhitungan saat baca** — ditolak; FR-069 menuntut pengiriman tanpa dipicu pengguna.
- **Hanya penjadwal dalam proses** — ditolak; kehilangan tenggat secara permanen bila proses sempat mati.
- **`pg_cron`** — ditolak; extension yang menambah pekerjaan latar di dalam Postgres, di luar jangkauan pengujian Go, dan logika bisnis akan tersebar ke dua tempat.

---

## R-08. Ketahanan sesi whatsmeow

**Decision**: whatsmeow berjalan sebagai goroutine di dalam proses backend dengan store sesi pada Postgres yang sama. Ditambah empat hal karena FR-002 menjadikan verifikasi nomor HP sebagai gerbang, sehingga sesi yang lepas berarti tidak ada akun baru yang dapat dibuat.

1. **Halaman admin QR dan status sambungan** — menampilkan QR saat sesi perlu disambungkan ulang, dan status terkini. Tanpa ini, satu sesi yang lepas menuntut akses SSH, dan itu tidak mungkin dilakukan saat demo berjalan.
2. **Endpoint health menyertakan status whatsmeow**, agar pemantau uptime eksternal memberi tahu sebelum juri menemukannya.
3. **Subcommand darurat** `devotion user:verify --phone` untuk memverifikasi akun secara manual. Bukan untuk pengguna, melainkan agar demo tidak hilang bila nomor terblokir satu jam sebelum penjurian.
4. **Email dipertahankan sebagai kanal kedua** — FR-052 tetap utuh, sehingga pemulihan akun (FR-003) tidak bergantung pada satu library tidak resmi.

Pengiriman WhatsApp mengikuti pola notifikasi umum: baris ditulis ke tabel di dalam transaksi kejadiannya, goroutine pengirim mencoba paling banyak tiga kali (FR-085), kegagalan tidak menggagalkan transaksi apa pun (FR-086), dan notifikasi di dalam platform tetap tampil (FR-054).

**Risiko yang diterima secara sadar**: whatsmeow memakai protokol WhatsApp Web multidevice, bukan API resmi. Mengirim kode sekali pakai ke banyak nomor yang belum pernah berinteraksi adalah pola yang paling mungkin dideteksi sebagai spam. Pembatasan laju per nomor dan per alamat asal mengurangi risiko itu, tidak menghilangkannya. Nomor yang dipakai adalah nomor khusus lomba, bukan nomor pribadi anggota tim, dan nilainya hanya ada di variabel lingkungan — tidak di repository, dokumentasi, maupun artefak perencanaan.

**Rationale**: API resmi Meta menuntut verifikasi bisnis yang tidak akan selesai sebelum tenggat, sementara dokumen sumber mencantumkan notifikasi WhatsApp sebagai bagian dari fitur yang dijanjikan [1]. whatsmeow adalah satu-satunya jalan, jadi keputusannya bukan apakah memakainya, melainkan bagaimana agar kegagalannya tidak menjatuhkan seluruh demo.

**Alternatives considered**:

- **API resmi WhatsApp Business** — ditolak; verifikasi bisnis tidak selesai pada waktunya.
- **Penyedia gateway tidak resmi pihak ketiga** — ditolak; memindahkan risiko blokir ke pihak lain tetapi menambah layanan luar, biaya, dan ketergantungan yang tidak dapat didiagnosis sendiri.
- **Menghapus WhatsApp dan hanya memakai email** — ditolak; menghilangkan fitur yang dijanjikan dokumen sumber, sementara mitigasinya sudah memadai.
- **Menjalankan whatsmeow sebagai proses terpisah** — ditolak; melanggar Gate I.

---

## R-09. Email transaksional lewat Mailjet dan penyiapan DNS

**Decision**: Mailjet melalui SMTP dengan `net/smtp` dari standard library, tanpa menambah dependency SDK. Pengirim `noreply@<domain>`. Tiga record DNS dipasang di Cloudflare **pada awal pengerjaan, bukan menjelang tenggat**: SPF, DKIM (nilainya diberikan Mailjet), dan DMARC dengan `p=none` pada tahap ini.

Verifikasi email memakai **kode enam digit**, bukan tautan sekali klik, karena penguji dapat membuka email di perangkat yang berbeda dari perangkat pendaftarannya. Pengiriman berjalan di goroutine — respons HTTP tidak pernah menunggu SMTP selesai — dan tombol kirim ulang selalu tersedia dengan jeda yang membesar.

**Rationale**: Reputasi domain adalah satu-satunya bagian dari daftar ini yang tidak dapat dipercepat di hari terakhir. Domain yang baru dibeli, belum pernah mengirim apa pun, lalu tiba-tiba mengirim email berisi kode angka adalah profil pengirim yang paling dicurigai, dan Gmail maupun Outlook akan menaruhnya di spam tanpa memberi galat apa pun yang dapat ditangkap aplikasi. Kegagalan email bersifat senyap: server penerima menjawab sukses lalu membuang pesannya. Dasbor Mailjet memberi satu-satunya cara mengetahui apakah pesan benar-benar terkirim, dan itu yang menyelamatkan proses diagnosis.

Karena FR-002 menjadikan verifikasi email sebagai gerbang, kegagalan pengiriman berarti **tidak ada satu pun user story yang dapat didemokan** — bukan sekadar fitur notifikasi yang rusak, melainkan pintu masuknya.

**Alternatives considered**:

- **Mail server sendiri di VPS** — ditolak; banyak penyedia memblokir port 25 keluar, alamat IP baru sering sudah masuk daftar hitam karena penyewa sebelumnya, dan membangun reputasi IP butuh berminggu-minggu.
- **SMTP Gmail dengan app password** — ditolak; kuota kecil, alamat pengirim `@gmail.com` terlihat tidak serius saat penjurian, dan pola otomatis dapat memicu pemblokiran.
- **SDK resmi Mailjet** — ditolak; `net/smtp` sudah memadai, dan Prinsip IV melarang dependency untuk hal yang dapat diselesaikan tanpanya.
- **Tautan sekali klik alih-alih kode** — ditolak; menyulitkan penguji yang berpindah perangkat.

---

## R-10. Sesi, kata sandi, dan pembatasan laju

**Decision**:

**Sesi** — cookie `httpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, dengan token acak 32 byte dari `crypto/rand` yang **hash-nya** disimpan di tabel sesi Postgres. Menyimpan hash, bukan token mentah, berarti kebocoran isi tabel tidak langsung berarti pengambilalihan akun. Masa berlaku 7 hari dengan perpanjangan bergulir. Keluar akun menghapus baris sesi, sehingga benar-benar berakhir.

**Kata sandi** — bcrypt dengan cost 10. Cost 12 lebih kuat tetapi menambah waktu dan penggunaan CPU per proses masuk pada mesin 2GB; 10 masih di atas ambang wajar dan dapat dinaikkan tanpa mengubah skema karena bcrypt menyimpan cost di dalam hash-nya.

**Pembatasan laju** memakai tabel Postgres, bukan penyimpanan dalam memori, agar tetap berlaku setelah proses dijalankan ulang:

| Sasaran | Batas | Menutup |
|---------|-------|---------|
| Percobaan masuk per akun | 5 per 15 menit, jeda membesar | Penebak kata sandi yang berpindah alamat IP |
| Kode sekali pakai per nomor | 3 per jam | Penekan tombol kirim ulang berulang |
| Kode sekali pakai per alamat asal | 10 nomor berbeda per jam | Pemutar nomor yang memancing blokir WhatsApp |
| Request kuota per pengguna | 20 per jam | Pembanjir seluruh subkontraktor (FR-029 mengizinkan kirim ke banyak kandidat sekaligus) |

Keempatnya adalah logika domain yang tidak dapat diserahkan ke Cloudflare, karena Cloudflare tidak mengetahui apa itu akun, nomor, maupun request kuota.

**Rationale**: Satu origin membuat JWT tidak diperlukan sama sekali, dan sesi di basis data memberi hal yang JWT tidak bisa: pembatalan seketika. Menyimpan token di `localStorage` sengaja dihindari — satu celah XSS akan langsung berarti pengambilalihan akun, dan aplikasi ini memuat dokumen identitas yang FR-009 wajib lindungi.

**Alternatives considered**:

- **JWT tanpa keadaan tersimpan** — ditolak; tidak dapat dibatalkan sebelum kedaluwarsa, dan tidak memberi manfaat apa pun pada satu origin dengan satu basis data.
- **argon2id** — lebih kuat, ditolak sebagai pilihan awal karena biaya memorinya perlu disetel hati-hati pada 2GB yang sudah berbagi dengan Postgres.
- **Pembatasan laju dalam memori** — ditolak; hilang setiap kali proses dijalankan ulang, dan penerapan versi baru menjadi cara termudah melewatinya.
- **Menyerahkan seluruh pembatasan laju ke Cloudflare** — ditolak; paket gratis memberi jumlah aturan yang sangat terbatas, dan tidak satu pun batas berbasis akun atau nomor dapat dinyatakan di sana.

---

## Ringkasan Status NEEDS CLARIFICATION

| Butir `plan.md` | Status | Catatan |
|-----------------|--------|---------|
| Penolakan koneksi non-Cloudflare | **Terjawab** (R-01) | Tiga lapisan. Rentang alamat wajib dicocokkan ke sumber resmi sebelum dipatok |
| Bentuk dan ketersediaan wilayah.id | **Terjawab bersyarat** (R-02) | Strategi dan bentuk sasaran ditetapkan; bentuk respons asli belum diperiksa dan tidak saya karang. Jalur mundur tersedia |
| Penyetelan Postgres untuk 2GB | **Terjawab** (R-03) | Angka konkret siap pakai, wajib diverifikasi dengan pengukuran setelah dipasang |

Enam keputusan lain (R-04 sampai R-10) tidak berasal dari daftar NEEDS CLARIFICATION tetapi mengikat Phase 1: tanpa R-04 dan R-05, `data-model.md` dan `contracts/` tidak dapat disusun secara benar.

Tidak ada NEEDS CLARIFICATION yang tersisa. Gate Constitution diperiksa ulang setelah Phase 1.

**Output berikutnya**: `data-model.md`, `contracts/` (OpenAPI 3.1), dan `quickstart.md` termasuk runbook VPS dari fresh install.