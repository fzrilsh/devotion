<div align="center">

<img src="https://gist.github.com/user-attachments/assets/69a93922-e7fe-4724-a4c7-05d5bc26ac35" alt="Devotion Logo" width="400" style="margin-bottom: 8px; height: auto;">

# Devotion
### Marketplace kapasitas produksi untuk UMKM konveksi

**Delegate Your Overload Production**

<br>

[![Live Demo](https://img.shields.io/badge/Live_Demo-Visit_Site-success?style=for-the-badge)](https://devotion.web.id/)
[![GitHub](https://img.shields.io/badge/GitHub-Repository-181717?style=for-the-badge&logo=github)](https://github.com/fzrilsh/devotion)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

<br><br>

**Submission for ITECHNO CUP 2026 — Web Development**

**By Indonesia Emas 74 Kg**

</div>

---

## Daftar isi

- [Tentang proyek](#tentang-proyek)
- [Fitur unggulan](#fitur-unggulan)
- [Demo dan screenshot](#demo-dan-screenshot)
- [Teknologi](#teknologi)
- [Arsitektur sistem](#arsitektur-sistem)
- [Instalasi dan setup](#instalasi-dan-setup)
- [Penggunaan](#penggunaan)
- [Dokumentasi API](#dokumentasi-api)
- [Testing](#testing)
- [Tim developer](#tim-developer)
- [Lisensi](#lisensi)

---

## Tim developer

| Nama | Peran | GitHub |
|---|---|---|
| **Juan Kevin Utomo** | Project lead | [@TrygerZ](https://github.com/TrygerZ) |
| **Fazril Syaveral Hillaby** | Backend developer | [@fzrilsh](https://github.com/fzrilsh) |
| **Chiko Maulana Ahmad** | Frontend developer | [@ChikoID](https://github.com/ChikoID) |

---

## Tentang proyek

### Latar belakang

Coba bayangkan pemilik konveksi kebanjiran order 5.000 potong, sementara kapasitasnya cuma 2.000. Yang biasanya terjadi: dia menelepon satu per satu kenalan yang nomornya masih tersimpan. Kalau semuanya penuh, sisa 3.000 potong itu dilepas, padahal di kota sebelah ada konveksi yang mesinnya menganggur minggu itu.

Dua masalah bertemu di titik yang sama. Yang kelebihan order tidak tahu siapa yang sedang kosong, dan yang sedang kosong tidak punya cara memberi tahu siapa pun. Selama pencarian mitra hanya lewat relasi pribadi, jangkauannya berhenti di batas daftar kontak.

Masalahnya bertambah karena kapasitas berubah cepat. Mitra yang minggu lalu longgar bisa penuh hari ini. Tanpa kalender ketersediaan yang bisa dilihat calon pemberi order, telepon demi telepon berakhir dengan jawaban "maaf, sedang penuh", dan waktu kedua pihak habis begitu saja.

### Solusi yang ditawarkan

Devotion mempertemukan dua sisi yang selama ini tidak saling melihat:

- **Subkontraktor**, UMKM konveksi yang punya kapasitas produksi menganggur.
- **Pemberi order**, UMKM atau brand yang ordernya melebihi kapasitas sendiri.

Alurnya sederhana. Subkontraktor memasang profil usaha, mencantumkan produk dan mesin yang dikuasai, lalu mengisi kapasitas per minggu di kalender ketersediaan. Pemberi order mengisi kriteria yang dia butuhkan, produk apa, mesin apa, berapa banyak, kapan tenggatnya, sejauh mana wilayah yang masih masuk hitungan. Dari hasil pencarian dia bisa memilih beberapa kandidat sekaligus dan mengirim satu request kuota ke semuanya. Penawaran yang diterima langsung berubah jadi work order, dan kapasitas yang dipakai tercatat per minggu.

Yang membedakan Devotion dari daftar kontak biasa adalah cara hasil pencarian disusun. Setiap kandidat dinilai dengan empat kriteria keras dan skor 0 sampai 4: produk cocok atau tidak, mesin cocok atau tidak, jeda kesiapan masih terjangkau atau tidak, dan kapasitas kalau dijumlah sampai tenggat cukup atau tidak. Reputasi, lencana verifikasi, kebaruan kalender, dan jarak sengaja tidak dimasukkan ke skor. Konsekuensinya penting: urutan yang muncul selalu bisa dijelaskan alasannya, dan pencarian yang sama menghasilkan urutan yang sama. Pengguna melihat kriteria mana yang tidak terpenuhi, bukan sekadar angka yang muncul entah dari mana.

### Tujuan proyek

- **Tujuan utama**: membuat kapasitas produksi yang menganggur bisa ditemukan lewat proses yang terbuka, bukan lewat kenalan.
- **Target pengguna**: pemilik konveksi yang mau menerima subkontrak, pemilik brand atau UMKM yang perlu melimpahkan order, dan admin yang menjaga operasional platform.
- **Yang didapat pengguna**: satu tempat untuk memasang, mencari, meminta, dan memantau kapasitas, dengan data yang sama dilihat kedua pihak. Tidak ada lagi versi kebenaran yang berbeda antara pemberi order dan subkontraktor.
- **Kaitan tema**: kalender kapasitas membuat informasi produksi ikut bergerak setiap kali ketersediaan berubah, dan di situlah sifat adaptifnya. Devotion menyentuh SDG 8 lewat akses order dan pemanfaatan kapasitas kerja yang tadinya terbuang, serta SDG 9 lewat digitalisasi proses matching antar pelaku industri kecil.

### Batasan produk

Ada dua hal yang sengaja tidak dikerjakan Devotion, dan keduanya keputusan sadar.

Devotion tidak menyentuh uang siapa pun. Tidak menahan, tidak menyalurkan, tidak memproses. Pembayaran terjadi langsung antar pihak, dan platform hanya mencatat pernyataan keduanya bahwa pembayaran sudah dikirim atau diterima. Tidak ada kolom jumlah uang sama sekali di catatan itu, karena begitu ada, platform mulai terlihat seperti perantara dana, dan itu bukan yang ingin dibangun di sini.

Antarmuka seluruhnya bahasa Indonesia, tanpa lapisan i18n. Sasarannya UMKM konveksi domestik, jadi menambah mekanisme multi-bahasa hanya akan menambah kerumitan tanpa ada yang memakainya.

---

## Fitur unggulan

### Fitur utama

Sembilan hal di bawah ini yang membentuk alur utama Devotion, dari mendaftar sampai pesanan selesai dan diulas.

| Fitur | Deskripsi | Kenapa ini penting |
|---|---|---|
| **Profil dan autentikasi usaha** | Registrasi dengan pilihan peran, login, sesi berbasis cookie `httpOnly`, verifikasi email dan nomor HP, pemulihan kata sandi, profil usaha dengan titik lokasi di peta. | Sebelum ada uang dan tenggat yang dipertaruhkan, kedua pihak sudah tahu sedang berhadapan dengan siapa. |
| **Listing kapasitas dan kalender mingguan** | Subkontraktor mengatur kapasitas mingguan, jeda kesiapan, jenis produk, jenis mesin, visibilitas listing, dan periode ketersediaan 12 minggu ke depan. Minggu yang sudah punya alokasi terkunci dari perubahan. | Kapasitas kosong akhirnya terlihat tanpa perlu dikenal lebih dulu. Kunci pada minggu yang sudah terpakai mencegah kalender berubah di belakang pesanan yang sedang berjalan. |
| **Pencarian dan skor kecocokan** | Filter produk, mesin, jumlah, tenggat, jeda maksimum, dan cakupan kota, provinsi, atau nasional. Skor 0 sampai 4 dari kriteria keras, kapasitas dijumlah lintas periode sampai tenggat, paginasi kursor dengan urutan deterministik. | Pengguna tidak cuma melihat siapa yang cocok, tapi juga kriteria mana yang tidak terpenuhi, jadi keputusannya bisa dipertanggungjawabkan. |
| **Request kuota multi-kandidat dan penawaran** | Satu request dikirim ke beberapa listing. Tiap kandidat punya status sendiri dan batas balasan 72 jam. Subkontraktor mengirim penawaran, pemberi order dapat menawar balik, rantai penawaran tercatat per babak. | Menawar ke lima calon mitra tidak lagi berarti lima percakapan terpisah yang harus diingat sendiri. |
| **Work order dengan alokasi kapasitas** | Penawaran yang diterima menjadi work order. Alokasi kapasitas ditulis dalam satu transaksi dengan penguncian baris terurut menurut minggu, sehingga dua kesepakatan berbarengan atas periode yang sama tidak dapat sama-sama berhasil. | Kapasitas yang sudah dijanjikan tidak pernah terjual dua kali, bahkan ketika dua pemberi order menekan tombol terima pada detik yang sama. |
| **Mesin keadaan pesanan dan konfirmasi otomatis** | Tujuh status pesanan dengan transisi yang dikirim backend lewat `allowed_transitions`. Pesanan berstatus dikirim dianggap diterima otomatis setelah 7 hari, dengan pengingat 2 hari sebelumnya, dan berhenti bila ada sengketa. | Pesanan punya batas waktu sendiri, jadi subkontraktor tidak tersangkut menunggu konfirmasi yang tak pernah datang. |
| **Pembatalan, sengketa, dan mediasi admin** | Pembatalan pra-produksi membalik seluruh baris alokasi. Sengketa menghentikan konfirmasi otomatis dan masuk ke antrean mediasi admin dengan hasil dilanjutkan, dikonfirmasi selesai, atau dibatalkan. | Kalau kesepakatan gagal sebelum produksi, kapasitasnya kembali bisa dijual. Kalau berselisih, ada jalur resmi, bukan saling telepon. |
| **Reputasi dan tingkat penyelesaian** | Ulasan setelah pesanan selesai, nilai reputasi turunan, dan tingkat penyelesaian yang hanya ditampilkan sebagai persentase bila datanya cukup. Pembatalan membebani pihak yang membatalkan. | Usaha baru tidak dihukum angka reputasi yang dihitung dari dua transaksi. Kalau datanya belum cukup, yang muncul keterangannya, bukan persentase yang menyesatkan. |
| **Panel admin** | Antrean verifikasi identitas, keputusan usulan item, pengelolaan daftar baku, pesanan telat, sengketa, moderasi ulasan, dan status sambungan WhatsApp beserta QR penyambungan ulang. | Semua urusan operasional bisa diselesaikan dari antarmuka, tanpa sekali pun membuka database. |

### Fitur tambahan

Yang di bawah ini tidak terlihat di alur utama, tapi ikut menentukan apakah aplikasi ini layak dipakai orang lain.

- **Master data produk, mesin, dan wilayah** membuat istilah pencarian tetap seragam. Kalau setiap orang mengisi jenis produk dengan kata-katanya sendiri, pencarian berhenti berfungsi.
- **Usulan item baru** untuk jenis produk atau mesin yang belum ada di daftar baku. Usulannya diputuskan admin, jadi daftar tetap rapi tanpa menutup pintu bagi kebutuhan yang belum terdaftar.
- **Verifikasi identitas usaha** lewat unggahan dokumen dan foto lokasi. Tipe berkas diperiksa dari magic bytes, bukan dari header yang bisa dipalsukan. Nama berkas dibuat sistem, metadata lokasi pada gambar dibuang, dan berkasnya hanya bisa diakses pemiliknya serta admin.
- **Notifikasi in-app** dengan penanda sudah dibaca, jumlah belum dibaca, dan pilihan kanal email atau WhatsApp. Notifikasi transaksional sengaja tidak bisa dimatikan.
- **Rate limiting berbasis data domain** untuk percobaan login, kode verifikasi per nomor dan per alamat asal, serta request kuota per pengguna. Penegakannya di aplikasi, tidak diserahkan ke proxy tepi.
- **Health check** untuk database, sambungan WhatsApp, dan ruang penyimpanan. Endpoint yang sama dipakai sebagai healthcheck container.
- **Galat yang konsisten** dalam format `application/problem+json`, dengan 34 kode mesin yang stabil dan `detail` bahasa Indonesia yang bisa dikutip penguji langsung ke laporan.
- **Swagger UI** di `/docs` saat mode development, membaca kontrak OpenAPI yang sama dengan yang disematkan ke binary.

---

## Demo dan screenshot

### Live demo

**[https://devotion.web.id/](https://devotion.web.id/)**

### Screenshot aplikasi

link gist : [devotion-screenshots](https://gist.github.com/TrygerZ/f6601f096885e7307b7210a750f92f7e)

| | |
|---|---|
| **Beranda**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/01-home.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-01-home.jpg" alt="Beranda" width="420" height="263"></a> | **Hasil pencarian dengan skor kecocokan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/40b-buyer-hasil-pencarian.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-40b-buyer-hasil-pencarian.jpg" alt="Hasil pencarian dengan skor kecocokan" width="420" height="263"></a> |
| **Kalender kapasitas mingguan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/25-sub-kalender-kapasitas.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-25-sub-kalender-kapasitas.jpg" alt="Kalender kapasitas mingguan" width="420" height="263"></a> | **Dasbor admin**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/60-admin-dasbor.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-60-admin-dasbor.jpg" alt="Dasbor admin" width="420" height="263"></a> |


<details>
<summary><b>Halaman publik</b> (6 page)</summary>

Bisa dibuka tanpa login. Beranda tidak diulang di sini karena sudah tampil di sorotan atas.

| | |
|---|---|
| **Tentang**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/02-tentang.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-02-tentang.jpg" alt="Tentang" width="420" height="263"></a> | **Profil usaha publik**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/06-profil-publik.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-06-profil-publik.jpg" alt="Profil usaha publik" width="420" height="263"></a> |
| **Bantuan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/03-bantuan.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-03-bantuan.jpg" alt="Bantuan" width="420" height="263"></a> | **Syarat dan ketentuan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/04-syarat-ketentuan.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-04-syarat-ketentuan.jpg" alt="Syarat dan ketentuan" width="420" height="263"></a> |
| **Kebijakan privasi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/05-kebijakan-privasi.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-05-kebijakan-privasi.jpg" alt="Kebijakan privasi" width="420" height="263"></a> | **Halaman tidak ditemukan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/13-404.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-13-404.jpg" alt="Halaman tidak ditemukan" width="420" height="263"></a> |

</details>

<details>
<summary><b>Autentikasi dan verifikasi</b> (6 page)</summary>

Registrasi sampai pemulihan kata sandi.

| | |
|---|---|
| **Registrasi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/07-register.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-07-register.jpg" alt="Registrasi" width="420" height="263"></a> | **Login**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/08-login.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-08-login.jpg" alt="Login" width="420" height="263"></a> |
| **Verifikasi email**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/09-verifikasi-email.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-09-verifikasi-email.jpg" alt="Verifikasi email" width="420" height="263"></a> | **Verifikasi nomor HP**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/10-verifikasi-telepon.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-10-verifikasi-telepon.jpg" alt="Verifikasi nomor HP" width="420" height="263"></a> |
| **Lupa kata sandi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/11-lupa-sandi.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-11-lupa-sandi.jpg" alt="Lupa kata sandi" width="420" height="263"></a> | **Atur ulang kata sandi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/12-reset-sandi.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-12-reset-sandi.jpg" alt="Atur ulang kata sandi" width="420" height="263"></a> |

</details>

<details>
<summary><b>Subkontraktor</b> (10 page)</summary>

Sisi pemilik kapasitas: memasang listing, mengatur kalender, menjawab permintaan.

| | |
|---|---|
| **Profil usaha saya**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/20-sub-profil-saya.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-20-sub-profil-saya.jpg" alt="Profil usaha saya" width="420" height="263"></a> | **Verifikasi identitas usaha**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/21-sub-verifikasi-identitas.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-21-sub-verifikasi-identitas.jpg" alt="Verifikasi identitas usaha" width="420" height="263"></a> |
| **Listing kapasitas**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/24-sub-listing.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-24-sub-listing.jpg" alt="Listing kapasitas" width="420" height="263"></a> | **Kalender kapasitas mingguan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/25-sub-kalender-kapasitas.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-25-sub-kalender-kapasitas.jpg" alt="Kalender kapasitas mingguan" width="420" height="263"></a> |
| **Permintaan kuota masuk**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/26-sub-permintaan-masuk.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-26-sub-permintaan-masuk.jpg" alt="Permintaan kuota masuk" width="420" height="263"></a> | **Detail permintaan dan penawaran**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/27-sub-permintaan-masuk-detail.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-27-sub-permintaan-masuk-detail.jpg" alt="Detail permintaan dan penawaran" width="420" height="263"></a> |
| **Daftar pesanan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/28-sub-pesanan.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-28-sub-pesanan.jpg" alt="Daftar pesanan" width="420" height="263"></a> | **Detail pesanan, sisi subkontraktor**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/29-sub-pesanan-detail.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-29-sub-pesanan-detail.jpg" alt="Detail pesanan, sisi subkontraktor" width="420" height="263"></a> |
| **Notifikasi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/22-sub-notifikasi.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-22-sub-notifikasi.jpg" alt="Notifikasi" width="420" height="263"></a> | **Preferensi notifikasi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/23-sub-preferensi-notifikasi.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-23-sub-preferensi-notifikasi.jpg" alt="Preferensi notifikasi" width="420" height="263"></a> |

</details>

<details>
<summary><b>Pemberi order</b> (8 page)</summary>

Sisi pencari kapasitas: mencari, meminta kuota, memantau pesanan.

| | |
|---|---|
| **Form pencarian kapasitas**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/40-buyer-pencarian.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-40-buyer-pencarian.jpg" alt="Form pencarian kapasitas" width="420" height="263"></a> | **Hasil pencarian dengan skor kecocokan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/40b-buyer-hasil-pencarian.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-40b-buyer-hasil-pencarian.jpg" alt="Hasil pencarian dengan skor kecocokan" width="420" height="263"></a> |
| **Buat permintaan kuota multi-kandidat**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/41-buyer-buat-permintaan.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-41-buyer-buat-permintaan.jpg" alt="Buat permintaan kuota multi-kandidat" width="420" height="263"></a> | **Permintaan terkirim**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/42-buyer-permintaan-terkirim.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-42-buyer-permintaan-terkirim.jpg" alt="Permintaan terkirim" width="420" height="263"></a> |
| **Detail permintaan dan perbandingan penawaran**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/43-buyer-permintaan-detail.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-43-buyer-permintaan-detail.jpg" alt="Detail permintaan dan perbandingan penawaran" width="420" height="263"></a> | **Daftar pesanan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/44-buyer-pesanan.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-44-buyer-pesanan.jpg" alt="Daftar pesanan" width="420" height="263"></a> |
| **Detail pesanan, sisi pemberi order**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/45-buyer-pesanan-detail.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-45-buyer-pesanan-detail.jpg" alt="Detail pesanan, sisi pemberi order" width="420" height="263"></a> | **Profil usaha saya**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/46-buyer-profil-saya.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-46-buyer-profil-saya.jpg" alt="Profil usaha saya" width="420" height="263"></a> |

</details>

<details>
<summary><b>Panel admin</b> (10 page)</summary>

Operasional platform, seluruhnya lewat antarmuka tanpa menyentuh database.

| | |
|---|---|
| **Dasbor admin**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/60-admin-dasbor.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-60-admin-dasbor.jpg" alt="Dasbor admin" width="420" height="263"></a> | **Antrean verifikasi identitas**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/61-admin-antrean-verifikasi.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-61-admin-antrean-verifikasi.jpg" alt="Antrean verifikasi identitas" width="420" height="263"></a> |
| **Daftar baku produk dan mesin**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/62-admin-master-item.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-62-admin-master-item.jpg" alt="Daftar baku produk dan mesin" width="420" height="263"></a> | **Usulan item baru**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/63-admin-usulan-item.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-63-admin-usulan-item.jpg" alt="Usulan item baru" width="420" height="263"></a> |
| **Pesanan telat**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/64-admin-pesanan-telat.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-64-admin-pesanan-telat.jpg" alt="Pesanan telat" width="420" height="263"></a> | **Detail pesanan, sisi admin**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/65-admin-pesanan-detail.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-65-admin-pesanan-detail.jpg" alt="Detail pesanan, sisi admin" width="420" height="263"></a> |
| **Sengketa dan mediasi**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/66-admin-sengketa.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-66-admin-sengketa.jpg" alt="Sengketa dan mediasi" width="420" height="263"></a> | **Moderasi ulasan**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/67-admin-moderasi-ulasan.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-67-admin-moderasi-ulasan.jpg" alt="Moderasi ulasan" width="420" height="263"></a> |
| **Status sambungan WhatsApp**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/68-admin-whatsapp.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-68-admin-whatsapp.jpg" alt="Status sambungan WhatsApp" width="420" height="263"></a> | **Kesehatan sistem**<br><a href="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/69-admin-sistem.png"><img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/t-69-admin-sistem.jpg" alt="Kesehatan sistem" width="420" height="263"></a> |

</details>


### Video demo

**Link video demo: belum tersedia.**

---

## Teknologi

### Tech stack

#### Frontend

```text
Framework    : React 18.3.1, TypeScript 5.8.3
Build tool   : Vite 8.2.0, hasil build di-embed ke binary Go
Styling      : Tailwind CSS 4.3.3, tanpa component library kedua
Server state : TanStack Query 5.102.1
Form         : React Hook Form 7.86.0 + Zod 4.4.3
Routing      : React Router 7.18.2
Peta         : Leaflet 1.9.4 + tile OpenStreetMap, tanpa kunci API
API client   : Fetch API, credentials include, tipe di-generate dari OpenAPI
```

#### Backend

```text
Runtime      : Go 1.25.0
HTTP         : net/http, router bawaan Go 1.22+
Database     : PostgreSQL 16, pgx/v5 5.7.5 + query hasil generate sqlc
Migration    : golang-migrate, otomatis saat startup di bawah advisory lock
Auth         : bcrypt, sesi cookie httpOnly, token disimpan sebagai hash
Notifikasi   : net/smtp ke Mailjet, whatsmeow untuk WhatsApp
Observability: log/slog JSON dengan request ID, Sentry opsional
```

#### DevOps dan tools

```text
Runtime      : Docker Compose, tepat 2 layanan (backend, postgres)
CI/CD        : GitHub Actions, image ke GitHub Container Registry
Edge dan TLS : Cloudflare Origin Certificate, TLS diselesaikan binary Go
Testing      : go vet, go test, ESLint, tsc
```

### Alasan pemilihan teknologi

Satu batasan menentukan hampir semua pilihan: aturan panitia yang membatasi layanan runtime menjadi dua.

| Teknologi | Alasan pemilihan |
|---|---|
| **React + Vite** | Hasil build disematkan ke binary Go lewat `embed.FS`, sehingga frontend menjadi berkas statis, bukan layanan runtime ketiga. |
| **Go + net/http** | Satu binary, router sudah ada di standard library sejak Go 1.22, jejak memori kecil. Framework HTTP tambahan tidak menyelesaikan masalah yang belum selesai. |
| **PostgreSQL 16** | Alokasi kapasitas butuh transaksi, CHECK constraint, dan penguncian baris. Dua kesepakatan bersamaan diselesaikan database, bukan logika aplikasi. |
| **sqlc, bukan ORM** | Query pencarian dan skor kecocokan adalah inti produk. SQL ditulis eksplisit agar dapat dibaca dan diaudit. |
| **OpenAPI + openapi-typescript** | Kontrak menjadi sumber tipe frontend. Perubahan bentuk respons memunculkan galat compiler, bukan bug runtime. |
| **Tailwind CSS** | Cukup untuk antarmuka mobile-first tanpa menambah component library yang gaya visualnya harus diselaraskan ulang. |
| **Leaflet + OpenStreetMap** | Peta tanpa kunci API dan tanpa tagihan. Jarak bersifat informatif dan tidak memengaruhi skor. |

### Dependencies utama

```text
Frontend  react, react-dom, vite, @tanstack/react-query,
          react-hook-form, zod, @hookform/resolvers, react-router-dom,
          tailwindcss, @tailwindcss/vite, leaflet, react-leaflet,
          motion, react-icons, clsx, tailwind-merge, qrcode,
          axios, react-compiler-runtime

Backend   github.com/jackc/pgx/v5
          github.com/golang-migrate/migrate/v4
          golang.org/x/crypto, golang.org/x/term
          go.mau.fi/whatsmeow
          github.com/getsentry/sentry-go
```

Daftar backend pendek karena disengaja. Log terstruktur (`log/slog`), email (`net/smtp`), pembuangan metadata gambar (`image/jpeg`), token acak (`crypto/rand`), UUID (`gen_random_uuid()`), dan jarak haversine semuanya diselesaikan standard library.

---

## Arsitektur sistem

### System architecture

Satu binary Go melayani API JSON dan React SPA dari proses yang sama. Cloudflare, Mailjet, WhatsApp, dan Sentry berstatus dependensi eksternal, bukan container.

```mermaid
flowchart TB
    subgraph client["Klien"]
        browser["Browser<br/>React 18 SPA<br/>cookie httpOnly"]
    end

    subgraph edge["Edge"]
        cf["Cloudflare<br/>proxy dan TLS<br/>Origin Certificate"]
    end

    subgraph host["Server, docker compose"]
        subgraph be["Layanan 1: backend, Go 1.25"]
            spa["Static handler<br/>embed.FS webdist"]
            api["API handler<br/>net/http router"]
            gate["Session dan role gate<br/>UncoveredAPIRoutes fail fast"]
            sched["Scheduler in process<br/>time.Ticker, advisory lock"]
            mig["Migrator<br/>golang-migrate<br/>pg_try_advisory_lock"]
        end
        pg[("Layanan 2: postgres 16<br/>max_connections 20")]
    end

    subgraph ext["Layanan eksternal, bukan container"]
        mail["Mailjet<br/>net/smtp"]
        wa["WhatsApp<br/>whatsmeow"]
        sentry["Sentry<br/>error tracking"]
    end

    browser -->|HTTPS| cf
    cf -->|"origin TLS, rentang IP dipatok"| spa
    cf -->|"/api"| api
    api --> gate
    gate -->|"pgx pool, maksimal 15 koneksi"| pg
    spa -.->|"non /api, SPA fallback"| browser
    mig -->|"saat startup"| pg
    sched -->|"tenggat dan antrean notifikasi"| pg
    sched -.-> mail
    sched -.-> wa
    api -.-> sentry

    classDef svc fill:#1f6feb,stroke:#0d419d,color:#ffffff
    classDef db fill:#238636,stroke:#196c2e,color:#ffffff
    classDef extn fill:#6e7681,stroke:#484f58,color:#ffffff
    class spa,api,gate,sched,mig svc
    class pg db
    class mail,wa,sentry extn
```

| Properti | Penerapan | Konsekuensi |
|---|---|---|
| **Satu proses, dua peran** | Static handler membaca `embed.FS`; path selain `/api` jatuh ke SPA fallback, `/api` tak dikenal membalas 404 JSON. | Frontend bukan layanan runtime, dan salah tulis endpoint tetap menghasilkan galat JSON. |
| **Gerbang peran wajib** | Setiap pola `/api` terdaftar publik atau bergerbang; `UncoveredAPIRoutes()` diperiksa saat `serve`. | Proses menolak menyala bila ada pola tanpa keputusan peran. |
| **Penjadwal in process** | `time.Ticker` di binary yang sama, tiap pekerjaan dibungkus advisory lock, tenggat juga dievaluasi saat data dibaca. | Tanpa worker terpisah, dan pekerjaan tidak dieksekusi ganda. |
| **Migrasi terkunci** | `golang-migrate` saat startup di bawah `pg_try_advisory_lock`. | Deployment tanpa langkah migrasi manual, dua instance tidak saling menimpa skema. |

### Database schema

**26 tabel domain**, plus `schema_migrations` dan tabel milik `whatsmeow`. ERD dipecah per konteks. Atribut dibatasi pada kunci dan kolom yang menentukan perilaku; `created_at` dan `updated_at` tidak diulang.

Entitas tanpa daftar atribut sudah dirinci di diagram lain. Kolom aktor admin (`decided_by`, `handled_by`, `hidden_by`, `changed_by`) menunjuk `user_account.id` dan kosong selama keputusan belum diambil; panahnya tidak digambar.

#### Identitas, wilayah, dan verifikasi

```mermaid
erDiagram
    province ||--o{ city : "membawahi"
    city ||--o{ business_profile : "melokasikan"
    user_account ||--o| business_profile : "memiliki"
    user_account ||--o{ session : "membuka"
    user_account ||--o{ verification_code : "menerima"
    business_profile ||--o{ uploaded_file : "mengunggah"
    business_profile ||--o{ verification_request : "mengajukan"
    uploaded_file ||--o{ verification_request : "melampirkan"

    province {
        text code PK "regex 2 digit"
        text name
    }
    city {
        text code PK "regex 4 digit"
        text province_code FK "2 digit awal sama"
        text name
    }
    user_account {
        uuid id PK
        citext email UK
        text phone UK "regex 62 diikuti 8 sampai 13 digit"
        text password_hash "bcrypt"
        boolean email_verified
        boolean phone_verified
        boolean role_subcontractor
        boolean role_buyer
        boolean role_admin "eksklusif dari peran usaha"
        boolean notif_nontx_email "preferensi kanal non transaksional"
        boolean notif_nontx_whatsapp "preferensi kanal non transaksional"
    }
    business_profile {
        uuid id PK
        uuid account_id FK,UK
        text business_name "minimal 3 karakter"
        text city_code FK
        numeric latitude "dibatasi wilayah Indonesia"
        numeric longitude "wajib berpasangan dengan latitude"
        text description
        boolean verified
    }
    session {
        uuid id PK
        uuid account_id FK
        bytea token_hash UK "hash, bukan token mentah"
        inet source_address
        timestamptz expires_at
        timestamptz accessed_at "dasar perpanjangan sesi"
    }
    verification_code {
        uuid id PK
        uuid account_id FK
        verification_purpose purpose "email, phone, recovery"
        bytea code_hash
        timestamptz expires_at
        timestamptz consumed_at
    }
    uploaded_file {
        uuid id PK
        uuid owner_profile_id FK
        file_type type "identity_document, location_photo"
        text original_name
        text mime_type "jpeg, png, pdf saja"
        integer size_bytes "maksimal 5 MB"
        text storage_path UK "nama dibuat sistem"
    }
    verification_request {
        uuid id PK
        uuid profile_id FK "satu pengajuan pending per profil"
        text identity_number
        uuid identity_file_id FK
        uuid location_file_id FK
        verification_status status
        text admin_note "wajib bila ditolak"
        uuid decided_by FK "user_account, wajib bila sudah diputus"
        timestamptz decided_at
        inet applicant_source_address
    }
```

#### Katalog, listing, dan kalender kapasitas

```mermaid
erDiagram
    business_profile ||--o| capacity_listing : "menerbitkan"
    business_profile ||--o{ item_proposal : "mengusulkan"
    capacity_listing ||--o{ availability_period : "menjadwalkan"
    capacity_listing ||--o{ listing_product : "menawarkan"
    capacity_listing ||--o{ listing_machine : "mengoperasikan"
    catalog_item ||--o{ listing_product : "dirujuk"
    catalog_item ||--o{ listing_machine : "dirujuk"
    catalog_item ||--o{ item_proposal : "dihasilkan"

    catalog_item {
        uuid id PK
        item_type type "product atau machine"
        text name "unik per tipe"
        boolean active
        integer sort_order
    }
    item_proposal {
        uuid id PK
        uuid profile_id FK
        item_type type
        text proposed_name
        proposal_status status
        text admin_note
        uuid item_id FK "terisi bila disetujui"
        uuid decided_by FK "user_account"
        timestamptz decided_at
    }
    capacity_listing {
        uuid id PK
        uuid profile_id FK,UK "satu listing per profil"
        integer weekly_capacity "harus positif"
        integer readiness_lead_days "0 sampai 365"
        boolean published
        timestamptz calendar_updated_at "dasar penanda kalender basi"
        date horizon_until "wajib hari Senin"
        timestamptz stale_notified_at "penanda pengingat kalender basi, direset saat kalender diperbarui"
    }
    availability_period {
        uuid id PK
        uuid listing_id FK
        date week_start "wajib hari Senin, unik per listing"
        integer total_capacity
        integer used_capacity "tidak melebihi total"
        boolean marked_full
    }
    listing_product {
        uuid listing_id PK,FK
        uuid item_id PK,FK
    }
    listing_machine {
        uuid listing_id PK,FK
        uuid item_id PK,FK
        integer machine_count "harus positif"
    }
```

#### Request kuota, penawaran, dan work order

```mermaid
erDiagram
    business_profile ||--o{ quota_request : "mengirim"
    business_profile ||--o{ request_candidate : "dijangkau sebagai subkontraktor"
    business_profile ||--o{ work_order : "menjadi pembeli atau subkontraktor"
    business_profile ||--o{ payment_record : "menyatakan pembayaran"
    business_profile ||--o{ review : "menulis dan menerima"
    business_profile ||--o{ dispute : "melaporkan"
    catalog_item ||--o{ quota_request : "menentukan produk"
    quota_request ||--o{ request_candidate : "menjangkau"
    capacity_listing ||--o{ request_candidate : "dinilai"
    request_candidate ||--o{ offer : "menampung"
    request_candidate ||--o| work_order : "menghasilkan"
    offer ||--o| work_order : "mendasari"
    work_order ||--o{ capacity_allocation : "memotong"
    availability_period ||--o{ capacity_allocation : "dipotong"
    work_order ||--o{ work_order_status_history : "mencatat"
    work_order ||--o{ payment_record : "mencatat pernyataan"
    work_order ||--o| dispute : "menimbulkan"
    work_order ||--o{ review : "membuka"

    quota_request {
        uuid id PK
        uuid buyer_id FK
        uuid product_item_id FK
        integer quantity "harus positif"
        text material
        date deadline
        text note
        timestamptz reply_due_at "72 jam sejak dibuat"
    }
    request_candidate {
        uuid id PK
        uuid request_id FK "satu kandidat per listing per request"
        uuid listing_id FK
        uuid subcontractor_id FK
        candidate_status status "6 nilai, hanya satu agreed per request"
        text rejection_reason
    }
    offer {
        uuid id PK
        uuid candidate_id FK
        integer sequence "unik per kandidat, satu babak penawaran"
        offer_party proposed_by "subcontractor atau buyer"
        bigint total_price "rupiah bulat, harus positif"
        integer readiness_lead_days
        text note
    }
    work_order {
        uuid id PK
        uuid candidate_id FK,UK
        uuid offer_id FK
        uuid buyer_id FK "wajib berbeda dari subkontraktor"
        uuid subcontractor_id FK
        integer quantity
        bigint total_price "rupiah bulat"
        date deadline
        date readiness_week_start "wajib hari Senin, tidak melewati deadline"
        work_order_status status "7 status"
        timestamptz shipped_at
        timestamptz auto_confirm_base_at "dasar hitung 7 hari"
        timestamptz confirmed_at
        boolean auto_confirmed
        timestamptz confirm_warn_sent_at "penanda pengingat sekali kirim"
        timestamptz late_notified_at "penanda pemberitahuan lewat tenggat"
        timestamptz deadline_warn_sent_at "penanda peringatan tenggat pengiriman mendekat"
        uuid cancelled_by_id FK
        text cancellation_reason
        timestamptz cancelled_at
    }
    capacity_allocation {
        uuid id PK
        uuid work_order_id FK
        uuid period_id FK "satu baris per pasangan order dan periode"
        integer quantity "harus positif"
        timestamptz reversed_at "terisi saat alokasi dibalik"
    }
    work_order_status_history {
        uuid id PK
        uuid work_order_id FK
        work_order_status old_status
        work_order_status new_status
        uuid changed_by FK "user_account, wajib bila bukan sistem"
        boolean by_system
        text note
    }
    payment_record {
        uuid id PK
        uuid work_order_id FK
        uuid profile_id FK
        payment_direction direction "sent atau received"
        date date "tanpa kolom jumlah uang"
        text note
    }
    dispute {
        uuid id PK
        uuid work_order_id FK "satu sengketa terbuka per pesanan"
        uuid reporter_id FK
        text report_body
        dispute_status status
        dispute_result result "cancelled, continued, confirmed"
        boolean allocation_reversed
        uuid liable_party_id FK
        text admin_note
        uuid handled_by FK "user_account"
        timestamptz resolved_at
    }
    review {
        uuid id PK
        uuid work_order_id FK
        uuid reviewer_id FK "tidak boleh sama dengan reviewee"
        uuid reviewee_id FK
        smallint rating "1 sampai 5"
        text text
        boolean hidden
        uuid hidden_by FK "user_account"
        timestamptz hidden_at
        text hidden_reason
    }
```

Tiga aturan di sini harus membaca tabel lain, jadi ditegakkan trigger, bukan `CHECK`: `trg_reject_self_request` menolak kandidat yang subkontraktornya sama dengan pembeli (FR-083), `trg_reject_allocation_before_readiness` menolak alokasi sebelum `readiness_week_start` (FR-087), `trg_reject_wrong_product_item` pada `listing_product` dan `trg_reject_wrong_machine_item` pada `listing_machine` mengikat setiap baris ke tipe item yang benar, keduanya memanggil fungsi `reject_wrong_item_type()` dengan argumen tipe yang berbeda.

#### Notifikasi dan pembatasan laju

```mermaid
erDiagram
    user_account ||--o{ notification : "menerima"
    notification ||--o{ notification_channel : "dikirim lewat"

    notification {
        uuid id PK
        uuid account_id FK
        event_type event "16 jenis kejadian"
        boolean transactional "tidak dapat dimatikan pengguna"
        text title
        text body
        text link
        timestamptz read_at
    }
    notification_channel {
        uuid id PK
        uuid notification_id FK "satu baris per kanal"
        notification_channel_type channel "email atau whatsapp"
        delivery_status status "pending, sent, failed_permanent"
        smallint attempts "maksimal 3"
        text last_error
        timestamptz attempted_at "urutan antrean pengiriman"
        timestamptz sent_at
    }
    rate_limit {
        uuid id PK
        rate_limit_target target "login, otp per nomor, otp per alamat, request kuota"
        text key
        timestamptz window_start "unik bersama target dan key"
        integer count
    }
```

`rate_limit` tanpa kunci asing. `key` menyimpan pengenal sasaran sebagai teks (id akun, nomor, atau alamat asal) sesuai `target`, sehingga pembatasan tetap berlaku bagi pihak yang belum punya akun.

#### Keputusan skema

| Keputusan | Penerapan | Alasan |
|---|---|---|
| **Uang bilangan bulat** | `offer.total_price` dan `work_order.total_price` `bigint`, `CHECK (total_price > 0)`. | Rupiah tidak dipecah di B2B ini, dan tipe pecahan menimbulkan galat pembulatan. |
| **Minggu selalu Senin** | `week_start`, `horizon_until`, `readiness_week_start` `date` dengan `CHECK (EXTRACT(ISODOW ...) = 1)`. | Periode tidak jatuh di tengah minggu, jadi kapasitas tidak berkurang dari periode salah. |
| **Platform tidak memegang dana** | `payment_record` tanpa kolom jumlah, hanya arah dan tanggal, unik per pesanan, pihak, dan arah. | Platform mencatat pernyataan kedua pihak tanpa jadi perantara dana. |
| **Kapasitas tidak terjual dua kali** | `capacity_allocation` unik per `work_order_id` dan `period_id`, `used_capacity <= total_capacity`, satu transaksi dengan `SELECT ... FOR UPDATE` terurut `week_start`. | Batas ditegakkan database, dan urutan kunci seragam mencegah deadlock. |
| **Jejak keputusan lengkap** | `dispute`, `item_proposal`, `verification_request`, `review` memakai CHECK gabungan: kolom pendukung wajib terisi begitu status keluar dari pending. | Status terminal tidak tersimpan tanpa catatan admin, waktu, dan pelaku. |
| **Token tidak disimpan mentah** | `session.token_hash` dan `verification_code.code_hash` `bytea` berisi hash. | Kebocoran isi tabel tidak langsung berarti pengambilalihan sesi. |

Definisi lengkap beserta indeks dan constraint ada di `docs/001-capacity-exchange-marketplace/data-model.md` dan `backend/db/migrations/`.

### Folder structure

```text
devotion/
├── backend/
│   ├── cmd/devotion/       # serve, admin:create, seed:*, reset:*, user:verify, health:check
│   ├── apidocs/            # Swagger UI dan salinan openapi.yaml yang di-embed
│   ├── internal/
│   │   ├── platform/       # clock, config, httpx, session, storage, scheduler,
│   │   │                   # ratelimit, cloudflare, health, migrate, observability, tlsconf
│   │   ├── account/        # akun, peran, profil, autentikasi, pemulihan
│   │   ├── verification/   # unggahan berkas, pengajuan, keputusan admin
│   │   ├── masterdata/     # daftar baku, wilayah, usulan item
│   │   ├── listing/        # listing kapasitas, kalender ketersediaan
│   │   ├── search/         # kriteria keras, skor, pemecah seri, keyset
│   │   ├── quota/          # request kuota, penawaran, counter-offer
│   │   ├── order/          # work order, alokasi, pembatalan, pembayaran, sengketa
│   │   ├── reputation/     # ulasan, nilai turunan, moderasi
│   │   ├── notification/   # antrean, pengiriman, percobaan ulang
│   │   ├── admin/          # status dan penyambungan WhatsApp
│   │   └── db/             # pool 15 koneksi, sqlcgen, testdb skema uji terpisah
│   ├── db/migrations/      # 22 migrasi golang-migrate
│   ├── db/queries/         # 15 berkas SQL sumber sqlc
│   └── webdist/            # hasil build frontend, disematkan lewat embed.FS
├── frontend/src/
│   ├── api/                # klien fetch dan tipe hasil generate dari OpenAPI
│   ├── components/         # layout, section, komponen bersama
│   ├── pages/              # 38 halaman, dikelompokkan per user story
│   ├── hooks/              # hook TanStack Query per domain
│   ├── schemas/            # skema Zod
│   ├── routes/             # GuestRoute, ProtectedRoute
│   ├── providers/          # QueryProvider
│   ├── lib/                # helper murni, termasuk predikat penawaran di offers.ts
│   ├── data/               # data statis bawaan antarmuka
│   ├── test/               # setup dan utilitas Jest
│   ├── styles/             # entri Tailwind dan gaya global
│   └── assets/             # gambar dan ikon
├── docs/                   # spec, plan, data-model, contracts, tasks, panduan operasional
├── .github/workflows/ci.yml
├── docker-compose.yml
├── .env.example
├── LICENSE
└── README.md
```

---

## Instalasi dan setup

### Prasyarat

**Git**, **Docker Engine** dengan Compose v2, **Go 1.25.0** atau lebih baru, dan **Node.js 20** atau lebih baru.

### Langkah instalasi

#### 1. Clone repository

```bash
git clone https://github.com/fzrilsh/devotion.git
cd devotion
```

#### 2. Siapkan environment variable

```bash
cp .env.example .env
```

Wajib di semua lingkungan hanya empat: `APP_ENV`, `APP_BASE_URL`, `DATABASE_URL`, `UPLOAD_PATH`. Sisanya diwajibkan saat `APP_ENV=production`.

```env
APP_ENV=development
APP_BASE_URL=http://localhost:8080
POSTGRES_USER=devotion
POSTGRES_PASSWORD=ganti_password_lokal
POSTGRES_DB=devotion
DATABASE_URL=postgres://devotion:***@127.0.0.1:5434/devotion?sslmode=disable
UPLOAD_PATH=/absolute/path/to/devotion/backend/uploads
UPLOAD_MAX_TOTAL_MB=500
UPLOAD_MAX_FILE_MB=5
```

Port `5434` bukan salah tulis: Compose menerbitkan Postgres di `127.0.0.1:5434` agar tidak bertabrakan dengan Postgres lain di 5432, dan ikatan loopback menjaganya tidak terekspos keluar mesin. Pengaturan TLS, Mailjet, WhatsApp, dan Sentry ada di `docs/001-capacity-exchange-marketplace/quickstart.md`. `.env` tidak pernah di-commit.

#### 3. Jalankan PostgreSQL

```bash
mkdir -p /absolute/path/to/devotion/backend/uploads
docker compose up -d postgres
```

Tanpa langkah migrasi manual. Migrasi jalan saat backend menyala, di bawah advisory lock.

#### 4. Jalankan backend

Binary Go membaca environment shell, bukan berkas `.env`; yang membaca `.env` hanya Docker Compose. Tanpa ekspor, `serve` berhenti dengan `variabel lingkungan wajib belum diisi: APP_ENV`.

```bash
set -a; . ./.env; set +a
cd backend
go run ./cmd/devotion serve
```

Backend mendengarkan di `http://localhost:8080`. Swagger UI di `/docs`, hanya saat `APP_ENV=development`.

#### 5. Isi data acuan dan buat akun

Sekali per database baru, dari `backend/` dengan environment yang sama.

```bash
go run ./cmd/devotion seed:regions
go run ./cmd/devotion seed:master-data
go run ./cmd/devotion admin:create
go run ./cmd/devotion seed:test-data      # menolak jalan bila APP_ENV=production
```

Data wilayah dibaca dari salinan di repository, tanpa memanggil layanan luar. Kredensial akun uji bisa dilihat pada bagian Testing atau [creedentials.txt](https://gist.github.com/fzrilsh/80783d8b07ac57dc2af454bc8796dd0d#file-creedentials-txt) di gist data dummy.

#### 6. Jalankan frontend

```bash
cd frontend
npm install
npm run dev
```

Vite jalan di `http://localhost:5173` dan memproksikan `/api` ke port 8080. Proxy ini wajib: tanpanya frontend dan backend terlihat sebagai origin berbeda, cookie `SameSite=Lax` tidak terkirim, dan setiap permintaan tampak belum login meski login berhasil.

> `npm run build` lolos di TypeScript 5.8.3, `tsconfig.app.json` dan `tsconfig.node.json` memakai `erasableSyntaxOnly` yang didukung sejak TypeScript 5.8.

---

## Penggunaan

### Menjalankan aplikasi

```bash
# Backend
cd backend && go run ./cmd/devotion serve

# Frontend, terminal lain
cd frontend && npm run dev

# Build dan periksa kualitas kode frontend
npm run build
npm run lint
```

Untuk production, CI membangun frontend, menyalin hasilnya ke `backend/webdist/`, lalu membangun image; server hanya menarik dan menjalankan. Build tidak dilakukan di server produksi, supaya proses build tidak berebut sumber daya dengan Postgres yang sedang hidup.

### Panduan pengguna

#### Untuk subkontraktor

1. Daftar sebagai **Subkontraktor** atau **Keduanya**, selesaikan verifikasi email dan nomor HP.
2. Lengkapi profil usaha beserta titik lokasi di peta.
3. Ajukan verifikasi identitas untuk mendapat lencana terverifikasi. Opsional, tapi menaikkan kepercayaan calon pemberi order.
4. Buat listing: produk yang dikuasai, mesin dan jumlah unit, kapasitas per minggu, jeda kesiapan.
5. Isi kalender ketersediaan 12 minggu ke depan, tandai minggu yang penuh.
6. Terbitkan listing. Sebelum diterbitkan, listing tidak muncul di pencarian.
7. Balas request kuota dengan harga total dan jeda kesiapan. Batas 72 jam, lewat itu dianggap tidak membalas.
8. Jalankan pesanan lewat status produksi, selesai, dikirim. Catat pernyataan pembayaran, buka sengketa bila ada ketidaksesuaian.

#### Untuk pemberi order

1. Masuk sebagai **Pemberi Order** atau **Keduanya**.
2. Isi kriteria pencarian: produk, mesin, jumlah, tenggat, jeda maksimum, cakupan wilayah.
3. Bandingkan kandidat lewat skor 0 sampai 4 beserta kriteria mana yang tidak terpenuhi.
4. Pilih beberapa kandidat, kirim satu request kuota untuk semuanya.
5. Bandingkan penawaran, tawar balik bila perlu, terima satu penawaran. Work order terbentuk dan kapasitas mitra langsung terpakai.
6. Konfirmasi begitu barang diterima. Bila dibiarkan, pesanan dianggap diterima otomatis setelah 7 hari, dengan pengingat 2 hari sebelumnya.
7. Beri ulasan setelah pesanan selesai.

#### Untuk admin

Akun admin pertama dibuat lewat `go run ./cmd/devotion admin:create`, bukan pendaftaran biasa. Panel `/admin` menampung antrean verifikasi identitas, keputusan usulan item, pengelolaan daftar baku, pesanan telat, sengketa beserta mediasi dan penyelesaiannya, moderasi ulasan, dan status sambungan WhatsApp dengan QR untuk menyambung ulang. Nomor layanan WhatsApp tidak pernah ditampilkan di antarmuka.

---

## Dokumentasi API

### Base URL

```text
Development  : http://localhost:8080/api
Production   : ditentukan oleh APP_BASE_URL
Health check : /api/health
Swagger UI   : /docs, hanya saat APP_ENV=development
```

Semua permintaan memakai cookie sesi `httpOnly`, tanpa token di badan respons. Path `/api` tak dikenal membalas 404 JSON, bukan `index.html`, sehingga salah tulis endpoint tetap dapat didiagnosis.

### Endpoints

#### Autentikasi, akun, dan profil

```http
POST  /api/auth/register              POST  /api/auth/login
POST  /api/auth/verify-email          POST  /api/auth/logout
POST  /api/auth/verify-phone          POST  /api/auth/recover/request
POST  /api/auth/resend-code           POST  /api/auth/recover/confirm
GET   /api/me                         PATCH /api/me/roles
GET   /api/profile/me                 PUT   /api/profile/me
GET   /api/profile/{profileId}        GET   /api/profile/{profileId}/reviews
```

#### Berkas, verifikasi, dan master data

```http
POST /api/files                       GET  /api/files/{fileId}
POST /api/verification                GET  /api/verification
GET  /api/master/products             GET  /api/master/machines
GET  /api/regions/provinces           GET  /api/regions/cities
POST /api/master/proposals
```

#### Listing dan kalender kapasitas

```http
GET /api/listing/me                   POST /api/listing/me
PUT /api/listing/me                   PUT  /api/listing/me/visibility
GET /api/listing/me/periods           PUT  /api/listing/me/periods
```

#### Pencarian, request kuota, dan penawaran

```http
GET  /api/search
POST /api/quota-requests              GET  /api/quota-requests
GET  /api/quota-requests/incoming     GET  /api/quota-requests/{requestId}
POST /api/candidates/{candidateId}/offers
POST /api/candidates/{candidateId}/reject
POST /api/offers/{offerId}/counter    POST /api/offers/{offerId}/accept
```

#### Work order

```http
GET  /api/work-orders                 GET  /api/work-orders/{workOrderId}
POST /api/work-orders/{workOrderId}/status
POST /api/work-orders/{workOrderId}/cancel
POST /api/work-orders/{workOrderId}/payments
POST /api/work-orders/{workOrderId}/disputes
POST /api/work-orders/{workOrderId}/reviews
```

#### Notifikasi, operasional, dan admin

```http
GET  /api/notifications               POST /api/notifications/{notificationId}/read
GET  /api/notifications/preferences   PUT  /api/notifications/preferences
GET  /api/health

GET   /api/admin/verification         POST  /api/admin/verification/{requestId}/decision
GET   /api/admin/proposals            POST  /api/admin/proposals/{proposalId}/decision
GET   /api/admin/master/items         POST  /api/admin/master/items
PATCH /api/admin/master/items/{itemId}
GET   /api/admin/late-orders          GET   /api/admin/disputes
POST  /api/admin/disputes/{disputeId}/mediate
POST  /api/admin/disputes/{disputeId}/resolve
POST  /api/admin/reviews/{reviewId}/hide
GET   /api/admin/whatsapp
```

Kontrak memuat 66 operasi pada 58 path, seluruhnya terdaftar di router termasuk `POST /api/work-orders/{workOrderId}/confirm` (`backend/internal/order/confirm.go:23`). Setiap pola `/api` wajib punya keputusan peran, publik atau bergerbang, dan `serve` menolak menyala bila ada satu pola tanpa keputusan itu.

### Contoh request

```bash
curl -i -c cookies.txt \
  -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}'

curl -i -b cookies.txt http://localhost:8080/api/me
```

Kredensial di atas hanya contoh. Pakai akun lokal sendiri saat menguji.

### Kontrak API lengkap

Kontrak OpenAPI 3.1 di [contracts/openapi.yaml](docs/001-capacity-exchange-marketplace/contracts/openapi.yaml), peta endpoint terhadap requirement di [contracts/README.md](docs/001-capacity-exchange-marketplace/contracts/README.md). Salinan yang di-embed disinkronkan lewat `backend/apidocs-sync.sh`, dan CI menggagalkan build bila salinannya basi.

### API eksternal

Satu API pihak ketiga: [wilayah.id](https://wilayah.id/), sumber wilayah administratif Indonesia. Tanpa kunci API. Seluruh pemanggilannya di `backend/internal/masterdata/regions.go`, satu-satunya `http.Client` keluar di backend.

```http
GET https://wilayah.id/api/provinces.json          → 38 provinsi
GET https://wilayah.id/api/regencies/{kode}.json   → kabupaten/kota per provinsi
```

Respons dibungkus objek `data` berisi `code` dan `name`. Kecamatan dan desa tidak diambil, tidak ada requirement yang memakainya.

| Hal | Keputusan |
|---|---|
| Kapan dipanggil | Hanya `seed:regions --refresh`, timeout 30 detik, satu galat membatalkan semuanya |
| Saat melayani pengguna | Tidak pernah, wilayah dibaca dari tabel `province` dan `city` |
| Sumber bawaan | `docs/master-data/regions.json`, 38 provinsi dan 514 kabupaten/kota |
| Idempotensi | Upsert pada kode wilayah, nama diperbarui, baris tidak pernah dihapus |
| Normalisasi | `NormalizeCityCode` membuang titik, `32.73` menjadi `3273`, tanpa itu `city_code_format` dan `city_belongs_to_province` menolak |

> Bila wilayah.id mati, aplikasi tetap jalan penuh. `seed:regions` tanpa `--refresh` membaca salinan JSON di repository.

Bentuk respons dan kueri verifikasi di [docs/master-data/README.md](docs/master-data/README.md), alasannya di `research.md` R-02. Cloudflare, Mailjet, Sentry, dan WhatsApp bukan API data, dicatat di `docs/layanan-luar.md`.

---

## Testing

### Data dummy untuk pengujian

**[Devotion, data dummy produksi](https://gist.github.com/fzrilsh/80783d8b07ac57dc2af454bc8796dd0d)**, disimpan di luar repository supaya dump besar tidak ikut ke riwayat kode. Isinya 60 usaha konveksi, 47 listing, dan 34 pesanan di tujuh status, plus antrean admin yang tidak kosong. Semua fiktif, tidak ada data pribadi orang sungguhan.

| Berkas | Isi |
|---|---|
| `dummy-data.sql` | seluruh data, satu transaksi |
| `creedentials.txt` | 61 akun uji beserta sandinya |
| `copy-files.sh` | penyalin 122 berkas unggahan tiruan, Linux dan macOS |
| `copy-files.ps1` | penyalin yang sama untuk Windows |

Prasyarat impor: migrasi sudah jalan lewat `serve`, lalu `seed:regions` dan `seed:master-data`. Tabel wilayah dan `catalog_item` tidak ikut di dump karena id-nya dibuat per database sedangkan profil dan listing menunjuknya lewat kunci asing. Bila belum ada, impor berhenti dengan pesan yang menyebut seed mana yang kurang.

```bash
docker compose exec -T postgres psql -U devotion -d devotion < dummy-data.sql
```

Keluaran terakhir harus `COMMIT`. Satu galat menggagalkan seluruh transaksi, jadi tidak ada risiko data separuh jadi, dan impor kedua gagal di pelanggaran UNIQUE alih-alih menghasilkan data ganda. Waktu di dalam dump dihitung relatif terhadap saat impor, jadi kalender kapasitas selalu terlihat baru. Jalankan salah satu skrip penyalin agar 122 baris `uploaded_file` punya berkas fisiknya di `UPLOAD_PATH`, tanpa itu halaman verifikasi admin menampilkan gambar rusak.

> Hanya untuk pengembangan dan demo. Jangan diimpor ke database yang memuat data sungguhan.

### Menjalankan pengujian

```bash
cd backend
go vet ./...
go test ./... -p 1                    # -p 1 menjaga koneksi di bawah max_connections 20

# beserta uji integrasi database
DATABASE_URL_TEST=postgres://devotion:***@127.0.0.1:5434/devotion?sslmode=disable \
  go test ./... -p 1

cd ../frontend
npm run lint
npm run build
```

`DATABASE_URL_TEST` wajib disebut eksplisit: bawaannya menunjuk port 5432 sedangkan Compose menerbitkan 5434, dan uji yang tidak menjangkau database memilih `t.Skip` daripada gagal. Tanpa variabel itu seluruh uji database dilewati diam-diam dan hasilnya tetap hijau.

Uji integrasi memakai skema terpisah pada layanan Postgres yang sama, bukan container tambahan. Uji bertenggat memakai `Clock` yang dapat digantikan, sehingga konfirmasi otomatis 7 hari diuji tanpa menunggu 7 hari.

### Hasil eksekusi pada branch staging

Go 1.25.0, tanpa `DATABASE_URL_TEST` (`npx tsc -b` exit 0, `vite build` lolos di TypeScript 5.8.3).

```text
go vet ./...           lulus, tanpa temuan
go test ./... -p 1     23 paket ok, 0 gagal
                       79 lulus, 274 dilewati karena database tidak dijangkau
apidocs-sync           salinan openapi.yaml identik dengan sumber
npm run lint           lulus, tanpa galat
npm run build          lolos (tsc -b + vite build, 723 modul)
npm test               13 suite lulus, 89 uji lulus
```

### Cakupan

Bukan persentase, tapi keterlacakan: **70 berkas uji Go**, **388 kasus uji**, setiap uji menyebut FR yang diverifikasinya di nama fungsinya (`TestPencarian_UrutanDapatDiulang_FR023_FR025_SC013`), sehingga **57 requirement** dapat ditelusuri dari nama uji ke spec.

Aturan yang paling mudah rusak diam-diam, karena itu diuji khusus: urutan hasil pencarian dapat diulang termasuk antar halaman; skor bebas dari pengaruh reputasi, verifikasi, kebaruan kalender, dan jarak; kapasitas terjumlah lintas periode sampai tenggat; dua kesepakatan berbarengan atas periode yang sama hanya satu berhasil; pembatalan pra-produksi membalik seluruh baris alokasi; request kuota ke listing sendiri ditolak; konfirmasi otomatis 7 hari dan penghentiannya oleh sengketa; tingkat penyelesaian membebani hanya pihak yang membatalkan; dokumen identitas tertutup selain bagi pemilik dan admin; validasi berkas dari magic bytes beserta batas ukuran, kuota storage, dan pembuangan metadata gambar; idempotensi horizon kalender; migrasi dan constraint PostgreSQL.

Uji end-to-end dijalankan manual oleh penguji di luar tim mengikuti `docs/001-capacity-exchange-marketplace/quickstart.md` bagian F, karena itu label dan pesan galat dibuat agar dapat dikutip apa adanya di laporan.

---

## Lisensi

Proyek ini memakai [MIT License](LICENSE).

---

<div align="center">

**Dibuat untuk ITECHNO CUP 2026**

</div>
