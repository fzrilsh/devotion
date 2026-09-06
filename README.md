<div align="center">

<img src="https://gist.githubusercontent.com/TrygerZ/f6601f096885e7307b7210a750f92f7e/raw/logo-bg.png" alt="Devotion Logo" width="240" style="margin-bottom: 8px; height: auto;">

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

## 📋 Daftar Isi

- [Tentang Proyek](#-tentang-proyek)
- [Fitur Unggulan](#-fitur-unggulan)
- [Demo & Screenshot](#-demo--screenshot)
- [Teknologi](#-teknologi)
- [Arsitektur Sistem](#-arsitektur-sistem)
- [Instalasi & Setup](#-instalasi--setup)
- [Penggunaan](#-penggunaan)
- [API Documentation](#-api-documentation)
- [Testing](#-testing)
- [Tim Developer](#-tim-developer)
- [Lisensi](#-lisensi)

---

## 👥 Tim Developer

| Nama | Peran | GitHub |
|---|---|---|
| **Juan Kevin Utomo** | Project Lead | [@TrygerZ](https://github.com/TrygerZ) |
| **Fazril Syaveral Hillaby** | Backend Developer | [@fzrilsh](https://github.com/fzrilsh) |
| **Chiko Maulana Ahmad** | Frontend Developer | [@ChikoID](https://github.com/ChikoID) |

---

## 🎯 Tentang Proyek

### Latar Belakang

Bayangkan pemilik konveksi kebanjiran order 5.000 potong, sementara kapasitasnya hanya 2.000. Yang biasanya terjadi: dia menelepon satu per satu kenalan yang nomornya masih tersimpan. Kalau semuanya penuh, sisa 3.000 potong itu dilepas, padahal di kota sebelah ada konveksi yang mesinnya menganggur minggu itu.

Dua masalah bertemu di titik yang sama. Yang kelebihan order tidak tahu siapa yang sedang kosong, dan yang sedang kosong tidak punya cara memberi tahu siapa pun. Selama pencarian mitra hanya lewat relasi pribadi, jangkauannya berhenti di batas daftar kontak.

Masalahnya bertambah karena kapasitas berubah cepat. Mitra yang minggu lalu longgar bisa penuh hari ini. Tanpa kalender ketersediaan yang bisa dilihat calon pemberi order, telepon demi telepon berakhir dengan jawaban "maaf, sedang penuh".

### Solusi yang Ditawarkan

Devotion mempertemukan dua sisi yang selama ini tidak saling melihat:

- **Subkontraktor**, UMKM konveksi yang punya kapasitas produksi menganggur.
- **Pemberi order**, UMKM atau brand yang ordernya melebihi kapasitas sendiri.

Subkontraktor memasang profil usaha, mencantumkan produk dan mesin yang dikuasai, lalu mengisi kapasitas per minggu di kalender ketersediaan. Pemberi order mengisi kriteria yang dibutuhkan: produk apa, mesin apa, berapa banyak, kapan deadline-nya, sejauh mana wilayah yang masih masuk hitungan. Dari hasil pencarian dia memilih beberapa kandidat sekaligus dan mengirim satu request kuota ke semuanya. Offer yang diterima langsung menjadi work order, dan kapasitas yang dipakai tercatat per minggu.

Yang membedakan Devotion dari daftar kontak biasa adalah cara hasil pencarian disusun. Setiap kandidat dinilai dengan empat kriteria keras dan skor 0 sampai 4: produk cocok atau tidak, mesin cocok atau tidak, jeda kesiapan masih terjangkau atau tidak, dan kapasitas kalau dijumlah sampai deadline cukup atau tidak. Reputasi, badge verifikasi, kebaruan kalender, dan jarak sengaja tidak masuk skor. Konsekuensinya: urutan yang muncul selalu bisa dijelaskan alasannya, dan pencarian yang sama menghasilkan urutan yang sama.

### Tujuan Proyek

- 🎯 **Tujuan Utama**: membuat kapasitas produksi yang menganggur bisa ditemukan lewat proses terbuka, bukan lewat kenalan.
- 📊 **Target Pengguna**: pemilik konveksi yang mau menerima subkontrak, pemilik brand atau UMKM yang perlu melimpahkan order, dan admin yang menjaga operasional platform.
- 💡 **Value Proposition**: satu tempat untuk memasang, mencari, meminta, dan memantau kapasitas, dengan data yang sama dilihat kedua pihak. Tidak ada lagi versi kebenaran yang berbeda antara pemberi order dan subkontraktor.
- 🌐 **Kaitan Tema dan SDG**: kalender kapasitas membuat informasi produksi ikut bergerak setiap kali ketersediaan berubah, dan di situlah sifat adaptifnya. Devotion menyentuh **SDG 8** lewat akses order dan pemanfaatan kapasitas kerja yang tadinya terbuang, serta **SDG 9** lewat digitalisasi proses matching antar pelaku industri kecil.

### Batasan Produk

Devotion tidak menyentuh uang siapa pun. Tidak menahan, tidak menyalurkan, tidak memproses. Pembayaran terjadi langsung antar pihak, dan platform hanya mencatat pernyataan keduanya bahwa pembayaran sudah dikirim atau diterima. Tidak ada kolom nominal sama sekali di catatan itu, karena begitu ada, platform mulai terlihat seperti perantara dana.

Antarmuka seluruhnya bahasa Indonesia, tanpa layer i18n. Sasarannya UMKM konveksi domestik, jadi menambah mekanisme multi-bahasa hanya menambah kerumitan tanpa ada yang memakainya.

---

## ✨ Fitur Unggulan

### Fitur Utama

| Fitur | Deskripsi | Keunggulan |
|---|---|---|
| **Profil dan autentikasi usaha** | Registrasi dengan pilihan peran, login, session berbasis cookie `httpOnly`, verifikasi email dan nomor HP, password recovery, profil usaha dengan titik lokasi di peta. | Sebelum ada uang dan deadline yang dipertaruhkan, kedua pihak sudah tahu sedang berhadapan dengan siapa. |
| **Listing kapasitas dan kalender mingguan** | Subkontraktor mengatur kapasitas mingguan, jeda kesiapan, jenis produk, jenis mesin, visibilitas listing, dan periode ketersediaan 12 minggu ke depan. Minggu yang sudah punya alokasi terkunci dari perubahan. | Kapasitas kosong akhirnya terlihat tanpa perlu dikenal lebih dulu, dan kalender tidak bisa berubah di belakang pesanan yang sedang berjalan. |
| **Pencarian dan skor kecocokan** | Filter produk, mesin, jumlah, deadline, jeda maksimum, dan cakupan kota, provinsi, atau nasional. Skor 0 sampai 4 dari kriteria keras, kapasitas dijumlah lintas periode sampai deadline, pagination cursor dengan urutan deterministik. | Pengguna melihat kriteria mana yang tidak terpenuhi, bukan sekadar angka tanpa penjelasan. |
| **Request kuota multi-kandidat dan offer** | Satu request dikirim ke beberapa listing. Tiap kandidat punya status sendiri dan batas balasan 72 jam. Subkontraktor mengirim offer, pemberi order dapat counter-offer, rantai offer tercatat per babak. | Menawar ke lima calon mitra tidak lagi berarti lima percakapan terpisah yang harus diingat sendiri. |
| **Work order dengan alokasi kapasitas** | Offer yang diterima menjadi work order. Alokasi kapasitas ditulis dalam satu transaction dengan row lock terurut menurut minggu. | Kapasitas yang sudah dijanjikan tidak pernah terjual dua kali, bahkan ketika dua pemberi order menekan tombol terima pada detik yang sama. |
| **State machine pesanan dan auto-confirm** | Tujuh status pesanan dengan transisi yang dikirim backend lewat `allowed_transitions`. Pesanan berstatus dikirim dianggap diterima otomatis setelah 7 hari, dengan reminder 2 hari sebelumnya, dan berhenti bila ada sengketa. | Pesanan punya batas waktu sendiri, jadi subkontraktor tidak tersangkut menunggu konfirmasi yang tak pernah datang. |
| **Pembatalan, sengketa, dan mediasi admin** | Pembatalan pra-produksi membalik seluruh baris alokasi. Sengketa menghentikan auto-confirm dan masuk ke antrean mediasi admin dengan hasil dilanjutkan, dikonfirmasi selesai, atau dibatalkan. | Kalau kesepakatan gagal sebelum produksi, kapasitasnya kembali bisa dijual. Kalau berselisih, ada jalur resmi. |
| **Reputasi dan completion rate** | Review setelah pesanan selesai, nilai reputasi turunan, dan completion rate yang hanya ditampilkan sebagai persentase bila datanya cukup. Pembatalan membebani pihak yang membatalkan. | Usaha baru tidak dihukum angka reputasi yang dihitung dari dua transaksi. |
| **Panel admin** | Antrean verifikasi identitas, keputusan usulan item, pengelolaan master data, pesanan telat, sengketa, moderasi review, dan status koneksi WhatsApp beserta QR untuk menyambung ulang. | Semua urusan operasional bisa diselesaikan dari antarmuka, tanpa sekali pun membuka database. |

### Fitur Tambahan

- **Master data produk, mesin, dan wilayah** menjaga istilah pencarian tetap seragam. Kalau setiap orang mengisi jenis produk dengan kata-katanya sendiri, pencarian berhenti berfungsi.
- **Usulan item baru** untuk produk atau mesin yang belum ada di master data. Usulannya diputuskan admin, jadi daftar tetap rapi tanpa menutup kebutuhan yang belum terdaftar.
- **Verifikasi identitas usaha** lewat upload dokumen dan foto lokasi. Tipe file diperiksa dari magic bytes, bukan dari header yang bisa dipalsukan. Nama file dibuat sistem, metadata lokasi pada gambar dibuang, dan file hanya bisa diakses pemiliknya serta admin.
- **Notifikasi in-app** dengan penanda sudah dibaca, jumlah belum dibaca, dan pilihan channel email atau WhatsApp. Notifikasi transaksional tidak bisa dimatikan.
- **Rate limiting berbasis data domain** untuk percobaan login, kode verifikasi per nomor dan per alamat asal, serta request kuota per pengguna. Penegakannya di aplikasi, tidak diserahkan ke edge proxy.
- **Health check** untuk database, koneksi WhatsApp, dan ruang penyimpanan. Endpoint yang sama dipakai sebagai healthcheck container.
- **Error response konsisten** dalam format `application/problem+json` (RFC 9457), dengan 34 error code stabil dan `detail` bahasa Indonesia yang bisa dikutip penguji langsung ke laporan.
- **Swagger UI** di `/docs` saat mode development, membaca kontrak OpenAPI yang sama dengan yang di-embed ke binary.

---

## 📸 Demo & Screenshot

### Live Demo

🔗 **[https://devotion.web.id/](https://devotion.web.id/)**

### Screenshot Aplikasi

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

### Video Demo

📹 **Link video demo: belum tersedia.**

---

## 🛠️ Teknologi

### Tech Stack

#### Frontend

```text
Framework    : React 18.3.1, TypeScript 5.8
Build tool   : Vite 8.2.0, hasil build di-embed ke binary Go
UI Library   : Tailwind CSS 4.3.3
State Mgmt   : TanStack Query 5.102.1
Validation   : React Hook Form 7.86.0 + Zod 4.4.3
Routing      : React Router 7.18.2
Peta         : Leaflet 1.9.4 + tile OpenStreetMap
API client   : Fetch API, tipe di-generate dari OpenAPI
```

#### Backend

```text
Runtime      : Go 1.25.0
Framework    : net/http, router bawaan Go 1.22+
Database     : PostgreSQL 16, pgx/v5 5.7.5 + query hasil generate sqlc
Migration    : golang-migrate, otomatis saat startup di bawah advisory lock
Auth         : bcrypt, session cookie httpOnly, token disimpan sebagai hash
Notifikasi   : net/smtp ke Mailjet, whatsmeow untuk WhatsApp
Observability: log/slog JSON dengan request ID, Sentry opsional
```

#### DevOps & Tools

```text
Deployment   : Docker Compose, tepat 2 service (backend, postgres)
CI/CD        : GitHub Actions, image ke GitHub Container Registry
Edge dan TLS : Cloudflare Origin Certificate, TLS diselesaikan binary Go
Testing      : go vet, go test, Jest, ESLint, tsc
```

### Alasan Pemilihan Teknologi

Satu batasan menentukan hampir semua pilihan: aturan panitia yang membatasi service runtime menjadi dua.

| Teknologi | Alasan Pemilihan |
|---|---|
| **React + Vite** | Hasil build di-embed ke binary Go lewat `embed.FS`, sehingga frontend menjadi file statis, bukan service runtime ketiga. |
| **Go + net/http** | Satu binary, router sudah ada di standard library sejak Go 1.22, jejak memori kecil. |
| **PostgreSQL 16** | Alokasi kapasitas butuh transaction, CHECK constraint, dan row lock. Dua kesepakatan bersamaan diselesaikan database, bukan logika aplikasi. |
| **sqlc, bukan ORM** | Query pencarian dan skor kecocokan adalah inti produk. SQL ditulis eksplisit agar dapat dibaca dan diaudit. |
| **OpenAPI + openapi-typescript** | Kontrak menjadi sumber tipe frontend. Perubahan bentuk response memunculkan compile error, bukan runtime bug. |
| **Tailwind CSS** | Cukup untuk antarmuka mobile-first tanpa menambah component library kedua. |
| **Leaflet + OpenStreetMap** | Peta tanpa API key dan tanpa tagihan. Jarak bersifat informatif dan tidak memengaruhi skor. |

### Dependencies Utama

```json
{
  "dependencies": {
    "react": "^18.3.1",
    "@tanstack/react-query": "^5.102.1",
    "react-hook-form": "^7.86.0",
    "zod": "^4.4.3",
    "react-router-dom": "^7.18.2",
    "tailwindcss": "^4.3.3",
    "leaflet": "^1.9.4"
  }
}
```

```text
Backend  github.com/jackc/pgx/v5 v5.7.5
         github.com/golang-migrate/migrate/v4 v4.18.3
         golang.org/x/crypto v0.54.0
         go.mau.fi/whatsmeow
         github.com/getsentry/sentry-go v0.35.3
```

Daftar backend pendek karena disengaja. Structured logging (`log/slog`), email (`net/smtp`), pembuangan metadata gambar (`image/jpeg`), token acak (`crypto/rand`), UUID (`gen_random_uuid()`), dan jarak haversine semuanya diselesaikan standard library.

---

## 🏗️ Arsitektur Sistem

### System Architecture

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
| **Satu proses, dua peran** | Static handler membaca `embed.FS`; path selain `/api` jatuh ke SPA fallback, `/api` tak dikenal membalas 404 JSON. | Frontend bukan service runtime, dan salah tulis endpoint tetap menghasilkan error JSON. |
| **Role gate wajib** | Setiap pola `/api` terdaftar publik atau bergerbang; `UncoveredAPIRoutes()` diperiksa saat `serve`. | Proses menolak menyala bila ada pola tanpa keputusan peran. |
| **Scheduler in-process** | `time.Ticker` di binary yang sama, tiap job dibungkus advisory lock, deadline juga dievaluasi saat data dibaca. | Tanpa worker terpisah, dan job tidak dieksekusi ganda. |
| **Migrasi terkunci** | `golang-migrate` saat startup di bawah `pg_try_advisory_lock`. | Deployment tanpa langkah migrasi manual, dua instance tidak saling menimpa schema. |

### Database Schema

**26 tabel domain**, plus `schema_migrations` dan tabel milik `whatsmeow`. ERD dipecah per konteks. Atribut dibatasi pada key dan kolom yang menentukan perilaku; `created_at` dan `updated_at` tidak diulang.

<details>
<summary><b>Identitas, wilayah, dan verifikasi (7 tabel)</b></summary>

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

</details>

<details>
<summary><b>Katalog, listing, dan kalender kapasitas (6 tabel)</b></summary>

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

</details>

<details>
<summary><b>Request kuota, offer, dan work order (10 tabel)</b></summary>

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

Tiga aturan harus membaca tabel lain, jadi ditegakkan trigger bukan `CHECK`: `trg_reject_self_request` (kandidat tidak boleh sama dengan pembeli), `trg_reject_allocation_before_readiness` (alokasi tidak boleh sebelum `readiness_week_start`), dan `trg_reject_wrong_product_item` / `trg_reject_wrong_machine_item` (setiap baris terikat ke tipe item yang benar).

</details>

<details>
<summary><b>Notifikasi dan rate limiting (3 tabel)</b></summary>

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

`rate_limit` tanpa foreign key. `key` menyimpan pengenal sasaran sebagai teks (id akun, nomor, atau alamat asal) sesuai `target`, sehingga pembatasan tetap berlaku bagi pihak yang belum punya akun.

</details>

#### Keputusan Schema

| Keputusan | Penerapan | Alasan |
|---|---|---|
| **Uang bilangan bulat** | `offer.total_price` dan `work_order.total_price` `bigint`, `CHECK (total_price > 0)`. | Rupiah tidak dipecah di B2B ini, dan tipe pecahan menimbulkan rounding error. |
| **Minggu selalu Senin** | `week_start`, `horizon_until`, `readiness_week_start` `date` dengan `CHECK (EXTRACT(ISODOW ...) = 1)`. | Periode tidak jatuh di tengah minggu, jadi kapasitas tidak berkurang dari periode salah. |
| **Platform tidak memegang dana** | `payment_record` tanpa kolom nominal, hanya arah dan tanggal, unik per pesanan, pihak, dan arah. | Platform mencatat pernyataan kedua pihak tanpa jadi perantara dana. |
| **Kapasitas tidak terjual dua kali** | `capacity_allocation` unik per `work_order_id` dan `period_id`, `used_capacity <= total_capacity`, satu transaction dengan `SELECT ... FOR UPDATE` terurut `week_start`. | Batas ditegakkan database, dan urutan lock seragam mencegah deadlock. |
| **Audit trail lengkap** | `dispute`, `item_proposal`, `verification_request`, `review` memakai CHECK gabungan: kolom pendukung wajib terisi begitu status keluar dari pending. | Status terminal tidak tersimpan tanpa catatan admin, waktu, dan pelaku. |
| **Token tidak disimpan mentah** | `session.token_hash` dan `verification_code.code_hash` `bytea` berisi hash. | Kebocoran isi tabel tidak langsung berarti session hijacking. |

Definisi lengkap beserta index dan constraint ada di [data-model.md](docs/001-capacity-exchange-marketplace/data-model.md) dan `backend/db/migrations/`.

### Folder Structure

```text
devotion/
├── backend/
│   ├── cmd/devotion/       # serve, admin:create, seed:*, reset:*, user:verify, health:check
│   ├── apidocs/            # Swagger UI dan salinan openapi.yaml yang di-embed
│   ├── internal/
│   │   ├── platform/       # clock, config, httpx, session, storage, scheduler,
│   │   │                   # ratelimit, cloudflare, health, migrate, observability, tlsconf
│   │   ├── account/        # akun, peran, profil, autentikasi, recovery
│   │   ├── verification/   # upload file, pengajuan, keputusan admin
│   │   ├── masterdata/     # master data, wilayah, usulan item
│   │   ├── listing/        # listing kapasitas, kalender ketersediaan
│   │   ├── search/         # kriteria keras, skor, tie-breaker, keyset pagination
│   │   ├── quota/          # request kuota, offer, counter-offer
│   │   ├── order/          # work order, alokasi, pembatalan, pembayaran, sengketa
│   │   ├── reputation/     # review, nilai turunan, moderasi
│   │   ├── notification/   # queue, pengiriman, retry
│   │   ├── admin/          # status dan koneksi WhatsApp
│   │   └── db/             # pool 15 koneksi, sqlcgen, testdb schema uji terpisah
│   ├── db/migrations/      # 22 migrasi golang-migrate
│   ├── db/queries/         # 15 file SQL sumber sqlc
│   └── webdist/            # hasil build frontend, di-embed lewat embed.FS
├── frontend/src/
│   ├── api/                # fetch client dan tipe hasil generate dari OpenAPI
│   ├── components/         # layout, section, komponen bersama
│   ├── pages/              # 38 halaman, dikelompokkan per user story
│   ├── hooks/              # hook TanStack Query per domain
│   ├── schemas/            # schema Zod
│   ├── routes/             # GuestRoute, ProtectedRoute, UnverifiedRoute
│   ├── providers/          # QueryProvider
│   ├── lib/                # helper murni
│   ├── test/               # setup dan utilitas Jest
│   └── styles/             # entri Tailwind dan style global
├── docs/                   # spec, plan, data-model, contracts, tasks, panduan operasional
├── .github/workflows/ci.yml
├── docker-compose.yml
├── .env.example
├── LICENSE
└── README.md
```

---

## ⚙️ Instalasi & Setup

### Prerequisites

Pastikan Anda telah menginstall:
- **Git**
- **Docker Engine** dengan Compose v2
- **Go** 1.25.0 atau lebih baru
- **Node.js** 20 atau lebih baru

### Langkah Instalasi

#### 1️⃣ Clone Repository

```bash
git clone https://github.com/fzrilsh/devotion.git
cd devotion
```

#### 2️⃣ Setup Environment Variables

```bash
cp .env.example .env
```

Wajib di semua environment hanya empat: `APP_ENV`, `APP_BASE_URL`, `DATABASE_URL`, `UPLOAD_PATH`. Sisanya diwajibkan saat `APP_ENV=production`.

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

> Port `5434` bukan salah tulis: Compose menerbitkan Postgres di `127.0.0.1:5434` agar tidak bertabrakan dengan Postgres lain di 5432, dan ikatan loopback menjaganya tidak terekspos keluar mesin.

Pengaturan TLS, Mailjet, WhatsApp, dan Sentry ada di [quickstart.md](docs/001-capacity-exchange-marketplace/quickstart.md). File `.env` tidak pernah di-commit.

#### 3️⃣ Setup Database

```bash
mkdir -p /absolute/path/to/devotion/backend/uploads
docker compose up -d postgres
```

Tanpa langkah migrasi manual. Migrasi jalan saat backend menyala, di bawah advisory lock.

#### 4️⃣ Run Backend

Binary Go membaca environment shell, bukan file `.env`; yang membaca `.env` hanya Docker Compose. Tanpa export, `serve` berhenti dengan `variabel lingkungan wajib belum diisi: APP_ENV`.

```bash
set -a; . ./.env; set +a
cd backend
go run ./cmd/devotion serve
```

Backend mendengarkan di `http://localhost:8080`. Swagger UI di `/docs`, hanya saat `APP_ENV=development`.

#### 5️⃣ Seed Data dan Buat Akun Admin

Sekali per database baru, dari `backend/` dengan environment yang sama.

```bash
go run ./cmd/devotion seed:regions
go run ./cmd/devotion seed:master-data
go run ./cmd/devotion admin:create
go run ./cmd/devotion seed:test-data      # menolak jalan bila APP_ENV=production
```

Data wilayah dibaca dari salinan di repository, tanpa memanggil layanan luar.

#### 6️⃣ Run Development Server

```bash
cd frontend
npm install
npm run dev
```

Aplikasi akan berjalan di `http://localhost:5173` dan memproksikan `/api` ke port 8080.

> Proxy ini wajib: tanpanya frontend dan backend terlihat sebagai origin berbeda, cookie `SameSite=Lax` tidak terkirim, dan setiap request tampak belum login meski login berhasil.

---

## 🚀 Penggunaan

### Menjalankan Aplikasi

```bash
# Development mode, backend
cd backend && go run ./cmd/devotion serve

# Development mode, frontend (terminal lain)
cd frontend && npm run dev

# Production build
npm run build

# Run tests
npm run test

# Linting
npm run lint
```

Untuk production, CI membangun frontend, menyalin hasilnya ke `backend/webdist/`, lalu membangun image. Server hanya menarik dan menjalankan, supaya proses build tidak berebut resource dengan Postgres yang sedang hidup.

### User Guide

#### Untuk Subkontraktor

1. **Registrasi**: daftar sebagai **Subkontraktor** atau **Keduanya**, selesaikan verifikasi email dan nomor HP.
2. **Profil usaha**: lengkapi profil beserta titik lokasi di peta.
3. **Verifikasi identitas**: ajukan untuk mendapat badge terverifikasi. Opsional, tapi menaikkan kepercayaan calon pemberi order.
4. **Buat listing**: produk yang dikuasai, mesin dan jumlah unit, kapasitas per minggu, jeda kesiapan.
5. **Isi kalender**: ketersediaan 12 minggu ke depan, tandai minggu yang penuh.
6. **Terbitkan listing**: sebelum diterbitkan, listing tidak muncul di pencarian.
7. **Balas request kuota**: kirim offer berisi harga total dan jeda kesiapan. Batas 72 jam, lewat itu dianggap tidak membalas.
8. **Jalankan pesanan**: lewat status produksi, selesai, dikirim. Catat pernyataan pembayaran, buka sengketa bila ada ketidaksesuaian.

#### Untuk Pemberi Order

1. **Registrasi**: masuk sebagai **Pemberi Order** atau **Keduanya**.
2. **Cari kapasitas**: isi kriteria produk, mesin, jumlah, deadline, jeda maksimum, cakupan wilayah.
3. **Bandingkan kandidat**: lihat skor 0 sampai 4 beserta kriteria mana yang tidak terpenuhi.
4. **Kirim request kuota**: pilih beberapa kandidat, satu request untuk semuanya.
5. **Terima offer**: bandingkan, counter-offer bila perlu, terima satu offer. Work order terbentuk dan kapasitas mitra langsung terpakai.
6. **Konfirmasi penerimaan**: bila dibiarkan, pesanan dianggap diterima otomatis setelah 7 hari, dengan reminder 2 hari sebelumnya.
7. **Beri review** setelah pesanan selesai.

#### Untuk Admin

1. **Akses Admin Panel**: akun admin pertama dibuat lewat `go run ./cmd/devotion admin:create`, bukan pendaftaran biasa. Panel ada di `/admin`.
2. **Verifikasi dan master data**: antrean verifikasi identitas, keputusan usulan item, pengelolaan master data produk dan mesin.
3. **Mediasi dan moderasi**: pesanan telat, sengketa beserta mediasi dan penyelesaiannya, moderasi review, status koneksi WhatsApp dengan QR untuk menyambung ulang.

Nomor layanan WhatsApp tidak pernah ditampilkan di antarmuka.

---

## 📚 API Documentation

### Base URL

```text
Development  : http://localhost:8080/api
Production   : ditentukan oleh APP_BASE_URL
Health check : /api/health
Swagger UI   : /docs, hanya saat APP_ENV=development
```

Semua request memakai session cookie `httpOnly`, tanpa token di body response. Path `/api` tak dikenal membalas 404 JSON, bukan `index.html`, sehingga salah tulis endpoint tetap dapat didiagnosis.

### Endpoints

Kontrak memuat **66 operasi pada 58 path**. Setiap pola `/api` wajib punya keputusan peran, publik atau bergerbang, dan `serve` menolak menyala bila ada satu pola tanpa keputusan itu.

#### Authentication

```http
POST  /api/auth/register              POST  /api/auth/login
POST  /api/auth/verify-email          POST  /api/auth/logout
POST  /api/auth/verify-phone          POST  /api/auth/recover/request
POST  /api/auth/resend-code           POST  /api/auth/recover/confirm
GET   /api/me                         PATCH /api/me/roles
GET   /api/profile/me                 PUT   /api/profile/me
GET   /api/profile/{profileId}        GET   /api/profile/{profileId}/reviews
```

#### Files, Verification, Master Data

```http
POST /api/files                       GET  /api/files/{fileId}
POST /api/verification                GET  /api/verification
GET  /api/master/products             GET  /api/master/machines
GET  /api/regions/provinces           GET  /api/regions/cities
POST /api/master/proposals
```

#### Listing & Capacity Calendar

```http
GET /api/listing/me                   POST /api/listing/me
PUT /api/listing/me                   PUT  /api/listing/me/visibility
GET /api/listing/me/periods           PUT  /api/listing/me/periods
```

#### Search, Quota Request, Offer

```http
GET  /api/search
POST /api/quota-requests              GET  /api/quota-requests
GET  /api/quota-requests/incoming     GET  /api/quota-requests/{requestId}
POST /api/candidates/{candidateId}/offers
POST /api/candidates/{candidateId}/reject
POST /api/offers/{offerId}/counter    POST /api/offers/{offerId}/accept
```

#### Work Order

```http
GET  /api/work-orders                 GET  /api/work-orders/{workOrderId}
POST /api/work-orders/{workOrderId}/status
POST /api/work-orders/{workOrderId}/confirm
POST /api/work-orders/{workOrderId}/cancel
POST /api/work-orders/{workOrderId}/payments
POST /api/work-orders/{workOrderId}/disputes
POST /api/work-orders/{workOrderId}/reviews
```

#### Notification & Admin

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

### Example Request

```bash
# Login, simpan session cookie
curl -i -c cookies.txt \
  -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}'

# Pakai cookie untuk endpoint bergerbang
curl -i -b cookies.txt http://localhost:8080/api/me
```

### Error Format

Semua error memakai `application/problem+json` (RFC 9457) dengan **34 error code** stabil yang bisa di-switch klien.

```json
{
  "type": "about:blank",
  "title": "Kapasitas tidak mencukupi",
  "status": 409,
  "code": "INSUFFICIENT_CAPACITY",
  "detail": "Kapasitas subkontraktor tidak cukup sampai tenggat yang diminta."
}
```

### Dokumentasi Lengkap

📖 Kontrak OpenAPI 3.1 di [contracts/openapi.yaml](docs/001-capacity-exchange-marketplace/contracts/openapi.yaml), peta endpoint terhadap requirement di [contracts/README.md](docs/001-capacity-exchange-marketplace/contracts/README.md). Salinan yang di-embed disinkronkan lewat `backend/apidocs-sync.sh`, dan CI menggagalkan build bila salinannya basi.

### API Eksternal

Satu API pihak ketiga: [wilayah.id](https://wilayah.id/), sumber wilayah administratif Indonesia. Tanpa API key. Seluruh pemanggilannya di `backend/internal/masterdata/regions.go`, satu-satunya `http.Client` keluar di backend.

```http
GET https://wilayah.id/api/provinces.json          → 38 provinsi
GET https://wilayah.id/api/regencies/{kode}.json   → kabupaten/kota per provinsi
```

| Hal | Keputusan |
|---|---|
| Kapan dipanggil | Hanya `seed:regions --refresh`, timeout 30 detik, satu error membatalkan semuanya |
| Saat melayani pengguna | Tidak pernah, wilayah dibaca dari tabel `province` dan `city` |
| Sumber bawaan | `docs/master-data/regions.json`, 38 provinsi dan 514 kabupaten/kota |
| Idempotensi | Upsert pada kode wilayah, nama diperbarui, baris tidak pernah dihapus |

> Bila wilayah.id mati, aplikasi tetap jalan penuh. `seed:regions` tanpa `--refresh` membaca salinan JSON di repository.

Cloudflare, Mailjet, Sentry, dan WhatsApp bukan API data, dicatat di [layanan-luar.md](docs/layanan-luar.md).

---

## 🧪 Testing

### Data Dummy untuk Pengujian

**[Devotion, data dummy produksi](https://gist.github.com/fzrilsh/80783d8b07ac57dc2af454bc8796dd0d)**, disimpan di luar repository supaya dump besar tidak ikut ke riwayat kode. Isinya 60 usaha konveksi, 47 listing, dan 34 pesanan di tujuh status, plus antrean admin yang tidak kosong. Semua fiktif, tidak ada data pribadi orang sungguhan.

| File | Isi |
|---|---|
| `dummy-data.sql` | seluruh data, satu transaction |
| `creedentials.txt` | 61 akun uji beserta passwordnya |
| `copy-files.sh` | penyalin 122 file upload tiruan, Linux dan macOS |
| `copy-files.ps1` | penyalin yang sama untuk Windows |

Prasyarat impor: migrasi sudah jalan lewat `serve`, lalu `seed:regions` dan `seed:master-data`.

```bash
docker compose exec -T postgres psql -U devotion -d devotion < dummy-data.sql
```

Output terakhir harus `COMMIT`. Satu error menggagalkan seluruh transaction, jadi tidak ada risiko data separuh jadi. Jalankan salah satu script penyalin agar 122 baris `uploaded_file` punya file fisiknya di `UPLOAD_PATH`.

> Hanya untuk development dan demo. Jangan diimpor ke database yang memuat data sungguhan.

### Running Tests

```bash
# Unit tests, backend
cd backend
go vet ./...
go test ./... -p 1

# Integration tests, backend (menjangkau PostgreSQL)
DATABASE_URL_TEST=postgres://devotion:***@127.0.0.1:5434/devotion?sslmode=disable \
  go test ./... -p 1

# Test coverage, backend
go test ./... -p 1 -coverprofile=cover.out
go tool cover -func=cover.out | tail -1

# Unit tests, frontend
cd ../frontend
npm run test -- --testTimeout=30000

# Test coverage, frontend
npm run test:coverage -- --testTimeout=30000

# Linting dan type check
npm run lint
npm run build
```

> `DATABASE_URL_TEST` wajib disebut eksplisit: bawaannya menunjuk port 5432 sedangkan Compose menerbitkan 5434, dan test yang tidak menjangkau database memilih `t.Skip` daripada gagal. Tanpa variabel itu seluruh test database dilewati diam-diam dan hasilnya tetap hijau.

> `--testTimeout=30000` wajib untuk suite OTP (`VerifyEmail`, `VerifyPhone`). Test itu mensimulasikan pengetikan enam digit lewat `userEvent`, dan pada mesin yang lambat prosesnya melewati timeout default Jest 5000 ms. Tanpa flag ini dua suite gagal karena timeout, bukan karena logika salah.

Integration test memakai schema terpisah pada service PostgreSQL yang sama, bukan container tambahan. Test berdeadline memakai `Clock` yang dapat digantikan, sehingga auto-confirm 7 hari diuji tanpa menunggu 7 hari.

---
### Test Coverage

Hasil eksekusi pada branch `staging`, commit `8f7bb51`, Go 1.25.0 dengan `DATABASE_URL_TEST` aktif.

**Backend (Go)**

```text
Statements   : 68.1% (kode aplikasi, 3.546/5.204)
             : 55.4% (termasuk sqlcgen hasil generate)
Packages     : 23 ok, 0 gagal
Test files   : 75 file, 419 fungsi Test
go vet       : lulus, tanpa temuan
```

**Frontend (TypeScript)**

```text
Statements   : 76.87% (768/999)
Branches     : 58.42% (312/534)
Functions    : 64.91% (161/248)
Lines        : 77.31% (702/908)
Test suites  : 26 lulus, 159 test lulus
ESLint       : lulus, tanpa error
tsc + build  : lulus (TypeScript 5.8, 723 modul)
```

Angka backend disajikan dua kali dengan alasan: `internal/db/sqlcgen` berisi 1.132 statement hasil generate sqlc yang tidak pernah dipanggil langsung oleh test, dan menyertakannya menurunkan angka tanpa menambah informasi. **68,1%** adalah coverage kode yang benar-benar ditulis tim.

**Coverage per paket backend, lima tertinggi dan tiga terendah**

| Paket | Coverage |
|---|---|
| `apidocs` | 100,0% |
| `platform/health` | 97,9% |
| `platform/cloudflare` | 95,0% |
| `platform/config` | 91,3% |
| `platform/httpx` | 90,1% |
| `internal/quota` | 81,9% |
| `internal/search` | 80,2% |
| ... | ... |
| `internal/admin` | 50,9% |
| `platform/scheduler` | 43,3% |
| `cmd/devotion` | 34,7% |

Tiga paket terbawah memang bukan target utama: `cmd/devotion` sebagian besar wiring subcommand, `scheduler` diverifikasi lewat integration test pada domain yang dipicunya, dan `admin` sebagian besar CRUD tipis di atas query yang sudah teruji di paketnya masing-masing.

---

## 📄 Lisensi

Proyek ini dilisensikan di bawah [MIT License](LICENSE) - lihat file LICENSE untuk detail lebih lanjut.

---

<div align="center">

**Made with ❤️ by Indonesia Emas 74 Kg for ITECHNO CUP 2026**

</div>
