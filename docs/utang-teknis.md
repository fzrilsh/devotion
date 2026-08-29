# Utang Teknis

Jalan pintas dan penyimpangan yang dicatat karena tenggat atau karena gerbang
konstitusi saling berbenturan. Setiap entri menyebut akibatnya.

## Empat risiko yang diterima sadar

Bukan pelanggaran aturan, tetapi penyimpangan dari mitigasi yang dokumen sumber
tetapkan. Dicatat lengkap di `plan.md` bagian Risiko yang Diterima Sadar dan di
Assumptions `spec.md`; diulang di sini agar terlihat saat implementasi.

**whatsmeow memakai protokol WhatsApp Web, bukan API resmi.** FR-002 menjadikan
verifikasi nomor HP sebagai gerbang, jadi nomor yang terblokir berarti tidak ada
akun baru yang bisa dibuat saat demo.

**Akibat:** risiko blokir nomor menghentikan alur inti. Mitigasi: halaman admin
QR dan status sambungan, subcommand `user:verify` sebagai jalan darurat, email
sebagai kanal kedua, dan pembatasan laju per nomor serta per alamat asal. Ini
tumpang tindih dengan entri layanan luar WhatsApp; lihat `docs/layanan-luar.md`.

**Escrow tidak dibangun.** Dokumen sumber menempatkan penahanan dana yang dirilis
saat pesanan dikonfirmasi selesai sebagai mitigasi utama risiko gagal bayar dan
alat tawar dalam sengketa. Versi ini menggantinya dengan pencatatan pernyataan
pembayaran, sejalan dengan Batas Keuangan yang melarang memproses dana.

**Akibat:** mediasi admin kehilangan salah satu daya paksanya. Tidak ada kolom
jumlah uang di `catatan_pembayaran`; platform hanya mencatat pernyataan pihak.

**Verifikasi identitas bukan gerbang.** Hasil pencarian dapat memuat usaha yang
belum diperiksa.

**Akibat:** lencana verifikasi menjadi satu-satunya pembeda antara usaha yang
sudah dan belum diperiksa; pengguna menanggung penilaian itu sendiri.

**Skor kecocokan tidak memuat faktor perilaku.** Penalti peringkat bagi
subkontraktor yang tidak memperbarui kalender tidak dipakai.

**Akibat:** penegakan hanya lewat pengingat dan penanda "Data Belum Diperbarui".
Ini justru menjaga skor tetap deterministik dan bebas dari reputasi, verifikasi,
serta kebaruan kalender, sesuai gerbang pengujian skor.

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

## BERISIKO: tingkat penyelesaian dihitung di dua tempat kelak

Perhitungan tingkat penyelesaian (FR-071, FR-072) saat ini hanya ada di satu
tempat, `SearchReputation` pada `db/queries/search.sql`, dengan aturan pembagi
FR-072 `FILTER (WHERE status <> 'cancelled' OR cancelled_by_id = pr.id)`. Sisi
profil publik di `internal/account/profile_http.go` masih memakai
`emptyReputation()` bawaan US1 dan belum punya kueri sungguhan.

**Alasan:** ketika halaman profil publik digarap, ia butuh angka yang sama.
Menyalin logika ke kueri kedua membuka celah divergensi: pembagi bisa keliru
memasukkan pembatalan pihak lawan, sehingga profil dan hasil pencarian
menampilkan persentase berbeda untuk usaha yang sama.

**Akibat:** belum ada duplikasi hari ini, jadi belum ada bug. Risikonya muncul
saat kueri kedua ditulis. Usul: satukan ke satu kueri bernama (mis.
`ProfileReputation` yang menerima satu atau banyak `profile_id`) yang dipakai
sisi pencarian dan sisi profil, agar aturan FR-072 hidup di satu tempat saja.

## Tipe TypeScript frontend basi terhadap `openapi.yaml`

Revisi kontrak pada sesi ini mengubah beberapa skema yang dipakai frontend,
sementara tipe hasil generate T004 belum ikut diperbarui. Tiga gelombang
perubahan: `Health` (`checks` menjadi `dependencies`, plus `version` dan
`storage` kini objek); `SearchCandidate` bertambah seluruh atribut keputusan
FR-027 plus `criteria` per kandidat; `RequestCandidate` bertambah `offers` dan
`Offer` bertambah `sequence`.

**Alasan:** frontend dikerjakan di branch terpisah (`develop/frontend`), jadi
regenerasi tipe dan penyesuaian komponen menunggu jalur [FE]. Menyentuhnya dari
sisi backend berisiko konflik lintas branch.

**Akibat:** sampai tipe di-generate ulang dari `openapi.yaml`, komponen yang
memakai tipe lama akan menyimpang diam-diam dari respons backend. Peringatan
regenerasi sudah dicatat di `tasks.md` pada T037 dan T044 supaya jalur [FE]
melakukannya sebelum task itu mulai.

## `checkStorage` menyusuri direktori unggahan tiap health check

`checkStorage` memanggil `filepath.WalkDir` atas direktori unggahan pada setiap
permintaan health check. Endpoint itu dipukul healthcheck Docker tiap 30 detik
plus monitor uptime tiap 5 menit, jadi direktori disusuri berkali-kali per menit.

**Alasan:** total unggahan dibatasi 500MB dengan 5MB per berkas, dan target demo
sekitar 50 usaha, jadi jumlah berkas kecil dan penyusuran murah. Konstitusi
menunda optimasi sampai terbukti perlu, jadi ini dicatat, bukan diperbaiki.

**Akibat:** tidak ada masalah pada skala demo. Bila volume unggahan tumbuh
sampai penyusuran terasa, kandidat perbaikannya cache pendek atas hasil
`checkStorage` (mis. beberapa detik) supaya health check tidak menyusuri disk
tiap kali dipanggil.

## Hasil audit `openapi.yaml` vs kode backend (6 domain)

Audit menyeluruh membandingkan `contracts/openapi.yaml` dengan handler Go
sebenarnya di seluruh domain (quota, account/verification, admin/masterdata,
notification, listing, search, order, reputation). Tujuannya memverifikasi
kontrak sudah cocok dengan implementasi sebelum jalur [FE] men-generate ulang
tipe dari kontrak. Temuan dikelompokkan menurut dampaknya. Yang belum
diputuskan arah perbaikannya (kontrak vs kode) ditandai jelas.

### A. Bentuk respons berbeda dari kontrak (paling berisiko bagi tipe FE)

Ini yang paling berbahaya: klien hasil generate akan salah tipe.

- **POST `/work-orders/{id}/payments`** balikin `WorkOrderDetail` penuh
  (`internal/order/payment.go:77-82`), kontrak bilang `PaymentRecord`
  (`openapi.yaml:1189`). **Sudah diperbaiki**: kontrak diselaraskan ke
  `WorkOrderDetail`.
- **POST `/work-orders/{id}/disputes`** balikin `WorkOrderDetail` penuh
  (`internal/order/dispute.go:58-63`), kontrak bilang `Dispute`
  (`openapi.yaml:1215`). **Sudah diperbaiki**: kontrak diselaraskan ke
  `WorkOrderDetail`.
- **GET `/admin/proposals`** balikin envelope `{items, pagination}` dengan
  `proposer_name` dan tanpa `reason` (`internal/masterdata/admin.go:209-216,265`),
  kontrak bilang array `ItemProposal` telanjang dengan `reason`
  (`openapi.yaml:1440-1447`). **Sudah diperbaiki**: kontrak jadi envelope
  `{items, pagination}` dengan `ItemProposal` bertambah `proposer_name`, dan
  `proposalQueueItem` kini mengisi `reason` dari `admin_note` yang sudah di-SELECT
  `ListItemProposalsPending`, jadi kontrak dan kode sepakat penuh.
- **`Notification.work_order_id`** diemit backend sebagai path, bukan UUID
  sebagaimana skema menuntut. **Sudah diperbaiki**: skema dan field respons
  diganti nama jadi `link`, string nullable tanpa `format: uuid`.

Kedua kasus order (payments, disputes) adalah pilihan desain yang konsisten di
kode: tiap mutasi pesanan memuat ulang dan balikin detail. Keputusan diambil:
kontrak diselaraskan ke `WorkOrderDetail`, sesuai aturan spec-first hanya bila
kontrak keliru; di sini kode yang benar dan kontrak yang menyimpang, jadi kontrak
yang berubah.

### B. Endpoint didokumentasikan tapi tidak ada di backend

- **POST `/work-orders/{id}/confirm`** ~~didokumentasikan
  (`openapi.yaml:1098-1114`) tetapi tidak ada handler mana pun~~. **Sudah
  diperbaiki**: handler ditambahkan di `internal/order/confirm.go` (gate peran
  buyer, query `PartyConfirmWorkOrder`, sengketa terbuka menahan konfirmasi).
  Konfirmasi manual buyer kini punya endpoint, di samping auto-confirm tujuh hari.
- **`confirmed` muncul di `allowed_transitions`** dari status `shipped`
  (`internal/order/accept.go:465-469`). ~~`/status` menolak `confirmed` dan
  `/confirm` tidak ada~~, sehingga FE yang render tombol dari array itu tidak
  punya tujuan kirim. **Sudah diperbaiki**: `/confirm` kini jadi tujuan kirim
  tombol itu. Gap alur order tertutup.

### C. Field selalu null atau tak terdokumentasi

- **`distance_km`** ada di skema SearchCandidate tetapi `viewOf` tidak pernah
  set (`internal/search/search.go:86,257-274`), selalu null. Sesuai catatan
  bahwa jarak informatif saja.
- **`verification_status`** di respons profil ~~hardcoded nil
  (`internal/account/profile_http.go:114,170`)~~. **Sudah diperbaiki**: query
  `LatestVerificationStatusByProfile` mengisi field dari pengajuan verifikasi
  terbaru; null selama profil belum pernah mengajukan.
- **`region_level` dan `relaxation`** diemit search
  (`search.go:100-103,111-112,202,220`) tetapi tidak ada di kontrak. Justru
  `relaxation` (menyimpan `most_restrictive` + `suggestion`) yang dibutuhkan T037
  untuk tombol perluas tier wilayah. **Sudah diperbaiki**: kontrak diperluas.
- **`rejection_reason`** ~~di-SELECT (`db/queries/request.sql:45,184`) tetapi tak
  pernah diserialisasi ke respons~~. **Sudah diperbaiki**: `candidateView` dan
  `detailCandidateView` menserialisasi `rejection_reason` (null sebelum ditolak),
  dan kontrak menambahkan field itu ke `RequestCandidate` (FR-035).

### D. Kode status dan security berbeda

- **Endpoint verify** wajib session padahal kontrak menandai `security: []`
  (`internal/account/handlers.go:156-166`).
- **PATCH `/me/roles`** ~~balikin 422, kontrak bilang 409
  (`handlers.go:388-389`)~~. **Sudah diperbaiki**: kode galat `ROLES_IN_USE`
  (409) ditambahkan dan dipakai `errRolesActive`; kasus kedua peran false tetap
  422 karena itu validasi masukan.
- **`/search`** balikin 422 untuk query buruk, kontrak dokumentasikan 400 dan
  tidak punya 403 (`internal/search/http.go:29`). Terkait entri "Kode status
  `'400'` pada path di luar User Story 1" di atas.
- **Offer note maxLength 500** di backend (`internal/quota/offer.go:51`) vs 1000
  di kontrak (`openapi.yaml:888`).
- **`max_lead_days`** default 365 di kontrak, backend memperlakukannya sebagai
  unset.

### E. Minor dan kosmetik

- **T045**: sisi baca incoming-request tidak pernah expose quantity, deadline,
  kapasitas-dalam-rentang, maupun can-fulfill (`internal/quota/detail.go:81-89`).
  Butuh perubahan backend plus kontrak, bukan FE saja.
- **`WorkOrderList` items** ~~menserialisasi `product_item_id: ""` (bukan UUID
  valid) dan `readiness_lead_days: 0` karena `workOrderView` tanpa `omitempty`
  (`internal/order/workorder.go:453-478`)~~. **Sudah diperbaiki**: kedua field
  diberi `omitempty` sehingga baris daftar tidak lagi mengirim UUID kosong.

### Yang sudah cocok (tidak perlu diapa-apakan)

WorkOrderDetail (`allowed_transitions`, `self_cancellable`, `auto_confirm_at`),
`completion_rate` sebagai integer persen 0..100, PaymentRecord, Review
(termasuk `transaction_date`), meta galat `INVALID_STATUS_TRANSITION` dan
`CANCELLATION_AFTER_PRODUCTION`, criteria per kandidat, AvailabilityPeriod,
stale_calendar, dan kursor keyset semua sudah selaras field demi field.

**Akibat:** sampai temuan A sampai D direkonsiliasi, tipe hasil generate FE akan
menyimpang dari respons backend pada endpoint terkait. Prioritas perbaikan:
kategori A (salah tipe), lalu B (alur order buntu), lalu C dan D. Rekonsiliasi
mengikuti aturan spec-first: bila kontrak yang benar, kode diubah; bila kode
yang benar dan spec keliru, spec diamandemen lebih dulu.

