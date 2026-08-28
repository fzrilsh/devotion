# Changelog Backend

Semua perubahan penting pada backend dicatat di sini. Format mengikuti
Conventional Commits, entri ditulis pada branch kerja yang sama dengan
perubahannya.

## [Belum dirilis]

### Ditambahkan
- Runbook VPS dilengkapi menjadi 16 langkah utuh (T077): pasang compose,
  menyalakan dan memverifikasi migrasi, seed wilayah dan daftar baku serta
  `admin:create`, menyambungkan WhatsApp, health check dan pemantau uptime,
  cadangan basis data, dan snapshot. Langkah firewall diubah menjadi gerbang
  berupa perintah `curl` dari luar VPS beserta hasil yang diharapkan, dengan
  peringatan bahwa port yang dipublikasikan Docker melewati `ufw`. Ditambah
  urutan eksekusi menuju penjurian dan checklist dua belas butir dari
  `quickstart.md` bagian H. Skrip `backup.sh` ditulis lengkap di langkah 15
  beserta rotasi tiga salinan, entri crontab, dan `rsync` keluar VPS (artefak
  T080/T081, eksekusi di server dilakukan pemilik VPS).
- `docs/menjalankan.md` dilengkapi alur pengembang lokal (Postgres via compose,
  `serve` dari host di port 5434, seed, uji) dan ringkasan alur server yang
  menunjuk ke `docs/setup-vps.md` (bagian dari T079).
- CI memisahkan gerbang mutu dari rilis (T007). Job `backend` (`go vet`,
  `go test ./... -p 1`, gerbang sinkronisasi apidocs) tetap wajib hijau pada
  setiap push dan pull request. Job baru `detect` memeriksa keberadaan
  `frontend/package.json`; selama frontend belum mendarat, job `image` dan
  `deploy` dilewati, bukan gagal, sehingga pipeline dapat hijau sebagai gerbang
  backend. Begitu T003 menambah `frontend/package.json`, keduanya menyala
  sendiri tanpa menyentuh workflow. Ditambah blok `concurrency` dengan
  `cancel-in-progress` dan cache npm pada `setup-node`.
- Kerangka dokumentasi dilengkapi (T006): `docs/pengujian.md` memuat cara
  menjalankan uji, aturan skema terpisah, `Clock` yang digantikan, penamaan uji
  yang menyebut FR, dan minimum tiga kasus per endpoint; `docs/temuan-penguji.md`
  memuat tabel siap isi beserta catatan bahwa WhatsApp dan email di luar cakupan.
- Blok YAML `docker-compose.yml` di `quickstart.md` B10 diselaraskan dengan
  compose sebenarnya (T005): port Postgres `127.0.0.1:5434:5432` untuk alur
  pengembangan lokal bagian D, dan bind unggahan `${UPLOAD_PATH}:${UPLOAD_PATH}`
  agar path host dan container ditulis satu kali. Ditambah catatan bahwa port
  loopback melewati `ufw` sehingga ikatan ke `127.0.0.1` yang menahannya tertutup.

### Diperbaiki
- Gerbang uji basis data di CI tidak pernah benar-benar berjalan sejak pipeline
  dibuat. `go test ./...` dijalankan tanpa `-p 1`, jadi paket-paket berjalan
  paralel dan menghabiskan koneksi Postgres (max_connections=20, pool 15),
  gagal dengan SQLSTATE 53300. Kegagalan itu tersamar karena uji yang butuh
  basis data melewati diri sendiri (skip) saat koneksi tak tersedia, sehingga
  paket tetap mencetak `ok`: build hijau tanpa pernah menyentuh jalur nyata.
  Akibatnya beberapa bug lolos ke branch dalam keadaan "lulus", antara lain
  `CreateAdmin` yang tidak menormalkan nomor telepon dan menabrak constraint
  `phone_format`, serta beberapa test yang mematok jumlah (versi migrasi 15,
  jumlah kode galat 31, nomor telepon ber-'+') yang basi begitu data bertambah.
  Perbaikannya menambahkan `-p 1` pada langkah `go test` di `ci.yml` supaya
  paket berjalan berurutan dalam batas koneksi, tanpa menaikkan
  `max_connections` (angka itu keputusan untuk VPS 2GB). Ini dicatat sebagai
  keputusan, bukan perbaikan diam-diam: gerbang yang selama ini hijau tidak
  menjamin apa pun, dan angka yang lolos sebelumnya harus dianggap belum teruji.

- `CreateAdmin` kini menormalkan nomor telepon (membuang '+' di depan) lewat
  `normalizePhone` yang sama dengan jalur registrasi, sebelum menyimpan. Jalur
  registrasi sudah menormalkan sejak awal, tapi `CreateAdmin` meneruskan nomor
  mentah ke `UpsertAdmin`, jadi admin dengan nomor ber-'+' menabrak constraint
  `phone_format` (`^62[0-9]{8,13}$`, SQLSTATE 23514). Satu fungsi normalisasi
  dipakai kedua jalur, bukan dua salinan.
  Akibat nyatanya: sebelum perbaikan ini, `admin:create` dipastikan gagal untuk
  nomor yang ditulis dalam bentuk E.164 dengan '+' di depan, bentuk yang paling
  wajar diketik operator. Bug itu laten selama gerbang uji basis data tidak
  pernah benar-benar berjalan, jadi tidak pernah muncul sebagai kegagalan build.
  Nomor berawalan '0' tetap ditolak dan memang seharusnya begitu: `normalizePhone`
  hanya membuang '+', tidak mengubah awalan lokal menjadi 62, dan constraint
  `phone_format` adalah penjaga terakhirnya.
- `GET /api/work-orders` sebelumnya membalas 500 pada setiap permintaan, termasuk
  jalur berhasil tanpa filter, sejak endpoint daftar itu ditulis: endpoint utama
  FR-038 tidak pernah berfungsi. Penyebabnya `ListWorkOrdersForParty` mengirim
  parameter `status_filter` bertipe `work_order_status[]`. pgx tidak dapat
  meng-encode slice dari tipe enum bernama tanpa OID enum itu terdaftar di
  koneksi pool, dan `parseStatusFilter` mengembalikan slice kosong non-nil pada
  setiap permintaan tanpa filter, jadi setiap permintaan menabrak kegagalan
  encode `cannot find encode plan`, bukan hanya jalur `?status=`. Perbaikannya
  meng-cast parameter ke `text[]` di query (`wo.status::text = ANY(...::text[])`)
  dan `parseStatusFilter` kini mengembalikan `[]string`, sehingga tidak perlu
  registrasi tipe per koneksi (Prinsip IV). Ini satu-satunya parameter
  enum-array di basis kode; filter kandidat kuota memakai enum skalar, bukan
  array, jadi tidak terpengaruh. (FR-038)

- Konfirmasi otomatis tujuh hari kini menyertakan keberadaan sengketa terbuka
  ke dalam keputusan domain, bukan sebagai pemeriksaan kedua yang berdiri
  sendiri. `IsAutoConfirmDue` menerima parameter `hasOpenDispute` dan
  mengembalikan `false` selama ada sengketa yang belum `resolved`.
  `GetWorkOrderForView` dan `ListWorkOrdersForParty` kini membawa
  `has_open_dispute` lewat `EXISTS`, dan lapisan baca (detail serta daftar)
  meneruskannya ke fungsi domain yang sama dengan yang dipakai penjadwal.
  Sebelumnya lapisan baca menghitung ulang status "confirmed" hanya dari
  `shipped_at` dan waktu sekarang, sementara penjadwal sudah benar melewati
  pesanan bersengketa lewat `NOT EXISTS`. Akibatnya satu pesanan bersengketa
  yang lewat tujuh hari terbaca "confirmed" di halaman detail dan daftar tapi
  tetap "shipped" oleh penjadwal: dua pihak melihat dua kenyataan berbeda atas
  pesanan yang sama, dan FR-070 hanya setengah ditegakkan. Kini kedua lapisan
  memakai flag sengketa yang sama sehingga selalu sepakat "belum dikonfirmasi".
  (FR-068, FR-070)
- Pengiriman kode verifikasi kini berjalan di goroutine sehingga respons HTTP
  registrasi tidak menunggu SMTP atau WhatsApp (R-09). Kegagalan kirim email
  maupun WhatsApp tidak lagi membatalkan registrasi yang sudah tersimpan, dan
  setiap kegagalan dicatat ke `slog` beserta alasannya (kegagalan email senyap
  di level protokol, jadi baris log ini satu-satunya jejaknya). Cabang senyap
  `if s.delivery == nil { return }` dihapus: pengirim yang belum terpasang kini
  memunculkan peringatan di log, bukan hilang tanpa jejak. Pada
  `APP_ENV=development` saja, kode verifikasi plaintext ikut dicatat ke `slog`
  karena kode hanya disimpan sebagai hash dan pengembangan lokal tidak punya
  cara lain membacanya; ini tidak pernah aktif di produksi. (FR-001, FR-002, R-09)
- `POST /api/auth/register` sekarang benar-benar mengirim kode verifikasi email
  dan nomor. Sebelumnya `account.New` dipasang dengan `delivery` nil di
  `serve.go`, jadi kode dibuat dan di-hash tapi tidak pernah diserahkan ke
  transport mana pun, sehingga email lewat Mailjet tidak keluar meski kredensial
  sudah diatur. Ditambahkan adapter `notification.CodeDelivery` yang memakai
  transport email (SMTP Mailjet) dan WhatsApp yang sama dengan job notifikasi,
  tapi di luar antrean (kode sekali pakai tidak menulis baris notifikasi in-app).
  Pengiriman tetap best effort: transport nil atau gagal kirim tidak menggagalkan
  registrasi yang sudah tersimpan. Manajer WhatsApp dan sender email kini dibangun
  sebelum `account.New` agar bisa dibagi. (FR-001)

### Ditambahkan
- Subcommand `seed:test-data` dan `reset:test-data` (T075). `seed:test-data`
  menyiapkan fixture demo deterministik: 50 usaha pengisi untuk paginasi
  pencarian, empat listing sorotan (kapasitas 500 jeda 0 hari untuk skenario
  3.000 potong, jeda 14 dan 21 hari untuk minggu kesiapan, kapasitas 200 untuk
  tenggat lima bulan), satu listing kalender basi 8 hari, satu request kuota
  kedaluwarsa tanpa kandidat, tiga pesanan terikat tenggat (dikirim 6 hari,
  dikirim 8 hari, satu telat), dua pengajuan verifikasi menunggu, dan satu usaha
  dengan hanya dua pesanan selesai sehingga nilai penyelesaian ditahan.
  `reset:test-data` mengosongkan seluruh tabel fixture lewat satu TRUNCATE
  CASCADE, menyisakan baris wilayah dan katalog. Baris terikat tenggat ditulis
  sudah dalam keadaan sasaran (`shipped_at` dipatok Clock.Now() dikurangi 6 atau
  8 hari), bukan dengan menggeser waktu, sehingga jendela konfirmasi otomatis
  tujuh hari jatuh di sisi yang benar tanpa menunggu. Kedua perintah menolak
  berjalan saat APP_ENV=production dan seluruh akun uji memakai domain `.test`.
  Data uji tidak memuat data pribadi orang sungguhan: nomor telepon dan nomor
  identitas dibuat sintetis. Uji `_T075` memverifikasi penolakan produksi lewat
  `guardNotProduction`, kehadiran seluruh fixture, domain `.test`, dan reset yang
  bersih terhadap skema Postgres terisolasi memakai Clock yang disuntikkan.
  `seedTestData` memanggil `resetTestData` lebih dulu sehingga dijalankan dua
  kali tidak menduplikasi, sama seperti idempotensi seed lain lewat upsert; uji
  `TestSeedTestData_Idempotent_T075` membuktikannya. Pembelahan tenggat pesanan
  dikirim diperiksa lewat fungsi domain yang sama dengan penjadwal
  (`order.IsAutoConfirmDue` atas `order.AutoConfirmBase`), bukan query ad-hoc:
  `TestSeedTestData_ShippedOrderAutoConfirmWindow_FR068` menegaskan pesanan
  dikirim 6 hari belum jatuh tempo dan yang 8 hari sudah. Kandidat kapasitas 500
  jeda 0 hari terbukti melewati kriteria kapasitas untuk 3.000 potong deadline
  delapan minggu lewat `TestSeedTestData_CapacityCandidateMeetsLargeOrder_SC019`.
- Uji cakupan sisi admin (T072, FR-007, FR-050, FR-060, FR-067). Ditinjau lebih
  dulu bahwa penolakan peran nyata lewat router sudah ada untuk keseluruhan tiga
  belas endpoint admin (WhatsApp status, tiga rute sengketa, pesanan telat, tiga
  rute daftar baku, dua rute usulan, dua rute verifikasi, sembunyikan ulasan),
  sehingga tak ada yang diduplikasi. Uji struktur seperti
  `TidakMeninggalkanRuteTanpaKeputusanPeran` tidak dihitung sebagai penolakan
  peran karena hanya memeriksa registrasi rute, bukan mengirim permintaan
  berperan salah. Satu-satunya celah keputusan mediasi yang belum teruji secara
  terpisah, cabang `cancelled` dengan catatan kosong, ditutup dengan
  `TestMediation_ResolveCancelledRequiresNote_FR067` yang membuktikan penolakan
  422 saat pihak penanggung sudah disebut tetapi catatannya kosong.
- Mediasi sengketa sisi admin (FR-046, FR-067, FR-072). Tiga endpoint di-gate ke
  peran admin saja: `GET /api/admin/disputes` mendaftar antrean sengketa sebagai
  array Dispute telanjang (bukan amplop paginasi), terbaru dulu, dengan filter
  `status` opsional yang menolak nilai tak dikenal sebagai galat validasi;
  `POST /api/admin/disputes/{id}/mediate` memindahkan pesanan ke "Dalam Mediasi"
  dalam satu transaksi, menghentikan hitungan konfirmasi otomatis tujuh hari
  (FR-070) dengan membiarkan pesanan tetap di himpunan pindaian `shipped`; dan
  `POST /api/admin/disputes/{id}/resolve` menutup sengketa dengan keputusan admin
  yang eksplisit. Hasil (`result`) adalah kolom tersimpan, bukan turunan status
  akhir, karena pesanan `continued` dapat sendiri mencapai `confirmed` sehingga
  tak terbedakan dari penutupan `confirmed` oleh admin. Cabang `cancelled`
  mewajibkan admin menentukan pihak yang menanggung (`liable_profile_id` ditulis
  ke `work_order.cancelled_by_id` sehingga hanya pihak itu masuk pembagi tingkat
  penyelesaian, FR-072), apakah alokasi dibalik, dan sebuah catatan; pembalikan
  memakai kembali `reverseAllocationInTx` sehingga hanya ada satu jalur pembalikan.
  Cabang `continued` mengembalikan pesanan ke status sebelum mediasi yang dibaca
  dari riwayat status, dan hanya bila status itu `shipped` jam konfirmasi otomatis
  dimulai ulang dari waktu penutupan mediasi lewat `auto_confirm_base_at` (bukan
  menimpa `shipped_at`) dengan `confirm_warn_sent_at` direset ke NULL. Cabang
  `confirmed` mengonfirmasi pesanan atas nama pemberi order dengan
  `auto_confirmed` tetap false, agar keputusan admin terbedakan dari penutupan
  sistem tujuh hari. Setiap cabang mencatat baris riwayat dengan admin sebagai
  pelaku (`by_system=false`) dan memberi tahu kedua pihak. Waktu diambil dari
  `Clock` yang disuntikkan (Rule 5). Migrasi `018_auto_confirm_base` menambah
  `work_order.auto_confirm_base_at` (dengan constraint urutan waktu dan indeks
  `idx_order_auto_confirm` yang membaca `COALESCE(auto_confirm_base_at,
  shipped_at)`); migrasi `019_dispute_result` menambah enum `dispute_result`,
  kolom `dispute.result`, dan memperluas `resolution_complete`.
 `GET /api/admin/late-orders`
  mendaftar setiap pesanan aktif (accepted, production, completed, shipped) yang
  tenggat pengirimannya sudah lewat, terbaru dulu, satu halaman keyset sekali
  jalan, di-gate ke peran admin saja. Dua lapisan seperti konfirmasi otomatis:
  daftar dihitung saat dibaca dari `order.PastDeadlineCutoff(now)` sehingga selalu
  mutakhir tanpa menunggu ticker, dan sebuah job penjadwal (`order:late-order`,
  advisory lock tersendiri) memberi tahu kedua pihak bahwa tenggat telah lewat.
  Ambang "lewat tenggat" ditulis satu kali di `order.PastDeadlineCutoff`, dipakai
  kedua lapisan, jadi sebuah pesanan tak mungkin muncul di daftar tapi dilewati
  job, atau sebaliknya. Kueri menumpang indeks parsial `idx_order_deadline_active`
  yang sudah ada, bukan menambah indeks baru; himpunan statusnya sama persis
  dengan predikat indeks itu, jadi pesanan terkonfirmasi, dibatalkan, atau sedang
  dimediasi otomatis di luar cakupan. Notifikasi idempoten lewat kolom
  `late_notified_at` dengan penjaga `IS NULL`: dua instance yang tumpang tindih
  saat rollover deploy masing-masing memberi tahu paling banyak sekali, dan kolom
  ini tidak di-reset saat mediasi ditutup. Waktu diambil dari `Clock` yang
  disuntikkan (Rule 5), tak ada `time.Now()` di logika bisnis.

  menyembunyikan satu ulasan yang melanggar aturan, di-gate ke peran admin saja.
  Menyembunyikan adalah keseluruhan tindakan: baik daftar publik maupun rata-rata
  rating sudah menyaring `NOT hidden` lewat satu query `SearchReputation`, jadi
  ulasan lenyap dari keduanya sekaligus tanpa aturan kedua yang bisa menyimpang.
  Baris dikunci lebih dulu (`FOR UPDATE`) supaya dua admin yang memutuskan ulasan
  yang sama terurut, dan yang kedua membaca keadaan yang sudah ditetapkan yang
  pertama, bukan menimpa alasan dan waktunya. Alasan penyembunyian wajib diisi;
  handler menolak yang kosong atau terlalu pendek dengan galat validasi lebih
  dulu sebelum constraint `hiding_complete` bicara, jadi admin membaca pesan yang
  bisa dikutip, bukan galat basis data mentah. Identitas admin (`hidden_by`) dan
  waktunya (`hidden_at`) tercatat. Test kritis membuktikan tidak ada penyaringan
  kedua: menyembunyikan satu ulasan mengubah rata-rata di profil maupun di hasil
  pencarian dan menghapusnya dari daftar publik, keduanya konsisten karena
  berasal dari satu query.
- Kelola daftar baku sisi admin (FR-059, FR-060, FR-061, FR-074). `GET /api/admin/master/items`
  menyajikan seluruh item satu jenis, aktif maupun tidak, untuk permukaan katalog
  admin; `POST /api/admin/master/items` menambah item baru; `PATCH /api/admin/master/items/{itemId}`
  mengganti nama atau membalik flag aktif secara terpisah lewat pointer opsional,
  mengunci baris (`FOR UPDATE`) lebih dulu lalu menerapkan perubahan parsial.
  Nama duplikat per jenis ditangkap sebagai galat field 422 dari constraint
  `item_name_unique_per_type`, bukan 500. Kelima rute di-gate ke peran admin saja.
  Menonaktifkan item hanya membalik flag `active`; baris listing yang memakainya
  tidak disentuh, dan karena `search.sql` tidak menyaring `catalog_item.active`,
  listing yang sudah terbit tetap dapat ditemukan lewat pencarian (FR-060). Test
  kritis membuktikan sisi sebaliknya yang mudah lolos: setelah item dinonaktifkan
  lewat rute admin, listing yang memakainya tetap `published` dan tetap muncul di
  `SearchCandidates` dengan skor cocok, bukan sekadar tak bisa dipilih di formulir
  baru. Permukaan HTTP keputusan usulan item menyusul di sini: `GET /api/admin/proposals`
  menyajikan antrean usulan `pending` ber-keyset dengan nama usaha pengusul, dan
  `POST /api/admin/proposals/{proposalId}/decision` menyetujui atau menolak satu
  usulan. Penolakan wajib menyertakan alasan, ditolak di handler dengan galat
  validasi lebih dulu sebelum constraint bicara; persetujuan membuat item katalog
  di transaksi yang sama. Setiap keputusan memberi tahu pengusul lewat notifikasi
  `item_proposal_decision` yang ditulis di dalam transaksi keputusan (FR-074).
- Sisi admin verifikasi identitas (FR-007, FR-008). `GET /api/admin/verification`
  menyajikan antrean pengajuan ber-keyset (`created_at`, `id` turun), dengan
  filter `status` opsional, di-gate ke peran admin saja. `POST /api/admin/verification/{requestId}/decision`
  menyetujui atau menolak satu pengajuan yang masih `pending`: keputusan mencatat
  status, identitas admin (`decided_by`), dan waktunya (`decided_at`) dalam satu
  transaksi. Baris dikunci lebih dulu (`FOR UPDATE`) supaya dua admin yang
  memutuskan pengajuan yang sama terurut, dan yang kedua membaca status yang sudah
  ditetapkan yang pertama. Penolakan wajib menyertakan alasan yang dibaca pemohon;
  handler menolaknya dengan galat validasi lebih dulu supaya pemohon menerima
  pesan field, bukan 500 dari CHECK `rejection_needs_reason`. Persetujuan
  membalik `business_profile.verified` di transaksi yang sama, dan karena
  `search.sql` sudah memilih `verified`, lencana ikut muncul di profil dan hasil
  pencarian tanpa perubahan lain. Penolakan tidak menyentuh listing (FR-010,
  FR-011). Kontrak: label `x-fr` pada `GET /admin/verification` dikoreksi dari
  `[FR-008, FR-081]` menjadi `[FR-007]` (FR-081 soal pengecualian listing sendiri,
  tidak berkaitan dengan antrean verifikasi).

  FR-071, FR-072, FR-073). `POST /api/work-orders/{workOrderId}/reviews` mencatat
  ulasan satu pihak atas lawan transaksinya: rating 1..5 dan teks opsional sampai
  2000 karakter, hanya pada pesanan yang sudah dikonfirmasi diterima (manual atau
  auto-konfirmasi malas lewat predikat yang sama dengan `order`), satu ulasan per
  pesanan per pihak, tidak anonim. Prasyarat status pesanan tidak bisa jadi CHECK
  (ia membaca tabel lain), jadi ditegakkan di aplikasi. `GET /api/profile/{profileId}/reviews`
  menyajikan daftar ulasan publik ber-keyset. Nilai reputasi (tingkat penyelesaian,
  rata-rata rating, jumlah ulasan) dihitung saat baca, tidak pernah disimpan:
  `reputation.Derive` adalah satu-satunya tempat ambang FR-073 (persentase ditahan
  sampai pembagi >= 3) dan pembulatan persennya berada, sementara aturan pembagi
  FR-072 (pembatalan masuk pembagi hanya bagi pihak yang membatalkan) tetap di
  kueri SQL `SearchReputation` yang sama dipakai profil dan pencarian, sehingga
  keduanya tidak mungkin melaporkan angka berbeda untuk usaha yang sama. Stub
  `emptyReputation()` di `account/profile_http.go` diganti nilai nyata dari kueri
  itu. Test: rantai murni `Derive` di kedua batas ambang (termasuk 0 dari 3 yang
  bernilai 0%, bukan nil), penolakan ulasan pada pesanan belum dikonfirmasi,
  penolakan pihak yang tak pernah bertransaksi, penolakan ulasan kedua oleh pihak
  sama, pembatalan yang menurunkan tingkat penyelesaian hanya pihak yang
  membatalkan dan tidak pihak lain, ulasan tersembunyi keluar dari rata-rata, dan
  satu test yang membuktikan profil dan pencarian menampilkan angka sama untuk
  usaha yang sama.
- Test rantai maju penuh mesin keadaan pesanan (FR-039, FR-044). Sebuah test
  menempuh tiap langkah legal yang dapat dijalankan subkontraktor lewat HTTP
  (accepted -> production -> completed -> shipped), memastikan tiap langkah
  diterima dan respons mencerminkan status baru, lalu membaca
  `work_order_status_history` langsung untuk membuktikan tiap perpindahan manusia
  tercatat atas nama subkontraktor yang memindahkan: `changed_by` akun itu dan
  `by_system` bernilai false. Ini pasangan dari test konfirmasi otomatis yang
  membuktikan penutupan sistem menulis baris `by_system` tanpa aktor manusia,
  jadi perpindahan maju milik manusia dan penutupan milik sistem, tak ada yang
  salah label. Melengkapi cakupan T058 yang sisanya sudah ada di test transisi,
  pembatalan, dan konfirmasi otomatis sebelumnya.
- Pencatatan pernyataan pembayaran pada pesanan (T056, FR-040, FR-041, FR-042,
  FR-043). `POST /api/work-orders/{workOrderId}/payments` memungkinkan salah satu
  pihak mencatat pernyataannya sendiri, arah `sent` atau `received`, dengan
  tanggal dan catatan bebas opsional. Tidak ada kolom jumlah uang: platform hanya
  mencatat pernyataan, tidak menahan, menyalurkan, maupun memverifikasi dana.
  Setiap pihak mencatat tiap arah paling banyak sekali; constraint
  `one_statement_per_party_per_direction` menegakkannya dan pelanggarannya
  diterjemahkan menjadi `PAYMENT_STATEMENT_EXISTS` 409 yang bisa dibaca pengguna,
  bukan galat 500. Pernyataan disimpan atas profil bisnis pihak, sehingga
  pernyataan kedua pihak tetap dapat dibedakan dan perbedaan di antara keduanya
  terlihat oleh admin saat sengketa ditengahi. Setiap jalur yang mengembalikan
  `WorkOrderDetail` kini mengisi array `payments`. Pihak lain diberi tahu saat
  sebuah pernyataan dicatat lewat event `payment_record`. Rute memeriksa peran
  (buyer atau subkontraktor) lalu menjaga bahwa pemanggil memang pihak pesanan
  itu; non-pihak menerima 404, bukan kebocoran keberadaan pesanan. (T056, FR-040,
  FR-041, FR-042, FR-043)
- Pelaporan sengketa pada pesanan (T057, FR-046, FR-070).
  `POST /api/work-orders/{workOrderId}/disputes` memungkinkan salah satu pihak
  melaporkan sengketa dengan uraian 10 sampai 2000 karakter yang dibaca admin
  saat menengahi. Hanya satu sengketa boleh terbuka per pesanan: laporan kedua
  selagi yang pertama belum diselesaikan melanggar indeks parsial
  `idx_one_open_dispute` dan diterjemahkan menjadi `DISPUTE_ALREADY_OPEN` 409 yang
  bisa dibaca pengguna. Melaporkan sengketa tidak memindahkan status pesanan:
  pesanan tetap pada statusnya sekarang (`shipped` misalnya), dan justru baris
  sengketa terbuka itu yang menghentikan hitung mundur konfirmasi otomatis tujuh
  hari, karena pemindaian auto-confirm mengecualikan pesanan mana pun yang punya
  sengketa belum selesai lewat penjaga `NOT EXISTS` pada tabel `dispute`. Admin
  yang memindahkan pesanan ke `in_mediation` saat menengahi (T071).
  `confirm_warn_sent_at` sengaja tidak direset. Pihak lain diberi tahu bahwa
  sengketa dilaporkan. Rute
  memeriksa peran (buyer atau subkontraktor) lalu menjaga keanggotaan pihak;
  non-pihak menerima 404. (T057, FR-046, FR-070)
  FR-069, FR-070). Pesanan berstatus `shipped` yang melewati jendela tujuh hari
  sejak `shipped_at` dianggap diterima secara otomatis: statusnya menjadi
  `confirmed`, kolom `auto_confirmed` disetel true untuk menandai bahwa penutupan
  itu dilakukan sistem, bukan salah satu pihak, dan baris riwayat status ditulis
  dengan `by_system = true` tanpa pelaku manusia. Kedua pihak diberi tahu bahwa
  penutupan terjadi otomatis. Dua hari sebelum tenggat, pemberi order (buyer)
  diperingatkan sekali lewat notifikasi yang menyebut tanggal konfirmasi otomatis;
  penanda `confirm_warn_sent_at` mencegah peringatan terkirim ulang tiap tick.
  Pesanan yang punya sengketa terbuka tidak pernah ditutup otomatis karena
  pemindaian mengecualikannya lewat penjaga `NOT EXISTS` pada tabel `dispute`,
  jadi baris sengketa yang menghentikan hitung mundur, bukan status pesanan. Tenggat dihitung satu kali di fungsi domain `AutoConfirmAt`/
  `IsAutoConfirmDue` yang dipakai kedua lapisan (R-07): lapisan baca lazy di
  `buildDetailView` dan `listItemView` membuat pesanan yang lewat tenggat langsung
  terbaca `confirmed` di halaman daftar maupun detail tanpa menunggu ticker, dan
  ticker dalam proses (`order:auto-confirm`, dibungkus advisory lock
  `LockKeyAutoConfirm`) yang menulis penutupan final serta mengirim peringatan
  FR-069. Migrasi 000016 menambah kolom `confirm_warn_sent_at`. (FR-068, FR-069,
  FR-070, R-07)

  FR-020, FR-065, FR-066, FR-072). Kedua pihak (pembeli atau subkontraktor) boleh
  membatalkan selama status masih `accepted`; rute digerbang kedua peran usaha dan
  handler menegakkan penjaga pihak, jadi bukan-pihak jadi 404 tanpa membocorkan
  keberadaan pesanan. Alasan wajib 5 sampai 500 karakter, di luar rentang itu
  ditolak `VALIDATION_FAILED` (422). Dalam satu transaksi: status dikunci lalu
  diperiksa, `cancelled_by_id` diisi id profil pihak pembatal (dasar perhitungan
  tingkat penyelesaian FR-072), alasan dan waktu dicatat, status menjadi
  `cancelled`, baris riwayat status ditulis dengan pelaku manusia (`by_system` =
  false), dan seluruh alokasi dibalik lewat `reverseAllocationInTx` sehingga
  perubahan status dan pengembalian kapasitas commit bersama (FR-020). Setelah
  pesanan lewat `accepted` (masuk produksi atau lebih), pembatalan sendiri tidak
  tersedia: `CANCELLATION_AFTER_PRODUCTION` (409) dengan meta `alternative_path`
  menuju jalur sengketa (FR-066). Pihak lawan diberi tahu beserta alasannya
  lewat kejadian `order_status_changed` (FR-051 mendaftar "perubahan status
  pesanan", pembatalan adalah transisi accepted ke cancelled, jadi tidak ada
  nilai enum baru), FR-065. Respons mengembalikan `WorkOrderDetail`
  yang sudah dimuat ulang.
- Mesin keadaan pesanan dan endpoint baca work-order (T053, FR-038, FR-039,
  FR-044). `GET /api/work-orders/{workOrderId}` mengembalikan `WorkOrderDetail`
  lengkap dengan `allowed_transitions` dan `self_cancellable` supaya frontend
  merender tombol dari array itu, bukan menyalin mesin keadaan. Penjaga pihak
  membandingkan akun pemanggil dengan pembeli dan subkontraktor pesanan, jadi
  bukan-pihak, id tak sah, dan pesanan tidak ada sama-sama jadi 404 tanpa
  membocorkan keberadaan pesanan. `POST /api/work-orders/{workOrderId}/status`
  digerbang subkontraktor (FR-005) untuk transisi maju (production, completed,
  shipped); lompatan di luar urutan ditolak `INVALID_STATUS_TRANSITION` yang
  menyebut urutan yang diizinkan (Diterima, Produksi, Selesai, Dikirim) beserta
  meta `current_status` dan `allowed_transitions`. Perubahan status mencatat
  pelaku (`changed_by` = id akun, `by_system` = false) dan memberi tahu pembeli.
  `GET /api/work-orders` melistkan pesanan per pihak dengan paginasi keyset
  (kursor opaque), filter `status[]` dan `role=as_buyer|as_subcontractor`.
- Pembalikan alokasi kapasitas satu pesanan dalam satu transaksi (`ReverseAllocation`
  di `internal/order`, FR-020). Setiap periode dikembalikan ke `used_capacity`
  sebelum pesanan terbentuk dengan mengurangi kuantitas baris alokasinya, dan
  baris alokasi ditandai `reversed_at` tanpa dihapus sehingga jejak audit tetap
  ada. Pola penguncian meniru pembentukan (R-04): kunci `work_order` lebih dulu,
  lalu baris alokasi aktif beserta periodenya menaik menurut `week_start` (pencegah
  deadlock). Baris yang sudah dibalik dilewati lewat penjaga `reversed_at IS NULL`,
  jadi pemanggilan ulang tidak mengembalikan kapasitas dua kali. Endpoint HTTP
  pembatalan (kedua pihak, alasan, CANCELLATION_AFTER_PRODUCTION) menyusul di T054.
  Query: `LockWorkOrderForReversal`, `ListActiveAllocationsForReversal`,
  `LowerUsedCapacity`, `MarkAllocationReversed`. (FR-020)
- Uji alokasi kalender dan pembalikan di `internal/order`: pengisian minggu paling
  awal lebih dulu dengan plafon per minggu (1.200 di 500/minggu jatuh 500/500/200,
  FR-018/FR-078), jeda kesiapan 14 hari melewati dua minggu pertama (SC-020/FR-087),
  periode ditandai penuh dilewati, pembalikan mengembalikan seluruh periode dan
  menandai baris (FR-020), propagasi FR-089 hanya menyentuh periode belum
  teralokasi, dan trigger `trg_reject_allocation_before_readiness` menolak alokasi
  sebelum minggu kesiapan (FR-087). (FR-018, FR-020, FR-078, FR-087, FR-089, SC-020)
- Handler HTTP dan pendaftaran rute untuk empat endpoint sisi pemohon verifikasi,
  paket `internal/verification`: `POST /api/files` (unggah berkas identitas atau
  foto lokasi), `GET /api/files/{fileId}` (unduh berkas milik sendiri),
  `POST /api/verification` (ajukan verifikasi), dan `GET /api/verification` (baca
  riwayat pengajuan sendiri, larik JSON polos). `POST /api/files`,
  `POST /api/verification`, dan `GET /api/verification` digerbangi dua peran
  usaha (`RoleSubcontractor`, `RoleBuyer`); admin tidak punya `business_profile`
  sehingga ditolak 403 di router sebelum mencapai handler. `GET /api/files/{fileId}`
  digerbangi dua peran usaha ditambah admin: SC-012 menuntut berkas terbaca oleh
  pemilik dan admin, dan antrean verifikasi Fase 7 bergantung pada akses admin
  itu. `POST /api/files`
  membatasi bodi permintaan pada batas per-berkas plus slack multipart sebelum
  `ParseMultipartForm`, lalu menyerahkan validasi magic byte, pembuangan EXIF, dan
  kuota ke paket `storage`; sentinel storage dipetakan ke 413 `FILE_TOO_LARGE`,
  415 `UNSUPPORTED_FILE_TYPE`, dan 507 `STORAGE_QUOTA_FULL`. `GET /api/files/{fileId}`
  meneruskan `storage.Caller{ProfileID, IsAdmin}` ke `storage.Open`, jadi berkas
  hanya terbaca oleh pemiliknya atau admin (bukan pemilik dan bukan admin -> 403),
  path bukan UUID -> 422, id sah tapi tak dikenal -> 404 tanpa membocorkan
  keberadaan. `POST /api/verification` menolak pengajuan kedua saat satu masih
  menunggu dengan 409 `VERIFICATION_PENDING` lewat index parsial
  `idx_one_pending_verification`, dan mengambil `created_at` dari `Clock` yang
  disuntikkan. Ditambahkan `Service.MaxFileBytes()` pada paket `storage` sebagai
  akses batas per-berkas untuk handler. Rangkaian uji tingkat router menutup
  minimum per endpoint (jalur berhasil, penolakan peran, masukan tidak sah) plus
  penolakan bukan-pemilik-bukan-admin pada unduh berkas, pembacaan admin atas
  dokumen orang lain (sisi positif SC-012), penolakan tipe menipu lewat magic
  byte (415), dan cakupan rute `UncoveredAPIRoutes` kosong. (FR-006, FR-009,
  FR-010, FR-011, SC-012)

  request kuota, melengkapi uji T043 yang sebelumnya hanya mencakup jalur berhasil
  dan penolakan peran. `TestQuotaRequest_ListRejectsInvalidQuery_FR030` menutup
  `GET /api/quota-requests` dengan `size=0`, `size=51`, dan `cursor=busuk` (422
  `VALIDATION_FAILED`); `TestDetail_RejectsMalformedRequestID_FR032` menutup
  `requestId` bukan UUID pada detail (422, dibedakan dari 404 milik id sah tapi
  bukan milik pemanggil); `TestIncoming_RejectsInvalidQuery_FR031` menutup
  `status=ngawur` dan `size=0` pada incoming. Ditambah pula
  `TestQuotaRequest_ListRejectsNonBuyer_FR030` yang membuktikan gerbang peran
  `GET /api/quota-requests` benar-benar menolak subkontraktor dengan 403; uji
  cakupan rute yang ada hanya memastikan keputusan peran terpasang, bukan menguji
  penolakannya secara fungsional. (FR-030, FR-031, FR-032)
- T043 (test request kuota) ditutup: ketujuh skenario yang didaftarkan task sudah
  tercakup uji tingkat router yang ada di `internal/quota/`, jadi tidak ditulis uji
  duplikat. Pemetaan skenario ke uji, supaya keterlacakannya tidak hilang:
  jalur berhasil `TestQuotaRequest_HappyPath_SendsToSeveralCandidates_FR029_FR030`
  (`quota_test.go`); penolakan peran `TestQuotaRequest_RejectsNonBuyer_FR029`
  (`quota_test.go`); masukan tidak sah `TestQuotaRequest_RejectsInvalidInput_FR029`
  (`quota_test.go`); request ke listing sendiri
  `TestQuotaRequest_RejectsSelfListing_FR083` (`quota_test.go`); kapasitas kurang
  beserta angka `remaining_capacity`/`quantity_requested`/`until_week`
  `TestOffer_RejectsQuantityBeyondCapacity_FR035` (`offer_test.go`); kesiapan
  melampaui deadline `TestOffer_RejectsReadinessAfterDeadline_FR090`
  (`offer_test.go`); request kedaluwarsa lewat `Clock` yang digeser 73 jam
  `TestOffer_RejectsAfterReplyWindow_FR082` (`offer_test.go`). (T043)
- Swagger UI di `GET /docs`, hanya saat `APP_ENV=development`, agar jalur
  frontend membaca kontrak tanpa membuka YAML mentah. Aset Swagger UI ditarik
  dari CDN jsdelivr dipatok `swagger-ui-dist@5.17.14` dengan Subresource
  Integrity (sha384) dan `crossorigin="anonymous"`, jadi CDN yang disusupi tidak
  bisa menyuntik kode. Kontrak disajikan di `GET /docs/openapi.yaml` dari salinan
  `apidocs/openapi.yaml` yang disematkan `embed.FS`, disegel byte-identik dengan
  `docs/001-capacity-exchange-marketplace/contracts/openapi.yaml` lewat uji
  (bukan hash, pembandingan isi dengan pesan gagal yang menyebut lokasi
  menyimpang) dan gerbang drift di CI (`apidocs-sync.sh` lalu `git diff
  --exit-code`). Rute didaftarkan hanya di pengembangan, jadi di produksi absen
  dan jatuh ke 404 yang sudah ada; tidak ada layanan runtime baru (Gate I tetap
  dua), tidak ada dependency Go baru. (T082)
- Modul Go `github.com/fzrilsh/devotion/backend` dengan toolchain dipatok
  `go 1.25.0`.
- Dispatcher subcommand di `cmd/devotion` dengan delapan perintah terdaftar:
  `serve`, `admin:create`, `seed:regions`, `seed:master-data`,
  `seed:test-data`, `reset:test-data`, `user:verify`, `health:check`. Semua
  masih stub kecuali dispatcher; diisi di branch berikutnya. (T002)
- `docker-compose.yml` dengan tepat dua layanan runtime (`postgres`,
  `backend`), penyetelan Postgres untuk 2GB dari research.md R-03, batas log
  `max-size 10m`/`max-file 3` di keduanya, volume `pgdata`, bind mount
  `${UPLOAD_PATH}`, dan `TZ: Asia/Jakarta`. Gate I: hitung entri di bawah
  `services:`, harus dua. (T005)
- Kerangka sembilan berkas `docs/*.md` (`menjalankan`, `pengujian`,
  `dependencies`, `utang-teknis`, `layanan-luar`, `temuan-penguji`,
  `cloudflare-ips`, `setup-vps`, `skenario-uji-manual`) plus
  `frontend/CHANGELOG.md`. Diisi sekarang: `layanan-luar.md`, `setup-vps.md`
  (ekstrak quickstart.md A-B), `skenario-uji-manual.md` (penunjuk ke §F),
  `utang-teknis.md` (tiga item Complexity Tracking). `cloudflare-ips.md`
  menyusul di T013. (T006)
- `backend/Dockerfile` multi-stage (build `golang:1.24.1-alpine`, runtime
  `alpine:3.20` non-root) dan `.github/workflows/ci.yml`. Urutan pipeline:
  `go vet` -> `go test` (Postgres sebagai layanan CI, bukan runtime, jadi Gate I
  tetap dua) -> build frontend -> salin `frontend/dist/.` ke `backend/webdist/`
  sebelum docker build -> push GHCR tag `<sha>` dan `latest` -> deploy SSH di
  `main`. (T007)
- `internal/platform/clock.go`: `Clock` interface, `SystemClock` (waktu
  ter-lokalisasi Asia/Jakarta), `TestClock` dengan `Set`/`Advance` ber-mutex,
  dan `WeekStart` (Senin awal minggu WIB) yang dipakai kedua lapisan penjadwal.
  Uji menyisir tree: `time.Now()` dilarang di luar `platform` dan `cmd`. (T008)
- `internal/platform/config`: `Load(getenv)` memvalidasi konfigurasi tanpa
  mengubah state proses. Wajib di semua environment: `APP_ENV`, `APP_BASE_URL`,
  `DATABASE_URL`, `UPLOAD_PATH`; wajib hanya di produksi: TLS, CF client CA,
  Mailjet, `MAIL_FROM`, `WHATSAPP_NUMBER`, `SENTRY_DSN`. Default
  `UPLOAD_MAX_TOTAL_MB=500`, `UPLOAD_MAX_FILE_MB=5`. `APP_ENV` tak dikenal
  adalah galat. Semua variabel hilang dikumpulkan dalam satu galat yang hanya
  memuat nama, tidak pernah nilai. `IsProduction()` untuk penjaga
  `seed:test-data`/`reset:test-data`. (T009)
- 15 migrasi SQL (`000001_extensions` sampai `000015_verification_code`, 30 berkas
  up/down) yang memetakan data-model.md §12 satu banding satu, ditambah runner
  `internal/platform/migrate`. Runner memakai `iofs` atas migrasi yang di-embed
  (`db/embed.go`), jalan di bawah `pg_try_advisory_lock` dengan kunci konstanta
  pada satu koneksi yang di-pin, dan mengembalikan nil tanpa galat bila lock
  dipegang proses lain (skip saat rollover deploy). Tanpa `DEFAULT now()` di
  mana pun; kolom waktu diisi aplikasi lewat `Clock`. Down migration kebalikan
  tepat dalam urutan mundur (trigger sebelum fungsi, fungsi sebelum tabel).
  Uji: versi 15 `dirty=false`, idempoten dua kali, down-up kembali ke versi 15,
  tiga fungsi trigger terpasang lewat `pg_trigger`, empat constraint kunci lewat
  `pg_constraint`, sapuan larangan `DEFAULT` waktu, dan kelengkapan 15 pasang
  migrasi. Uji integrasi memakai skema terpisah pada Postgres yang sama dan
  `t.Skip` bila `DATABASE_URL_TEST` tak terjangkau. (T010)
- Lapisan akses data: `sqlc.yaml` (engine postgresql, `schema: db/migrations`
  supaya tipe tak menyimpang dari database yang dimigrasi, `queries: db/queries`,
  `sql_package: pgx/v5`, `emit_json_tags: false`), `db/queries/health.sql`, dan
  hasil `sqlc generate` di `internal/db/sqlcgen` (di-commit). `internal/db/pool.go`
  menyetel pool ke angka R-03: `MaxConns 15`, `MinConns 2`, lifetime 30m, idle 5m,
  health check 1m, dengan `Ping` saat buka. `tx.go` `WithTx` rollback pada galat
  maupun panic lalu re-panic. Harness `internal/db/testdb`: `New(t, name)` membuat
  skema `test_<name>`, memigrasinya, mengembalikan pool
  `search_path=test_<name>,public`, TRUNCATE (bukan DROP) saat cleanup.
  Kunci advisory migrasi dipindah ke bentuk dua-int
  `pg_try_advisory_lock(class, hashtext(current_schema()))` sehingga skema uji
  yang berbeda tak saling memblokir; ekstensi `citext`/`pgcrypto` dipasang
  `WITH SCHEMA public` dan down 000001 dikosongkan karena ekstensi milik seluruh
  database, bukan per skema. (T011)
- `internal/platform/cloudflare`: rentang IP Cloudflare resmi dipatok sebagai
  konstanta, di-parse sekali di `init` menjadi `[]*net.IPNet` dan panic pada
  entri rusak supaya typo gagal saat startup. `RealIP` memisah `RemoteAddr`,
  mengembalikan kosong bila tak terurai, host mentah bila koneksi di luar rentang
  Cloudflare (tanpa menyentuh header), dan baru mempercayai `CF-Connecting-IP`
  bila koneksi dari rentang tersebut. Konstanta `RetrievedAt` dan `docs/cloudflare-ips.md`
  dijaga sinkron oleh uji. Daftar diverifikasi ke sumber resmi, bukan dari
  research.md R-01. (T013)
- `internal/platform/httpx`: lapisan HTTP dasar. `codes.go` mentranskripsi 29
  kode galat dari openapi.yaml sebagai `type Code string` dan memegang peta
  `Code -> {Status, Title}` sehingga status HTTP diturunkan dari kode dan tidak
  bisa berbeda antar handler. `problem.go` menulis `application/problem+json`
  (RFC 9457) lewat `WriteProblem`, `WriteValidation` (membawa `errors[]` bentuk
  `ProblemValidation`), dan `WriteInternal` (500 generik). `logger.go`:
  `contextHandler` yang menarik `request_id` ke tiap record slog sehingga tak ada
  call site yang bisa lupa. `middleware.go`: rantai dari luar `RequestID` ->
  `Recover` (panic jadi 500 problem+json, stack ke slog bukan ke klien) ->
  `Logger` (JSON: method, path, status, duration_ms) -> `RealIP`. `router.go`:
  `http.ServeMux` pola method+path Go 1.22 dengan catch-all `/api/` yang
  mengembalikan 404 `Problem`, bukan HTML. (T012)
- `internal/platform/ratelimit`: empat jendela R-10 di `map[Target]window`
  sebagai satu-satunya sumber angka (login 5/15m per akun, OTP 3/jam per nomor,
  OTP 10 nomor berbeda/jam per alamat asal, request kuota 20/jam per pengguna).
  State di tabel `rate_limit`, bukan memori, sehingga redeploy bukan jalan
  pintas. `Check` menaikkan penghitung dan membandingkan dalam satu transaksi;
  `INSERT ... ON CONFLICT DO UPDATE` mengunci baris sehingga dua pemanggil
  berbarengan tak bisa sama-sama membaca hitungan sama lalu lolos. `CheckAddress`
  menghitung **nomor berbeda**, bukan percobaan: kirim ulang ke nomor yang sudah
  dihitung tak memakan kuota alamat, hanya nomor baru; sebuah
  `pg_advisory_xact_lock` per alamat men-serialisasi hitung-lalu-catat. Semua
  timestamp dari `Clock` sehingga uji kedaluwarsa jendela menggeser waktu, bukan
  tidur; 429 membawa `Retry-After` sampai jendela bergulir. Kueri di
  `db/queries/ratelimit.sql`, hasil `sqlc generate` di-commit. (T016)
- `internal/platform/session` dan `internal/account`: sesi dan seluruh
  permukaan auth. Token 32 byte `crypto/rand` di-encode base64url pada cookie
  `devotion_session` (`httpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, TTL 7
  hari dengan perpanjangan bergulir); yang tersimpan `token_hash` SHA-256, bukan
  token mentah, dan logout menghapus baris. Sepuluh endpoint sesuai
  `openapi.yaml`: `register`, `verify-email`, `verify-phone`, `resend-code`,
  `login`, `logout`, `recover/request`, `recover/confirm`, `GET /me`,
  `PATCH /me/roles`. `GET /me` mengembalikan bentuk `MyAccount`. bcrypt cost 10;
  login menjalankan rate limit T016 **sebelum** perbandingan bcrypt. Kode enam
  digit R-09 untuk email dan telepon, tersimpan sebagai hash SHA-256, sekali
  pakai lewat `consumed_at`, kedaluwarsa 15 menit dari `Clock`. `POST
  /auth/recover/request` selalu 202 dengan waktu respons dikonstankan lewat
  `platform.ConstantTimeFloor` (wall-clock, bukan `Clock`, karena kebocorannya
  sinyal waktu nyata) agar tak membocorkan keberadaan akun; `recover/confirm`
  mengakhiri semua sesi lain. `golang.org/x/crypto` naik dari indirect menjadi
  dependency langsung, dicatat di `docs/dependencies.md`. (T014)
- Middleware peran di `internal/platform/httpx/auth.go`: peran sebagai bitmask
  `Role` (`RoleSubcontractor`, `RoleBuyer`, `RoleAdmin`; bit admin terpisah,
  tak pernah tersirat oleh peran usaha, satu akun boleh memegang dua peran
  usaha). `RequireAuth` mengizinkan tiap pemanggil terautentikasi dan menaruh
  `Principal` di konteks; `RequireRole` mengizinkan yang memegang salah satu
  peran yang diminta. Kegagalan auth 401, peran salah 403, galat resolusi 500,
  sehingga hiccup basis data tak disalahartikan sebagai sesi absen. httpx tak
  mengimpor `account`: interface `Authenticator` disuntikkan, dan `account`
  mengimplementasinya (`Authenticate` memvalidasi cookie, memuat akun segar,
  melipat tiga flag boolean jadi bitmask). Router mencatat tiap pola sebagai
  publik eksplisit atau ter-gate; `UncoveredAPIRoutes` melaporkan pola `/api/*`
  non-publik yang tak ter-gate, dan uji menuntutnya kosong, sehingga endpoint
  tak bisa terbit tanpa keputusan peran. `logout` dipindah dari Public ke Gated
  (kontrak menuntut 401), `GET /me` dan `PATCH /me/roles` lewat `RequireAuth`
  plus adapter `fromPrincipal`. Uji: matriks penolakan tiap kombinasi peran tak
  berwenang, auth mendahului cek peran, dan rute akun nol tak tercakup. (T015)
- Penyimpanan berkas unggahan di `internal/platform/storage`, satu-satunya
  tempat byte klien menyentuh disk. `Save` menjalankan urutan yang mengikat:
  `io.LimitReader` ke batas per-berkas lebih dulu agar decode bomb tak menguras
  memori, lalu `http.DetectContentType` atas magic bytes (nama dan
  `Content-Type` dari klien tak pernah dipercaya), lalu decode dan re-encode
  gambar lewat `image/jpeg`/`image/png` untuk membuang EXIF (foto lokasi dari
  ponsel membawa koordinat GPS), PDF divalidasi magic bytes lalu disimpan apa
  adanya, lalu cek total 500MB, baru tulis dengan nama acak `crypto/rand`
  berekstensi dari tipe terverifikasi. `Open` menyelesaikan berkas lewat id dan
  menegakkan pemilik-atau-admin (FR-009); tak ada path statis sama sekali. Query
  `db/queries/uploaded_file.sql` (Create/Get/SumBytes), hasil `sqlc generate`
  di-commit. Uji menunjuk FR-006/FR-009: orang asing ditolak `ErrForbidden`,
  berkas berekstensi menipu ditolak `ErrUnsupportedType`, kuota penuh
  `ErrQuotaFull`, kelebihan ukuran `ErrTooLarge`, dan penanda EXIF hilang setelah
  re-encode. (T017)
- Aritmetika tenggat di `internal/order/deadline.go`, satu-satunya tempat waktu
  dihitung untuk kedua lapisan penjadwal (research.md R-07), sehingga sebuah
  pesanan tak pernah tampak beda status di halaman berbeda. Setiap fungsi
  menerima instan sekarang sebagai parameter, tak ada `time.Now()`:
  `ReadinessDeadline`, `AutoConfirmAt`, `IsAutoConfirmDue`,
  `IsAutoConfirmApproaching` (FR-068/FR-069), `IsCalendarStale` (FR-021),
  `IsRequestExpired` (FR-037/FR-082). Batas inklusif diuji per nanodetik.
- Penjadwal lapisan 2 di `internal/platform/scheduler`: satu `time.Ticker` lima
  menit dalam goroutine yang dinyalakan `serve`, bukan proses/cron/container
  kedua, jadi Gate I tetap dua layanan. Tiap job dibungkus
  `pg_try_advisory_lock(class, key)` pada koneksi pool yang di-pin, dilepas lewat
  `defer` pada koneksi sama dengan `context.Background()`, sehingga saat rollover
  deploy container lama melewatkan job alih-alih mengantre. `LockKey` konstanta
  literal dalam satu blok; pendaftaran job kosong (diisi T023). Uji menunjuk
  R-07: dua penjadwal pada database sama menaikkan counter tepat sekali, lock
  terlepas diperiksa lewat `pg_locks`. (T018)
- Modul `internal/masterdata` plus subcommand `seed:regions` dan
  `seed:master-data`, dan empat endpoint baca publik (`security: []`):
  `GET /api/master/products`, `/api/master/machines`, `/api/regions/provinces`,
  `/api/regions/cities` (filter opsional `?province=`). `NormalizeCityCode`
  membuang titik pada kode kota wilayah.id (`32.73` jadi `3273`) sebelum
  disimpan, karena `city_code_format` dan `city_belongs_to_province` menolak
  bentuk lain. `seed:regions` default membaca salinan `docs/master-data/regions.json`;
  `--refresh` mengambil dari wilayah.id, menormalkan, menulis ulang salinan,
  lalu mengisi database. Seeder idempoten pada kode/nama: sisip bila absen,
  perbarui nama bila ada, tak pernah menghapus karena `business_profile`
  merujuknya. Handler memetakan kolom DB (`id`/`type`) ke nama kontrak
  (`item_id`/`kind`). Uji: `NormalizeCityCode` langsung, seed dua kali idempoten,
  nol baris kota dengan `left(code,2) <> province_code`, dan keempat endpoint
  baca. (T019)
- Subcommand `admin:create` di `cmd/devotion`: membuat admin pertama atau
  mereset kata sandinya bila email sudah ada (idempoten lewat
  `INSERT ... ON CONFLICT (email) DO UPDATE`, query `UpsertAdmin`). Kata sandi
  dibaca dari prompt tanpa echo (`golang.org/x/term`, dikonfirmasi dua kali),
  tidak pernah lewat flag karena flag masuk riwayat shell; `--email` dan
  `--phone` dari flag. `account.CreateAdmin` memakai satu jalur bcrypt yang sama
  (`hashPassword`, cost 10) sehingga hashing kata sandi cuma satu tempat; baris
  admin punya `role_admin` true dan kedua peran usaha false, diterima
  `has_at_least_one_role` dan `admin_has_no_business_role`. `golang.org/x/term`
  dinaikkan dari indirect ke dependency langsung, dicatat di
  `docs/dependencies.md`. Uji: dua kali jalan tidak menduplikasi admin, panggilan
  kedua mengganti kata sandi. (T020)
- Penyajian SPA tersemat dan TLS produksi, plus `serve` yang sesungguhnya.
  `embed.go` (`package web`) menyematkan `webdist/` lewat `//go:embed
  all:webdist`; awalan `all:` wajib atau chunk Vite berawalan `_` tersaring
  diam-diam. `webdist/index.html` placeholder di-commit supaya direktif embed
  ter-compile; CI menimpanya dengan hasil build. `httpx.NewStatic` menegakkan
  urutan R-06: `/api/*` ke handler API (termasuk 404 `Problem` untuk path API
  tak dikenal, bukan `index.html`), berkas nyata di `webdist` dengan
  `Cache-Control: public, max-age=31536000, immutable` untuk aset ber-hash,
  sisanya jatuh ke `index.html` dengan `Cache-Control: no-cache`. `tlsconf.Load`
  membangun `tls.Config` dengan `ClientAuth: RequireAndVerifyClientCert`,
  `ClientCAs` dari CA klien Cloudflare, dan `MinVersion: TLS12`, sehingga
  Authenticated Origin Pulls menolak koneksi yang melewati edge (R-01). `serve`
  memuat config, menyambung pool, menjalankan migrasi, membangun router, mendaftar
  `account` dan `masterdata`, menyalakan penjadwal sebagai goroutine, lalu listen
  dengan `ReadHeaderTimeout: 10s` dan shutdown anggun pada SIGINT/SIGTERM: TLS di
  `:443` saat produksi, HTTP polos di `:8080` di pengembangan. Cookie `Secure`
  mati hanya bila `APP_BASE_URL` memakai `http://`, dan pengecualian itu nyaring
  di log. (T022)
- Modul notifikasi `internal/notification`: antrean transaksional plus
  pengiriman kanal. `Enqueue` menerima `pgx.Tx`, bukan pool, sehingga baris
  notifikasi selalu ditulis di dalam transaksi kejadiannya (FR-086);
  notifikasi dalam platform selalu tersimpan meski semua kanal dimatikan
  preferensi, karena feed adalah satu-satunya jalur observasi penguji manual
  (FR-054). `IsTransactional` adalah fungsi atas enum `event_type`, bukan
  kolom yang bisa disalah-set pemanggil: hanya `calendar_stale`,
  `deadline_approaching`, dan `rating_request` yang non-transaksional, sisanya
  transaksional dan tidak bisa dibungkam preferensi (FR-091). Event
  transaksional mengantre email dan WhatsApp tanpa syarat; event
  non-transaksional hanya mengantre kanal yang masih diaktifkan akun. Kanal
  dikirim oleh job penjadwal `Deliver` (didaftarkan lewat `DeliverJob`),
  maksimal tiga percobaan lalu `failed_permanent`, hitungannya di
  `notification_channel` (FR-085); kegagalan kirim tidak pernah menyentuh baris
  notifikasi dalam platform. Email lewat `net/smtp` ke Mailjet tanpa SDK. Empat
  endpoint terjaga peran: daftar feed dengan kursor keyset opaque dan
  `unread_count`, tandai dibaca (idempoten, 404 bila bukan milik pemanggil atau
  id tak sah), GET dan PUT preferensi non-transaksional. Uji menunjuk FR-051,
  FR-054, FR-055, FR-085, FR-086, FR-091. (T023)
- Tautan WhatsApp di `internal/admin`: klien whatsmeow berjalan sebagai
  goroutine di dalam proses `serve` (research.md R-08), bukan layanan kedua,
  jadi Gate I tetap dua. Sesinya disimpan di Postgres yang sama lewat handle
  `database/sql` kedua (driver `pgx` stdlib), dibatasi `SetMaxOpenConns(2)` dan
  dianggarkan dari lima koneksi yang disisakan di luar pool 15. `Manager`
  membuka store, meng-upgrade skemanya, dan mengurus siklus hidup klien:
  `Start` menyalurkan kode QR ke status saat perangkat belum terpasang lalu
  `Connect`, `onEvent` membedakan `Connected` dari `LoggedOut`. `SendText`
  memenuhi `notification.WhatsAppSender` sehingga kanal WhatsApp kini terkirim.
  `GET /api/admin/whatsapp` khusus admin mengembalikan `WhatsAppStatus`
  (`connected`, `qr_code`, `last_error`); nomor layanan tidak pernah muncul di
  respons, log, maupun Sentry (FR-082), ditegakkan secara struktural karena
  tipe status tak punya field untuk membawanya. Subcommand `user:verify
  --email`/`--phone` memverifikasi akun tanpa antarmuka supaya nomor yang
  terblokir sesaat tak menghalangi pembuatan akun. Dependency `go.mau.fi/whatsmeow`
  dicatat di `docs/dependencies.md`. Uji menunjuk FR-082: gate admin (401/403/200),
  null saat kosong, dan QR/galat sampai ke body. (T024a)
- `internal/platform/health` menyajikan `GET /health` publik (`security:[]`)
  yang memeriksa tiga ketergantungan: ping basis data, tautan WhatsApp lewat
  `Manager.Connected()`, dan ruang sisa volume unggahan lewat `Statfs`. Balasan
  503 bila salah satu gagal, dengan status per ketergantungan (`ok`/`down`) di
  body; enum status itu tak punya ruang untuk nomor layanan (FR-082).
  `internal/platform/observability` menyalakan Sentry dengan `BeforeSend`
  berbentuk allowlist: event keluar dibangun ulang dari field aman saja, jadi
  request, cookie, user, Extra, dan Contexts dibuang alih-alih disaring, dan
  field sensitif baru aman secara default. Subcommand `health:check` menyelidik
  `GET /health` lewat HTTP untuk healthcheck kontainer tanpa `curl` di image.
  Dependency `github.com/getsentry/sentry-go` dicatat di `docs/dependencies.md`.
  Uji menunjuk FR-082: 503 saat tiap ketergantungan mati, dan scrub membuang
  kata sandi, token, nomor telepon, serta rujukan dokumen identitas. (T025)
- Kontrak `openapi.yaml` diselaraskan dengan data-model.md dan migrasi untuk
  jalur User Story 1. `RegisterRequest` kini membawa `city_code` dan `roles`
  wajib (profil usaha lahir dalam transaksi register, jadi `GET /profile/me`
  tak pernah 404). `ListingRequest`/`Listing` memakai `weekly_capacity`,
  `readiness_lead_days`, `product_item_ids`, dan `machines` dengan
  `machine_count`, menyamai `listing_product`/`listing_machine`, bukan lagi item
  tunggal. `AvailabilityPeriod`/`PeriodUpdateItem` mendapat `marked_full`.
  `ProfileUpdateRequest` membuang `address`/`province_code` yang tak berkolom;
  `MyProfile`/`PublicProfile` menurunkan `city_name`/`province_*` dari `city`.
  Kode galat `LISTING_ALREADY_EXISTS` (409) ditambahkan ke enum `Problem.code`
  dan `httpx` (`codes.go` kini 30 kode). Kunci respons validasi `'400'` pada
  path US1 diseragamkan ke `'422'` (status nyata `httpx.StatusFor`); sisa `'400'`
  path story lain dicatat sebagai utang di `docs/utang-teknis.md`. (T026-kontrak)
- Profil usaha (`internal/account/profile.go`, `profile_http.go`,
  `db/queries/profile.sql`): `GET /api/profile/me` dan `PUT /api/profile/me` di
  balik autentikasi, `GET /api/profile/{profileId}` publik. Profil kini lahir
  bersama akun dalam satu transaksi `POST /api/auth/register` (`CreateAccount` +
  `CreateProfile` di `db.WithTx`), sehingga `RegisterRequest` mewajibkan
  `city_code`/`business_name` dan `GET /profile/me` tak pernah 404. Kota tak
  dikenal menjadi 422 `FieldError` pada `city_code`, bukan pelanggaran foreign
  key 500. `PUT` memvalidasi nama minimal 3 karakter, koordinat lengkap atau
  kosong sebagai pasangan, dan rentang dalam wilayah Indonesia (menyalin
  `coordinates_within_indonesia`). `MyProfile`/`PublicProfile` menurunkan
  `city_name`/`province_code`/`province_name` dari join `city`+`province`;
  `verification_status` null dan reputasi kosong di US1. Id profil yang cacat
  atau tak dikenal pada path publik dijawab 404 tanpa membedakan keduanya.
  (T026)
- Fondasi listing kapasitas (T027, sedang berjalan): kueri SQL listing dan
  kalender di `db/queries/listing.sql` (`CreateListing`, `GetListingByProfile`,
  `GetListingByID`, `LockListingByProfile` `FOR UPDATE`, `UpdateListing`,
  `SetListingPublished`, `TouchCalendarUpdatedAt`, `RaiseHorizonUntil` dengan
  `GREATEST` agar horizon tak pernah mundur, `InsertPeriodsUpToWeek` idempoten
  lewat `ON CONFLICT`, `ListPeriodsInRange`, `LockPeriodByWeek`, `UpsertPeriod`,
  `PropagateCapacityToFuturePeriods`, `FindFutureAllocatedPeriodOverCapacity`,
  `PeriodHasActiveAllocation`, `CountActiveCatalogItemsOfType`, dan tautan
  `listing_product`/`listing_machine`), hasil `sqlc generate` di
  `internal/db/sqlcgen`, `internal/platform/dateid.go` (`FormatDateID` menghasilkan
  "24 Agustus 2026" ter-lokalisasi Asia/Jakarta untuk periode dan notifikasi),
  serta kerangka paket `internal/listing/listing.go` (`Service{pool, clock}`,
  `New`, `queries()`, `InitialHorizonWeeks = 14` yang memenuhi sekaligus minimal
  13 periode FR-088 dan minimal 3 bulan FR-017, `MaxPeriodBatch = 26`).
- Listing kapasitas subkontraktor (T027): enam rute di `internal/listing/http.go`,
  semuanya di belakang `httpx.RequireRole(auth, RoleSubcontractor)` sehingga tak
  ada yang lolos gerbang `UncoveredAPIRoutes()` dan peran salah ditolak 403.
  `GET /api/listing/me`, `POST /api/listing/me` (FR-010: listing langsung tayang
  tanpa gerbang verifikasi, `published` true sejak insert), `PUT /api/listing/me`,
  `PUT /api/listing/me/visibility` (nonaktif sementara lalu aktifkan kembali,
  FR-015), `GET /api/listing/me/periods`, dan `PUT /api/listing/me/periods`.
  `EnsureHorizon` menjamin setiap periode mingguan sampai minggu target ada lalu
  menaikkan `horizon_until`; idempoten dan aman dipanggil bersamaan tanpa
  advisory lock karena duplikasi dicegah `one_period_per_week`, kemunduran
  horizon dicegah `GREATEST`, dan deadlock dicegah urutan lock tetap (baris
  listing sebelum `availability_period`). Batas minggu dihitung satu tempat lewat
  `platform.WeekStart`, jadi `week_start_is_monday` dan `horizon_is_monday` tak
  bisa dilanggar. Propagasi kapasitas FR-089 di `PUT /api/listing/me`:
  `FindFutureAllocatedPeriodOverCapacity` menolak seluruh permintaan dengan 409
  `CAPACITY_ALREADY_ALLOCATED` bila ada periode mendatang yang pemakaiannya sudah
  melebihi kapasitas baru, `PropagateCapacityToFuturePeriods` menulis kapasitas
  baru hanya ke minggu `>= minggu berjalan` yang belum teralokasi, dan periode
  teralokasi dibiarkan utuh. Pra-pemeriksaan `CountActiveCatalogItemsOfType`
  memvalidasi tipe item sebelum insert, sehingga id mesin yang dikirim sebagai
  produk dijawab 422 menyebut `product_item_ids`, bukan 500 dari
  `trg_reject_wrong_product_item`. `PUT /api/listing/me/periods` memvalidasi
  seluruh batch (1..26 elemen, tiap `week_start` hari Senin dalam rentang minggu
  berjalan sampai 26 minggu ke depan, kapasitas non-negatif, tanpa minggu ganda)
  sebelum menulis apa pun, lalu dalam satu transaksi mengunci listing,
  memperpanjang horizon, dan mengunci tiap periode urut menaik; kapasitas di
  bawah pemakaian jadi 409 `CAPACITY_ALREADY_ALLOCATED` dan tanda penuh saat ada
  alokasi aktif jadi 409 `PERIOD_ALREADY_ALLOCATED`. `TouchCalendarUpdatedAt`
  adalah satu-satunya jalur yang memajukan `calendar_updated_at` (FR-021).
  `platform.ParseDate` mengurai `week_start` sebagai tengah malam Asia/Jakarta
  agar tanggal Senin tetap Senin, bukan bergeser sehari akibat lokalisasi UTC.
  Terpasang di `cmd/devotion/serve.go` lewat `listing.New(pool, clock).Register(router, acc)`
  sebelum gerbang rute. Uji pendamping di `internal/listing/listing_test.go`
  (jalur berhasil, penolakan peran, dan masukan tak sah tiap rute; propagasi
  FR-089; idempotensi dan konsistensi horizon; `FormatDateID` di
  `internal/platform/dateid_test.go`).
- Kalender awal dan horizon (T028): kemampuan FR-017 dan FR-088 sudah terkirim
  utuh di dalam T027 lewat `EnsureHorizon` (`internal/listing/calendar.go`) yang
  membuat periode mingguan minimal 13 minggu ke depan saat listing dibuat,
  memakai kapasitas mingguan sebagai kapasitas total, menyimpan periode terjauh
  di `horizon_until` konsisten dengan `MAX(week_start)`, dan memperpanjang
  horizon secara idempoten serta aman dipanggil bersamaan tanpa membuat baris
  ganda. T035 memanggil fungsi perpanjangan ini sebagai API internal, bukan kode
  yang hanya dipakai saat pembuatan listing. Ditandai selesai di `tasks.md`;
  tanpa perubahan kode baru karena cakupannya sudah dites di
  `internal/listing/listing_test.go` (`TestCreateListing_HorizonAwal*`,
  `TestEnsureHorizon_*`).
- Usulan item daftar baku (T029, FR-061): `POST /api/master/proposals` di
  `internal/masterdata/http.go` menerima usulan item baru dari pemanggil
  bisnis, digerbang ke `RoleSubcontractor` dan `RoleBuyer` (keduanya memilih
  item dari daftar baku yang sama, FR-022). Body divalidasi (`kind` product
  atau machine, `proposed_name` 2..80 karakter, 422 per-field), usulan tersimpan
  berstatus `pending` dengan `created_at` dari `Clock` (Rule 5). Metode domain
  `DecideProposal` (`internal/masterdata/proposal.go`) menerapkan keputusan admin
  dan mengantre notifikasi `item_proposal_decision` ke pengusul di dalam satu
  transaksi (FR-086), memenuhi syarat FR-061 bahwa pengusul diberi tahu saat
  usulannya diputus; permukaan HTTP admin `/admin/proposals` menyusul di T068.
  Kueri `InsertItemProposal`, `GetItemProposalByID`, `DecideItemProposal` di
  `db/queries/masterdata.sql` dan hasil `sqlc generate`. Diuji di
  `internal/masterdata/proposal_test.go` (`TestCreateProposal_Success_FR061`,
  `TestCreateProposal_RejectsRole_FR061`, `TestCreateProposal_RejectsInvalidInput_FR061`,
  `TestDecideProposal_NotifiesProposer_FR061`).
  Dua hal sengaja ditunda ke T068 (permukaan HTTP admin, FR-058): (1)
  `DecideProposal` memakai UPDATE ber-guard `WHERE status = 'pending'`, jadi
  keputusan atas proposal yang sudah diputus membuat `DecideItemProposal`
  mengembalikan `pgx.ErrNoRows` yang naik jadi galat transaksi; T068 harus
  memetakannya ke 409, bukan 500 mentah. (2) `DecideProposal` menerima `itemID`
  dari pemanggil untuk constraint `approved_yields_item`; belum ada pembuat item
  katalog dari proposal yang disetujui, jadi T068 perlu menyambungkan pembuatan
  `catalog_item` sesungguhnya saat approve.
- Test backend US1 (T030): melengkapi cakupan uji US1 di modul `account` dan
  `listing` tanpa menulis ulang yang sudah ada. Audit menemukan `listing` sudah
  memenuhi trio (jalur berhasil, penolakan peran, penolakan masukan) untuk
  seluruh rute-nya plus kasus horizon awal dan perpanjangan idempoten, jadi
  celah terpusat di `account`. Ditambah `internal/account/us1_test.go`:
  `TestPatchRoles_MenambahPeran_Berhasil_FR001`,
  `TestPatchRoles_TanpaSesi_Unauthorized_FR001`,
  `TestPatchRoles_MencabutSemuaPeran_Ditolak_FR001` (trio PATCH /me/roles),
  `TestVerifyPhone_JalurBerhasil_FR002`, `TestVerifyPhone_KodeSalah_Ditolak_FR002`,
  `TestResendCode_ChannelTidakSah_Ditolak_FR002`, `TestResendCode_SelaluDiterima_FR002`
  (gerbang verifikasi dua kanal FR-002), dan
  `TestPublicProfile_IdTidakDikenal_NotFound_FR016`. `handlePatchRoles`
  (`internal/account/handlers.go`) kini menolak permintaan yang mencabut kedua
  peran dengan 422 (FR-001), menghindari agar constraint `has_at_least_one_role`
  muncul sebagai 500. Komentar pada
  `TestCreateListing_TanpaPengajuanVerifikasi_TetapTayang_FR010` diperjelas: spec
  kita sengaja menyimpang dari status "Menunggu Verifikasi" dokumen sumber, dan
  test itu mengunci keputusan tersebut (FR-010).
- Mesin pencarian (T035): modul baca `internal/search` dengan rute tunggal
  `GET /api/search` bergerbang `RoleBuyer`. Kueri `SearchCandidates` di
  `db/queries/search.sql` mengikuti `data-model.md` §10: rentang kapasitas per
  kandidat dari minggu kesiapan (Senin dari tanggal pencarian + `readiness_lead_days`)
  sampai minggu deadline, minggu di luar `horizon_until` dihitung berkapasitas
  penuh (FR-088), empat kriteria keras dijumlahkan menjadi skor 0-4 tanpa
  pembobotan maupun normalisasi (FR-023, FR-024), filter yang tidak diisi
  dihitung terpenuhi dan dilaporkan tidak dievaluasi (FR-026), pemecah seri lima
  tingkat berakhir di `listing_id` (FR-025), keyset pagination opaque (FR-080),
  dan pengecualian listing milik pencari (FR-081). Perluasan wilayah kota →
  provinsi → nasional lewat parameter `region_level`, saran pelonggaran saat
  hasil kosong di tingkat nasional (FR-028), dan perpanjangan horizon kandidat
  lolos di transaksi tersendiri di luar kueri baca (FR-088). Skor tidak
  terpengaruh reputasi, verifikasi, kebaruan kalender, maupun jarak (FR-024).
- Uji determinisme dan rentang kapasitas mesin pencarian (T036) di
  `internal/search`: urutan identik pada pengulangan dan stabil antar halaman
  meski ada listing baru disisipkan di tengah penelusuran (SC-013, FR-025); skor
  tak berubah saat rating, verifikasi, dan kebaruan kalender diubah (FR-024);
  3.000 potong pada 500/minggu lolos di deadline 8 minggu dan gagal di 4 minggu
  (SC-019); jeda kesiapan 14 hari membuang dua minggu pertama sehingga totalnya
  di bawah kandidat jeda nol (SC-020); minggu kesiapan yang melampaui deadline
  menghasilkan kapasitas nol dan kriteria (d) tak terpenuhi (SC-020); kapasitas
  di luar horizon awal tetap dihitung penuh sampai deadline lalu periodenya
  benar-benar dimaterialisasi (SC-021, FR-088); filter mesin kosong membuat
  kriterianya terpenuhi dan dilaporkan tidak dievaluasi (FR-023, FR-026).
- Modul request kuota (T039): paket tulis `internal/quota` dengan dua rute
  `POST /api/quota-requests` dan `GET /api/quota-requests`, keduanya digerbang
  peran pembeli. Satu aksi mengirim satu request ke beberapa listing kandidat
  sekaligus, tiap kandidat membawa statusnya sendiri (FR-029), dan pembeli
  melihat daftar request-nya sendiri dengan keyset pagination kursor opaque
  terbaru dulu (FR-030, FR-080). Jendela balasan 72 jam (`reply_due_at`) dan
  `created_at` keduanya diambil dari `Clock` yang disuntikkan, bukan
  `time.Now()` (FR-082, Aturan 5). Listing tak dikenal atau belum tayang jadi
  422; listing milik pembeli sendiri ditolak 409 `SELF_REQUEST` sebelum ada
  insert apa pun, trigger basis data hanya jaring pengaman (FR-083). Tiap
  kandidat memicu notifikasi `request_received` di dalam transaksi yang sama
  sehingga kegagalan antrean membatalkan seluruh request.
- Penawaran dan negosiasi kuota (T040): melengkapi `internal/quota` dengan lima
  rute. `POST /api/candidates/{candidateId}/offers` (digerbang subkontraktor)
  membalas kandidat dengan harga rupiah bulat `int64` dan kesiapan dalam hari,
  memvalidasi pemilik listing, menolak kesiapan yang melewati tenggat
  (`READINESS_AFTER_DEADLINE`, 422, FR-090) dan jumlah melebihi sisa kapasitas
  lintas minggu kesiapan..tenggat (`INSUFFICIENT_CAPACITY`, 409 dengan meta
  `quantity_requested`, `remaining_capacity`, `until_week`, FR-035).
  `POST /api/candidates/{candidateId}/reject` menolak kandidat dengan alasan,
  tanpa notifikasi (FR-031). `POST /api/offers/{offerId}/counter` (digerbang
  kedua peran) merantai penawaran balik sebagai baris baru, bukan pembaruan,
  bergiliran antar pihak dan menyimpan seluruh riwayat (FR-033).
  `GET /api/quota-requests/{requestId}` (digerbang pembeli) menampilkan detail
  request dengan tiap kandidat membawa penawaran terakhirnya berdampingan,
  memakai penjaga akun pembeli sehingga request milik pembeli lain jadi 404
  bukan 403 (FR-030, FR-032). `GET /api/quota-requests/incoming` (digerbang
  subkontraktor) menampilkan satu halaman keyset kandidat masuk dengan filter
  status opsional (FR-030, FR-031). Semua waktu diambil dari `Clock` yang
  disuntikkan (Aturan 5).
- Pembentukan kesepakatan dan alokasi kapasitas (T041): paket `internal/order`
  dengan rute `POST /api/offers/{offerId}/accept`, digerbang peran pembeli, yang
  mengubah satu penawaran diterima menjadi pesanan lewat transaksi R-04. Dalam
  satu transaksi: kunci listing, tumbuhkan kalender sampai minggu tenggat
  (FR-088), kunci tiap periode kandidat urut menaik `week_start` (pencegah
  deadlock R-04), jumlahkan sisa kapasitas lintas periode tak-penuh dan tolak
  kekurangan dengan angka sebenarnya (FR-035), sisipkan work order yang menyimpan
  minggu kesiapannya (FR-084), isi minggu paling awal dulu melewati periode penuh
  atau habis (FR-018/FR-078) dengan satu baris alokasi per periode terpakai
  (FR-077), tandai kandidat pemenang `agreed`, tutup dan beri tahu kandidat lain
  (FR-034), lalu catat transisi status pembuka. Penjaga akun pembeli membuat
  penawaran milik pembeli lain jadi 404, bukan bocor keberadaannya. Balasan
  `WorkOrderDetail` membawa `allowed_transitions` dan `self_cancellable` supaya
  frontend merender tombol dari array itu, tidak menduplikasi mesin keadaan
  (FR-039). Semua waktu diambil dari `Clock` yang disuntikkan (Aturan 5).
- Uji perilaku dan balapan pembentukan kesepakatan (T042) di `internal/order`:
  dua kesepakatan berbarengan atas periode yang sama dari dua request berbeda,
  hanya satu menang dan yang kalah menerima `CAPACITY_ALREADY_TAKEN` dengan
  `used_capacity` berakhir di satu pesanan bukan dua (FR-036); dua kesepakatan
  berbarengan atas request yang sama lewat dua listing berkecukupan, yang kalah
  menerima `REQUEST_ALREADY_AGREED` dari indeks unik parsial
  `idx_one_agreement_per_request` dengan tepat satu kandidat `agreed` (FR-034);
  `RaiseUsedCapacity` melampaui `total_capacity` ditolak constraint
  `used_capacity_within_total` SQLSTATE 23514, jaring pengaman tingkat
  penyimpanan yang jalur accept tak pernah menyentuh karena penjumlahan di bawah
  lock menolak lebih dulu (FR-079/SC-018); tenggat di luar horizon tersimpan
  berhasil karena `EnsureHorizon` memmaterialisasi minggu yang kurang di dalam
  transaksi, membuktikan estimasi optimistik pra-lock tak memalsukan positif
  (FR-088). Balapan digerakkan `runConcurrent` dengan barrier agar keduanya benar
  berebut lock. Ketiga uji minimum per-endpoint menembus router: penolakan peran
  bukan-pembeli 403 (FR-005), sesi absen 401 (FR-005), id penawaran tak sah 422
  (respons kontrak, tanpa FR), plus satu jalur berhasil yang menembus gerbang,
  `parseUUID`, transaksi, dan pemetaan 201 lalu memeriksa serialisasi
  `WorkOrderDetail` (`work_order_id`, `status` accepted, `allowed_transitions`
  tak kosong, `self_cancellable` ada). Uji memakai skema terpisah pada Postgres
  yang sama dan `Clock` yang digantikan.
- Detail request kini melampirkan seluruh rantai penawaran tiap kandidat, bukan
  hanya penawaran terakhir: `RequestCandidate` bertambah larik `offers` terurut
  `sequence` menaik dan `Offer` bertambah field wajib `sequence`, sehingga
  pembeli melihat tiap putaran negosiasi dan tawar-balik berurutan (FR-032,
  langkah manual 3.7). `latest_offer` tetap diisi dari elemen terakhir rantai.
  Rantai dibangun dari `ListOffersByRequest` yang sudah dibaca, satu round-trip,
  tanpa kueri atau endpoint baru.
- `WorkOrderDetail` pada jalur pembentukan kesepakatan kini membawa larik
  `payments`, kosong saat pesanan baru terbentuk karena belum ada pernyataan
  pembayaran. Field mengikuti skema `PaymentRecord` tanpa kolom jumlah uang
  (Batas Keuangan). Pengisian sungguhannya milik US5 (T056, FR-041..FR-043);
  jalur accept hanya menyiapkan larik kosong agar bentuk respons sesuai kontrak.

### Dihapus
- Tiga kueri sqlc tanpa pemanggil dibuang beserta kode Go hasil generate-nya:
  `ListOffersByCandidate` (digantikan `ListOffersByRequest` yang mengambil rantai
  seluruh kandidat sekali jalan), `MaxPeriodWeek`, dan `CountPeriods`. `go build
  ./...` tetap bersih setelah `sqlc generate`.

### Diperbaiki
- `GET /api/profile/me` dan `PUT /api/profile/me` menolak akun admin dengan 403
  (`FORBIDDEN`, "Akun admin tidak memiliki profil usaha.") alih-alih 500. Akun
  admin tidak punya baris `business_profile` karena `admin:create` tidak menulis
  profil dan peran admin bukan peran usaha, jadi kueri profil dulu memberi
  `ErrNoRows` yang tidak dipetakan handler dan jatuh ke 500. Akar masalahnya
  kedua rute profil hanya digerbang autentikasi, bukan peran; menambah
  pemeriksaan `RoleAdmin` per-handler hanya akan membuat rute profil ketiga nanti
  lupa lagi, persis cara bug ini lahir. Kedua rute kini berada di balik gerbang
  peran usaha di router (`RequireRole` subkontraktor atau pemberi order), jadi
  admin ditolak dengan 403 sebelum handler jalan, sebentuk dengan setiap endpoint
  usaha lain, dan pemeriksaan `RoleAdmin` di handler GET dicabut karena jadi
  mubazir. Tidak ada pemanggil sah kedua rute ini yang tanpa peran usaha:
  registrasi mewajibkan minimal satu peran usaha dan `admin_has_no_business_role`
  melarang admin memegangnya. 403 dipilih, bukan 404, agar jaminan kontrak bahwa
  endpoint ini tak pernah 404 untuk akun hasil registrasi tetap utuh. Untuk akun
  usaha, profil lahir bersama akun, jadi baris yang hilang di sana tetap
  pelanggaran invarian (500 plus catatan `slog` berlevel error dengan
  `account_id`, karena bila terjadi itu tanda data rusak, bukan salah pemanggil),
  bukan 404. Uji lewat router membuktikan admin memperoleh 403 berkode
  `FORBIDDEN` pada GET dan PUT, dan akun usaha tetap 200 dengan profilnya.
  Kontrak menambahkan respons 403 pada `PUT /profile/me`, sebentuk dengan GET.
  Audit endpoint lain yang membaca `profile_id` dari sesi menunjukkan tidak ada
  yang ikut cacat: `GET /api/me` mengembalikan `profile_id: null`, sedangkan
  jalur kuota, usulan item, dan listing berada di balik gerbang peran usaha yang
  menolak admin dengan 403 sebelum kueri profil dijalankan. (FR-005)
- `GET /api/health` memisahkan liveness dari readiness: hanya basis data gagal
  (`database: fail`) atau penyimpanan penuh (`storage.status: full`) yang
  menggerakkan 503, sedangkan WhatsApp terputus kini menghasilkan 200 dengan
  `status: degraded` dan tetap terlihat di `dependencies.whatsapp`. Alasannya
  restart loop: `docker-compose.yml` memakai `restart: unless-stopped` bersama
  healthcheck yang memanggil `devotion health:check`, jadi bila WhatsApp
  terputus mengembalikan 503, container ditandai tidak sehat dan di-restart,
  padahal pemulihan sesi whatsmeow menuntut pemindaian QR manual lewat halaman
  admin. Restart tidak menyambungkan apa pun dan hanya menjatuhkan seluruh situs
  yang basis data dan web-nya sehat. `health:check` diselaraskan agar menilai
  kode status HTTP saja (200 hidup, 503 mati), bukan lagi mengurai body dan
  mensyaratkan `status` `"ok"`, supaya body `degraded` tidak ikut memicu restart
  loop; rute sudah terdaftar sehingga 200 pasti dari handler health, bukan shell
  SPA. `checkStorage` mencatat `slog.Error` yang menamai path dan galat saat
  direktori unggahan tak terbaca, karena body hanya membawa enum tetap dan tidak
  boleh membocorkan path. Pemantau uptime wajib dikonfigurasi mencocokkan
  `"whatsapp":"connected"` pada isi respons agar terputusnya tetap ter-alert
  tanpa restart. Uji lewat router membuktikan keempat keadaan (WhatsApp terputus
  200 degraded, DB gagal 503, storage penuh 503, semua sehat 200 ok), dan uji
  `health:check` mengunci bahwa body `degraded` pada 200 tetap berhasil.
  (R-08, T025)
- Perutean statis (T022, T025): `Static.ServeHTTP` kini mengonsultasi mux untuk
  setiap path sebelum jatuh ke berkas statis lalu fallback `index.html`,
  memakai `ServeMux.Handler` untuk mendeteksi rute terdaftar. Sebelumnya hanya
  path berawalan `/api/` yang diarahkan ke mux, sehingga rute yang terdaftar di
  luar `/api/` ditelan fallback SPA dan mengembalikan 200 HTML. Rute health
  dipindah dari `GET /health` ke `GET /api/health` agar selaras dengan prefiks
  `servers` `/api` di `openapi.yaml` dan referensi quickstart (B4, B14, checklist),
  dan `health:check` kini mengurai body serta mensyaratkan `status` `"ok"` supaya
  200 dengan body HTML (shell SPA) tidak lagi dilaporkan sehat. Uji regresi di
  `httpx` membuktikan rute non-`/api/` benar-benar terjangkau, path tak terdaftar
  tetap jatuh ke shell SPA, dan `/api` tak dikenal tetap 404 problem+json.
  ditambahi `p.readiness_week, p.deadline_week`, kolom `param` yang muncul di
  `SELECT` lewat `uncreated_remaining` tapi tak teragregasi. Tanpa keduanya
  Postgres menolak kueri saat runtime dengan SQLSTATE 42803 (grouping error).
  sqlc menghasilkan kode Go yang tetap ter-compile dan `go vet` lolos, jadi
  cacatnya tak tertangkap build maupun uji: sepanjang belum ada uji yang benar
  menjalankan jalur accept, penjaga pra-lock (`INSUFFICIENT_CAPACITY`) tak pernah
  sekali pun tereksekusi sejak T041 menulisnya. Uji T042 yang menembus kueri ini
  itulah yang memunculkannya. `internal/db/sqlcgen/order.sql.go` diregenerasi.
- Keyset pagination mesin pencarian (T036): klausa `WHERE` kursor yang memakai
  satu perbandingan row-value `<` untuk kelima kolom urut diganti rantai OR
  leksikografis eksplisit. Perbandingan row-value tunggal keliru karena urutan
  mencampur arah (skor dan sisa kapasitas menurun, sedangkan jeda, nama, dan
  `listing_id` menaik), sehingga kandidat berskor sama bisa muncul dua kali
  antar halaman. Tiap tingkat kini dibandingkan pada arahnya sendiri setelah
  tingkat di atasnya seri (SC-013, FR-025).
- CI: `GO_VERSION` diselaraskan dengan directive `go` di `backend/go.mod`
  (1.25.0), sehingga runner tak lagi mengunduh toolchain terpisah tiap run.
  `actions/setup-go` diberi `cache-dependency-path: backend/go.sum` supaya cache
  modul menemukan berkas checksum yang ada di subdirektori `backend/`, bukan di
  root repository.
- `TestCodes_EveryCodeMapsToOneStatus`: daftar kode uji diselaraskan menjadi 31
  kode setelah `READINESS_AFTER_DEADLINE` masuk peta kode dan enum `openapi.yaml`.
  Test masih menegakkan jumlah kode uji sama dengan jumlah entri peta status,
  jadi kode baru tanpa status akan tetap ketahuan.
- `internal/platform/config`: nama variabel kuota unggahan diselaraskan dengan
  `.env.example` dan quickstart menjadi `UPLOAD_MAX_TOTAL_MB` dan
  `UPLOAD_MAX_FILE_MB` (sebelumnya `UPLOAD_TOTAL_LIMIT_MB`/`UPLOAD_FILE_LIMIT_MB`).
  Nama lama tak pernah terbaca, sehingga `parseLimit` jatuh ke default 500/5
  secara senyap dan kuota tak bisa dikonfigurasi sama sekali.
- `docker-compose.yml`: bind mount TLS memakai path cermin
  `/opt/devotion/tls:/opt/devotion/tls:ro` (sebelumnya `:/tls:ro`), sejalan
  dengan keputusan path-di-dalam-container-sama-dengan-host agar `.env` tak
  perlu dua versi.
- `GET /health`: bentuk balasan diselaraskan dengan skema `Health` di
  `openapi.yaml`. Blok per-ketergantungan berganti nama dari `checks` ke
  `dependencies`, ditambah `version`; `database` memakai enum `[ok, fail]`,
  `whatsapp` `[connected, disconnected]`, dan `storage` menjadi objek
  `{status, used_mb, limit_mb}` dengan status `[ok, near_full, full]`. Status
  keseluruhan pada 503 kini `degraded`, bukan `down`. `used_mb`/`limit_mb`
  dihitung dari kuota TOTAL unggahan, bukan batas per berkas. (T025)
- `SearchCandidate`: kandidat pencarian kini membawa seluruh atribut informatif
  FR-027 yang sebelumnya hilang dari kontrak dan struct Go, yaitu `city_code`,
  `city_name`, `machine_types`, `weekly_capacity`, `readiness_week`,
  `readiness_lead_days`, `total_capacity_until_deadline`, `completed_jobs`,
  blok `reputation`, dan penanda `stale_calendar` (FR-021). Tanpa atribut ini
  US2 tak bisa lewat karena langkah uji manual 2.1, 2.6, dan 2.7 bergantung
  padanya. Reputasi dihitung saat dibaca lewat kueri kedua `SearchReputation`
  atas profil satu halaman, bukan dimaterialisasi ke kolom listing atau profil
  (data-model.md bagian 19, FR-071). Ambang FR-073 ditegakkan di service:
  `completion_rate` baru terisi setelah pembagi mencapai tiga pesanan
  disepakati, di bawah itu `enough_data` tetap false dan `completion_rate` nil,
  sama seperti skema Reputation publik. `stale_calendar` dihitung dari `Clock`
  yang disuntikkan, bukan `time.Now` (Aturan 5), dan bersifat informatif saja,
  tak pernah mengubah urutan.


