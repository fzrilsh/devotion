# Devotion Constitution

**Project**: Devotion (*devotion*, kesetiaan). Platform Capacity Exchange:
marketplace subkontrak B2B yang mempertemukan UMKM konveksi dengan kapasitas
produksi menganggur dan UMKM yang order-nya melebihi kapasitas sendiri [1].

**Konteks**: submission lomba, bukan sistem produksi. Penilaian bertumpu pada
demo yang berjalan dan kualitas dokumen, di bawah tenggat ketat. Setiap
keputusan dalam dokumen ini tunduk pada kenyataan itu.

## Core Principles

### I. Monolith-First (NON-NEGOTIABLE)

Satu repository, satu aplikasi yang dijalankan. Modularitas dicapai lewat batas
modul di dalam codebase, bukan lewat batas jaringan.

Aturan yang mengikat:

- Backend dan frontend berada dalam satu repository yang sama.
- Frontend MUST disajikan oleh proses backend yang sama, bukan oleh proses
  penyaji tersendiri. Hasil build frontend disematkan ke dalam biner atau
  dilayani dari satu direktori oleh backend.
- Proses runtime yang diizinkan hanya dua: satu proses aplikasi backend (yang
  sekaligus menyajikan frontend dan menghabiskan TLS) dan satu basis data.
- Dilarang: memisah service, message broker, worker atau queue sebagai proses
  terpisah, cron service terpisah, cache server terpisah, reverse proxy sebagai
  proses tersendiri, dan penambahan proses runtime lain dalam bentuk apa pun.
- Pekerjaan terjadwal (kalender basi, request kedaluwarsa, pesanan lewat
  deadline, konfirmasi otomatis) MUST diselesaikan dengan salah satu dari dua
  cara: dihitung saat data dibaca, atau penjadwal di dalam proses backend yang
  sama. Bukan dengan proses kedua.
- Pengiriman notifikasi MUST berjalan di dalam proses backend yang sama.
  Kegagalan pengiriman MUST TIDAK menggagalkan transaksi yang memicunya.
- Komunikasi antar modul MUST berupa pemanggilan fungsi langsung di dalam satu
  proses, bukan pemanggilan jaringan ke diri sendiri.
- Perkakas yang hanya berjalan saat pengembangan (test runner, linter,
  generator kode, generator migrasi) bukan proses runtime dan tidak melanggar
  prinsip ini. Batasnya: perkakas itu MUST TIDAK menambah layanan pada
  `docker-compose.yml` dan MUST TIDAK diperlukan agar aplikasi berjalan.
- Perintah sekali jalan (pembuatan admin, seed data) bukan proses runtime.
  Dijalankan lewat subcommand pada biner yang sama, lalu berhenti.
- Pengujian MUST berjalan terhadap layanan basis data yang sama dengan nama
  basis data atau skema terpisah. Dilarang menambah layanan basis data khusus
  pengujian.
- Layanan di luar repository yang tidak dijalankan di server sendiri (DNS dan
  proxy tepi, penyedia email, pemantau uptime, pelacak error) bukan proses
  runtime dan tidak dihitung dalam batas ini. Setiap layanan semacam itu MUST
  dicatat di `docs/` beserta akibatnya bila layanan itu mati.

Cara memeriksa: hitung entri layanan pada `docker-compose.yml`. Lebih dari dua,
atau ada layanan di luar backend dan basis data, berarti pelanggaran.

### II. Demo-Ready Over Complete

Setiap user story MUST dapat didemokan dari awal sampai akhir melalui antarmuka,
tanpa menyentuh basis data secara manual dan tanpa penjelasan lisan untuk
menutup bagian yang belum jadi.

Aturan yang mengikat:

- Sebuah user story dinyatakan selesai hanya jika seluruh Acceptance Scenario-nya
  dapat dijalankan lewat antarmuka.
- Fitur yang tidak terlihat dalam demo MUST diturunkan prioritasnya di bawah
  fitur yang terlihat, meskipun secara teknis lebih menarik.
- Setiap user story MUST memiliki data contoh yang cukup untuk mendemokan
  keadaan berhasil sekaligus setidaknya satu keadaan gagal atau kosong.
- Daftar baku jenis produk, daftar jenis mesin, dan data wilayah MUST sudah
  terisi sebelum demo dimulai, dan pengisiannya MUST dapat dilakukan dengan satu
  perintah. Antarmuka pencocokan tidak dapat didemokan tanpa ketiganya.
- Demo MUST TIDAK bergantung pada layanan luar yang dapat mati tanpa
  pemberitahuan. Setiap kanal luar MUST punya jalur pengamatan pengganti di
  dalam platform.
- Jalan pintas demi demo diizinkan, tetapi MUST dicatat sebagai utang di
  `docs/` beserta akibatnya.

### III. Traceability to Spec

Setiap satuan pekerjaan MUST menunjuk ke nomor requirement atau user story di
`001-capacity-exchange-marketplace/spec.md`.

Aturan yang mengikat:

- Setiap task di `tasks.md` MUST mencantumkan FR atau user story yang
  dilayaninya.
- Setiap pengujian otomatis MUST menyebutkan FR atau Acceptance Scenario yang
  diverifikasinya, pada nama pengujian atau komentar di atasnya. Pengujian yang
  tidak dapat ditelusuri ke spec dianggap tidak menambah nilai.
- Setiap skenario untuk penguji eksternal MUST menunjuk Acceptance Scenario yang
  diperiksanya.
- Kode yang tidak dapat ditelusuri ke requirement mana pun MUST dihapus atau
  requirement-nya ditambahkan lebih dulu ke spec.
- Perubahan perilaku MUST diubah di spec sebelum diubah di kode. Spec adalah
  sumber kebenaran, bukan catatan setelah kejadian.
- Bila kode menyimpang dari spec karena tenggat, penyimpangan itu MUST dicatat
  di Complexity Tracking pada `plan.md`.

### IV. Minimal Dependencies

Setiap dependency baru MUST dibenarkan. Default-nya adalah library standar
bahasa dan framework yang sudah dipilih.

Aturan yang mengikat:

- Sebuah dependency hanya boleh ditambahkan jika ia menghemat pekerjaan yang
  nyata, dan alasannya MUST dicatat di `docs/`.
- Dilarang menambah dependency untuk hal yang dapat diselesaikan dengan puluhan
  baris kode sendiri.
- Versi MUST dipatok tepat, bukan rentang terbuka.
- Dilarang menambah dependency yang menuntut proses runtime tambahan. Prinsip I
  menang.
- Untuk setiap kemampuan, hanya satu dependency. Tidak ada dua library untuk
  urusan yang sama. Termasuk pengujian: satu kerangka pengujian untuk backend,
  satu untuk frontend, tidak lebih.
- Dependency khusus pengembangan dan pengujian tunduk pada aturan yang sama,
  tetapi dinilai lebih longgar karena tidak menambah beban pada aplikasi yang
  dijalankan.

### V. Deterministic Behavior Over Cleverness

Perilaku yang dapat diulang mengalahkan perilaku yang terasa pintar.

Aturan yang mengikat:

- Pencarian yang sama atas data yang sama MUST menghasilkan urutan hasil yang
  identik pada setiap pengulangan, termasuk antar halaman ketika hasil
  dipaginasi.
- Skor kecocokan MUST hanya dihitung dari kriteria keras yang ditetapkan spec.
  Dilarang memasukkan pembobotan, pembelajaran mesin, personalisasi, maupun
  pengacakan.
- Setiap keputusan yang ditampilkan ke pengguna (urutan hasil, penolakan
  penawaran, angka tingkat penyelesaian) MUST dapat dijelaskan kepada pengguna
  dalam satu kalimat.
- Hitungan yang bergantung waktu MUST memakai satu sumber waktu yang sama, dan
  sumber waktu itu MUST dapat digantikan saat pengujian. Tidak ada pemanggilan
  waktu sistem yang tersebar di dalam logika bisnis.
- Batas periode mingguan MUST dihitung pada zona waktu Asia/Jakarta dengan hari
  Senin sebagai awal minggu, dan MUST disimpan sebagai tanggal tanpa zona waktu.
  Waktu kejadian disimpan sebagai waktu berzona dan hanya dikonversi saat
  ditampilkan.
- Nilai uang MUST berupa bilangan bulat rupiah. Dilarang memakai bilangan
  pecahan untuk uang.
- Data acuan yang berasal dari layanan luar MUST diambil sekali lalu disimpan di
  basis data sendiri, dan MUST punya salinan di dalam repository sebagai
  cadangan. Dilarang memanggil layanan luar itu saat permintaan pengguna
  dilayani.

## Batasan Tambahan

### Struktur Repository (mengikat)

```text
devotion/
├── README.md            # template dari panitia, tidak diubah strukturnya
├── LICENSE              # MIT
├── docker-compose.yml   # maksimal 2 layanan: backend, basis data
├── backend/
├── frontend/
└── docs/
```

Berkas pengujian berada di dalam `backend/` dan `frontend/`, mengikuti kebiasaan
kerangka kerja masing-masing. Dilarang membuat direktori pengujian di tingkat
atas. Menambah direktori tingkat atas di luar daftar ini MUST dicatat sebagai
pelanggaran di Complexity Tracking.

### Batas Keuangan (mengikat)

Platform MUST TIDAK menahan, menyalurkan, maupun memproses dana pihak mana pun.
Dilarang memasang integrasi payment gateway, escrow, maupun dompet internal.
Pembayaran terjadi langsung antar pihak; platform hanya mencatat pernyataan
mereka. Batas ini berasal dari FR-040 dan tidak boleh dilewati tanpa mengubah
spec lebih dulu.

### Keamanan

- Dokumen identitas dan foto lokasi usaha MUST hanya dapat diakses pemiliknya
  dan admin. Ini MUST dibuktikan dengan pengujian akses otomatis, bukan
  diasumsikan.
- Berkas unggahan MUST TIDAK dilayani lewat path statis. Setiap permintaan MUST
  melewati pemeriksaan peran lebih dulu.
- Nama berkas yang disimpan MUST dibuat sistem, bukan diambil dari nama yang
  dikirim pengguna. Ukuran dan tipe berkas MUST divalidasi dari isinya.
- Metadata lokasi pada gambar yang diunggah MUST dibuang.
- Setiap endpoint MUST memeriksa peran pemanggil. Endpoint tanpa pemeriksaan
  peran dianggap cacat, bukan sekadar belum lengkap.
- Origin MUST menolak koneksi yang tidak datang lewat proxy tepi, dan koneksi
  antara proxy tepi dan origin MUST terenkripsi. Konfigurasi yang membuat
  segmen itu berjalan tanpa enkripsi dilarang.
- Alamat asal permintaan yang dikirim proxy tepi MUST hanya dipercaya bila
  koneksinya memang berasal dari rentang alamat proxy tersebut.
- Pembatasan laju yang bergantung pada data domain (percobaan masuk per akun,
  pengiriman ulang kode sekali pakai per nomor, dan pengiriman request kuota per
  pengguna) MUST ditegakkan di dalam aplikasi, tidak diserahkan ke proxy tepi.
- Kredensial MUST berada di variabel lingkungan. Dilarang menuliskannya di
  dalam kode maupun meng-commit berkas `.env`. Nomor telepon layanan, kunci API,
  dan kredensial basis data MUST TIDAK muncul di repository, di dokumentasi,
  maupun di artefak perencanaan.
- Kata sandi MUST disimpan dalam bentuk hash dengan algoritma yang memang
  ditujukan untuk kata sandi.
- Data yang dikirim ke pelacak error MUST dibersihkan dari kata sandi, token,
  nomor telepon, dan apa pun yang menyangkut dokumen identitas.
- Data uji MUST TIDAK memuat data pribadi orang sungguhan, termasuk nomor
  telepon dan dokumen identitas milik anggota tim maupun penguji eksternal.

### Standar Performa dan Batas Sumber Daya

- Hasil pencarian tampil dalam 3 detik pada koneksi seluler lambat.
- Seluruh antarmuka MUST dapat digunakan sepenuhnya pada layar ponsel dan
  berbahasa Indonesia.
- Membangun frontend maupun backend MUST TIDAK dilakukan di server. Artefak
  dibangun di luar server, dan server hanya menarik dan menjalankannya.
- Ukuran log kontainer MUST dibatasi. Log yang tumbuh tanpa batas dapat mengisi
  penyimpanan dan menghentikan basis data.
- Total penyimpanan berkas unggahan MUST dibatasi, dan unggahan yang melampaui
  batas MUST ditolak dengan pesan yang jelas.
- Jumlah koneksi basis data MUST disesuaikan dengan memori server, tidak
  dibiarkan pada nilai bawaan.
- Cadangan basis data MUST dibuat terjadwal dan salinannya MUST disimpan di luar
  server. Jumlah salinan yang disimpan MUST dibatasi.
- Optimasi di luar hal-hal di atas MUST ditunda sampai ada bukti masalah.

## Alur Kerja dan Gerbang Mutu

### Urutan Pengerjaan

1. Fase Setup dan Foundational diselesaikan lebih dulu, termasuk penyiapan
   server, pemasangan kerangka pengujian, dan pengisian data acuan.
2. User story dikerjakan menurut prioritas, satu per satu sampai dapat
   didemokan dan pengujiannya lulus.
3. Berhenti di setiap checkpoint dan buktikan story itu berjalan sendiri
   sebelum lanjut.

### Pengujian (mengikat)

Pengujian otomatis diwajibkan. Cakupannya ditetapkan sebagai kewajiban yang
dapat diperiksa, bukan sebagai angka persentase, karena persentase mudah
dipenuhi tanpa menambah keyakinan.

Kewajiban minimum:

- Setiap endpoint MUST memiliki sekurang-kurangnya dua pengujian: satu jalur
  berhasil, dan satu penolakan karena peran pemanggil tidak berwenang.
- Setiap endpoint yang menerima masukan pengguna MUST memiliki
  sekurang-kurangnya satu pengujian penolakan masukan yang tidak sah.
- Setiap FR yang perilakunya dapat diamati dari luar MUST tercakup oleh
  sekurang-kurangnya satu pengujian.
- Aturan yang paling mudah rusak diam-diam MUST diuji secara khusus: urutan
  hasil pencarian dan sifat dapat-diulangnya termasuk antar halaman, larangan
  faktor selain kriteria keras pada skor, penjumlahan kapasitas lintas periode
  beserta cara alokasinya, pengembalian seluruh alokasi saat pembatalan
  pra-produksi, larangan pembatalan sendiri setelah produksi, larangan mengirim
  request kuota ke listing milik sendiri, konfirmasi otomatis beserta
  penghentiannya oleh sengketa, rumus tingkat penyelesaian beserta pembebanan
  pada pihak yang membatalkan, perluasan wilayah satu tingkat, dan pembatasan
  akses dokumen identitas.
- Alokasi kapasitas MUST diuji terhadap dua kesepakatan yang terjadi berbarengan
  atas periode yang sama, dan basis data MUST menolak kapasitas terpakai yang
  melampaui kapasitas total meskipun logika aplikasi bocor.
- Pengujian yang menyangkut tenggat waktu MUST memakai sumber waktu yang
  digantikan, bukan menunggu waktu nyata.
- Seluruh pengujian MUST dapat dijalankan dengan satu perintah yang
  didokumentasikan di `docs/`.

Yang tidak diwajibkan: pengujian atas kode kerangka kerja, pengujian tampilan
yang hanya meneruskan data tanpa logika, dan angka cakupan tertentu.

### Pengujian End-to-End oleh Penguji Eksternal (mengikat)

Pengujian menyeluruh dijalankan secara manual oleh orang di luar tim, mengikuti
skenario yang tim sediakan. Karena mereka tidak memiliki akses ke basis data
maupun kode, sistem MUST menyediakan segalanya yang mereka butuhkan melalui
antarmuka.

Kewajiban penyiapan:

- MUST tersedia akun uji yang sudah disiapkan untuk kedua peran usaha dan untuk
  admin, beserta kredensialnya di dalam `docs/`.
- MUST tersedia dokumen skenario yang ditulis untuk orang yang belum pernah
  melihat sistem ini: setiap langkah menyebutkan akun yang dipakai, tindakan
  yang dilakukan, dan hasil yang diharapkan, dalam bahasa sehari-hari tanpa
  istilah internal.
- Setiap skenario MUST memuat kolom untuk penguji menuliskan apa yang benar-benar
  terjadi, bukan hanya penanda lulus atau gagal.
- Alur bertenggat MUST dapat diperiksa tanpa menunggu waktu nyata, dengan cara
  menyiapkan data yang sudah berada pada keadaan itu: pesanan yang tenggat
  konfirmasinya mendekat, pesanan yang tenggatnya sudah terlampaui, listing yang
  kalendernya sudah basi, dan request yang sudah kedaluwarsa.
- Notifikasi MUST dapat diamati penguji tanpa perangkat atau nomor sungguhan.
  Notifikasi di dalam platform menjadi jalur pengamatan utamanya, dan kanal luar
  yang tidak dapat mereka verifikasi MUST dinyatakan di luar cakupan pengujian
  mereka di dalam dokumen skenario.
- Data uji MUST dapat dipulihkan ke keadaan awal dengan satu perintah, agar
  penguji berikutnya tidak mewarisi kekacauan dari penguji sebelumnya.
- Label, pesan kesalahan, dan judul halaman MUST jelas dan berbahasa Indonesia,
  sehingga penguji dapat mengutipnya dalam laporan.
- Penanda elemen antarmuka yang stabil dianjurkan tetapi tidak diwajibkan,
  karena pengujian dijalankan manusia. Bila pengujian otomatis end-to-end
  kemudian diminta, penanda ini menjadi wajib.
- Temuan penguji MUST dicatat di `docs/` beserta keputusan: diperbaiki, atau
  diterima sebagai utang dengan alasannya.

### Gerbang Sebelum Sebuah Story Dinyatakan Selesai

- Seluruh Acceptance Scenario story itu dapat dijalankan lewat antarmuka.
- Pengujian otomatis untuk story itu ada, menunjuk FR yang diuji, dan lulus.
- Seluruh pengujian yang sudah ada sebelumnya tetap lulus.
- Skenario uji manual untuk story itu sudah tertulis, beserta data yang
  dibutuhkannya.
- Jumlah layanan pada `docker-compose.yml` tetap dua.
- Dependency baru, bila ada, sudah dicatat alasannya.
- Pemeriksaan peran terpasang pada endpoint yang ditambahkan, dan ada pengujian
  yang membuktikannya menolak peran yang tidak berwenang.

### Dokumentasi

`docs/` MUST memuat: cara menyiapkan server dari keadaan kosong, cara
menjalankan sistem, cara menjalankan pengujian, daftar skenario uji manual
beserta kredensial akun uji, catatan temuan penguji dan keputusannya, daftar
layanan luar beserta akibat bila mati, daftar utang teknis beserta akibatnya,
dan alasan setiap dependency. Dokumen lain hanya ditambahkan bila memang
dipakai.

## Governance

Constitution ini mengalahkan preferensi teknis apa pun, termasuk preferensi
penulis kode dan kebiasaan yang dianggap praktik terbaik di luar konteks ini.

- Setiap pelanggaran MUST dicatat di tabel Complexity Tracking pada `plan.md`,
  memuat pelanggarannya, alasan kebutuhannya, dan alternatif lebih sederhana
  yang ditolak beserta sebabnya.
- Pelanggaran yang tidak tercatat berarti pekerjaan itu belum selesai.
- Prinsip I tidak dapat dikesampingkan dengan alasan apa pun karena berasal
  dari aturan panitia. Pelanggaran atasnya menggugurkan submission, bukan
  sekadar menambah utang.
- Prinsip II sampai V dapat dikesampingkan per kasus, asalkan tercatat.
- Kewajiban pengujian dapat dikesampingkan hanya untuk satu story tertentu dan
  hanya karena tenggat, dengan catatan di Complexity Tracking yang menyebutkan
  story mana dan risiko apa yang ditanggung. Pengecualian menyeluruh tidak
  diizinkan.
- Perubahan constitution MUST menaikkan nomor versi: angka pertama bila ada
  prinsip atau gerbang mutu yang dihapus atau dibalik maknanya, angka kedua bila
  ada prinsip, batasan, atau gerbang baru, angka ketiga untuk perbaikan
  redaksional.
- Saat terjadi pertentangan antara constitution dan spec, constitution menang
  pada hal teknis, spec menang pada hal perilaku produk. Pertentangan yang
  tidak dapat diselesaikan dengan aturan itu MUST diangkat ke pemilik project.

## Riwayat Amandemen

- **2.1.0 (2026-08-21)**: Frontend wajib disajikan oleh proses backend yang
  sama; batas layanan runtime turun dari tiga menjadi dua (backend dan basis
  data), karena proxy tepi tidak lagi berjalan di server sendiri. Penanda elemen
  antarmuka yang stabil turun dari wajib menjadi anjuran karena pengujian
  end-to-end dijalankan manusia, digantikan kewajiban dokumen skenario yang
  ditulis untuk orang luar beserta kolom temuan. Ditambahkan: aturan layanan
  luar tidak dihitung sebagai proses runtime tetapi wajib dicatat; kewajiban
  keamanan berkas unggahan, enkripsi segmen origin, kepercayaan alamat asal, dan
  pembatasan laju berbasis data domain; batas sumber daya untuk membangun di
  luar server, ukuran log, kuota unggahan, koneksi basis data, dan cadangan;
  aturan determinisme untuk batas minggu, nilai uang, paginasi, dan data acuan
  dari layanan luar; kewajiban pengujian atas kapasitas lintas periode dan
  balapan alokasi. Larangan mencantumkan nomor telepon layanan dan kredensial di
  repository maupun artefak perencanaan dinyatakan eksplisit.
- **2.0.0 (2026-08-21)**: Pengujian otomatis berubah dari tidak diwajibkan
  menjadi diwajibkan, dengan cakupan minimum yang dapat diperiksa. Ditambahkan
  gerbang mutu baru untuk pengujian end-to-end oleh penguji eksternal. Prinsip I
  diperjelas bahwa perkakas pengembangan bukan proses runtime dan pengujian
  tidak boleh menambah layanan basis data. Prinsip III mewajibkan setiap
  pengujian menunjuk FR. Prinsip V memperketat sumber waktu menjadi wajib dapat
  digantikan saat pengujian. Prinsip IV mengatur dependency pengujian. Aturan
  penomoran versi diperluas agar mencakup gerbang mutu, bukan hanya prinsip.
- **1.0.0 (2026-08-21)**: Versi pertama.

**Version**: 2.1.0 | **Ratified**: 2026-08-21 | **Last Amended**: 2026-08-21