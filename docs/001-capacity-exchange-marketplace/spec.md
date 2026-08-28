# Feature Specification: Capacity Exchange, Marketplace Subkontrak Kapasitas Konveksi (MVP)

**Feature Branch**: `001-capacity-exchange-marketplace`

**Created**: 2026-08-21

**Status**: Draft

**Input**: User description: "Ide fitur ada di PDF terlampir (Dokume Lomba New.pdf), marketplace subkontrak B2B yang mempertemukan UMKM konveksi berkapasitas menganggur dengan UMKM yang order-nya melebihi kapasitas. Scope MVP = daftar fitur Must-have; konteks tambahan (pengguna, project baru/existing, reuse, out-of-scope, tests) tidak diisi dan digantikan asumsi."

## Clarifications

### Session 2026-08-21

- Q: Apakah verifikasi identitas admin wajib dilewati sebelum listing bisa ditemukan pihak lain? → A: Tidak. Listing langsung tayang; verifikasi hanya menambah lencana pada profil dan hasil pencarian.
- Q: Apakah versi pertama menahan dana pihak lain di platform (escrow)? → A: Tidak. Untuk keperluan lomba dan bukan production, platform hanya mencatat kesepakatan dan konfirmasi pembayaran yang terjadi langsung antar pihak.
- Q: Faktor apa yang menentukan urutan hasil pencarian? → A: Kecocokan keras saja: kesesuaian jenis produk, spesifikasi mesin, lead time, dan ketersediaan kapasitas. Reputasi dan status verifikasi tidak mempengaruhi urutan.
- Q: Apakah jenis produk dan spesifikasi mesin dipilih dari daftar baku yang dikelola admin, atau diisi bebas sebagai teks? → A: Daftar baku tertutup untuk keduanya; subkontraktor dan pemberi order hanya memilih dari daftar itu, dan admin yang menambah item baru.
- Q: Siapa yang boleh membatalkan sebuah pesanan, dan sampai tahap mana pembatalan masih diizinkan? → A: Sebelum status "Produksi" kedua pihak boleh membatalkan sendiri dengan alasan dan kapasitas langsung kembali; setelah "Produksi" hanya lewat mediasi admin.
- Q: Bagaimana "tingkat penyelesaian" dihitung, dan apakah pembatalan ikut menurunkannya? → A: Pesanan selesai dibagi seluruh pesanan yang disepakati, tetapi pembatalan hanya dihitung merugikan pihak yang membatalkan.
- Q: Lokasi usaha dicatat sebagai apa, nama wilayah administratif saja, atau titik koordinat yang bisa dihitung jaraknya? → A: Wilayah administratif berjenjang, dan titik koordinat usaha juga dicatat sebagai informasi tampilan.
- Q: Apa yang terjadi jika pemberi order tidak pernah mengonfirmasi penerimaan padahal barang sudah dikirim? → A: Otomatis dianggap diterima setelah 7 hari sejak status "Dikirim", disertai pemberitahuan sebelum tenggat itu jatuh.
- Q: Bagaimana pesanan yang jumlahnya melampaui kapasitas satu minggu diperlakukan? → A: Kapasitas dijumlahkan dari minggu berjalan sampai minggu deadline pemberi order. Lolos bila totalnya cukup, tertolak bila deadline-nya terlalu dekat.
- Q: Bagaimana kapasitas dialokasikan setelah kesepakatan terbentuk? → A: Mengisi periode mingguan paling awal lebih dulu sampai jumlah pesanan terpenuhi, dicatat per periode sebagai baris alokasi tersendiri.
- Q: Kapasitas dinyatakan dalam satuan apa? → A: Satu satuan tunggal, potong per minggu, untuk seluruh listing. Jenis produk hanya menyatakan kemampuan mengerjakan, tanpa angka kapasitas sendiri.
- Q: Lead time berarti apa? → A: Jeda kesiapan mulai, yaitu jumlah hari sejak kesepakatan sampai produksi dapat dimulai. Bukan durasi menyelesaikan pekerjaan.
- Q: Berapa tingkat hierarki wilayah dan bagaimana perluasan pencarian bekerja? → A: Tiga tingkat (kota/kabupaten, provinsi, seluruh Indonesia) memakai pembagian administratif resmi tanpa pengelompokan buatan sendiri.
- Q: Bolehkah satu akun mengirim request kuota ke listing miliknya sendiri? → A: Tidak boleh.
- Q: Berapa batas waktu balasan sebuah request kuota, dan siapa yang menentukan? → A: Sistem menetapkan 72 jam, bukan pemberi order.
- Q: Berapa kali pengiriman notifikasi diulang sebelum dianggap gagal? → A: Tiga kali, lalu dicatat gagal permanen sementara notifikasi di dalam platform tetap tampil.
- Q: Apakah hasil pencarian dipaginasi? → A: Ya, dengan urutan yang tetap stabil antar halaman.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Subkontraktor mempublikasikan kapasitas produksinya (Priority: P1)

Pak Budi punya 6 mesin jahit yang menganggur separuh minggu. Ia mendaftar sebagai subkontraktor, mengisi profil usahanya termasuk kota dan titik lokasi bengkelnya, lalu membuat satu listing kapasitas: jenis mesin yang dimiliki dan jenis produk yang bisa dikerjakan dipilih dari daftar yang sudah tersedia, ditambah kapasitas dalam potong per minggu dan jeda kesiapan mulai. Setelah disubmit, listing langsung tayang dan dapat ditemukan pihak lain tanpa menunggu persetujuan siapa pun.

**Why this priority**: Tanpa pasokan kapasitas yang terdaftar, tidak ada apa pun untuk dicari atau ditransaksikan. Ini sisi pertama dari masalah dua-sisi dan satu-satunya story yang bernilai walau berdiri sendiri, menjawab pencarian subkontraktor yang selama ini hanya lewat relasi personal sehingga jangkauannya terbatas dan tidak ada mekanisme matching sistematis [1].

**Independent Test**: Daftar sebagai subkontraktor, buat listing lengkap, lalu buka halaman publik listing tersebut sebagai pengunjung lain dan pastikan seluruh atribut kapasitas tampil benar.

**Acceptance Scenarios**:

1. **Given** saya sudah login sebagai subkontraktor dengan email dan nomor HP terverifikasi, **When** saya mengisi form listing dengan memilih jenis mesin dan jenis produk dari daftar yang tersedia serta mengisi kapasitas potong per minggu dan jeda kesiapan mulai, **Then** listing tersimpan, berstatus "Tayang", dan langsung dapat ditemukan melalui pencarian.
2. **Given** saya mengerjakan beberapa jenis produk dengan mesin yang sama, **When** saya memilih beberapa jenis produk pada listing, **Then** sistem menerima pilihan itu sebagai pernyataan kemampuan dan tetap memakai satu angka kapasitas untuk seluruh listing, tanpa meminta angka per jenis produk.
3. **Given** saya tidak menemukan jenis produk yang saya kerjakan di dalam daftar, **When** saya mengusulkan item baru, **Then** usulan itu terkirim ke admin, saya dapat menyimpan listing dengan item yang paling mendekati, dan saya diberi tahu ketika usulan diputuskan.
4. **Given** saya belum pernah mengajukan verifikasi identitas usaha, **When** listing saya tayang, **Then** listing tetap muncul di hasil pencarian tanpa lencana terverifikasi.
5. **Given** saya mengosongkan salah satu atribut wajib (jenis mesin, kapasitas mingguan, jenis produk, atau jeda kesiapan mulai), **When** saya submit, **Then** listing tidak tersimpan dan saya melihat pesan yang menyebutkan atribut mana yang kurang.
6. **Given** listing saya sudah tayang, **When** saya mengubah kapasitas mingguan atau jeda kesiapan mulai, **Then** perubahan langsung tercermin pada tampilan publik listing.

---

### User Story 2 - Pemberi order menemukan subkontraktor yang cocok (Priority: P2)

Bu Sari menerima order 3.000 kaos polos yang melebihi kapasitasnya. Ia mencari subkontraktor dengan memilih jenis produk dan spesifikasi mesin dari daftar yang sama seperti yang dipakai subkontraktor, menetapkan kota, mengisi jumlah yang dibutuhkan dan deadline. Hasilnya diurutkan berdasarkan seberapa banyak kriteria kerasnya terpenuhi, dan kandidat yang total kapasitasnya sampai deadline mencukupi ikut muncul meski kapasitas satu minggunya lebih kecil dari jumlah pesanan. Ketika hasilnya kosong, ia memperluas cakupan satu tingkat ke provinsi, lalu ke seluruh Indonesia.

**Why this priority**: Ini sisi permintaan yang mengubah direktori pasif menjadi mekanisme matching. Bergantung pada adanya listing (US1), tetapi dapat diuji dengan data listing yang sudah ada.

**Independent Test**: Dengan sekumpulan listing yang sudah tayang, jalankan pencarian dengan kombinasi filter produk, kota, jumlah, dan deadline; pastikan hanya kandidat yang memenuhi kriteria muncul, terurut sesuai aturan skor kecocokan, lalu perluas cakupan satu tingkat dan pastikan kandidat dari luar kota bertambah.

**Acceptance Scenarios**:

1. **Given** saya login sebagai pemberi order, **When** saya memilih jenis produk "Kaos Polos", kota "Bandung", jumlah 3.000 potong, dan deadline delapan minggu dari sekarang, **Then** sistem menampilkan kandidat yang cocok terurut berdasarkan skor kecocokan, dan setiap hasil menampilkan jeda kesiapan mulai serta total kapasitas tersedia sampai deadline saya.
2. **Given** seorang kandidat berkapasitas 500 potong per minggu dan deadline saya delapan minggu, **When** saya mencari 3.000 potong, **Then** kandidat itu terhitung memenuhi kriteria ketersediaan kapasitas karena total kapasitasnya sampai deadline mencukupi.
3. **Given** kandidat yang sama, **When** deadline saya hanya empat minggu, **Then** kandidat itu terhitung tidak memenuhi kriteria ketersediaan kapasitas dan turun peringkat di bawah kandidat yang memenuhi.
4. **Given** dua kandidat memenuhi jumlah kriteria keras yang sama, **When** hasil ditampilkan, **Then** urutan keduanya ditentukan oleh aturan pemecah seri yang tetap dan menghasilkan urutan yang sama pada pencarian yang diulang.
5. **Given** hasil pencarian lebih banyak dari satu halaman, **When** saya berpindah antar halaman lalu kembali, **Then** urutan seluruh hasil tetap sama dan tidak ada kandidat yang muncul dua kali maupun terlewat.
6. **Given** satu kandidat terverifikasi dan satu lagi tidak, tetapi keduanya memenuhi kriteria keras yang sama, **When** hasil ditampilkan, **Then** status verifikasi tidak mengubah posisi keduanya dan hanya tampil sebagai lencana.
7. **Given** tidak ada hasil yang cocok di tingkat kota, **When** saya memperluas cakupan pencarian, **Then** sistem naik satu tingkat ke provinsi tanpa mengubah filter lain, menyebutkan tingkat yang sedang dipakai, dan menampilkan hasil baru bila ada.
8. **Given** hasil masih kosong di tingkat provinsi, **When** saya memperluas sekali lagi, **Then** sistem mencakup seluruh Indonesia; bila masih kosong, sistem menyebutkan filter mana yang paling membatasi dan menyarankan pelonggaran yang konkret.
9. **Given** sebuah subkontraktor menandai seluruh periode sampai deadline saya sebagai penuh, **When** saya mencari, **Then** subkontraktor tersebut tidak muncul di hasil.
10. **Given** sebuah kandidat mencantumkan titik lokasi usahanya, **When** saya membuka profilnya, **Then** saya melihat posisi usaha itu pada peta beserta perkiraan jarak dari lokasi saya, dan informasi itu tidak mengubah urutan hasil pencarian.

---

### User Story 3 - Kirim request kuota ke beberapa kandidat dan bandingkan penawaran (Priority: P3)

Bu Sari memilih tiga kandidat, mengirim satu request kuota berisi spesifikasi produk, jumlah, bahan, dan deadline ke ketiganya sekaligus, lalu menerima estimasi harga dan jeda kesiapan mulai dari yang membalas dalam 72 jam. Ia membandingkan penawaran, boleh mengajukan counter-offer, dan menerima satu penawaran.

**Why this priority**: Titik konversi dari penemuan menjadi transaksi. Inti nilai platform, tetapi tidak bisa diuji sebelum US1 dan US2 ada.

**Independent Test**: Kirim satu request ke tiga akun subkontraktor, balas dari dua di antaranya dengan harga berbeda, dan pastikan pemberi order melihat perbandingan penawaran beserta status per kandidat, lalu terima salah satu.

**Acceptance Scenarios**:

1. **Given** saya memilih 3 kandidat subkontraktor, **When** saya submit request dengan spesifikasi produk, jumlah, bahan, dan deadline, **Then** ketiga kandidat menerima notifikasi, status per kandidat tercatat "Menunggu Balasan", dan batas waktu balasan 72 jam ditampilkan kepada kedua pihak.
2. **Given** saya juga terdaftar sebagai subkontraktor dan punya listing sendiri, **When** saya mencari kandidat, **Then** listing milik saya sendiri tidak muncul sebagai kandidat dan tidak dapat saya kirimi request kuota.
3. **Given** satu kandidat membalas dengan estimasi harga, **When** saya membuka detail request, **Then** saya melihat penawaran beserta jeda kesiapan mulai yang dijanjikan dan dapat membandingkannya berdampingan dengan penawaran lain.
4. **Given** saya mengajukan counter-offer harga, **When** subkontraktor menyetujui, **Then** penawaran menjadi kesepakatan dan sebuah pesanan terbentuk dengan harga, jumlah, dan deadline yang disepakati.
5. **Given** saya menerima satu penawaran, **When** kesepakatan terbentuk, **Then** kandidat lain pada request yang sama otomatis berstatus "Tidak Dilanjutkan" dan mendapat notifikasi.
6. **Given** sebuah kandidat menyanggupi jumlah yang melampaui total kapasitas tersisanya sampai deadline, **When** ia mengirim penawaran, **Then** sistem menolak penawaran itu dan menjelaskan berapa total kapasitas yang sebenarnya tersisa sampai deadline tersebut.
7. **Given** sebuah request melewati 72 jam, **When** tidak ada kandidat yang membalas, **Then** request berstatus "Kedaluwarsa" dan pemberi order diberi tahu.

---

### User Story 4 - Kalender ketersediaan tetap aktual (Priority: P4)

Pak Budi menandai minggu-minggu yang sudah penuh di kalender ketersediaannya, dan ketika ia menerima pesanan, kapasitas berkurang otomatis dari minggu-minggu paling awal yang masih tersedia tanpa ia perlu mengubahnya manual. Ketika sebuah pesanan dibatalkan sebelum produksi, seluruh kapasitas yang tadinya dipakai pesanan itu kembali dengan sendirinya.

**Why this priority**: Mengatasi listing kapasitas yang statis dan cepat kedaluwarsa sehingga informasinya tidak aktual dan transaksi gagal [1]. Diturunkan di bawah alur transaksi karena pencarian dan request masih berfungsi, dengan akurasi lebih rendah, tanpanya.

**Independent Test**: Tandai satu minggu sebagai penuh dan pastikan kandidat tidak dihitung memenuhi kriteria untuk periode itu; konfirmasi sebuah pesanan besar dan pastikan kapasitas berkurang dari minggu paling awal lebih dulu; lalu batalkan pesanan itu sebelum produksi dan pastikan seluruh kapasitas kembali ke angka semula.

**Acceptance Scenarios**:

1. **Given** saya punya listing tayang, **When** saya menandai minggu tertentu sebagai "Penuh", **Then** kalender tampil penuh dan kapasitas minggu itu tidak lagi ikut dijumlahkan dalam pencarian.
2. **Given** kapasitas saya 500 potong per minggu dan saya menerima pesanan 1.200 potong dengan deadline lima minggu, **When** pesanan dikonfirmasi, **Then** kapasitas berkurang 500 pada minggu pertama yang tersedia, 500 pada minggu berikutnya, dan 200 pada minggu ketiga, sementara minggu keempat dan kelima tetap utuh.
3. **Given** sebuah minggu sudah ditandai penuh, **When** alokasi pesanan baru dihitung, **Then** minggu itu dilewati dan alokasi berpindah ke minggu berikutnya yang masih tersedia.
4. **Given** sebuah pesanan dibatalkan sebelum status "Produksi", **When** pembatalan tersimpan, **Then** seluruh baris alokasi pesanan itu dibalik dan kapasitas setiap periode terkait kembali seperti sebelum pesanan itu terbentuk.
5. **Given** total kapasitas tersisa saya sampai suatu tanggal menjadi nol, **When** ada pencarian dengan deadline tanggal itu, **Then** listing saya tidak dihitung memenuhi kriteria ketersediaan tanpa perlu tindakan manual dari saya.
6. **Given** saya belum memperbarui kalender lebih dari 7 hari, **When** batas itu terlampaui, **Then** saya menerima pengingat dan listing saya ditandai "Data Belum Diperbarui" di hasil pencarian tanpa perubahan posisi urutan.

---

### User Story 5 - Kedua pihak memantau pesanan sampai tuntas (Priority: P5)

Setelah kesepakatan terbentuk, Bu Sari dan Pak Budi melihat pesanan yang sama di dashboard masing-masing dan mengikuti perkembangannya dari diterima, produksi, selesai, sampai dikirim. Pembayaran mereka lakukan langsung antar pihak, lalu keduanya menandai konfirmasinya di platform sebagai catatan. Bila salah satu pihak berubah pikiran sebelum produksi dimulai, ia dapat membatalkan dengan menyebut alasan. Bila Bu Sari lupa mengonfirmasi barang yang sudah dikirim, pesanan menutup sendiri setelah tujuh hari.

**Why this priority**: Membuat transaksi bisa dieksekusi dan diselesaikan, prasyarat bagi rating. Tidak diperlukan untuk membuktikan matching bekerja.

**Independent Test**: Dari satu pesanan yang sudah terbentuk, jalankan seluruh transisi status dari sisi subkontraktor dan pastikan pemberi order melihat status serta waktu perubahan yang sama; lalu pada pesanan kedua, biarkan status "Dikirim" melewati tujuh hari dan pastikan pesanan tertutup otomatis.

**Acceptance Scenarios**:

1. **Given** sebuah pesanan aktif, **When** subkontraktor mengubah status menjadi "Produksi", **Then** pemberi order melihat status baru beserta waktu perubahannya.
2. **Given** pesanan berstatus "Selesai", **When** pemberi order mengonfirmasi penerimaan, **Then** pesanan berpindah ke riwayat dan kedua pihak diminta memberi rating.
3. **Given** pesanan berstatus "Dikirim" dan pemberi order tidak mengonfirmasi apa pun, **When** tujuh hari terlampaui, **Then** pesanan ditandai dikonfirmasi otomatis, kedua pihak diberi tahu, dan keduanya dapat memberi rating.
4. **Given** pesanan berstatus "Dikirim" dan tenggat otomatis tinggal dua hari, **When** batas itu tercapai, **Then** pemberi order menerima pemberitahuan yang menyebutkan tanggal pesanan akan dianggap diterima.
5. **Given** pemberi order melaporkan sengketa sebelum tenggat tujuh hari, **When** laporan tersimpan, **Then** hitungan konfirmasi otomatis dihentikan dan pesanan menunggu mediasi admin.
6. **Given** sebuah pesanan belum berstatus "Produksi", **When** salah satu pihak membatalkan dengan menyebutkan alasan, **Then** pesanan berstatus "Dibatalkan", pihak lain diberi tahu beserta alasannya, dan seluruh alokasi kapasitasnya dibalik.
7. **Given** sebuah pesanan sudah berstatus "Produksi", **When** salah satu pihak ingin membatalkan, **Then** pembatalan sendiri tidak tersedia dan pihak itu diarahkan untuk melaporkan sengketa agar ditengahi admin.
8. **Given** pembayaran dilakukan langsung antar pihak, **When** salah satu pihak menandai pembayaran terkirim atau diterima, **Then** catatan itu tampil pada pesanan bagi kedua pihak beserta keterangan bahwa platform tidak menahan maupun menjamin dana.
9. **Given** sebuah transisi status tidak sah (misalnya dari "Menunggu" langsung ke "Dikirim"), **When** dicoba, **Then** sistem menolak dan menjelaskan urutan status yang diizinkan.

---

### User Story 6 - Reputasi terbentuk dari transaksi nyata (Priority: P6)

Setelah pesanan tuntas, kedua pihak saling memberi rating 1–5 dan ulasan tertulis. Profil publik menampilkan rating rata-rata, jumlah pekerjaan selesai, dan tingkat penyelesaian yang menghitung pembatalan sebagai beban pihak yang membatalkan, sehingga UMKM yang belum saling kenal punya dasar untuk percaya.

**Why this priority**: Menjawab tidak adanya sistem reputasi antar UMKM yang belum saling kenal, yang membuat trust rendah dan transaksi antar pihak asing terhambat [1], tetapi baru bermakna setelah ada transaksi selesai.

**Independent Test**: Selesaikan satu pesanan, isi rating dan ulasan dari kedua sisi, lalu buka profil publik dan pastikan rata-rata rating serta hitungan pekerjaan selesai berubah sesuai; lalu batalkan sebuah pesanan lain dari satu sisi dan pastikan hanya tingkat penyelesaian pihak yang membatalkan yang turun.

**Acceptance Scenarios**:

1. **Given** sebuah pesanan sudah dikonfirmasi selesai, baik secara manual maupun otomatis, **When** saya mengirim rating 1–5 dan ulasan tertulis, **Then** ulasan tampil di profil publik pihak lain beserta identitas pemberi ulasan dan tanggal transaksi.
2. **Given** saya membatalkan sebuah pesanan sebelum produksi, **When** profil publik dihitung ulang, **Then** tingkat penyelesaian saya turun karena pesanan itu masuk pembagi tanpa masuk hitungan selesai, sementara tingkat penyelesaian pihak lain tidak terpengaruh sama sekali.
3. **Given** sebuah usaha baru menyepakati dua pesanan, **When** profilnya dibuka, **Then** tingkat penyelesaian belum ditampilkan sebagai persentase dan diganti keterangan bahwa datanya belum cukup.
4. **Given** saya belum bertransaksi dengan sebuah usaha, **When** saya mencoba memberi rating pada usaha itu, **Then** sistem menolak.
5. **Given** sebuah ulasan dilaporkan, **When** admin menyembunyikannya, **Then** ulasan itu tidak lagi tampil publik dan tidak dihitung dalam rata-rata rating, dan tindakan tersebut tercatat.

---

### User Story 7 - Admin mengelola daftar baku, lencana verifikasi, dan mediasi (Priority: P7)

Tim Ops mengelola daftar jenis produk dan jenis mesin yang menjadi tulang punggung pencocokan, memeriksa dokumen identitas untuk memberi atau menolak lencana terverifikasi, memantau pesanan yang melewati deadline, dan menengahi ketika salah satu pihak melapor atau ketika pembatalan diminta setelah produksi berjalan. Keputusan verifikasi tidak menahan listing siapa pun dari tayang.

**Why this priority**: Daftar baku adalah prasyarat data bagi US1 dan US2, sementara lencana dan mediasi adalah lapisan di atas alur yang sudah jalan. Pengisian awal daftar baku dan data wilayah dilakukan lewat perintah sekali jalan, bukan lewat antarmuka, sehingga bagian antarmuka admin dapat tetap berada di prioritas terakhir.

**Independent Test**: Tambahkan satu jenis produk baru dan pastikan item itu langsung dapat dipilih pada form listing dan filter pencarian; lalu setujui dan tolak satu pengajuan verifikasi dan pastikan lencana serta notifikasi berubah sesuai sementara listing tetap tayang pada kedua kasus.

**Acceptance Scenarios**:

1. **Given** saya admin, **When** saya menambah jenis produk atau jenis mesin baru ke daftar baku, **Then** item itu langsung tersedia pada form listing dan pada filter pencarian.
2. **Given** sebuah item daftar baku sudah dipakai listing yang tayang, **When** saya menonaktifkannya, **Then** item itu tidak lagi dapat dipilih untuk listing baru sementara listing yang sudah memakainya tetap utuh dan tetap dapat ditemukan.
3. **Given** ada pengguna mengusulkan item baru, **When** saya menyetujui atau menolaknya, **Then** pengusul diberi tahu beserta hasil keputusannya.
4. **Given** ada UMKM mengirim dokumen identitas usaha dan foto lokasi, **When** saya membuka panel verifikasi, **Then** saya bisa menyetujui atau menolak dengan catatan alasan.
5. **Given** saya menyetujui, **When** keputusan tersimpan, **Then** usaha tersebut mendapat lencana terverifikasi pada profil dan hasil pencarian, dan pengguna mendapat notifikasi.
6. **Given** saya menolak dengan alasan, **When** keputusan tersimpan, **Then** listing usaha itu tetap tayang tanpa lencana, pengguna mendapat notifikasi beserta alasannya, dan dapat mengajukan ulang.
7. **Given** ada pesanan berstatus "Produksi" melewati deadline, **When** keterlambatan terdeteksi, **Then** muncul alert di dashboard admin dan notifikasi dikirim ke kedua pihak.
8. **Given** ada sengketa atau permintaan pembatalan setelah produksi berjalan, **When** salah satu pihak melapor, **Then** saya bisa melihat riwayat lengkap pesanan dan menandai pesanan sebagai "Dalam Mediasi".
9. **Given** sebuah pesanan dalam mediasi, **When** saya menutupnya sebagai dibatalkan dengan catatan, **Then** saya menentukan apakah seluruh alokasi kapasitasnya dibalik dan pihak mana yang menanggung pembatalan dalam perhitungan tingkat penyelesaian, dan keputusan itu tercatat.

---

### Edge Cases

- Pesanan yang jumlahnya melampaui total kapasitas sampai deadline, tetapi hanya selisih kecil.
- Deadline pemberi order jatuh di tengah minggu. Apakah minggu itu dihitung penuh atau tidak dihitung sama sekali?
- Jeda kesiapan mulai subkontraktor membuat produksi baru bisa dimulai setelah beberapa minggu pertama terlewat, sehingga kapasitas yang benar-benar terpakai bukan minggu paling awal.
- Subkontraktor menurunkan kapasitas mingguannya setelah punya alokasi berjalan, sehingga kapasitas terpakai melebihi kapasitas total yang baru.
- Subkontraktor menandai minggu penuh padahal minggu itu sudah punya alokasi dari pesanan berjalan.
- Dua pemberi order menerima penawaran yang memakai periode kapasitas yang sama secara bersamaan.
- Alokasi sebuah pesanan gagal di tengah jalan setelah beberapa periode terisi. Apakah seluruhnya dibatalkan atau sebagian tersimpan?
- Semua kandidat pada satu request menolak, atau menolak setelah lewat 72 jam.
- Pemberi order membatalkan berkali-kali sebelum produksi sehingga alokasi kapasitas subkontraktor bolak-balik terpakai dan bebas.
- Pembatalan diajukan tepat ketika pihak lain sedang mengubah status menjadi "Produksi".
- Pesanan dibatalkan setelah pembayaran sudah dicatat terkirim oleh pemberi order.
- Kedua pihak mencatat status pembayaran yang bertentangan.
- Pemberi order melaporkan sengketa pada hari keenam untuk menghentikan konfirmasi otomatis, lalu tidak menanggapi mediasi.
- Pesanan tertutup otomatis padahal barang sebenarnya tidak pernah sampai.
- Usaha dengan satu atau dua pesanan saja sehingga tingkat penyelesaiannya menyesatkan.
- Akun berperan ganda mencoba mengirim request kuota ke listing miliknya sendiri, langsung maupun lewat tautan yang disalin.
- Hasil pencarian berubah karena ada listing baru saat pengguna sedang berpindah halaman.
- Admin menonaktifkan item daftar baku yang masih dipakai banyak listing tayang.
- Subkontraktor tidak menemukan jenis produk atau mesinnya di daftar baku dan tidak dapat menyelesaikan listing.
- Pemberi order mencari jenis produk yang belum ada di daftar baku sama sekali.
- Sebuah kota tidak ada dalam data wilayah yang tersimpan.
- Pencarian di kota yang secara praktik satu klaster dengan kota di provinsi lain, sehingga perluasan satu tingkat belum menjangkau tetangganya.
- Titik koordinat yang dimasukkan pengguna berada jauh dari kota yang ia pilih.
- Listing tayang tanpa satu pun periode ketersediaan terisi.
- Pencarian menghasilkan nol hasil bahkan setelah diperluas ke seluruh Indonesia.
- Beberapa kandidat memenuhi kriteria keras yang sama persis sehingga skor kecocokannya identik.
- Satu usaha mendaftarkan beberapa akun untuk mengakali rating buruk atau tingkat penyelesaian yang rendah.
- Pengajuan verifikasi identitas dengan dokumen kabur atau tidak sesuai nama usaha.
- Ulasan berisi ancaman, nomor kontak, atau ajakan bertransaksi di luar platform.
- Pengiriman notifikasi gagal tiga kali sementara kejadian yang memicunya sudah tersimpan.

## Requirements *(mandatory)*

Nomor FR bersifat tetap dan tidak digunakan ulang. Requirement yang lahir dari revisi ditambahkan mulai FR-075, sehingga rujukan yang sudah ada tidak bergeser. Revisi 2026-08-22 menambahkan FR-087 sampai FR-091 dari empat pertentangan antar artefak yang ditemukan `/analyze`, seluruhnya menyangkut rentang kapasitas, horizon kalender, propagasi perubahan kapasitas, kesiapan mulai pada penawaran, dan penggolongan notifikasi.

### Istilah yang Mengikat

- **Kapasitas mingguan**: jumlah potong yang dapat diselesaikan sebuah listing dalam satu minggu, berlaku untuk seluruh jenis produk yang dinyatakan listing itu. Bukan angka per jenis produk.
- **Jenis produk pada listing**: pernyataan kemampuan mengerjakan, tanpa angka kapasitas tersendiri.
- **Jeda kesiapan mulai** (sebelumnya disebut "lead time"): jumlah hari sejak kesepakatan terbentuk sampai produksi dapat dimulai. Bukan durasi menyelesaikan pekerjaan.
- **Durasi penyelesaian**: dihitung sistem dari jumlah pesanan dibagi kapasitas mingguan, bukan diisi pengguna.
- **Periode mingguan**: satu minggu yang dimulai hari Senin menurut zona waktu Asia/Jakarta.
- **Minggu kesiapan mulai**: periode mingguan yang memuat tanggal acuan ditambah jeda kesiapan mulai. Tanggal acuannya adalah tanggal kesepakatan pada sebuah pesanan, dan tanggal pencarian pada perhitungan kandidat. Ini periode paling awal yang boleh dihitung maupun dialokasikan; minggu sebelum itu tidak ikut, karena subkontraktor sendiri menyatakan produksi belum dapat dimulai.
- **Rentang kapasitas**: rentang periode mingguan dari minggu kesiapan mulai sampai periode yang memuat deadline pemberi order, inklusif. Seluruh penjumlahan dan alokasi kapasitas memakai rentang ini, bukan dihitung dari minggu berjalan. Bila minggu kesiapan mulai jatuh setelah minggu deadline, rentangnya kosong dan kapasitas yang terhitung nol.

### Functional Requirements

**Akun, Peran, dan Identitas**

- **FR-001**: Sistem MUST memungkinkan pendaftaran akun usaha dengan memilih peran subkontraktor, pemberi order, atau keduanya, dan peran dapat ditambahkan kemudian tanpa membuat akun baru.
- **FR-002**: Sistem MUST memverifikasi alamat email dan nomor HP (melalui kode sekali pakai) sebelum akun dapat mempublikasikan listing atau mengirim request kuota.
- **FR-003**: Pengguna MUST dapat masuk, keluar, dan memulihkan akses akun tanpa bantuan admin.
- **FR-004**: Sistem MUST menyediakan profil usaha yang memuat nama usaha, kota/kabupaten yang dipilih dari data wilayah, titik koordinat lokasi usaha, deskripsi singkat, dan status lencana verifikasi.
- **FR-005**: Sistem MUST membatasi setiap tindakan hanya pada peran yang berwenang: hanya subkontraktor mengelola listing dan kalender, hanya pemberi order mengirim request kuota, hanya admin mengelola daftar baku serta memutuskan verifikasi dan mediasi.
- **FR-006**: Sistem MUST memungkinkan pengguna mengajukan verifikasi identitas usaha dengan mengunggah nomor identitas usaha atau identitas pemilik beserta foto lokasi usaha, dan MUST menampilkan status pengajuan.
- **FR-007**: Admin MUST dapat menyetujui atau menolak pengajuan verifikasi dengan catatan alasan, dan keputusan tersebut MUST tercatat beserta identitas admin dan waktunya.
- **FR-008**: Sistem MUST menampilkan lencana terverifikasi pada profil publik dan pada setiap hasil pencarian bagi usaha yang pengajuannya disetujui.
- **FR-009**: Sistem MUST membatasi akses ke dokumen identitas dan foto lokasi hanya pada pemiliknya dan admin yang bertugas melakukan verifikasi.
- **FR-010**: Sistem MUST menayangkan listing segera setelah disimpan tanpa menunggu keputusan verifikasi identitas, dan status verifikasi MUST TIDAK mempengaruhi apakah sebuah listing dapat ditemukan, urutannya di hasil pencarian, maupun kemampuannya menerima request kuota dan membentuk kesepakatan.
- **FR-011**: Sistem MUST mengizinkan pengajuan ulang verifikasi setelah penolakan, dan penolakan MUST TIDAK menurunkan atau menyembunyikan listing yang sudah tayang.
- **FR-057**: Sistem MUST memberi tahu pengguna secara jelas bahwa titik koordinat lokasi usahanya akan tampil publik pada profil, dan MUST mengizinkan pengguna mengubah titiknya sendiri kapan saja.

**Daftar Baku dan Wilayah**

- **FR-058**: Sistem MUST menyediakan daftar baku tertutup untuk jenis produk dan untuk jenis mesin, dan seluruh pengisian maupun penyaringan atas kedua atribut itu MUST memakai item dari daftar tersebut, bukan teks bebas.
- **FR-059**: Admin MUST dapat menambah item baru, mengubah nama item, dan menonaktifkan item pada daftar baku.
- **FR-060**: Menonaktifkan sebuah item MUST TIDAK mengubah atau menghapus data listing yang sudah memakainya, dan listing tersebut MUST tetap dapat ditemukan melalui pencarian.
- **FR-061**: Pengguna MUST dapat mengusulkan item baru ketika tidak menemukan yang sesuai, MUST tetap dapat menyimpan listingnya dengan item yang tersedia, dan MUST diberi tahu ketika usulannya diputuskan.
- **FR-062**: Sistem MUST menyimpan data wilayah dua tingkat administratif (provinsi dan kota/kabupaten), dan setiap kota/kabupaten MUST termasuk dalam tepat satu provinsi. Pengelompokan wilayah di luar pembagian administratif resmi MUST TIDAK dipakai.
- **FR-075**: Sistem MUST dapat mengisi data wilayah dan daftar baku dalam satu tindakan yang dapat diulang tanpa menduplikasi data, MUST menyimpan salinannya sendiri, dan MUST TIDAK bergantung pada sumber data luar saat melayani permintaan pengguna.

**Listing Kapasitas**

- **FR-012**: Subkontraktor MUST dapat membuat listing kapasitas yang memuat jenis dan jumlah mesin serta jenis produk yang dipilih dari daftar baku, satu angka kapasitas mingguan dalam potong yang berlaku untuk seluruh jenis produk pada listing itu, dan jeda kesiapan mulai dalam hari.
- **FR-013**: Sistem MUST menolak penyimpanan listing jika salah satu atribut wajib pada FR-012 kosong, dan MUST menyebutkan atribut mana yang kurang.
- **FR-014**: Sistem MUST memvalidasi bahwa kapasitas mingguan dan jeda kesiapan mulai berupa angka bulat tidak negatif, dan kapasitas mingguan lebih besar dari nol.
- **FR-015**: Subkontraktor MUST dapat mengubah, menonaktifkan sementara, dan mengaktifkan kembali listingnya sendiri.
- **FR-016**: Sistem MUST menampilkan profil publik subkontraktor yang menggabungkan atribut listing, ketersediaan kapasitas terkini, titik lokasi usaha, dan ringkasan reputasi.
- **FR-076**: Sistem MUST TIDAK meminta maupun menyimpan angka kapasitas terpisah per jenis produk, dan MUST menghitung seluruh alokasi terhadap satu kapasitas mingguan bersama.

**Kalender Ketersediaan dan Alokasi Kapasitas**

- **FR-017**: Subkontraktor MUST dapat menandai ketersediaan kapasitas per periode mingguan pada horizon minimal 3 bulan ke depan.
- **FR-018**: Sistem MUST mengurangi kapasitas secara otomatis ketika sebuah pesanan dikonfirmasi, dengan mengisi periode mingguan paling awal yang masih tersedia lebih dulu sampai jumlah pesanan terpenuhi, tanpa tindakan manual dari subkontraktor.
- **FR-019**: Sistem MUST mengecualikan sebuah periode dari penjumlahan kapasitas apabila kapasitas tersisanya nol atau periode itu ditandai penuh.
- **FR-020**: Sistem MUST membalik seluruh baris alokasi sebuah pesanan segera setelah pesanan dibatalkan sebelum status "Produksi", sehingga kapasitas setiap periode terkait kembali ke angka sebelum pesanan itu terbentuk.
- **FR-021**: Sistem MUST menandai listing yang kalendernya tidak diperbarui lebih dari 7 hari dan MUST mengirim pengingat kepada pemiliknya; penanda ini bersifat informatif dan MUST TIDAK mengubah urutan hasil pencarian.
- **FR-077**: Sistem MUST mencatat alokasi kapasitas sebagai baris tersendiri per pasangan pesanan dan periode mingguan, memuat jumlah yang dialokasikan pada periode itu, sehingga alokasi satu pesanan dapat tersebar ke beberapa periode dan dapat dibalik secara utuh.
- **FR-078**: Sistem MUST melewati periode yang ditandai penuh atau kapasitasnya sudah habis ketika mengisi alokasi, dan MUST melanjutkan ke periode berikutnya yang masih tersedia.
- **FR-079**: Sistem MUST menjaga agar kapasitas terpakai sebuah periode tidak pernah melampaui kapasitas totalnya, dan jaminan ini MUST tetap berlaku bahkan ketika logika aplikasi keliru.
- **FR-087**: Sistem MUST menghitung penjumlahan dan alokasi kapasitas hanya di dalam rentang kapasitas, yaitu mulai dari minggu kesiapan mulai, bukan dari minggu berjalan. Sebuah pesanan MUST menyimpan minggu kesiapan mulainya sendiri saat kesepakatan terbentuk, dan alokasi MUST TIDAK menyentuh periode mingguan sebelum minggu itu.
- **FR-088**: Sistem MUST memperlakukan periode mingguan yang belum dibuat tetapi jatuh sebelum deadline yang diminta sebagai berkapasitas penuh saat menilai kandidat, dan MUST membuat periode yang kurang itu ketika sebuah kesepakatan benar-benar terbentuk, tanpa bergantung pada penjadwal bergulir.
- **FR-089**: Ketika kapasitas mingguan sebuah listing diubah, sistem MUST memperbarui kapasitas total seluruh periode mendatang yang belum memiliki alokasi aktif, dan MUST membiarkan periode yang sudah memiliki alokasi tetap seperti semula.

**Pencarian dan Matching**

- **FR-022**: Pemberi order MUST dapat menyaring subkontraktor berdasarkan jenis produk dan spesifikasi mesin yang dipilih dari daftar baku, wilayah, jeda kesiapan mulai, jumlah yang dibutuhkan, dan deadline, serta mengombinasikan filter-filter tersebut.
- **FR-023**: Sistem MUST mengurutkan hasil pencarian berdasarkan skor kecocokan yang dihitung hanya dari empat kriteria keras berikut, masing-masing bernilai terpenuhi atau tidak: (a) jenis produk yang dicari termasuk dalam jenis produk listing, (b) spesifikasi mesin yang dicari dimiliki listing, (c) jeda kesiapan mulai listing tidak melebihi jeda yang dapat diterima pemberi order, dan (d) total kapasitas tersisa listing di dalam rentang kapasitas tidak kurang dari jumlah yang dibutuhkan. Kandidat yang memenuhi lebih banyak kriteria MUST berada di atas.
- **FR-024**: Skor kecocokan MUST TIDAK dipengaruhi oleh rating, tingkat penyelesaian, jumlah pekerjaan selesai, status verifikasi, kebaruan pembaruan kalender, jarak koordinat, tanggal pendaftaran, maupun imbalan komersial apa pun.
- **FR-025**: Ketika beberapa kandidat memiliki skor kecocokan sama, sistem MUST memakai pemecah seri yang tetap dan dapat diulang: total kapasitas tersisa terbesar sampai deadline yang diminta, lalu jeda kesiapan mulai terpendek, lalu urutan abjad nama usaha, lalu pengenal listing.
- **FR-026**: Sistem MUST dapat menjelaskan kepada pemberi order kriteria mana yang terpenuhi dan tidak terpenuhi untuk setiap kandidat di hasil pencarian.
- **FR-027**: Setiap hasil pencarian MUST menampilkan atribut yang dipakai pemberi order untuk memutuskan: kota/kabupaten, jenis mesin, kapasitas mingguan, total kapasitas tersisa sampai deadline yang diminta, jeda kesiapan mulai, rating rata-rata, jumlah pekerjaan selesai, tingkat penyelesaian, dan lencana verifikasi bila ada.
- **FR-028**: Ketika pencarian tidak menghasilkan apa pun pada tingkat wilayah tertinggi yang tersedia, sistem MUST menyebutkan filter mana yang paling membatasi dan menyarankan pelonggaran yang konkret.
- **FR-063**: Sistem MUST menyediakan tindakan perluasan pencarian yang menaikkan cakupan wilayah tepat satu tingkat, dengan urutan kota/kabupaten, lalu provinsi, lalu seluruh Indonesia, tanpa mengubah filter lain, MUST menyebutkan tingkat yang sedang dipakai pada hasil, dan MUST menghentikan perluasan setelah tingkat seluruh Indonesia.
- **FR-064**: Sistem MUST menampilkan titik lokasi usaha pada peta di profil publik beserta perkiraan jarak dari lokasi pemberi order, dan informasi jarak ini MUST bersifat informatif saja: tidak menjadi filter dan tidak mempengaruhi urutan hasil pencarian.
- **FR-080**: Sistem MUST memaginasi hasil pencarian, dan urutan seluruh hasil MUST tetap sama antar halaman sehingga tidak ada kandidat yang muncul dua kali maupun terlewat selama data tidak berubah.
- **FR-081**: Sistem MUST mengecualikan listing milik akun yang sedang mencari dari hasil pencarian yang ia lakukan sendiri.

**Request Kuota dan Kesepakatan**

- **FR-029**: Pemberi order MUST dapat mengirim satu request kuota berisi jenis produk (dari daftar baku), jumlah, bahan, deadline, dan catatan tambahan kepada beberapa kandidat sekaligus dalam satu tindakan.
- **FR-030**: Sistem MUST memelihara status per kandidat pada setiap request (menunggu balasan, ditawar, ditolak, kedaluwarsa, tidak dilanjutkan, disepakati) dan MUST menampilkannya kepada kedua pihak.
- **FR-031**: Subkontraktor MUST dapat membalas request dengan estimasi harga dan jeda kesiapan mulai yang dijanjikan, atau menolak dengan alasan.
- **FR-032**: Pemberi order MUST dapat melihat seluruh penawaran atas satu request secara berdampingan untuk dibandingkan.
- **FR-033**: Kedua pihak MUST dapat mengajukan counter-offer harga sampai tercapai kesepakatan atau salah satu pihak menghentikan negosiasi, dan seluruh riwayat penawaran MUST tersimpan.
- **FR-034**: Sistem MUST membentuk pesanan dengan harga, jumlah, dan deadline yang disepakati saat sebuah penawaran diterima, dan MUST menutup kandidat lain pada request yang sama dengan notifikasi.
- **FR-035**: Sistem MUST menolak penawaran yang jumlahnya melampaui total kapasitas tersisa subkontraktor di dalam rentang kapasitas sampai periode yang memuat deadline yang diminta, dan MUST menyebutkan total kapasitas yang sebenarnya tersisa sampai deadline tersebut.
- **FR-036**: Sistem MUST memastikan kapasitas satu periode hanya dialokasikan kepada satu kesepakatan ketika dua kesepakatan atas periode yang sama terjadi berbarengan, sehingga hanya satu yang berhasil dan pihak yang gagal MUST diberi tahu beserta alasannya.
- **FR-037**: Sistem MUST menandai request yang tidak dibalas melewati batas waktu balasan sebagai kedaluwarsa dan memberi tahu pemberi order.
- **FR-082**: Sistem MUST menetapkan batas waktu balasan setiap request kuota sebesar 72 jam sejak request dikirim, MUST TIDAK meminta pemberi order menentukannya, dan MUST menampilkan batas itu kepada kedua pihak.
- **FR-083**: Sistem MUST menolak pengiriman request kuota kepada listing yang dimiliki akun pengirim sendiri, termasuk ketika permintaan dikirim tanpa melalui hasil pencarian.
- **FR-084**: Sistem MUST membentuk pesanan dan seluruh baris alokasi kapasitasnya dalam satu tindakan yang utuh: bila salah satu periode gagal dialokasikan, seluruh pembentukan pesanan MUST dibatalkan dan tidak ada kapasitas yang tersisa terpakai.
- **FR-090**: Sistem MUST menolak penawaran yang minggu kesiapan mulainya jatuh setelah periode yang memuat deadline yang diminta, karena produksi tidak akan dapat dimulai sebelum deadline terlampaui, dan MUST menjelaskan alasan itu kepada subkontraktor.

**Pesanan, Pembatalan, dan Pencatatan Pembayaran**

- **FR-038**: Sistem MUST menampilkan daftar pesanan aktif dan riwayat pesanan selesai kepada kedua pihak, masing-masing hanya untuk pesanan yang melibatkan dirinya.
- **FR-039**: Sistem MUST melacak status pesanan melalui tahap diterima, produksi, selesai, dikirim, dikonfirmasi diterima, serta status akhir dibatalkan dan dalam mediasi, dan MUST mencatat waktu setiap perubahan status beserta pelakunya.
- **FR-040**: Sistem MUST TIDAK menahan, menyalurkan, maupun memproses dana pihak mana pun. Pembayaran terjadi langsung antar kedua pihak di luar platform.
- **FR-041**: Kedua pihak MUST dapat menandai pembayaran sebagai terkirim dan sebagai diterima pada sebuah pesanan, beserta tanggal dan catatan bebas, sebagai catatan yang dapat dilihat kedua pihak.
- **FR-042**: Sistem MUST menyatakan secara jelas pada halaman pesanan bahwa catatan pembayaran adalah pernyataan pengguna dan platform tidak menjamin maupun memverifikasi terjadinya pembayaran.
- **FR-043**: Sistem MUST menandai perbedaan pernyataan pembayaran antara kedua pihak dan MUST menampilkan perbedaan itu kepada admin ketika sengketa dilaporkan.
- **FR-044**: Sistem MUST menolak transisi status pesanan yang tidak mengikuti urutan yang ditetapkan.
- **FR-045**: Sistem MUST menandai pesanan yang melewati deadline dan memberi tahu kedua pihak serta admin.
- **FR-046**: Salah satu pihak MUST dapat melaporkan sengketa atas sebuah pesanan, dan admin MUST dapat menandai pesanan tersebut "Dalam Mediasi" serta melihat seluruh riwayat request, penawaran, status, alokasi kapasitas, catatan pembayaran, alasan pembatalan, dan lampirannya.
- **FR-065**: Sebelum pesanan berstatus "Produksi", kedua pihak MUST dapat membatalkan pesanan atas kehendak sendiri dengan menyebutkan alasan, dan sistem MUST memberi tahu pihak lain beserta alasan tersebut.
- **FR-066**: Setelah pesanan berstatus "Produksi", pembatalan atas kehendak sendiri MUST TIDAK tersedia bagi kedua pihak, dan sistem MUST mengarahkan pihak yang ingin membatalkan untuk melaporkan sengketa agar ditengahi admin.
- **FR-067**: Admin MUST dapat menutup pesanan yang berada dalam mediasi sebagai dibatalkan dengan catatan, dan pada saat itu MUST menentukan secara eksplisit apakah seluruh alokasi kapasitasnya dibalik dan pihak mana yang menanggung pembatalan tersebut dalam perhitungan tingkat penyelesaian.
- **FR-068**: Sistem MUST menandai pesanan berstatus "Dikirim" sebagai dikonfirmasi diterima secara otomatis setelah 7 hari sejak status tersebut ditetapkan, dan MUST memberi tahu kedua pihak bahwa penutupan terjadi secara otomatis.
- **FR-069**: Sistem MUST memberi tahu pemberi order sebelum tenggat konfirmasi otomatis jatuh, dengan menyebutkan tanggal pesanan akan dianggap diterima.
- **FR-070**: Sistem MUST menghentikan hitungan konfirmasi otomatis ketika sengketa dilaporkan sebelum tenggat, dan pesanan MUST menunggu keputusan mediasi admin.

**Reputasi**

- **FR-047**: Kedua pihak MUST dapat memberi rating 1–5 dan ulasan tertulis hanya atas pesanan yang sudah dikonfirmasi diterima, baik secara manual maupun otomatis, dan hanya satu kali per pesanan per pihak.
- **FR-048**: Sistem MUST menampilkan rating rata-rata, jumlah pekerjaan selesai, dan tingkat penyelesaian pesanan pada profil publik.
- **FR-049**: Setiap ulasan yang tampil MUST menyertakan tanggal transaksi dan identitas pemberi ulasan, sehingga tidak ada ulasan anonim.
- **FR-050**: Admin MUST dapat menyembunyikan ulasan yang melanggar ketentuan, dan ulasan yang disembunyikan MUST dikeluarkan dari perhitungan rata-rata rating.
- **FR-071**: Sistem MUST menghitung tingkat penyelesaian sebuah usaha sebagai jumlah pesanan yang dikonfirmasi diterima dibagi jumlah seluruh pesanan yang pernah disepakati oleh usaha tersebut, dan MUST menampilkan kedua angka penyusunnya agar persentasenya dapat ditelusuri.
- **FR-072**: Sebuah pesanan yang dibatalkan MUST masuk pembagi tingkat penyelesaian hanya bagi pihak yang membatalkan, dan MUST TIDAK mempengaruhi tingkat penyelesaian pihak lain sama sekali.
- **FR-073**: Sistem MUST menahan tampilan tingkat penyelesaian sebagai persentase sampai sebuah usaha memiliki minimal 3 pesanan yang disepakati, dan sebelum ambang itu MUST menampilkan keterangan bahwa data belum cukup.

**Notifikasi**

- **FR-051**: Sistem MUST mengirim notifikasi pada kejadian berikut: request kuota diterima, penawaran masuk, counter-offer, kesepakatan terbentuk, perubahan status pesanan, catatan pembayaran dibuat pihak lain, deadline mendekat dan terlampaui, keputusan verifikasi, dan permintaan rating.
- **FR-052**: Sistem MUST mengirim notifikasi melalui email dan pesan WhatsApp, serta menampilkannya di dalam platform.
- **FR-053**: Pengguna MUST dapat mengatur kanal notifikasi mana yang ia terima untuk notifikasi non-transaksional, sementara notifikasi transaksional tidak dapat dimatikan.
- **FR-054**: Sistem MUST tetap menampilkan notifikasi di dalam platform meskipun pengiriman ke kanal eksternal gagal.
- **FR-074**: Sistem MUST mengirim notifikasi juga pada kejadian berikut: pesanan dibatalkan oleh pihak lain beserta alasannya, tenggat konfirmasi otomatis mendekat, pesanan tertutup secara otomatis, dan keputusan admin atas usulan item daftar baku.
- **FR-085**: Sistem MUST mencoba mengirim ulang notifikasi yang gagal ke kanal eksternal paling banyak 3 kali, lalu MUST menandainya gagal permanen beserta alasannya.
- **FR-086**: Kegagalan pengiriman notifikasi MUST TIDAK menggagalkan maupun membatalkan kejadian yang memicunya, dan catatan notifikasi di dalam platform MUST tetap tersimpan bersama kejadian tersebut.
- **FR-091**: Sistem MUST menggolongkan setiap notifikasi sebagai transaksional atau non-transaksional. Notifikasi non-transaksional (pengingat kalender basi, deadline mendekat, dan permintaan rating) MUST tunduk pada preferensi kanal pengguna, sementara notifikasi transaksional MUST TIDAK dapat dimatikan karena menyangkut jalannya pesanan.

**Antarmuka dan Bahasa**

- **FR-055**: Seluruh antarmuka MUST menggunakan bahasa Indonesia dan dapat digunakan sepenuhnya pada layar ponsel.
- **FR-056**: Seluruh alur inti MUST dapat diselesaikan menggunakan keyboard dan terbaca oleh pembaca layar, dengan label yang jelas pada setiap kolom masukan.

### Key Entities

- **Akun Pengguna**: Identitas login satu orang di platform; memuat kredensial, status verifikasi email dan nomor HP, serta satu atau dua peran usaha, atau peran admin. Akun berperan usaha memiliki tepat satu Profil Usaha. Akun berperan admin tidak memiliki Profil Usaha: peran admin bukan peran usaha, sehingga endpoint yang mengembalikan profil usaha menolak akun admin dengan 403, bukan mencari baris yang memang tidak ada.
- **Profil Usaha**: Identitas usaha yang tampil publik: nama usaha, kota/kabupaten (merujuk Wilayah), titik koordinat lokasi usaha, deskripsi, status lencana verifikasi, ringkasan reputasi. Menjadi pemilik Listing Kapasitas dan menjadi pihak dalam Request Kuota dan Pesanan.
- **Wilayah**: Satuan lokasi administratif dua tingkat (provinsi dan kota/kabupaten) beserta kode resminya. Setiap kota/kabupaten termasuk dalam tepat satu provinsi. Menjadi dasar penyaringan dan perluasan pencarian, dengan tingkat ketiga berupa seluruh Indonesia yang tidak memerlukan data tersendiri.
- **Item Daftar Baku**: Satu item pada daftar jenis produk atau daftar jenis mesin, beserta penanda aktif atau nonaktif. Dikelola admin, dirujuk oleh Listing Kapasitas dan oleh filter pencarian, dan tidak dapat diisi bebas oleh pengguna.
- **Usulan Item**: Permintaan pengguna untuk menambah item baru ke daftar baku, beserta keputusan admin dan waktunya.
- **Pengajuan Verifikasi Identitas**: Berkas identitas usaha atau pemilik dan foto lokasi yang diajukan sebuah Profil Usaha, beserta keputusan admin, alasan, dan waktu keputusan. Bersifat rahasia, tidak pernah tampil publik, dan tidak menentukan ketayangan listing.
- **Listing Kapasitas**: Penawaran kapasitas satu Profil Usaha subkontraktor: rujukan ke Item Daftar Baku untuk jenis mesin dan jenis produk, jumlah mesin, satu angka kapasitas mingguan dalam potong, jeda kesiapan mulai dalam hari, waktu pembaruan terakhir, dan status tayang. Memiliki banyak Periode Ketersediaan.
- **Periode Ketersediaan**: Satu periode mingguan pada sebuah Listing Kapasitas, ditandai dengan tanggal Senin awal minggu, beserta kapasitas total periode itu, kapasitas terpakai, dan penanda penuh. Kapasitas terpakai tidak boleh melampaui kapasitas total.
- **Alokasi Kapasitas**: Pemakaian kapasitas sejumlah tertentu oleh satu Pesanan pada satu Periode Ketersediaan. Satu pesanan dapat memiliki beberapa baris alokasi pada periode yang berurutan, dan seluruh barisnya dibalik bersama ketika pesanan dibatalkan.
- **Request Kuota**: Permintaan pekerjaan dari Profil Usaha pemberi order: jenis produk, jumlah, bahan, deadline, catatan, dan batas waktu balasan 72 jam yang ditetapkan sistem. Dikirim ke beberapa kandidat dan memiliki satu status per kandidat.
- **Penawaran**: Balasan seorang subkontraktor atas sebuah Request Kuota: harga dalam rupiah bulat, jeda kesiapan mulai yang dijanjikan, catatan. Dapat memiliki rangkaian counter-offer, dan satu di antaranya dapat diterima menjadi Pesanan.
- **Pesanan**: Kesepakatan yang mengikat dua Profil Usaha: harga, jumlah, spesifikasi, deadline, status berjalan, riwayat perubahan status, serta bila dibatalkan: pihak yang membatalkan, alasan, dan waktunya. Memiliki satu atau beberapa Alokasi Kapasitas dan menjadi dasar satu Ulasan dari setiap pihak.
- **Catatan Pembayaran**: Pernyataan salah satu pihak atas sebuah Pesanan bahwa pembayaran telah dikirim atau diterima, beserta tanggal dan catatan. Tidak mewakili aliran dana di dalam platform.
- **Ulasan**: Rating 1–5 dan teks dari satu pihak atas satu Pesanan yang sudah dikonfirmasi diterima, dengan penanda disembunyikan oleh admin atau tidak.
- **Sengketa**: Laporan salah satu pihak atas sebuah Pesanan, beserta status mediasi, catatan admin, keputusan pengembalian alokasi kapasitas, pihak yang menanggung pembatalan, dan waktu penyelesaian.
- **Notifikasi**: Pesan yang ditujukan ke satu Akun Pengguna atas satu kejadian, beserta kanal pengiriman, jumlah percobaan pengiriman, status pengiriman per kanal, dan status baca.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Subkontraktor baru dapat menyelesaikan pendaftaran sampai listing kapasitasnya tayang dalam waktu di bawah 10 menit tanpa pendampingan dan tanpa menunggu persetujuan admin.
- **SC-002**: Pemberi order dapat menemukan minimal 3 kandidat yang relevan dalam waktu di bawah 3 menit sejak masuk ke platform.
- **SC-003**: Terdapat 200 usaha aktif pada bulan ketiga sejak peluncuran, dengan komposisi tidak lebih tidak seimbang dari 70:30 antara kedua peran.
- **SC-004**: Minimal 60% subkontraktor aktif memperbarui kalender ketersediaannya setidaknya sekali per minggu.
- **SC-005**: Minimal 70% request kuota menerima setidaknya satu balasan dalam 24 jam.
- **SC-006**: Minimal 50% request kuota berujung pada satu kesepakatan.
- **SC-007**: Minimal 80% pesanan yang selesai mendapat rating dari kedua pihak dalam 7 hari.
- **SC-008**: Tidak lebih dari 5% pesanan berujung pada sengketa yang memerlukan mediasi admin.
- **SC-009**: 90% pengajuan verifikasi identitas mendapat keputusan dalam 2 hari kerja.
- **SC-010**: Hasil pencarian tampil dalam 3 detik pada koneksi seluler lambat.
- **SC-011**: Minimal 80% peserta uji coba dari kalangan pemilik konveksi menyelesaikan alur "cari kandidat lalu kirim request kuota" pada percobaan pertama tanpa bantuan.
- **SC-012**: Tidak ada dokumen identitas atau foto lokasi yang dapat diakses oleh pengguna selain pemiliknya dan admin, dibuktikan melalui pengujian akses.
- **SC-013**: Pencarian yang sama dengan data yang sama menghasilkan urutan hasil yang identik pada setiap pengulangan, termasuk ketika hasilnya ditelusuri halaman demi halaman.
- **SC-014**: Tidak lebih dari 10% pesanan yang pernah disepakati berakhir dibatalkan sebelum produksi.
- **SC-015**: Tidak lebih dari 15% pesanan tertutup melalui konfirmasi otomatis alih-alih konfirmasi manual pemberi order.
- **SC-016**: Minimal 90% subkontraktor menemukan jenis produk dan jenis mesin yang mereka kerjakan di dalam daftar baku tanpa perlu mengusulkan item baru.
- **SC-017**: Setiap usulan item daftar baku mendapat keputusan admin dalam 2 hari kerja.
- **SC-018**: Kapasitas terpakai sebuah periode tidak pernah melampaui kapasitas totalnya, dibuktikan melalui pengujian dua kesepakatan yang terjadi berbarengan atas periode yang sama.
- **SC-019**: Pesanan yang jumlahnya melampaui kapasitas satu minggu tetap dapat disepakati selama total kapasitas sampai deadline mencukupi, dibuktikan dengan skenario 3.000 potong pada kapasitas 500 potong per minggu dengan deadline delapan minggu.
- **SC-020**: Kapasitas kandidat yang jeda kesiapan mulainya menggeser produksi ke minggu-minggu berikutnya hanya dijumlahkan mulai dari minggu kesiapan mulai, dibuktikan dengan kandidat berjeda 14 hari yang dua minggu pertamanya tidak ikut dihitung sehingga total kapasitasnya lebih kecil dari kandidat berjeda 0 hari pada deadline yang sama.
- **SC-021**: Pencarian dengan deadline yang melampaui horizon kalender yang sudah dibuat tetap menilai kandidat berdasarkan kapasitas penuh sampai deadline, dan periode yang kurang benar-benar terbentuk hanya ketika sebuah kesepakatan atas kandidat itu terbentuk, dibuktikan dengan pencarian berdeadline lima bulan pada listing yang horizon awalnya baru 3 bulan.

## Assumptions

Seluruh butir di bawah ini menggantikan konteks tambahan yang belum diisi pemilik project. Koreksi yang mana pun, dan spec ini akan disesuaikan.

- **Tujuan pembuatan**: keperluan lomba, bukan sistem produksi. Ini alasan langsung di balik keputusan tanpa escrow dan tanpa gerbang verifikasi, dan menjadi konteks bagi seluruh trade-off scope di bawah.
- **Pengguna**: publik, bukan internal. Tiga kelompok: pemilik konveksi subkontraktor, pemilik brand/UMKM pemberi order, dan tim ops internal sebagai admin.
- **Target pasar dan data demo adalah dua hal berbeda.** SC-003 menargetkan 200 usaha aktif pada bulan ketiga sebagai sasaran bisnis, mengikuti estimasi pasar dokumen sumber yang memproyeksikan sekitar 1.500 UMKM aktif pada tahun ketiga [1]. Data demo untuk lomba diasumsikan sekitar 50 usaha, dan angka itu tidak menggantikan SC-003.
- **Project**: dibangun baru dari nol, tanpa komponen yang dipakai ulang.
- **Tests**: pengujian otomatis diwajibkan pada tingkat fitur dan endpoint, dan pengujian menyeluruh dijalankan secara manual oleh penguji di luar tim mengikuti skenario yang tim sediakan. Alur bertenggat karenanya harus dapat diperiksa tanpa menunggu waktu nyata.
- **Satu akun boleh memegang dua peran**, dan antarmukanya sama untuk kedua peran tanpa pemilih konteks. Konsekuensinya, listing sendiri harus dikecualikan dari pencarian dan request kuota ke diri sendiri harus ditolak (FR-081, FR-083).
- **Periode kapasitas berbasis mingguan**, dimulai hari Senin, dihitung pada zona waktu Asia/Jakarta, dan disimpan sebagai tanggal awal minggu tanpa zona waktu. Waktu kejadian disimpan berzona dan dikonversi hanya saat ditampilkan.
- **Satu satuan kapasitas untuk seluruh listing.** Konsekuensi yang diterima: subkontraktor yang produktivitasnya berbeda jauh antar jenis produk menyatakan satu angka rata-rata, dan penyesuaiannya terjadi saat ia menawar harga dan menyanggupi pekerjaan. Alternatif kapasitas per jenis produk ditolak karena mesin dan tenaga kerjanya berbagi, sehingga angka terpisah akan mengizinkan penyanggupan ganda pada minggu yang sama.
- **Jeda kesiapan mulai dan durasi penyelesaian dipisahkan.** Konsekuensi yang diterima: istilah "lead time" pada dokumen sumber ditafsirkan sebagai jeda kesiapan mulai, karena durasi penyelesaian sudah dapat dihitung dari kapasitas dan jumlah pesanan sehingga tidak dapat menjadi atribut tetap listing.
- **Alokasi mengisi periode paling awal lebih dulu.** Konsekuensi yang diterima: pilihan ini lebih mudah dijelaskan kepada pengguna dan sesuai kebiasaan mengerjakan pesanan sesegera mungkin, tetapi tidak menyisakan ruang bagi subkontraktor untuk menahan minggu terdekat bagi pesanan mendesak yang mungkin datang kemudian.
- **Data wilayah memakai pembagian administratif resmi** yang diambil sekali dari sumber data publik, disimpan di basis data sendiri, dan disalin ke dalam repository sebagai cadangan. Kecamatan dan desa tidak diambil karena tidak ada requirement yang memakainya.
- **Perluasan wilayah tiga tingkat tanpa pengelompokan buatan.** Konsekuensi yang diterima: pencarian di sebuah kota yang secara praktik satu klaster dengan kota di provinsi lain (misalnya Jakarta dengan Bekasi dan Tangerang) baru menjangkau tetangganya pada perluasan tingkat terakhir. Pengelompokan wilayah manual ditolak karena menuntut data yang harus dirawat sendiri dan memperlambat penyiapan.
- **Nilai uang berupa bilangan bulat rupiah.** Tidak ada pecahan, karena harga subkontrak konveksi dalam praktik tidak memakai satuan di bawah rupiah.
- **Cakupan wilayah awal**: lima kota besar yang disebut dokumen sumber sebagai basis segmen pasar aktif [1]; platform tidak dibatasi teknis pada kota-kota itu.
- **Komisi platform 5–10% per transaksi** adalah model pendapatan yang diasumsikan di dokumen sumber; karena platform tidak menyentuh dana, penagihan dan pembukuan komisi tidak tercakup dalam versi ini.
- **Di luar scope versi ini**, mengikuti bagian Won't-have dan Non-Goals dokumen sumber: marketplace bahan baku, pembiayaan atau pinjaman UMKM, penjualan ke konsumen akhir, manufaktur presisi tinggi, penanganan sengketa secara legal formal, verifikasi kualitas produk secara fisik, dukungan multi-bahasa, dan aplikasi ponsel native.
- **Escrow dan pemrosesan pembayaran ditunda seluruhnya.** Konsekuensi yang diterima secara sadar: mitigasi utama risiko gagal bayar dan penipuan pada risk register dokumen sumber, yang menempatkan escrow wajib sebagai penangkalnya [1], tidak berlaku di versi ini. Tidak ada mekanisme yang menahan transaksi pindah ke luar platform selain pencatatan dan reputasi.
- **Verifikasi identitas bukan gerbang.** Konsekuensi yang diterima: pasokan kapasitas terisi lebih cepat, tetapi hasil pencarian dapat memuat usaha yang belum diperiksa, dan lencana menjadi satu-satunya pembeda.
- **Skor kecocokan tidak memuat faktor perilaku.** Konsekuensi yang diterima: mitigasi berupa penalti penurunan skor pencarian bagi subkontraktor yang tidak memperbarui kalender kapasitas [1] tidak dipakai; penegakannya bergantung pada pengingat dan penanda "Data Belum Diperbarui" saja.
- **Pembatalan pra-produksi tidak dikenai denda apa pun** selain turunnya tingkat penyelesaian pihak yang membatalkan, karena platform tidak memproses uang.
- **Ambang 3 pesanan** sebelum tingkat penyelesaian tampil sebagai persentase adalah angka pilihan penulis spec; belum divalidasi ke pengguna.
- **Tenggat 7 hari** untuk konfirmasi otomatis dan **72 jam** untuk batas waktu balasan request adalah angka pilihan penulis spec; keduanya diasumsikan cukup untuk pengiriman dan kebiasaan berbalas antar kota di Indonesia.
- **Ditunda dari MVP** meskipun disebut di dokumen sumber: chat di dalam platform, integrasi ekspedisi, dan analitik kapasitas. Komunikasi rinci antar pihak sementara terjadi di luar platform.