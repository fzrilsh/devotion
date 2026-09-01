# Changelog

## [Unreleased]

### Diperbaiki

* Menyatukan aturan tombol Terima Penawaran di halaman request terkirim dan masuk ke fungsi bersama `canAcceptOffer` di `lib/offers.ts`, sehingga buyer dan subcontractor memakai predikat yang sama dan tidak menilai ulang status negosiasi secara berbeda. Aturan ini membatasi aksi menerima hanya saat status `offered` dan `latest.offer_id` benar-benar ada, jadi tombol tidak lagi muncul untuk ronde yang tidak valid dan backend `FORBIDDEN` tetap menjadi satu-satunya sumber kebenaran untuk penolakan oleh pihak yang mengajukan ronde terakhir.

* Menambahkan komentar invariant pada `SentDetail.tsx` dan `lib/offers.ts` untuk menjaga dua keputusan penting tetap terlihat saat refactor: `acceptMutation` sengaja dimiliki induk karena satu penerimaan membuat pesanan bersama, menonaktifkan kandidat lain, dan menampilkan pesanan yang dibuat; `canAcceptOffer` sengaja mensyaratkan status `offered`, bukan sekadar status non-terminal, serta hanya menyaring tampilan. Backend tetap menjadi penegak aturan sebenarnya melalui `FORBIDDEN`/`own_offer` pada penerimaan penawaran.

* Meregenerasi `src/api/types.ts` dari `openapi.yaml` agar `POST /offers/{offerId}/accept` memuat respons `403` yang menjelaskan bahwa pihak yang mengajukan penawaran terakhir tidak boleh menerima tawarannya sendiri dan bahwa persetujuan harus datang dari pihak lawan (FR-033). Ini menjaga `Problem` yang dipakai UI tetap sinkron dengan kontrak backend dan mencegah tipe yang ketinggalan dari versi kontrak lama.

* Memperbaiki daftar aksi work-order sehingga `allowed_transitions` difilter berdasarkan sisi pengguna (`buyer` atau `subcontractor`) melalui `transitionsForSide` sebelum tombol ditampilkan. Ini adalah filter tampilan, bukan state machine kedua: halaman hanya menampilkan transisi yang sudah diizinkan backend, dan backend tetap menjadi sumber kebenaran untuk apa yang valid. Ketika server mempersempit kumpulan transisi, UI mempersempitnya ke kumpulan yang sama untuk pihak yang sedang bertindak, bukan membuat atau menghitung ulang kumpulan aturan terpisah.

* Menghapus pemilih arah pembayaran dari panel pembayaran work-order dan menentukan arah pernyataan berdasarkan sisi pengguna melalui `paymentDirectionForSide` dan `paymentDirectionLabel`. Pengguna tidak pernah memilih “saya sudah membayar” atau “saya sudah menerima pembayaran” sebagai pilihan bebas, karena arahnya ditentukan dari siapa yang sedang melihat order. Ini sesuai dengan kontrak FR-043, ketika sistem membandingkan dua pernyataan pembayaran dari kedua pihak untuk mendeteksi ketidaksesuaian; buyer yang menyatakan bahwa ia telah menerima pembayaran bukan merupakan pernyataan yang nyata dan tidak boleh ditawarkan sebagai aksi yang dapat dipilih.

### Ditambahkan

* Penawaran yang dikembalikan oleh `POST /candidates/{candidateId}/offers` dan `POST /offers/{offerId}/counter` disimpan selama sisa sesi penjelajahan di `lib/offerSession.ts`, dengan kunci `candidateId` dan dibaca melalui `useSyncExternalStore`. Tanpa mekanisme ini, ronde yang baru saja dikirim subcontractor menghilang dari layar segera setelah daftar masuk dimuat ulang, karena daftar tersebut tidak membawa rantai penawaran. Hanya data yang benar-benar dikembalikan backend yang disimpan, nilai `offer_id` duplikat diabaikan, dan penyimpanan hanya hidup di memori: tidak ada `localStorage`, tidak ada `sessionStorage`, sehingga pemuatan ulang halaman akan mengosongkannya dan halaman kembali ke kondisi "riwayat tidak tersedia", bukan merekonstruksi ronde yang tidak pernah dikirim server.

* `LocationPicker` mendapatkan kotak pencarian tempat. Pengetikan nama kota, jalan, atau tempat akan melakukan kueri ke Nominatim, layanan geocoding dari proyek OpenStreetMap yang sama yang menyediakan ubin peta, sehingga tidak diperlukan API key dan tidak ada dependensi npm baru: `lib/geocode.ts` menampung client dan `hooks/usePlaceSearch.ts` melakukan debounce input selama 600 ms dengan `AbortController`, tetap berada dalam batas penggunaan wajar satu permintaan per detik. Memilih hasil hanya menggeser dan memperbesar peta; titik pin tidak pernah dipindahkan, karena titik yang disimpan ditampilkan secara publik (FR-057) dan harus merupakan titik yang sengaja ditandai pengguna, bukan centroid suatu wilayah yang kebetulan namanya cocok. Pencarian yang gagal menyatakan kegagalan tersebut dan mengarahkan pengguna kembali untuk mengeklik peta. Pencarian dan "gunakan lokasi saya" sekarang berada di dalam peta sebagai kontrol Leaflet yang dirender melalui React portal, sehingga keduanya mempertahankan state masing-masing sambil berada di luar subtree DOM `MapContainer`: pencarian berada di kiri bawah (diciutkan menjadi pil sampai diklik, dengan daftar hasil yang berkembang ke atas agar panel tidak pernah melampaui tepi bawah peta) dan tombol lokasi berada di kanan atas, sementara kontrol zoom tetap berada di kiri atas. Berbeda dengan pencarian, fitur lokasi menulis titik langsung melalui `onChange`, karena hasil GPS merupakan jawaban yang sengaja diberikan pengguna sendiri terhadap pertanyaan "di mana Anda berada".

* Halaman Identity Verification mendapatkan tombol yang sebelumnya hilang: setiap pengiriman dalam riwayat sekarang menautkan dokumen identitas dan foto lokasi melalui handler `/files/{fileId}` yang melakukan pemeriksaan peran, banner pending memiliki tombol "Periksa keputusan" untuk memuat ulang data (form sepenuhnya dikunci ketika pengiriman sedang ditinjau, sehingga sebelumnya halaman tidak menawarkan aksi apa pun), banner approved menautkan ke profil yang memiliki lencana, dan baris riwayat menampilkan timestamp keputusan di samping waktu pengiriman.

* Menambahkan unit test untuk dua modul bersama yang baru: `lib/datetime.test.ts` mencakup parsing RFC 3339 dan format yang dipisahkan spasi, fallback "-" untuk input yang tidak dapat diparse, tampilan WIB, dan rupiah integer; `lib/statusFilters.test.ts` mencakup pembentukan dan pengurutan chip, pertumbuhan ketika sebuah status ditambahkan ke peta label, serta penolakan key `Object.prototype` oleh `isStatusFilter`.

* Menambahkan komponen form yang dapat digunakan kembali di bawah `components/form`: `FormField` (input teks dan select), `PasswordField` (toggle tampil/sembunyikan), dan `RoleOption` (kartu peran bergaya radio).

* Menambahkan `DashboardLayout`, layout sidebar generik untuk dashboard terautentikasi (sidebar tetap pada desktop, drawer pada mobile, header sticky dengan menu akun dan logout), serta `AdminLayout` dengan item navigasi admin.

* Menambahkan `AppLayout`, layout sidebar untuk pengguna terautentikasi non-admin dengan item navigasi yang mengikuti peran akun (buyer, subcontractor, atau keduanya). Semua route non-admin yang dilindungi sekarang berada di dalam layout ini.

* Menambahkan konten placeholder untuk semua halaman admin (dashboard, antrean verifikasi, master item, proposal, order terlambat, sengketa, moderasi ulasan, WhatsApp) dan memasukkan route admin ke dalam `AdminLayout` di `App.tsx`.

* Menambahkan konten placeholder untuk halaman verification, notifications, notification preferences, work orders, listing, listing calendar, requests (incoming, sent, create, detail), dan search.

* Mengimplementasikan halaman profil bisnis publik (`/profile/:profileId`): header identitas dengan lencana terverifikasi, kartu reputasi yang mengikuti aturan `enough_data` (FR-073), detail listing kapasitas (produk, mesin, kapasitas mingguan), serta kondisi not-found yang ramah.

* Menambahkan `VerifiedRoute`, route guard untuk halaman yang hanya masuk akal ketika verifikasi belum lengkap. Akun dengan email dan telepon yang keduanya sudah terverifikasi diarahkan keluar dari `/verification` menuju profilnya.

* Menghubungkan halaman Forgot Password dan Reset Password ke API: meminta kode pemulihan (`/auth/recover/request`) dari halaman lupa kata sandi, lalu mengonfirmasi kode dan kata sandi baru (`/auth/recover/confirm`) pada halaman reset, dengan email dibawa melalui router state dan pengalihan kembali ke forgot-password jika email tersebut tidak ada.

* Menghubungkan halaman Verify Phone ke API (`/auth/verify-phone`, `/auth/resend-code`): pengiriman OTP nyata, hitung mundur pengiriman ulang, nomor telepon yang disamarkan dari akun, dan pengalihan ke halaman default sesuai peran setelah berhasil.

* Menambahkan `LocationPicker`, peta Leaflet klik-untuk-menandai yang digunakan pada form edit My Profile untuk menetapkan latitude dan longitude, menggantikan input koordinat manual. Mode tampilan menampilkan titik tersimpan pada peta read-only.

* Mendesain ulang halaman My Profile: kartu identitas dengan inisial avatar, lokasi, lencana status verifikasi, dan panel reputasi (rating ditambah completion rate yang mengikuti aturan `enough_data`). Payload pembaruan sekarang hanya mengirim field `ProfileUpdateRequest` (`province_code` tetap menjadi filter kota khusus form), memperbaiki penolakan 422 "Format permintaan tidak sah".

* Mengerjakan ulang `DashboardLayout`: sisi kanan header sekarang memiliki lonceng notifikasi dengan lencana unread (dari `/notifications`) dan identitas akun yang menautkan ke profil; kartu profil berada di atas tombol logout pada footer sidebar. Item sidebar Notifications dihapus dan digantikan oleh lonceng.

* Menghubungkan halaman Notifications ke API: daftar dengan ikon event dan styling per tipe, filter semua/unread, timestamp relatif dalam WIB, penandaan telah dibaca per item (yang juga memperbarui lencana header), tautan ke work order terkait, dan "Muat lebih banyak" yang meneruskan `next_cursor` opaque tanpa perubahan.

* Menghubungkan halaman Notification Preferences ke API (`GET`/`PUT /notifications/preferences`): toggle kanal email dan WhatsApp untuk notifikasi non-transaksional, dengan catatan bahwa notifikasi transaksional tidak dapat dinonaktifkan (FR-054).

* Mengimplementasikan Admin Dashboard: empat kartu ringkasan antrean (antrean verifikasi, proposal item yang menunggu, sengketa terbuka, order terlambat) ditambah lima entri terbaru dari setiap antrean dengan tautan ke halaman admin terkait, yang mengambil data dari `/admin/verification`, `/admin/proposals`, `/admin/disputes`, dan `/admin/late-orders`.

* Memisahkan halaman My Profile berdasarkan tipe akun: admin sekarang melihat profil khusus akun (tanpa profil bisnis), sedangkan akun buyer/subcontractor tetap menggunakan profil bisnis. Kedua versi menampilkan bagian akun dengan status verifikasi email dan telepon serta tombol yang membuka halaman Verify Email / Verify Phone yang sudah ada untuk kanal yang belum terverifikasi.

* Mengizinkan pengguna yang sudah login membuka `/auth/verify-email` dan `/auth/verify-phone` dari profil mereka (GuestRoute tidak lagi mengeluarkan mereka; kanal yang sudah terverifikasi diarahkan ke `/profile/me`). Halaman Verify Email menggunakan email akun sebagai fallback ketika tidak ada router state, dan verifikasi yang berhasil kembali ke profil, bukan ke alur autentikasi. `/verification` sekarang berada di bawah ProtectedRoute umum dan `VerifiedRoute` dihapus.

* Menghubungkan halaman Listing ke API (`/listing/me`, `/listing/me/visibility`, `/master/products`, `/master/machines`, `/master/proposals`): form create/edit dengan chip produk, checkbox mesin dengan jumlah unit, input kapasitas mingguan dan lead kesiapan, proposal item inline untuk katalog master, serta toggle publish dengan banner visibilitas.

* Menghubungkan halaman Listing Calendar ke API (`GET`/`PUT /listing/me/periods`): grid 12 minggu yang dipaginasi mulai Senin saat ini dalam WIB, input kapasitas per minggu dengan toggle marked-full, minggu dengan alokasi aktif dikunci dari perubahan, serta sticky save bar yang hanya mengirim minggu yang berubah.

* Menghubungkan halaman Work Orders ke API (`/orders`): halaman daftar memiliki chip filter status untuk ketujuh status, filter peran buyer/subcontractor, dan pagination dengan opaque cursor; halaman detail merender tombol aksi dari `allowed_transitions` dan `self_cancellable` (FR-039) dengan panel pembatalan, pernyataan pembayaran (arah dan tanggal, tanpa nominal, FR-040/042), laporan sengketa, dan ulasan, ditambah banner hitung mundur auto-confirm, alokasi, riwayat pembayaran, dan timeline status.

* Menghubungkan halaman Search ke API (`/search`): form kriteria (produk, mesin, kuantitas, deadline, maksimum hari lead, kota/provinsi/nasional dari wilayah akun), kartu kandidat dengan skor 4 kriteria dan penjelasan per kriteria, lencana verified dan kalender stale, reputasi yang mengikuti `enough_data`, serta sticky selection bar yang menyerahkan listing terpilih ke form permintaan kuota. Pagination cursor meneruskan `next_cursor` tanpa perubahan.

* Menghubungkan halaman permintaan kuota ke API (`/quota-requests`, `/candidates/{id}/offers`, `/offers/{id}/counter`, `/offers/{id}/accept`): form create menerima kandidat melalui router state dari search dan memblokir self-request; daftar terkirim menampilkan ringkasan kandidat per permintaan; halaman detail membandingkan kandidat dengan status per kandidat, rantai penawaran lengkap yang diurutkan berdasarkan ronde, panel counter-offer, dan accept-offer yang berpindah ke work order yang dibuat.

* Menghubungkan halaman Identity Verification ke API (`POST /files`, `GET`/`POST /verification`): form pengiriman dengan nomor identitas 8-32 karakter, dua slot upload (dokumen identitas dan foto lokasi usaha, batas 5 MB dengan pemeriksaan ukuran di sisi client), banner pending/approved/rejected berdasarkan status verifikasi akun, serta daftar riwayat pengiriman. `apiClient` sekarang melewati JSON Content-Type untuk body `FormData` agar fetch dapat menetapkan multipart boundary sendiri.

* Menghubungkan halaman Incoming Requests ke API (`/quota-requests/incoming`): halaman daftar memfilter berdasarkan status kandidat dan menautkan ke halaman detail; halaman detail menampilkan header permintaan, rantai penawaran lengkap, dan panel untuk mengirim penawaran (harga total dan lead kesiapan), melakukan counter terhadap penawaran buyer, atau menolak dengan alasan. Menambahkan `request_id` ke schema `RequestCandidate` dalam `openapi.yaml` (type hasil regenerasi diperbarui agar sesuai) karena item incoming sebelumnya tidak dapat menaut ke detail permintaannya.

* Menghubungkan semua halaman admin yang tersisa ke API. Verification Queue (`/admin/verification` + decision) memfilter berdasarkan status, menautkan dokumen identitas dan foto lokasi, serta mewajibkan alasan penolakan. Item Proposals (`/admin/proposals` + decision) menyetujui item ke katalog master atau menolaknya dengan alasan. Master Items (`/admin/master/items`) menambah, mengganti nama, dan mengaktifkan/menonaktifkan item katalog. Disputes (`/admin/disputes` + mediate/resolve) memulai mediasi dan menyelesaikan dengan hasil continued/confirmed/cancelled, membalik alokasi ketika dibatalkan (FR-071/072). Late Orders (`/admin/late-orders`) menampilkan order yang melewati deadline kesiapan. Reviews Moderation (`/profile/{id}/reviews` + hide) mencari ulasan profil berdasarkan ID dan menyembunyikan ulasan yang melanggar dengan alasan (FR-050). WhatsApp (`/admin/whatsapp`) menampilkan status sesi, error terakhir, dan QR code relinking dengan refresh otomatis 30 detik, tidak pernah menampilkan nomor layanan (FR-082).

* WhatsApp sekarang mengubah payload QR sesi menjadi PNG yang dapat dipindai menggunakan `qrcode`, sekaligus menerima image data URL dari API.

* Memperbarui sidebar dashboard dan admin agar menggunakan aset logo, menambahkan tujuan `/admin/orders` yang sebelumnya hilang, dan mencegah permintaan business-profile sebelum akun non-admin selesai dimuat.

* Menyesuaikan payload mutasi admin dengan type `paths` OpenAPI yang dihasilkan, melakukan encoding pada identifier path admin, dan membangun parameter query master-item dengan `URLSearchParams`.

* Menormalisasi respons daftar proposal admin dan sengketa sebelum rendering serta melewati business-profile query untuk akun admin, yang memang tidak memiliki profil bisnis.

* Merender payload QR WhatsApp menjadi gambar QR yang dapat dipindai menggunakan package `qrcode`, sambil mempertahankan dukungan terhadap image data URL.

* Menambahkan halaman supervisi order admin read-only di `/admin/orders/:workOrderId` (`GET /work-orders/{workOrderId}`, yang boleh dibaca admin menurut FR-045 dan FR-046): angka-angka order, riwayat status, tabel alokasi per minggu, tautan ke profil kedua pihak, pernyataan pembayaran, serta deadline kesiapan dan order. Halaman tersebut menyatakan bahwa admin sedang membaca sebagai supervisor dan tidak dapat mengubah status, memiliki tombol kembali ke antrean tempat halaman tersebut dibuka, dan menampilkan sengketa yang diajukan terhadap order dengan tautan ke antrean sengketa (FR-043, FR-046). Perubahan status tetap menjadi tanggung jawab pihak transaksi; admin bertindak melalui resolusi sengketa.

* Meregenerasi `api/types.ts` dari `openapi.yaml`, dengan mengambil `LateOrderSummary`, `LateOrderList`, dan `POST /admin/whatsapp/reconnect`.

* Menambahkan panel kontak pihak lawan ke halaman detail work order, menghubungkan `GET /work-orders/{id}/contacts` melalui `getWorkOrderContacts` dan `useWorkOrderContacts` baru: nama bisnis, sisi transaksi, tautan email `mailto:`, dan tautan WhatsApp `wa.me`. Ini menjadi penghubung menuju koordinasi pembayaran dan produksi yang berlangsung langsung antar pihak (FR-040, FR-092). Query hanya berjalan ketika viewer merupakan pihak dalam order, karena non-pihak memang mendapatkan 404 dan respons membawa email serta nomor telepon pihak lainnya.

* Menambahkan daftar ulasan ke profil bisnis publik. `GET /profile/{profileId}/reviews` merupakan endpoint publik tetapi client-nya sebelumnya hanya berada di `api/admin.ts`, sehingga ulasan hanya dapat diakses dari halaman moderasi admin: `getProfileReviews` dipindahkan ke `api/profile.ts` dan `useProfileReviews` ke `hooks/useProfile.ts` (halaman admin sekarang mengimpornya dari sana). Setiap entri menampilkan rating, nama bisnis pemberi ulasan yang menaut ke profilnya, tanggal transaksi, dan teks, sehingga tidak ada ulasan yang tampil secara anonim (FR-048, FR-049). Ulasan tersembunyi tidak pernah sampai ke client.

* Menambahkan peta lokasi bisnis dan perkiraan jarak informasional ke profil publik, dengan merender `LocationMap` yang sudah ada menggunakan koordinat profil. Jarak dihitung di sisi client dalam `lib/geo.ts` yang baru (haversine, tanpa PostGIS dan tanpa map API key) terhadap koordinat bisnis milik viewer sendiri, dan hanya ditampilkan kepada akun bisnis yang sudah menandai lokasinya, serta tidak pernah ditampilkan pada profil sendiri. Panel menyatakan bahwa jarak tidak digunakan untuk memfilter atau mengurutkan ulang hasil pencarian (FR-064).

* Menambahkan manajemen peran ke My Profile, yang merupakan satu-satunya pemanggil `useUpdateMyRoles` / `PATCH /me/roles` yang sebelumnya tidak memiliki pemakai. Peran buyer atau subcontractor dapat diaktifkan pada akun yang sama setelah registrasi; tombol untuk peran terakhir yang tersisa dinonaktifkan, dan penolakan ketika masih ada order aktif menampilkan pesan backend `ROLES_IN_USE` secara apa adanya.

* Menambahkan indikator payment mismatch pada sisi order dan sisi admin (FR-043). `WorkOrderDetail` mendapatkan `payment_mismatch` nullable dalam `openapi.yaml` (`missing_counterpart`, atau `date_differs` dengan jumlah hari) dan `PaymentMismatchNotice` bersama merendernya pada halaman detail work order, halaman supervisi order admin, dan setiap kartu sengketa terbuka di antrean sengketa admin, sesuai lokasi sinyal dalam spesifikasi. Perbandingan tetap berada di domain: client tidak menghitung mismatch dari array `payments`, sehingga tampilan pihak dan tampilan admin tidak dapat berbeda.

* Menambahkan halaman status sistem admin di `/admin/system` (`GET /health`) dengan `api/system.ts` dan `hooks/useSystem.ts` baru: status keseluruhan `ok`/`degraded`, dependensi database dan WhatsApp, serta storage dengan bar used-versus-limit agar `near_full` terlihat sebelum upload mulai ditolak karena batas 500 MB. Dashboard admin membawa banner singkat untuk sinyal yang sama yang menaut ke halaman tersebut. Request yang gagal dibaca sebagai instance yang tidak sehat, bukan sekadar network error, karena 503 memang merupakan jawaban untuk database yang gagal atau storage yang penuh.

* Menambahkan tombol "Sambungkan Ulang" pada halaman WhatsApp admin yang memanggil `POST /admin/whatsapp/reconnect` melalui `reconnectWhatsApp` dan `useReconnectWhatsApp` baru, menulis respons langsung ke status cache sehingga QR baru muncul tanpa menunggu polling berikutnya. "Muat Ulang" tetap hanya membaca status, dan halaman sekarang menjelaskan fungsi masing-masing tombol.

* Menambahkan `can_record_payment` dan `can_review` ke `WorkOrderDetail`, serta rantai `latest_offer`/`offers` ke `IncomingCandidate`, dalam `openapi.yaml`, lalu meregenerasi `api/types.ts`. Halaman incoming request sudah merender rantai penawaran; schema sebelumnya hanya memiliki rantai tersebut pada `RequestCandidate`, yaitu bentuk yang digunakan di sisi buyer.

### Diubah

* `usePlaceSearch` tidak lagi memanggil `setState` dari dalam body effect. Modul tersebut menyimpan query yang sudah diselesaikan bersama hasilnya, sehingga state pencarian dibaca dengan membandingkan query tersimpan dengan query saat ini, yang juga mencegah hasil lama ditampilkan seolah-olah menjawab query baru. Bentuk lama memicu `react-hooks/set-state-in-effect`.

* `jest.config.js` menetapkan `target` ts-jest ke `es2023` agar sesuai dengan `tsconfig.app.json`. ts-jest sebelumnya menggunakan default yang lebih lama, sehingga `Array.prototype.at` dapat diperiksa tipenya pada build aplikasi tetapi gagal saat test dijalankan.

* Chip filter status sekarang diturunkan dari peta label status, bukan ditulis ulang sebagai literal. `lib/statusFilters.ts` membangunnya dari `Record<Enum, …>`, sehingga nilai enum baru di `openapi.yaml` akan menyebabkan typecheck gagal pada peta label dan chip akan muncul tanpa perlu mengedit halaman. Diterapkan pada daftar work orders, incoming requests, admin disputes, admin verification queue, dan admin item proposals. Dua konsekuensi yang perlu diketahui: filter incoming requests sekarang juga menawarkan "Tidak Dilanjutkan", yang sebelumnya tidak ada pada daftar buatan tangan, dan chip "offered" di sana sekarang berbunyi "Ada Penawaran" seperti di tempat lain, bukan "Sudah Dibalas". `parseStatusParam` pada daftar work orders melakukan validasi terhadap peta yang sama melalui `isStatusFilter`.

* Hasil mediasi sengketa pada halaman admin sekarang berasal dari `Record<DisputeResult, …>` daripada array literal, dan baris sengketa yang telah diselesaikan langsung mencari label dari peta tersebut daripada memindai array.

* `disputeStatusMeta` berada di `pages/Admin/meta.ts` dan digunakan bersama oleh halaman sengketa dan dashboard admin. Dua salinan sebelumnya tidak sesuai: `in_mediation` terbaca "Dalam Mediasi" di satu halaman dan "Mediasi" di halaman lain, sehingga satu sengketa memiliki dua label tergantung dari mana sengketa tersebut dibuka. Salinan bersama mempertahankan "Dalam Mediasi".

* Pemformatan tanggal dan rupiah dipindahkan ke `lib/datetime.ts`. Dua belas halaman masing-masing memiliki `formatDateTimeId`/`formatDateId` sendiri, dan hanya `WorkOrders/meta.ts` serta dashboard admin yang menormalisasi bentuk timestamp yang dipisahkan spasi (`2026-08-31 10:00:00`) yang tidak diwajibkan untuk diterima oleh ECMAScript. Setiap halaman sekarang melewati `parseApiDate`, sehingga nilai dengan format dipisahkan spasi dirender sama di Safari dan Chrome, bukannya menampilkan "Invalid Date" di beberapa halaman dan tanggal di halaman lain. `WorkOrders/meta.ts` dan `Requests/meta.ts` melakukan re-export dari modul tersebut, sehingga import halaman tetap tidak berubah. Formatter mengembalikan "-" untuk input yang tidak dapat diparse, bukan "Invalid Date". Waktu relatif notifikasi juga melakukan parsing melalui modul tersebut.

* `useIncomingCandidate` sekarang mengikuti `has_next` sampai akhir daftar incoming, bukan berhenti setelah 5 halaman, sehingga kandidat yang terdorong jauh ke bawah daftar masih dapat dibuka dari notifikasinya. Batas 200 halaman dan pemeriksaan cursor yang sudah pernah dilihat tetap dipertahankan sebagai pengaman terhadap cursor yang berhenti maju, bukan sebagai batas jangkauan. Endpoint `GET /candidates/{candidateId}` akan menghilangkan kebutuhan untuk melakukan pemindaian seluruh daftar; kontrak saat ini belum memiliki endpoint tersebut.

* Cakupan wilayah pada halaman Search sekarang menjadi perluasan satu langkah yang dipandu, bukan tiga toggle bebas (FR-063). Pencarian dimulai dari cakupan tersempit yang dimiliki akun (kota, jika tidak ada maka provinsi, jika tidak ada maka nasional), header hasil menyebutkan cakupan yang benar-benar digunakan oleh respons, dan satu aksi "Perluas ke ..." menaikkan cakupan tepat satu tingkat tanpa menyentuh filter lain. Aksi tersebut menghilang pada tingkat nasional. Akun tanpa kode kota atau provinsi tidak lagi melihat cakupan yang tidak dapat mereka cari.

* Hasil pencarian kosong sekarang menyebutkan filter paling ketat dan satu cara konkret untuk melonggarkannya (FR-028), dengan membaca `SearchResult.relaxation` ketika backend mengirimkannya dan membuat saran dari filter yang dikirim jika tidak ada. Kalimat umum "coba perluas wilayah atau longgarkan kriteria" dihapus.

* `WhatsAppStatus` sekarang membawa enum `status` (`connected`, `pairing`, `disconnected`) alih-alih boolean `connected`, dalam `openapi.yaml` dan type hasil regenerasi, dan halaman WhatsApp admin menampilkan ketiga status tersebut dengan kata-kata masing-masing. Boolean tidak dapat membedakan pairing dari disconnected, sehingga sesi yang sudah menampilkan QR yang dapat dipindai sebelumnya digambarkan sebagai disconnected dan halaman meminta admin melakukan reconnect tanpa alasan.

* Mendesain ulang halaman Register untuk kejelasan dan aksesibilitas: panel branding datar yang disederhanakan, pemilihan peran melalui radiogroup yang benar, field kata sandi biasa dengan aturan minimum panjang yang terlihat, checkbox persetujuan syarat yang eksplisit, pesan validasi bahasa Indonesia yang lebih ramah, serta CTA utama yang lebih jelas dengan spinner saat loading.

* Menata ulang halaman Login, Register, Forgot Password, dan Reset Password ke desain dua panel yang sama: panel brand deep-navy dengan bentuk abstrak pada desktop dan panel form yang ringkas, dengan tetap mempertahankan palet warna dan seluruh logika form yang ada.

* Menata ulang halaman My Profile agar sesuai dengan layout dashboard: bagian berbasis kartu, dan tombol simpan sekarang mencerminkan status pending dari mutation update.

* Mendesain ulang Final CTA pada halaman utama: kartu dua kolom dengan bentuk abstrak yang sesuai dengan halaman autentikasi, panel sorotan fitur, tombol rounded dengan tautan `react-router-dom`, serta panah hover pada aksi sekunder.

* Menata ulang `DashboardLayout` menjadi sidebar terang dan blur: item aktif dengan warna aksen, submenu yang dapat diciutkan, dan tombol logout yang dipasang di footer sidebar. Area akun pada header disederhanakan menjadi blok identitas statis. Navigasi admin sekarang mengelompokkan master data dan supervisi order ke dalam submenu.

* Mendesain ulang footer publik: layout empat kolom (brand dengan deskripsi pertukaran kapasitas dan catatan bahwa dana tidak ditahan, navigasi dalam halaman, tautan platform, tautan perusahaan) dengan bottom bar untuk copyright dan tautan legal.

* Halaman Master Items admin diperbarui: badge Aktif/Nonaktif sekarang disembunyikan saat baris masuk mode edit (`{!editing && <span ...>}`), grid diubah dari `md:grid-cols-2 lg:grid-cols-3` menjadi `lg:grid-cols-2` agar jumlah kolom lebih konsisten di tablet dan desktop, dan node spasi yang tidak perlu di antara tombol aksi dihapus (`{\" \"}` hasil format ulang yang menciptakan text node di DOM tanpa dampak visual).

### Diperbaiki

* Kedua sisi work order tidak lagi melihat tombol milik pihak lainnya. `allowed_transitions` mendeskripsikan order, bukan pemanggil, sehingga merendernya apa adanya membuat buyer melihat "Mulai Produksi", "Tandai Selesai", dan "Tandai Dikirim", serta subcontractor melihat "Konfirmasi Diterima": empat tombol pada setiap halaman yang sebenarnya milik pihak lain, semuanya menghasilkan 403 ketika diklik. `transitionsForSide` di `pages/WorkOrders/meta.ts` melakukan intersect array tersebut dengan aksi yang dapat dilakukan posisi pemanggil, berdasarkan `getWorkOrderSide`. Buyer mempertahankan konfirmasi, pembatalan, dan sengketa; subcontractor mempertahankan tiga status produksi, pembatalan, dan sengketa. Ini tetap merupakan filter, bukan state machine kedua: tidak ada sesuatu yang muncul jika backend tidak mengirimkannya, sehingga jawaban server yang lebih sempit tetap mempersempit halaman. Panel pembayaran mengikuti alasan yang sama dan menghapus pemilih arah, karena buyer yang menyatakan "saya sudah menerima" bukan pernyataan yang nyata, dan deteksi mismatch (FR-043) memang membandingkan deklarasi kedua sisi. Tombol cancel dan payment sekarang juga mensyaratkan sisi yang diketahui, sehingga viewer yang bukan salah satu pihak tidak mendapatkan tombol aksi sama sekali.

* Lencana verifikasi pada My Profile tidak lagi mendorong baris email melewati tepi kartu. Baris tersebut sebelumnya menempatkan avatar, label, alamat, dan badge dalam satu baris, dan alamat panjang seperti `umkm-03@devotion.test` membuat badge tidak memiliki ruang untuk menyusut. Status dan tombol "Verifikasi" dipindahkan ke bawah alamat, dipisahkan dengan divider, dan perlakuan yang sama diterapkan pada kartu peran dalam "Peran Usaha" ketika tombol "Cabut"/"Aktifkan" terdesak oleh deskripsi dua baris. Alasan peran terakhir tidak dapat dicabut sekarang ditulis di bawah tombol yang dinonaktifkan, bukan disembunyikan dalam tooltip `title` yang tidak pernah muncul pada perangkat sentuh.

* `npm run lint` kembali bersih. `VerificationGate.tsx` mengekspor hook `useAccountVerification` bersamaan dengan komponen, yang memicu `react-refresh/only-export-components`: Vite dapat melakukan hot-swap pada modul yang seluruh export-nya merupakan komponen, sedangkan satu export non-komponen menyebabkan full page reload dan menghilangkan apa pun yang sudah diketik pengguna pada form yang berada di atas gate tersebut. Hook sekarang berada di `hooks/useAccountVerification.ts` bersama hook lainnya, dan tiga halaman yang mengimpornya telah diperbarui.

* Jest kembali melakukan type-check terhadap matcher jest-dom. `ts-jest` mengompilasi setiap file test secara terpisah, sehingga import `@testing-library/jest-dom` di `src/test/setup.ts` tidak pernah sampai ke `VerificationGate.test.tsx` dan `toBeInTheDocument`, `toHaveAttribute`, serta `toHaveTextContent` semuanya ditolak dengan TS2339. tsconfig inline di `jest.config.js` sekarang mendeklarasikan `types: ["jest", "node", "@testing-library/jest-dom"]`; `node` harus dicantumkan secara eksplisit karena menyebutkan `types` apa pun akan menggantikan pemindaian otomatis `@types`, yang digunakan `src/test/polyfills.ts` untuk `node:util`.

* Alur incoming request tidak lagi bergantung pada rantai penawaran yang tidak dikirim server. `latest_offer` dan `offers` keduanya diperlakukan sebagai opsional: `lib/offers.ts` yang baru menyelesaikan satu rantai dari field mana pun yang dibawa respons, dan setiap gate membaca rantai tersebut, bukan status saja. `canCounter` sebelumnya hanya menjadi `status === "offered"`, yang tetap true ketika chain tidak ada, sehingga tombol menunjuk ke `offer_id` yang tidak ada. Sekarang tombol tersebut membutuhkan `Offer` nyata dengan `offer_id` yang berasal dari backend. Ketika status mengatakan `offered` tetapi tidak ada ronde, halaman menyatakan bahwa riwayat tidak diterima, mempertahankan "Tolak" agar tetap dapat digunakan, dan menawarkan reload; halaman tidak pernah membuat ronde atau `offer_id` sendiri, dan tidak pernah memanggil `GET /quota-requests/{requestId}` yang hanya tersedia untuk buyer untuk mengisi kekosongan tersebut. Halaman daftar membaca ronde terbaru melalui resolver yang sama, dan halaman sent-request menggunakannya bersama agar kedua sisi tidak dapat berbeda.

* Mengeklik peta di `LocationPicker` tidak lagi memperkecil zoom kembali. Komponen `Recenter` lama menerbitkan ulang `setView(position, MARKER_ZOOM)` dari effect yang bergantung pada position, sehingga setiap klik mengatur ulang tampilan ke zoom 13 dan membuang zoom yang sudah dinaikkan pengguna untuk menempatkan pin secara presisi. Pergerakan peta sekarang menjadi command object eksplisit: peta melakukan pan satu kali ke titik yang sudah tersimpan ketika profil dimuat, menggunakan `fitBounds` dengan batas maksimum zoom untuk hasil pencarian yang dipilih, dan selebihnya mempertahankan zoom yang sedang digunakan.

* Tautan notifikasi tidak lagi mengirim subcontractor ke route khusus buyer. `normalizeLink` sebelumnya meneruskan `/quota-requests/<requestId>` secara langsung, tetapi route tersebut berada di balik role gate buyer, sehingga tombol "Lihat request masuk" pada notifikasi counter-offer mengarahkan subcontractor kembali ke profil mereka. Fungsi tersebut sekarang menerima peran pemanggil dan mengembalikan null untuk tujuan yang tidak dapat dibuka oleh peran tersebut, sehingga fallback event (`/requests/incoming`) mengambil alih. Notifikasi negosiasi membawa `request_id`, sedangkan halaman detail subcontractor menggunakan `candidate_id`, sehingga tidak ada route detail yang setara untuk ditulis ulang.

* Halaman detail incoming request tidak lagi merender kartu negosiasi yang mati ketika rantai penawaran hilang dari respons API. `GET /api/quota-requests/incoming` mengembalikan kandidat dengan `status: "offered"` tetapi tanpa `latest_offer`/`offers`, meskipun salinan kontrak pada branch ini menyatakan kedua field tersebut ada, sehingga `canOffer`, `canCounter`, `waitingBuyer`, dan `terminal` semuanya false: tidak ada tombol, tidak ada pesan, dan tidak ada yang dapat digunakan untuk counter. Halaman sekarang mendeteksi kondisi tersebut, menyatakan bahwa riwayat penawaran tidak termuat sementara negosiasi masih terbuka, menawarkan tombol reload, dan mempertahankan aksi reject agar tetap dapat digunakan. Perbaikan sebenarnya berada di backend, yang `incomingCandidateView`-nya tidak membawa field penawaran.

* Daftar incoming request tidak lagi mengatakan "Belum ada penawaran" untuk kandidat yang statusnya `offered`. Status tersebut berarti chain tersedia, sehingga label sebelumnya bertentangan dengan chip di sebelahnya; sekarang dinyatakan bahwa negosiasi sedang berjalan dan riwayat belum dimuat.

* Kartu kandidat pencarian sekarang menentukan denominator kriteria dari `candidate.criteria.length` daripada angka 4 yang ditulis secara hardcoded. Badge skor sebelumnya membaca `{score}/4` dan tombol "Pilih" dinonaktifkan pada `unmet.length === 4`, sehingga perubahan jumlah kriteria wajib di backend akan menghasilkan denominator yang salah dan secara diam-diam berhenti menonaktifkan kandidat yang tidak memenuhi apa pun.

* Completion rate tidak lagi dikalikan 100 pada halaman hasil pencarian dan My Profile. `Reputation.completion_rate` sudah merupakan persentase (0-100) menurut `openapi.yaml`, sehingga nilai 92 sebelumnya ditampilkan sebagai "9200%". Halaman profil publik sudah benar. Gate `enough_data` (FR-073) sudah benar di ketiga tempat dan tidak berubah.

* Mencatat pembayaran, mengajukan sengketa, atau mengirim ulasan sekarang langsung memperbarui detail work order. Invalidator bersama sebelumnya hanya menginvalidasi key daftar, dan ketiga mutation tersebut tidak mengirim order yang diperbarui, sehingga cache detail tetap stale selama `staleTime` 30 detik: baris pembayaran baru tidak terlihat, dan order terus menunjukkan status lama dengan tombol sengketa masih aktif setelah berpindah ke `in_mediation`. `POST /work-orders/{id}/payments` dan `POST /work-orders/{id}/disputes` mengembalikan `WorkOrderDetail` lengkap menurut `openapi.yaml`, sehingga kedua mutation sekarang langsung menulisnya ke cache detail seperti yang sudah dilakukan status change, confirm, dan cancel; `recordPayment` dan `reportDispute` sebelumnya diketik sebagai `PaymentRecord` dan `Dispute`, yang sekarang tidak lagi sesuai dengan kontrak.

* Tombol "Catat Pembayaran" dan "Beri Ulasan" sekarang dikendalikan oleh flag capability server, bukan daftar status yang ditulis di React. "Catat Pembayaran" sebelumnya dirender tanpa syarat, sehingga muncul pada order yang dibatalkan dan bagi viewer yang bukan pihak, sedangkan "Beri Ulasan" dikendalikan oleh `order.status === "confirmed"` tanpa mengetahui apakah pemanggil sudah pernah memberikan ulasan. `WorkOrderDetail` mendapatkan `can_record_payment` dan `can_review` dalam `openapi.yaml`, dan kedua tombol membaca nilai tersebut, sehingga state machine order tetap berada di satu tempat (FR-039, FR-041, FR-047).

* Deep link notifikasi tidak lagi berakhir buntu ketika event tidak membawa target. Resolusi link dipindahkan ke `lib/notificationLinks.ts`, membaca field `link` dari schema (memetakan path API seperti `/work-orders/{id}` ke route client `/orders/{id}` dan menolak URL di luar situs), serta menggunakan tujuan fallback per-event untuk seluruh 16 event. `rating_request` diarahkan ke `/orders?status=confirmed`, dan daftar work orders sekarang membaca filter status dari query string sehingga tautan tersebut tiba dalam kondisi sudah terfilter (FR-047, FR-051, FR-074).

* Banner "Lihat Pesanan" pada halaman detail sent quota request sekarang muncul setelah transaksi berhasil disepakati. Mutation accept-offer dipindahkan ke komponen halaman dan diteruskan ke setiap kartu kandidat, sehingga work order yang dibuat dapat dibaca di tempat banner dirender; sebelumnya setiap kartu memiliki instance mutation sendiri dan copy pada halaman tidak pernah berjalan.

* Antrean admin sekarang selalu membuka halaman supervisi order admin. Baris late-order di dashboard dan kartu sengketa sebelumnya menaut ke `/orders/{id}` (tampilan pihak transaksi), sedangkan halaman late orders menaut ke `/admin/orders/{id}`, sehingga satu order dapat menampilkan dua halaman berbeda tergantung dari titik masuknya.

* Tautan dokumen identitas dan foto lokasi pada antrean verifikasi admin sekarang melewati base URL API, bukan prefix `/api` hardcoded, yang sebelumnya membuat kedua tombol rusak ketika `VITE_API_URL` menunjuk ke host lain. Menambahkan `apiUrl()` ke `api/client.ts` untuk tautan yang dibuka di luar `fetch`.

* `getLateOrders` sekarang mengembalikan `LateOrderList` alih-alih `WorkOrderList`. Endpoint daftar memang sengaja tidak menyertakan riwayat status, alokasi, dan catatan pembayaran; type yang lebih luas sebelumnya membuat field tersebut terlihat tersedia (FR-045).

* Notifikasi offer dan counter-offer untuk buyer sekarang menaut ke `/quota-requests` alih-alih `/requests/sent`, yaitu route yang tidak ada dan sebelumnya membawa pengguna ke halaman 404.

* Registrasi sekarang melanjutkan ke halaman Verify Email dengan membawa email yang didaftarkan alih-alih kembali ke login, sehingga kode enam digit yang dikirim ke email dan WhatsApp memiliki layar untuk dimasukkan (FR-002).

* Daftar work orders sekarang menampilkan state loading ketika halaman pertama sedang diambil, bukan empty state "tidak ada order pada filter ini".

* My Profile tidak lagi memanggil setState saat render. Form melakukan sinkronisasi melalui opsi `values` dari React Hook Form dan filter provinsi dibaca dengan `useWatch`, menghilangkan double render di bawah StrictMode.

* Forgot Password sekarang mengatakan bahwa kode enam digit dikirim, sesuai dengan yang benar-benar diharapkan API dan halaman Reset Password, bukan menjanjikan tautan.

* Menghapus angka-angka yang dibuat-buat dan klaim jaminan pembayaran dari landing page. "Pembayaran terproteksi dengan jaminan kepuasan" bertentangan dengan batas platform yang tidak pernah menahan, mentransfer, atau memproses dana; jumlah partner tidak memiliki dukungan data.

* Navigasi internal pada bagian hero dan header publik sekarang menggunakan `Link` alih-alih `<a href>`, sehingga berpindah dari landing page tidak lagi me-reload SPA dan membuang cache TanStack Query, termasuk `/me`.

* Halaman Verify Email tidak lagi menampilkan judul masalah mentah "Belum masuk". Error 401 sekarang menjelaskan bahwa sesi tidak terbaca dan menawarkan tautan "Masuk kembali", sedangkan 410 menjelaskan bahwa kode telah kedaluwarsa dengan petunjuk untuk mengirim ulang.

* Registrasi sekarang melanjutkan ke halaman Verify Email dengan email yang didaftarkan alih-alih mengembalikan pengguna ke login, dan verifikasi email yang berhasil melanjutkan ke Verify Phone.

* Menghubungkan manifest aplikasi web ke index.html dan memperbaiki konfigurasinya: `site.webmanifest` sekarang dilintas dengan `<link rel="manifest">`, nama dan short_name diubah ke "Devotion", theme_color disesuaikan ke #0f172a (sesuai brand), icon diperkecil menjadi hanya favicon.svg yang benar-benar ada, dan menghapus referensi ke PNG icon yang sudah tidak digunakan dari public/. Bersamanya, aset ikon yang tidak dirujuk dari HTML (apple-touch-icon.png, favicon-96x96.png, favicon.ico, web-app-manifest-192x192.png, web-app-manifest-512x512.png) dihapus dari folder public/, mengurangi ukuran build.

* Halaman Master Items mengubah dua perilaku visual: badge status Aktif/Nonaktif kini disembunyikan saat baris masuk mode edit sehingga status tidak muncul dua kali (sekali di input readonly, sekali di badge), dan grid daftar item berubah dari kolom tiga level tablet (`md:grid-cols-2 lg:grid-cols-3`) menjadi kolom dua level desktop saja (`lg:grid-cols-2`), mengurangi kolom di tablet dari 2 menjadi 1 kolom penuh dan di desktop dari 3 menjadi 2 kolom. Halaman ini juga memiliki banyak node teks spasi berlebihan (`{" "}`) yang tersisip di antara tombol dari hasil format ulang Prettier; node tersebut tidak berdampak visual (kontainer flex dengan gap), tetapi dapat dibersihkan pada penyentuhan berkas berikutnya.
### Dihapus

* Menghapus komponen dead code `TrustStatsSection` dan `FinalCTASection`. Keduanya tidak pernah diimpor setelah commit yang diklaim mengekstraksi mereka, sehingga tetap disimpan di berkas dirinya sendiri tanpa digunakan dari `pages/Home.tsx`. Bagian tersebut digantikan oleh redesign Final CTA yang lebih sederhana.

* Menghapus dependensi `axios`. Tidak ada kode yang mengimpornya: setiap request berjalan melalui wrapper `fetch` di `api/client.ts`, yang membutuhkan `credentials: "include"` untuk session cookie.

* Menghapus `react-compiler-runtime`, yang tertinggal dari template Vite. React Compiler tidak pernah diaktifkan, `babel-plugin-react-compiler` tidak terpasang, dan `vite.config.ts` tidak meneruskan `babel.plugins` ke `@vitejs/plugin-react`, sehingga runtime tersebut dikirim ke bundle tanpa kegunaan. Paragraf template "React Compiler is enabled" dihapus dari `frontend/README.md` agar sesuai.

* Menghapus `getProducts` dan `getMachines` dari `api/master.ts`. Keduanya tidak pernah diimpor: listing hooks menggunakan `getMasterProducts`/`getMasterMachines` dari `api/listing.ts`, yang memanggil dua endpoint yang sama. Re-export `CatalogItem` yang kini tidak digunakan ikut dihapus.

* Menghapus duplikat `pages/Search/SearchPage.tsx` yang tidak digunakan; halaman search berada di `pages/Search.tsx`.

* Melonggarkan schema nomor telepon agar menerima format `08`, `62`, dan `+62`, serta membuat pesan error konfirmasi kata sandi menjadi lebih jelas.

* Menginisialisasi frontend React 19 + TypeScript + Vite.

* Menambahkan Tailwind CSS 4 melalui plugin Vite.

* Menambahkan struktur source awal untuk components, pages, hooks, routes, API, schemas, utilities, dan assets.

* Menambahkan dependensi UI: `motion`, `react-icons`, `clsx`, `tailwind-merge`, dan `react-router-dom`.

* Menambahkan application shell awal, entry point, shared utility, dan starter assets.

* Menambahkan script pengembangan untuk `dev`, `build`, `lint`, dan `preview`.

* Mengaktifkan React Compiler dan konfigurasi ESLint untuk TypeScript dan React.
