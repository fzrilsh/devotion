# Specification Quality Checklist: Capacity Exchange, Marketplace Subkontrak Kapasitas Konveksi (MVP)

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-21
**Last Revised**: 2026-08-22
**Feature**: `docs/001-capacity-exchange-marketplace/spec.md` (91 FR, revisi 2026-08-22)

## Content Quality

- [ ] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

### Hasil penilaian ulang: 16/16 → 15/16

Satu butir turun status. Ini bukan regresi akibat revisi 2026-08-22. Butir itu **seharusnya sudah gagal sejak versi sebelumnya**, dan penilaian 16/16 yang saya berikan waktu itu tidak akurat. `/analyze` yang menemukannya (temuan W-3).

| Butir | Sebelum | Sekarang | Sebab |
|-------|---------|----------|-------|
| No implementation details | `[x]` | `[ ]` | Dua requirement memuat cara secara sadar; tiga sisanya sudah dibersihkan |
| No implementation details leak (Feature Readiness) | `[x]` | `[x]` | Tetap lolos, lihat penjelasan di bawah |

Dua butir berbunyi mirip tetapi menilai hal berbeda. **Content Quality** menilai apakah spec menyebut mekanisme teknis di mana pun; **Feature Readiness** menilai apakah kebocoran itu sampai mengunci pilihan teknologi sehingga spec tidak lagi netral. Yang pertama gagal, yang kedua masih lolos karena tidak satu pun kebocoran menyebut bahasa, framework, atau produk tertentu.

### Lima requirement yang pernah bocor, dua yang masih dipertahankan

Tiga sudah dibersihkan (FR-036, FR-075, FR-079); dua dipertahankan secara sadar karena keterujiannya menuntutnya (FR-077, FR-088).

**FR-036** dahulu berbunyi "dengan mengunci baris periode yang terlibat di dalam transaksi yang sama dengan pembentukan pesanan", yang menyebut penguncian baris dan transaksi, dua mekanisme basis data. Sudah dibersihkan pada revisi 2026-08-22 menjadi pernyataan perilaku murni: hanya satu kesepakatan yang berhasil, dan pihak yang gagal diberi tahu beserta alasannya. Caranya ada di `research.md` R-04, tempatnya yang benar. Disebut di sini sebagai riwayat.

**FR-075** ada dua istilah teknis: "satu perintah sekali jalan" adalah CLI dan "basis data sendiri" adalah teknologi penyimpanan. Sudah diperbaiki menjadi "satu tindakan" dan "menyimpan salinannya sendiri"; maknanya utuh dan keterujiannya tidak berkurang, sehingga berbeda dari FR-077 dan FR-088 yang dipertahankan secara sadar.

**FR-079** dahulu berbunyi "ditegakkan pada tingkat penyimpanan data sehingga tetap berlaku meskipun logika aplikasi keliru". Sudah dibersihkan pada revisi 2026-08-22 menjadi pernyataan perilaku murni. Disebut di sini sebagai riwayat.

**FR-077** berbunyi "mencatat alokasi kapasitas sebagai baris tersendiri per pasangan pesanan dan periode mingguan". Kata "baris" adalah istilah basis data. Pembelaan yang dapat diterima: tanpa menyatakan alokasi terpisah per periode, FR-020 (membalik seluruh alokasi saat pembatalan) tidak dapat dinyatakan secara testabel. Ini kebocoran yang **dipertahankan secara sadar**, bukan kelalaian.

**FR-088** berbunyi "membuat periode yang belum ada sampai periode yang memuat deadline". Menyiratkan penyimpanan periode sebagai catatan yang dibuat dan tidak dibuat. Sama seperti FR-077: dipertahankan karena SC-021 tidak dapat diuji tanpanya.

### Kenapa butir ini dibiarkan gagal alih-alih diperbaiki

FR-077 dan FR-088 bisa dibersihkan secara redaksional, tetapi hasilnya akan menjadi requirement yang lebih kabur dan lebih sulit diuji, dan konstitusi menempatkan keterujian di atas kemurnian bentuk. Menandai butir ini `[ ]` beserta empat contohnya lebih jujur daripada menulis ulang requirement sampai kehilangan ketajamannya, atau membiarkan `[x]` yang tidak akurat.

Butir ini **tidak memblokir implementasi**. Yang memblokir adalah requirement yang tidak dapat diuji atau bertentangan, dan tidak ada lagi yang seperti itu setelah C-1 sampai C-4 ditutup.

### Yang diperiksa ulang dan tetap lolos

**Requirements are testable and unambiguous**, sebelumnya tertahan oleh empat kontradiksi yang ditemukan `/analyze`. Keempatnya kini punya requirement penyelesai: FR-087 (minggu kesiapan mulai), FR-088 (horizon diperpanjang sampai deadline), FR-023 yang diperjelas (kriteria yang filternya tidak diisi dihitung terpenuhi), dan penghapusan `DEFAULT now()` yang memindahkan seluruh sumber waktu ke `Clock` sehingga alur bertenggat dapat diuji.

**Scope is clearly bounded**, dua penambahan memperjelas alih-alih memperluas: FR-089 (propagasi perubahan kapasitas mingguan) dan FR-091 (penggolongan notifikasi transaksional versus non-transaksional). Keduanya menutup pertanyaan yang sebelumnya menggantung, bukan menambah fitur baru.

**Success criteria are technology-agnostic**, 21 kriteria diperiksa satu per satu. SC-018 menyebut "dua kesepakatan yang terjadi berbarengan" dan SC-013 menyebut "halaman demi halaman"; keduanya perilaku yang dapat diamati pengguna, bukan mekanisme. Lolos.

**Edge cases are identified**, 34 butir, naik dari 31. Empat kasus jeda kesiapan mulai yang sebelumnya hanya tercatat sebagai edge case tanpa requirement penyelesai kini punya FR-087 dan FR-090.

### Yang tetap perlu ditinjau pemilik produk

Bukan kegagalan checklist, tetapi keputusan yang saya ambil dan belum divalidasi:

- **Ambang 3 pesanan** sebelum tingkat penyelesaian tampil sebagai persentase (FR-073), angka pilihan penulis spec.
- **Tenggat 7 hari** konfirmasi otomatis (FR-068) dan **72 jam** batas balasan request (FR-082), keduanya angka pilihan penulis spec, diasumsikan cukup untuk kebiasaan pengiriman dan berbalas antar kota di Indonesia.
- **Satu listing per profil usaha**, penyederhanaan model yang diambil karena seluruh Acceptance Scenario memakai bentuk tunggal. Spec tidak pernah menyatakan batas ini secara eksplisit; bila satu konveksi perlu beberapa listing terpisah, spec harus diubah lebih dulu.
- **`admin_tidak_berperan_usaha`**, pemisahan admin dari peran usaha tidak diminta FR mana pun. Alasannya konflik kepentingan pada FR-005, tetapi ia menutup kasus admin yang juga punya konveksi.

### Empat penyimpangan dari dokumen sumber

Tercatat lengkap di Assumptions spec, diringkas di sini karena semuanya keputusan sadar yang mengurangi mitigasi yang dokumen sumber tetapkan:

**Escrow tidak dibangun.** Modul transaksi dokumen sumber memuat penahanan dana yang dirilis saat pesanan dikonfirmasi selesai; versi ini menggantinya dengan pencatatan pernyataan pembayaran, sehingga mediasi admin kehilangan salah satu daya paksanya. Mediasi admin sendiri memang jalur yang dipilih dokumen sumber untuk fase awal, karena penanganan sengketa secara legal formal menuntut tim hukum dan asuransi [1].

**Verifikasi identitas bukan gerbang.** Dokumen sumber menempatkan validasi manual oleh admin sebagai sub-fitur Verifikasi Identitas UMKM bersama unggah NIB/NIK dan foto lokasi usaha [1]; di versi ini keputusan admin hanya menambah lencana dan tidak menahan listing dari tayang.

**Skor kecocokan tanpa faktor perilaku.** Pencarian tetap mengurutkan hasil berdasarkan skor kecocokan seperti yang diminta dokumen sumber [1], tetapi hanya dari empat kriteria keras. Penalti peringkat bagi subkontraktor yang tidak memperbarui kalender tidak dipakai, dan penegakannya hanya lewat pengingat serta penanda "Data Belum Diperbarui".

**Perlindungan data identitas dibatasi pada kontrol akses.** Enkripsi tingkat penyimpanan dan pengujian penetrasi berkala tidak tercakup; yang ditegakkan adalah pembatasan akses berbasis peran, dibuktikan lewat pengujian akses otomatis.

Dua non-goal dokumen sumber diikuti tanpa penyimpangan: verifikasi kualitas produk secara fisik diganti sistem rating dan sampel foto, dan dukungan multi-bahasa ditunda karena fokus pasar domestik [1].

### Status akhir

15/16 lolos. Satu butir yang gagal bersifat kualitas redaksional dan tidak memblokir implementasi. Tidak ada penanda `[NEEDS CLARIFICATION]` tersisa, tidak ada requirement yang bertentangan, dan seluruh 91 FR punya task pemilik di `tasks.md`.