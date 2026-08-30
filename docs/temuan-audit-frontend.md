# Temuan Audit Frontend

Audit read-only frontend pada branch `staging`, setelah merge `develop/frontend`
dan `develop/backend` terbaru. Tidak ada perubahan kode saat audit. Perbaikan
dipegang tim frontend, dilacak lewat issue GitHub (lihat tabel di akhir).

Kesimpulan umum: frontend sudah substansial lengkap. Semua tujuh user story punya
alur UI nyata, bukan stub. Aturan penting CLAUDE.md sebagian besar dipatuhi:
`credentials: 'include'` di satu-satunya wrapper fetch, tidak ada token di
`localStorage`/`sessionStorage`, cursor paginasi opaque di semua daftar, kriteria
pencarian per kandidat dirender dengan chip terpenuhi/tidak, completion rate
digerbang `enough_data`, uang integer rupiah (`maximumFractionDigits: 0`), tombol
status utama dirender dari `allowed_transitions`/`self_cancellable`, kalender
Senin-WIB dengan penguncian minggu beralokasi, Leaflet tanpa API key. Gap
terkonsentrasi di beberapa titik spesifik di bawah.

## Bug dan logika

### Cache detail work order basi setelah payment, dispute, review
`frontend/src/hooks/useWorkOrders.ts:29-39, 73, 82, 91`

`useDetailInvalidator` hanya `invalidateQueries(["work-orders","list"])`, tidak
pernah invalidasi `workOrderKeys.detail(id)`. `useRecordPayment`,
`useReportDispute`, dan `useSubmitReview` memanggil `invalidate()` tanpa argumen,
jadi cache detail tidak di-`setQueryData` maupun diinvalidasi. Akibat: setelah
catat pembayaran, baris pembayaran tidak muncul; setelah lapor sengketa, order
tidak langsung pindah ke `in_mediation` dan tombol dispute tidak hilang, sampai
`staleTime` 30 detik lewat atau user keluar-masuk halaman. Status, confirm, dan
cancel selamat karena mengoper `WorkOrderDetail` penuh.

### Tipe notifikasi drift dari openapi, `request_expired` tidak tertangani
`frontend/src/api/types.ts:3630`, `frontend/src/pages/Notifications.tsx:34-50`

Enum backend punya 16 event; `types.ts` cuma 15 (berhenti di `calendar_stale`,
`request_expired` tidak ada). Melanggar aturan generate tipe dari `openapi.yaml`.
Notifikasi `request_expired` jatuh ke label fallback "Notifikasi" tanpa ikon dan
tanpa deep link.

### Dua tombol WorkOrder pakai status hardcoded, bukan flag server
`frontend/src/pages/WorkOrders/Detail.tsx:244, 256`

"Catat Pembayaran" dirender tanpa syarat, bahkan pada order `cancelled`. "Beri
Ulasan" digerbang `order.status === "confirmed"` di React. Kelas duplikasi mesin
keadaan yang dilarang CLAUDE.md; tidak membedakan buyer vs subcontractor.

### Notifikasi tanpa deep link saat `work_order_id` null
`frontend/src/pages/Notifications.tsx:12-31`

Banyak event (rating_request, agreement_formed, order_status_changed, dan lain
lain) hanya membuat link bila ada `work_order_id`. Kalau null, ada label tapi
tanpa navigasi. `rating_request` khususnya tak punya rute fallback untuk menulis
ulasan.

## FR dengan endpoint tersedia tapi tanpa UI

### FR-092: kontak counterparty tidak pernah ditampilkan
Endpoint `GET /work-orders/{workOrderId}/contacts` (openapi.yaml:1119) tidak punya
api fn, hook, maupun render. Alur inti koordinasi luar-platform (nama usaha,
email, WhatsApp lawan transaksi) tidak muncul di mana pun. Gap fungsional paling
nyata.

### FR-049: daftar ulasan individual di profil publik
`frontend/src/pages/Profile/PublicProfile.tsx` hanya menampilkan agregat. Endpoint
`GET /profile/{profileId}/reviews` dan hook `useProfileReviews` sudah ada tapi
cuma dipakai di `Admin/ReviewsModeration`. Publik tidak bisa lihat ulasan per
transaksi dan identitas pengulas.

### FR-064: peta dan estimasi jarak di profil publik
`LocationMap` hanya dipakai di `MyProfile`. Profil publik cuma teks lokasi.

### `PATCH /me/roles` orphan
`updateMyRoles` dan `useUpdateMyRoles` terdefinisi tapi tidak dipanggil dari mana
pun. Tidak ada UI menambah peran buyer atau subcontractor setelah registrasi.

## Admin dan operasional

### FR-043: indikator selisih pembayaran
Tidak ada tanda ketidakcocokan pernyataan buyer vs subcon, baik di
`WorkOrders/Detail.tsx`, `Admin/OrderDetail.tsx`, maupun `Admin/Disputes.tsx`.
Spec mengarahkan selisih pembayaran ke admin saat sengketa; sinyalnya tidak punya
UI.

### `POST /admin/whatsapp/reconnect` tidak dipakai
Tombol "Muat Ulang" di `Admin/WhatsApp.tsx` hanya `refetch()` GET status, bukan
POST reconnect. Status juga dipipihkan jadi boolean `connected`, bukan enum
`connected/disconnected`, sehingga tidak bisa mewakili `degraded`.

### Tidak ada UI health, readiness, storage
Enum `degraded`, `database: fail`, `storage: near_full/full`
(openapi.yaml:2566-2581) tidak punya surface operator. Skema ada di types tapi tak
dipakai. Mengingat batas 500MB, warning storage berguna.

## Kualitas pencarian (US2)

### FR-063: ekspansi wilayah terpandu
`Search.tsx:15-19, 269-278` memakai toggle manual tiga tombol (city, province,
national), bukan saran ekspansi satu tingkat terpandu yang diminta spec.

### FR-028: saran filter paling membatasi
`Search.tsx:307` empty-state generik, bukan saran konkret kriteria mana yang
paling banyak mengeliminasi kandidat.

### Hardcode denominator "/4" pada kartu hasil
`Search.tsx:58, 100` mengasumsikan tepat empat kriteria (`score === 4`,
`unmet.length === 4`). Kalau backend mengubah jumlah kriteria, denominator salah
dan tombol disable rusak diam-diam. Denominator sebaiknya diturunkan dari
`candidate.criteria.length`.

## Ketahanan dan kebersihan kode

- Daftar enum hardcoded yang bisa drift (literal, bukan `Record<Type,...>`
  exhaustive): `WorkOrders/List.tsx:12-20`, `Requests/Incoming.tsx:11-17`,
  `Admin/Disputes.tsx:12-22`, `Admin/Dashboard.tsx:90`.
- `useIncomingCandidate` menyerah setelah scan lima halaman
  (`useQuota.ts:66-97`); kandidat di luar itu tak terjangkau.
- Parsing tanggal tidak seragam: `Requests/*` pakai `new Date(iso)` langsung, beda
  dari `WorkOrders/meta.ts` yang menormalkan timestamp berspasi. Latent bug di
  Safari kalau backend kirim format berspasi.
- Dead code: `getProducts`/`getMachines` di `api/master.ts:9,13` duplikat dari
  varian `listing.ts` yang dipakai hook.

## Pelacakan issue

| Issue | Tema |
|-------|------|
| #2 | Bug dan logika |
| #3 | FR dengan endpoint tersedia tapi tanpa UI |
| #4 | Admin dan operasional |
| #5 | Kualitas pencarian US2 |
| #6 | Ketahanan dan kebersihan kode |
| #7 | Ringkasan dan tracking |
