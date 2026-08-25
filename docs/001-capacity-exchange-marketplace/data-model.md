# Data Model: Capacity Exchange, Devotion

**Feature**: `docs/001-capacity-exchange-marketplace/`
**Date**: 2026-08-21
**Last Revised**: 2026-08-22
**Input**: `spec.md` (91 FR, 16 entitas), `research.md` (R-03, R-04, R-05), `docs/memory/constitution.md` v2.1.0

16 entitas domain diwujudkan pada 26 tabel. Angka ini adalah sumber tunggal jumlah tabel; dokumen lain merujuk ke sini, tidak menyebut angkanya sendiri.

## Aturan yang Berlaku di Seluruh Model

| Aturan | Penerapan | Sumber |
|--------|-----------|--------|
| Uang bilangan bulat rupiah | `bigint`, tanpa tipe pecahan di mana pun | Prinsip V |
| Periode mingguan | `date` berisi tanggal Senin awal minggu, tanpa zona waktu | Prinsip V |
| Waktu kejadian | `timestamptz`, dikonversi ke WIB hanya saat ditampilkan | Prinsip V |
| **Seluruh waktu berasal dari aplikasi** | **Tidak ada `DEFAULT now()` pada tabel mana pun.** Aplikasi mengirim setiap nilai waktu dari `Clock` yang disuntikkan | Prinsip V |
| Pengenal baris | `uuid` dengan `gen_random_uuid()`, tanpa dependency | Prinsip IV |
| Nilai turunan | Rating rata-rata dan tingkat penyelesaian **dihitung saat dibaca**, tidak disimpan sebagai kolom | R-07, FR-071 |
| Penghapusan | Tidak ada penghapusan fisik pada data acuan dan riwayat; pakai penanda nonaktif atau status | FR-060 |

Nama tabel, kolom, tipe, dan nilai enum ditulis bahasa Inggris sesuai aturan penamaan kode di `CLAUDE.md`. Prosa penjelasan tetap bahasa Indonesia karena ini dokumen, bukan kode.

**Larangan `DEFAULT now()` adalah perubahan dari versi sebelumnya**, dan alasannya bukan kerapian. Prinsip V mewajibkan seluruh hitungan bertenggat dapat diuji dengan waktu yang digantikan. `DEFAULT now()` melewati `Clock` sepenuhnya, sehingga pengujian yang menggeser waktu akan menghasilkan baris yang waktunya tidak konsisten dengan waktu uji, padahal FR-068, FR-037, serta FR-021 semuanya bergantung pada perbandingan waktu. Satu-satunya cara menutup celah itu adalah melarangnya di seluruh model, bukan sebagian.

Konsekuensi yang harus dijaga: setiap `INSERT` menyertakan kolom waktunya secara eksplisit. Bila sebuah kolom waktu ternyata lupa diisi, `NOT NULL` yang gagal akan menangkapnya saat pengujian, bukan menghasilkan waktu yang salah secara senyap.

Nilai turunan sengaja tidak dimaterialisasi. Kolom yang harus diperbarui setiap kali ulasan disembunyikan atau pesanan dibatalkan adalah sumber ketidaksesuaian yang paling sering muncul, dan pada 50 usaha demo maupun 200 usaha target SC-003 biaya menghitungnya saat dibaca tidak terasa.

---

## Peta Entitas ke Tabel

| # | Entitas Spec | Tabel | Catatan |
|---|--------------|-------|---------|
| 1 | Akun Pengguna | `user_account` | |
| 2 | Profil Usaha | `business_profile` | |
| 3 | Wilayah | `province`, `city` | Dua tabel, lihat alasan di bawah |
| 4 | Item Daftar Baku | `catalog_item` | Satu tabel untuk produk dan mesin, dibedakan kolom `type` |
| 5 | Usulan Item | `item_proposal` | |
| 6 | Pengajuan Verifikasi Identitas | `verification_request` | |
| 7 | Listing Kapasitas | `capacity_listing` | + `listing_product`, `listing_machine` |
| 8 | Periode Ketersediaan | `availability_period` | Diperpanjang saat dibutuhkan (FR-088) |
| 9 | Alokasi Kapasitas | `capacity_allocation` | |
| 10 | Request Kuota | `quota_request` | + `request_candidate` |
| 11 | Penawaran | `offer` | |
| 12 | Pesanan | `work_order` | + `work_order_status_history` |
| 13 | Catatan Pembayaran | `payment_record` | |
| 14 | Ulasan | `review` | |
| 15 | Sengketa | `dispute` | |
| 16 | Notifikasi | `notification` | + `notification_channel` |

**Tabel penopang** yang tidak berdiri sebagai entitas domain:

| Tabel | Dituntut oleh | Alasan tidak jadi entitas |
|-------|---------------|---------------------------|
| `session` | FR-003 | Mekanisme autentikasi, bukan konsep bisnis |
| `verification_code` | FR-002, R-09 | Kode enam digit email/telepon/pemulihan, disimpan sebagai hash |
| `uploaded_file` | FR-006, FR-009 | Metadata penyimpanan; entitas pemiliknya Pengajuan Verifikasi |
| `listing_product`, `listing_machine` | FR-012, FR-076 | Relasi banyak-ke-banyak; `listing_machine` menyimpan jumlah mesin per jenis |
| `request_candidate` | FR-030 | Satu status per kandidat pada satu request; tidak dapat jadi kolom |
| `work_order_status_history` | FR-039 | Waktu dan pelaku setiap perubahan status; tabel, bukan kolom |
| `notification_channel` | FR-085 | Jumlah percobaan dan status per kanal, terpisah per kanal |
| `rate_limit` | R-10 | Pembatasan laju yang harus bertahan setelah proses dijalankan ulang |

**Wilayah menjadi dua tabel, bukan satu tabel berhierarki.** Alasannya integritas: `business_profile` harus menunjuk kota/kabupaten, bukan provinsi. Dengan satu tabel berhierarki, tidak ada kunci asing yang dapat mencegah profil menunjuk provinsi. Karena tingkatnya tepat dua dan tetap (FR-062), memisahkannya tidak menimbulkan biaya.

---

## 1. Akun, Sesi, dan Profil

```sql
CREATE TABLE user_account (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email              citext NOT NULL,
    phone              text   NOT NULL,
    password_hash      text   NOT NULL,
    email_verified     boolean NOT NULL DEFAULT false,
    phone_verified     boolean NOT NULL DEFAULT false,
    role_subcontractor boolean NOT NULL DEFAULT false,
    role_buyer         boolean NOT NULL DEFAULT false,
    role_admin         boolean NOT NULL DEFAULT false,
    notif_nontx_email  boolean NOT NULL DEFAULT true,
    notif_nontx_whatsapp boolean NOT NULL DEFAULT true,
    created_at         timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL,

    CONSTRAINT email_unique UNIQUE (email),
    CONSTRAINT phone_unique UNIQUE (phone),
    CONSTRAINT phone_format CHECK (phone ~ '^62[0-9]{8,13}$'),
    CONSTRAINT has_at_least_one_role CHECK (
        role_subcontractor OR role_buyer OR role_admin
    ),
    CONSTRAINT admin_has_no_business_role CHECK (
        NOT role_admin OR (NOT role_subcontractor AND NOT role_buyer)
    )
);
```

Peran sebagai tiga kolom boolean, bukan enum, karena FR-001 mengizinkan satu akun memegang dua peran usaha sekaligus.

`admin_has_no_business_role` memisahkan admin dari pengguna usaha: admin memutuskan verifikasi dan mediasi (FR-005), dan membiarkannya juga bertransaksi menciptakan konflik kepentingan. **Constraint ini tidak diminta FR mana pun**, ia keputusan model. Bila kasus admin yang juga punya konveksi perlu didukung, lepaskan dan naikkan ke spec.

Nomor HP dinormalkan ke `62…` tanpa tanda plus agar keunikannya bermakna: `08…` dan `+628…` yang sama tidak boleh jadi dua akun.

`notif_nontx_email` dan `notif_nontx_whatsapp` menyimpan preferensi kanal untuk notifikasi non-transaksional (FR-053, FR-091). Keduanya default `true`; mematikannya hanya memengaruhi notifikasi yang bergolongan non-transaksional pada tabel `notification`. Notifikasi transaksional selalu terkirim ke kanal yang tersedia dan tidak membaca dua kolom ini. Preferensi disimpan di akun, bukan tabel terpisah, karena hanya dua tombol per akun dan tidak berkembang jadi pemetaan per kejadian.

```sql
CREATE TABLE session (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id     uuid NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    token_hash     bytea NOT NULL,
    source_address inet,
    expires_at     timestamptz NOT NULL,
    created_at     timestamptz NOT NULL,
    accessed_at    timestamptz NOT NULL,

    CONSTRAINT token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_session_account ON session (account_id);
CREATE INDEX idx_session_expires ON session (expires_at);
```

Yang disimpan adalah hash token, bukan token mentah (R-10).

```sql
CREATE TYPE verification_purpose AS ENUM ('email', 'phone', 'recovery');

CREATE TABLE verification_code (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    purpose     verification_purpose NOT NULL,
    code_hash   bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL
);

CREATE INDEX idx_verification_code_lookup ON verification_code (account_id, purpose);
```

Kode enam digit untuk verifikasi email dan telepon (R-09) serta pemulihan kata sandi. Yang disimpan adalah hash SHA-256 kodenya, bukan kode mentah, dengan alasan yang sama dengan token sesi: kebocoran basis data tidak boleh memberi kode yang masih berlaku. `consumed_at` menandai kode yang sudah dipakai agar satu kode hanya sah sekali; `expires_at` diisi aplikasi lewat `Clock`, bukan `DEFAULT`. `purpose` memisah tiga alur (verifikasi email, verifikasi telepon, pemulihan) sehingga kode satu alur tidak dapat dipakai di alur lain.

```sql
CREATE TABLE business_profile (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
    business_name text NOT NULL,
    city_code     text NOT NULL REFERENCES city(code) ON DELETE RESTRICT,
    latitude      numeric(9,6),
    longitude     numeric(9,6),
    description   text,
    verified      boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL,

    CONSTRAINT one_profile_per_account UNIQUE (account_id),
    CONSTRAINT business_name_not_empty CHECK (length(trim(business_name)) >= 3),
    CONSTRAINT coordinates_complete_or_empty CHECK (
        (latitude IS NULL AND longitude IS NULL) OR (latitude IS NOT NULL AND longitude IS NOT NULL)
    ),
    CONSTRAINT coordinates_within_indonesia CHECK (
        latitude IS NULL OR (latitude BETWEEN -11.5 AND 6.5 AND longitude BETWEEN 94.5 AND 141.5)
    )
);

CREATE INDEX idx_profile_city ON business_profile (city_code);
CREATE INDEX idx_profile_name ON business_profile (business_name);
```

`verified` adalah cache keputusan admin terakhir, disimpan karena dibaca pada setiap hasil pencarian (FR-008, FR-027). Ia tidak pernah mempengaruhi ketayangan maupun urutan (FR-010, FR-024). Ini menyimpang dari kriteria penerimaan dokumen sumber yang menempatkan validasi manual admin sebelum listing aktif [1]; penyimpangannya tercatat di Assumptions spec.

Yang tidak dapat ditegakkan basis data: titik koordinat yang berada jauh dari kota yang dipilih. Itu pemeriksaan aplikasi dengan peringatan, bukan penolakan, karena batas kota tidak tersedia di data yang kita simpan.

---

## 2. Wilayah dan Daftar Baku

```sql
CREATE TABLE province (
    code text PRIMARY KEY,
    name text NOT NULL,
    CONSTRAINT province_code_format CHECK (code ~ '^[0-9]{2}$')
);

CREATE TABLE city (
    code          text PRIMARY KEY,
    province_code text NOT NULL REFERENCES province(code) ON DELETE RESTRICT,
    name          text NOT NULL,
    CONSTRAINT city_code_format CHECK (code ~ '^[0-9]{4}$'),
    CONSTRAINT city_belongs_to_province CHECK (left(code, 2) = province_code)
);

CREATE INDEX idx_city_province ON city (province_code);
```

`city_belongs_to_province` memanfaatkan sifat kode wilayah resmi: dua digit pertama kode kabupaten/kota adalah kode provinsinya. Ini menangkap kesalahan pemetaan saat seed, dan gagal keras di sana alih-alih senyap saat pencarian. Bila R-02 menemukan kode dari sumber tidak mengikuti pola itu, constraint ini yang pertama gagal.

Tingkat ketiga perluasan pencarian, seluruh Indonesia (FR-063), tidak memerlukan tabel: ia berarti tidak ada penyaringan wilayah.

```sql
CREATE TYPE item_type AS ENUM ('product', 'machine');

CREATE TABLE catalog_item (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type       item_type NOT NULL,
    name       text NOT NULL,
    active     boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL,

    CONSTRAINT item_name_unique_per_type UNIQUE (type, name)
);

CREATE INDEX idx_item_active ON catalog_item (type, active) WHERE active;
```

Menonaktifkan item tidak menghapusnya (FR-060), sehingga listing yang sudah memakainya tetap utuh dan tetap dapat ditemukan.

```sql
CREATE TYPE proposal_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE item_proposal (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id    uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    type          item_type NOT NULL,
    proposed_name text NOT NULL,
    status        proposal_status NOT NULL DEFAULT 'pending',
    admin_note    text,
    decided_by    uuid REFERENCES user_account(id),
    decided_at    timestamptz,
    item_id       uuid REFERENCES catalog_item(id),
    created_at    timestamptz NOT NULL,

    CONSTRAINT decision_complete CHECK (
        (status = 'pending' AND decided_at IS NULL AND decided_by IS NULL)
        OR (status <> 'pending' AND decided_at IS NOT NULL AND decided_by IS NOT NULL)
    ),
    CONSTRAINT approved_yields_item CHECK (status <> 'approved' OR item_id IS NOT NULL)
);

CREATE INDEX idx_proposal_pending ON item_proposal (created_at) WHERE status = 'pending';
```

---

## 3. Berkas Unggahan dan Verifikasi Identitas

```sql
CREATE TYPE file_type AS ENUM ('identity_document', 'location_photo');

CREATE TABLE uploaded_file (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_profile_id  uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    type              file_type NOT NULL,
    original_name     text NOT NULL,
    mime_type         text NOT NULL,
    size_bytes        integer NOT NULL,
    storage_path      text NOT NULL,
    created_at        timestamptz NOT NULL,

    CONSTRAINT max_size CHECK (size_bytes > 0 AND size_bytes <= 5 * 1024 * 1024),
    CONSTRAINT allowed_type CHECK (mime_type IN ('image/jpeg', 'image/png', 'application/pdf')),
    CONSTRAINT storage_path_unique UNIQUE (storage_path)
);

CREATE INDEX idx_file_owner ON uploaded_file (owner_profile_id);
```

`original_name` hanya metadata tampilan; `storage_path` memakai UUID yang dibuat sistem. Batas 5MB ditegakkan di basis data selain di aplikasi, agar tidak ada jalur tulis yang melewatinya.

Yang wajib di aplikasi: pemeriksaan tipe dari magic bytes (bukan dari header yang dikirim), pembuangan metadata lokasi gambar, dan batas total penyimpanan 500MB.

```sql
CREATE TYPE verification_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE verification_request (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id           uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    identity_number      text NOT NULL,
    identity_file_id     uuid NOT NULL REFERENCES uploaded_file(id) ON DELETE RESTRICT,
    location_file_id     uuid NOT NULL REFERENCES uploaded_file(id) ON DELETE RESTRICT,
    status               verification_status NOT NULL DEFAULT 'pending',
    admin_note           text,
    decided_by           uuid REFERENCES user_account(id),
    decided_at           timestamptz,
    applicant_source_address inet,
    created_at           timestamptz NOT NULL,

    CONSTRAINT verification_decision_complete CHECK (
        (status = 'pending' AND decided_at IS NULL)
        OR (status <> 'pending' AND decided_at IS NOT NULL AND decided_by IS NOT NULL)
    ),
    CONSTRAINT rejection_needs_reason CHECK (status <> 'rejected' OR admin_note IS NOT NULL)
);

CREATE UNIQUE INDEX idx_one_pending_verification
    ON verification_request (profile_id) WHERE status = 'pending';
CREATE INDEX idx_verification_queue
    ON verification_request (created_at) WHERE status = 'pending';
```

Indeks unik parsial mengizinkan pengajuan ulang setelah penolakan (FR-011) sekaligus mencegah dua pengajuan menunggu sekaligus. Dokumen sumber menuntut unggah NIB/NIK dan foto lokasi usaha sebagai bagian verifikasi identitas [1]; keduanya diwakili dua kunci asing wajib di sini.

---

## 4. Listing Kapasitas

```sql
CREATE TABLE capacity_listing (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id          uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    weekly_capacity     integer NOT NULL,
    readiness_lead_days integer NOT NULL,
    published           boolean NOT NULL DEFAULT true,
    calendar_updated_at timestamptz NOT NULL,
    horizon_until       date NOT NULL,
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL,

    CONSTRAINT one_listing_per_profile UNIQUE (profile_id),
    CONSTRAINT capacity_positive CHECK (weekly_capacity > 0),
    CONSTRAINT lead_not_negative CHECK (readiness_lead_days >= 0 AND readiness_lead_days <= 365),
    CONSTRAINT horizon_is_monday CHECK (EXTRACT(ISODOW FROM horizon_until) = 1)
);

CREATE INDEX idx_listing_published ON capacity_listing (id) WHERE published;
CREATE INDEX idx_listing_calendar_stale ON capacity_listing (calendar_updated_at) WHERE published;
CREATE INDEX idx_listing_horizon ON capacity_listing (horizon_until) WHERE published;
```

**`horizon_until` adalah kolom baru untuk FR-088.** Ia menyimpan periode mingguan terjauh yang sudah pernah dibuat, sehingga pencarian dapat memeriksa satu kolom alih-alih menghitung `MAX(week_start)` dari `availability_period` pada setiap permintaan. Ketika deadline yang diminta melampaui nilai ini, aplikasi membuat periode yang kurang lalu memperbarui kolomnya.

Satu angka `weekly_capacity` untuk seluruh listing, tanpa kolom kapasitas per jenis produk (FR-076). Dokumen sumber memang meminta input kapasitas harian/mingguan dan jenis produk sebagai dua hal terpisah [1]; angka terpisah per produk ditolak karena mesin dan tenaga kerjanya berbagi.

`one_listing_per_profile` adalah penyederhanaan model: spec tidak pernah menyebut satu usaha punya beberapa listing, dan seluruh Acceptance Scenario memakai bentuk tunggal. Melepasnya nanti tidak mengubah tabel lain, tetapi kueri pencarian dan alokasi perlu disesuaikan.

`calendar_updated_at` terpisah dari `updated_at` karena FR-021 mengukur kebaruan kalender: mengubah harga atau deskripsi tidak boleh menghapus penanda "Data Belum Diperbarui".

```sql
CREATE TABLE listing_product (
    listing_id uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE CASCADE,
    item_id    uuid NOT NULL REFERENCES catalog_item(id) ON DELETE RESTRICT,
    PRIMARY KEY (listing_id, item_id)
);

CREATE TABLE listing_machine (
    listing_id    uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE CASCADE,
    item_id       uuid NOT NULL REFERENCES catalog_item(id) ON DELETE RESTRICT,
    machine_count integer NOT NULL,
    PRIMARY KEY (listing_id, item_id),
    CONSTRAINT machine_count_positive CHECK (machine_count > 0)
);

CREATE INDEX idx_listing_product_item ON listing_product (item_id);
CREATE INDEX idx_listing_machine_item ON listing_machine (item_id);
```

Indeks pada `item_id` adalah arah yang dipakai pencarian: dari jenis produk yang dicari menuju listing yang menyatakannya (FR-023 kriteria a dan b).

Yang tidak dapat ditegakkan kunci asing: bahwa `item_id` pada `listing_product` berjenis `product`, dan `item_id` pada `listing_machine` berjenis `machine`. Ditegakkan trigger, karena `CHECK` tidak boleh merujuk tabel lain. Satu fungsi dipakai bersama, jenis yang benar diteruskan sebagai argumen trigger sehingga tidak ada dua fungsi yang bisa menyimpang satu sama lain:

```sql
CREATE FUNCTION reject_wrong_item_type() RETURNS trigger AS $$
DECLARE
    v_type          text;
    v_expected_type text := TG_ARGV[0];
BEGIN
    SELECT i.type INTO v_type
      FROM catalog_item i WHERE i.id = NEW.item_id;

    IF v_type IS DISTINCT FROM v_expected_type THEN
        RAISE EXCEPTION
            'item % has type %, this table only accepts type %',
            NEW.item_id, v_type, v_expected_type;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reject_wrong_product_item
    BEFORE INSERT OR UPDATE ON listing_product
    FOR EACH ROW EXECUTE FUNCTION reject_wrong_item_type('product');

CREATE TRIGGER trg_reject_wrong_machine_item
    BEFORE INSERT OR UPDATE ON listing_machine
    FOR EACH ROW EXECUTE FUNCTION reject_wrong_item_type('machine');
```

Aplikasi tetap wajib menolak jenis yang salah lebih dulu dengan pesan yang jelas, trigger ini gerbang terakhir agar data tidak pernah rusak meski logika aplikasi keliru, bukan pengganti validasi masukan.

---

## 5. Periode Ketersediaan dan Alokasi Kapasitas

Inti FR-018 sampai FR-020, FR-077 sampai FR-079, FR-087, FR-088, dan FR-089, sekaligus tempat paling mungkin data rusak diam-diam.

```sql
CREATE TABLE availability_period (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id     uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE CASCADE,
    week_start     date NOT NULL,
    total_capacity integer NOT NULL,
    used_capacity  integer NOT NULL DEFAULT 0,
    marked_full    boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,

    CONSTRAINT one_period_per_week UNIQUE (listing_id, week_start),
    CONSTRAINT week_start_is_monday CHECK (EXTRACT(ISODOW FROM week_start) = 1),
    CONSTRAINT total_capacity_not_negative CHECK (total_capacity >= 0),
    CONSTRAINT used_capacity_within_total CHECK (
        used_capacity >= 0 AND used_capacity <= total_capacity
    )
);

CREATE INDEX idx_period_listing_week ON availability_period (listing_id, week_start);
CREATE INDEX idx_period_available ON availability_period (listing_id, week_start)
    WHERE NOT marked_full AND used_capacity < total_capacity;
```

Tiga constraint yang menanggung beban terbesar:

`week_start_is_monday` menegakkan Prinsip V pada tingkat data. Tanpa ini, satu galat perhitungan batas minggu menghasilkan periode tumpang tindih, dan penjumlahan kapasitas jadi salah tanpa gejala.

`used_capacity_within_total` adalah penegakan FR-079 dan gerbang SC-018. Bila logika alokasi keliru, transaksi gagal keras alih-alih menghasilkan kapasitas minus. Spec sengaja tidak menyebut cara penegakannya. Itu keputusan model, dan alasannya ada di `research.md` R-04.

`one_period_per_week` membuat penguncian baris pada R-04 bermakna: satu minggu satu baris.

**Dua keadaan yang tidak dilarang basis data dan wajib ditangani aplikasi**, keduanya edge case yang sudah tercatat di spec:

FR-089: menurunkan `weekly_capacity` listing memperbarui `total_capacity` seluruh periode mendatang **yang belum memiliki alokasi**. Periode yang sudah punya alokasi tidak diubah, karena menurunkannya di bawah `used_capacity` akan ditolak constraint. Aplikasi harus menyaring periode berdasarkan ada tidaknya baris alokasi aktif, bukan mencoba memperbarui semuanya lalu menangkap galat.

Penandaan penuh atas periode yang sudah teralokasi harus ditolak aplikasi dengan pesan yang menyebut minggu mana beserta jumlah terpakainya, bukan meneruskan galat basis data mentah.

```sql
CREATE TABLE capacity_allocation (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id uuid NOT NULL REFERENCES work_order(id) ON DELETE RESTRICT,
    period_id    uuid NOT NULL REFERENCES availability_period(id) ON DELETE RESTRICT,
    quantity     integer NOT NULL,
    created_at   timestamptz NOT NULL,
    reversed_at  timestamptz,

    CONSTRAINT one_allocation_per_order_period UNIQUE (work_order_id, period_id),
    CONSTRAINT allocation_quantity_positive CHECK (quantity > 0)
);

CREATE INDEX idx_allocation_order ON capacity_allocation (work_order_id);
CREATE INDEX idx_allocation_period ON capacity_allocation (period_id) WHERE reversed_at IS NULL;
```

Satu pesanan memiliki beberapa baris alokasi pada minggu berurutan mulai dari minggu kesiapan mulai (FR-077, FR-087). Pesanan 3.000 potong pada kapasitas 500 per minggu menghasilkan enam baris.

`reversed_at` menyimpan jejak pembatalan alih-alih menghapus baris, sehingga riwayat mediasi tetap dapat dibaca admin (FR-046).

**Trigger untuk FR-087.** Alokasi tidak boleh menyentuh periode sebelum minggu kesiapan mulai pesanan. Perbandingannya melintasi tiga tabel, jadi `CHECK` tidak dapat dipakai:

```sql
CREATE FUNCTION reject_allocation_before_readiness() RETURNS trigger AS $$
DECLARE
    v_period_week    date;
    v_readiness_week date;
BEGIN
    SELECT p.week_start INTO v_period_week
      FROM availability_period p WHERE p.id = NEW.period_id;

    SELECT o.readiness_week_start INTO v_readiness_week
      FROM work_order o WHERE o.id = NEW.work_order_id;

    IF v_period_week < v_readiness_week THEN
        RAISE EXCEPTION
            'FR-087: allocation on week % precedes readiness start week %',
            v_period_week, v_readiness_week;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reject_allocation_before_readiness
    BEFORE INSERT OR UPDATE ON capacity_allocation
    FOR EACH ROW EXECUTE FUNCTION reject_allocation_before_readiness();
```

Aturan ini mudah dilanggar tanpa disadari: alokasi yang naif akan mulai dari minggu berjalan, dan itu berarti menjadwalkan pekerjaan pada minggu yang menurut pernyataan subkontraktor sendiri belum dapat dipakai. Bug seperti itu tidak akan terlihat pada pengujian manual karena angka totalnya tetap benar.

---

## 6. Request Kuota dan Penawaran

```sql
CREATE TABLE quota_request (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id       uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    product_item_id uuid NOT NULL REFERENCES catalog_item(id) ON DELETE RESTRICT,
    quantity       integer NOT NULL,
    material       text NOT NULL,
    deadline       date NOT NULL,
    note           text,
    reply_due_at   timestamptz NOT NULL,
    created_at     timestamptz NOT NULL,

    CONSTRAINT request_quantity_positive CHECK (quantity > 0),
    CONSTRAINT reply_due_after_created CHECK (reply_due_at > created_at)
);

CREATE INDEX idx_request_buyer ON quota_request (buyer_id, created_at DESC);
CREATE INDEX idx_request_due ON quota_request (reply_due_at);
```

**Perbaikan C-3.** Versi sebelumnya memasang `CHECK (reply_due_at = created_at + interval '72 hours')` bersama `created_at DEFAULT now()`. Kombinasi itu **selalu gagal**: aplikasi menghitung `reply_due_at` dari `Clock.Now()` sementara basis data mengisi `created_at` dari `now()`, dan keduanya berbeda mikrodetik. `DEFAULT now()` juga melewati `Clock`, sehingga pengujian kedaluwarsa dengan waktu digeser tidak dapat menghasilkan baris yang konsisten.

Sekarang aplikasi mengirim kedua nilai dari `Clock`, dan constraint hanya menjaga urutannya. Angka 72 jam sendiri ditegakkan aplikasi dan diuji, bukan oleh basis data, karena basis data tidak boleh punya sumber waktu sendiri.

```sql
CREATE TYPE candidate_status AS ENUM (
    'awaiting_reply', 'offered', 'rejected', 'expired', 'not_continued', 'agreed'
);

CREATE TABLE request_candidate (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id        uuid NOT NULL REFERENCES quota_request(id) ON DELETE CASCADE,
    listing_id        uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE RESTRICT,
    subcontractor_id  uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    status            candidate_status NOT NULL DEFAULT 'awaiting_reply',
    rejection_reason  text,
    updated_at        timestamptz NOT NULL,

    CONSTRAINT one_candidate_per_request UNIQUE (request_id, listing_id)
);

CREATE INDEX idx_candidate_subcon ON request_candidate (subcontractor_id, status);
CREATE INDEX idx_candidate_request ON request_candidate (request_id);
CREATE UNIQUE INDEX idx_one_agreement_per_request
    ON request_candidate (request_id) WHERE status = 'agreed';
```

`idx_one_agreement_per_request` menegakkan FR-034: menerima satu penawaran menutup kandidat lain, sehingga tidak mungkin ada dua kesepakatan dari satu request.

FR-081 dan FR-083, larangan request ke listing sendiri, tidak dapat dinyatakan `CHECK` karena membandingkan `subcontractor_id` di tabel ini dengan `buyer_id` di tabel lain:

```sql
CREATE FUNCTION reject_self_request() RETURNS trigger AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM quota_request r
        WHERE r.id = NEW.request_id AND r.buyer_id = NEW.subcontractor_id
    ) THEN
        RAISE EXCEPTION 'FR-083: quota request cannot be sent to your own listing';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reject_self_request
    BEFORE INSERT OR UPDATE ON request_candidate
    FOR EACH ROW EXECUTE FUNCTION reject_self_request();
```

Aplikasi menolaknya lebih awal dengan pesan yang dapat dibaca pengguna; trigger adalah jaring pengaman untuk jalur yang dikirim tanpa melalui hasil pencarian, yang disebut eksplisit di FR-083.

```sql
CREATE TYPE offer_party AS ENUM ('subcontractor', 'buyer');

CREATE TABLE offer (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id        uuid NOT NULL REFERENCES request_candidate(id) ON DELETE CASCADE,
    sequence            integer NOT NULL,
    proposed_by         offer_party NOT NULL,
    total_price         bigint NOT NULL,
    readiness_lead_days integer NOT NULL,
    note                text,
    created_at          timestamptz NOT NULL,

    CONSTRAINT sequence_unique_per_candidate UNIQUE (candidate_id, sequence),
    CONSTRAINT price_positive CHECK (total_price > 0),
    CONSTRAINT offer_lead_reasonable CHECK (readiness_lead_days >= 0 AND readiness_lead_days <= 365)
);

CREATE INDEX idx_offer_candidate ON offer (candidate_id, sequence);
```

Setiap counter-offer adalah baris baru dengan `sequence` bertambah, sehingga seluruh riwayat negosiasi tersimpan (FR-033). Dokumen sumber menempatkan negosiasi harga sebagai kirim estimasi, lalu terima, tolak, atau ajukan counter-offer [1], dan rangkaian itulah yang direkam kolom `sequence` dan `proposed_by`.

`total_price` bertipe `bigint` dalam rupiah bulat.

---

## 7. Pesanan, Riwayat Status, dan Pembayaran

```sql
CREATE TYPE work_order_status AS ENUM (
    'accepted', 'production', 'completed', 'shipped', 'confirmed', 'cancelled', 'in_mediation'
);

CREATE TABLE work_order (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id        uuid NOT NULL REFERENCES request_candidate(id) ON DELETE RESTRICT,
    offer_id            uuid NOT NULL REFERENCES offer(id) ON DELETE RESTRICT,
    buyer_id            uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    subcontractor_id    uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    quantity            integer NOT NULL,
    total_price         bigint NOT NULL,
    deadline            date NOT NULL,
    readiness_week_start date NOT NULL,
    status              work_order_status NOT NULL DEFAULT 'accepted',
    shipped_at          timestamptz,
    confirmed_at        timestamptz,
    auto_confirmed      boolean NOT NULL DEFAULT false,
    cancelled_by_id     uuid REFERENCES business_profile(id),
    cancellation_reason text,
    cancelled_at        timestamptz,
    created_at          timestamptz NOT NULL,

    CONSTRAINT one_order_per_candidate UNIQUE (candidate_id),
    CONSTRAINT two_distinct_parties CHECK (buyer_id <> subcontractor_id),
    CONSTRAINT order_quantity_positive CHECK (quantity > 0),
    CONSTRAINT order_price_positive CHECK (total_price > 0),
    CONSTRAINT readiness_is_monday CHECK (EXTRACT(ISODOW FROM readiness_week_start) = 1),
    CONSTRAINT readiness_not_past_deadline CHECK (readiness_week_start <= deadline),
    CONSTRAINT cancellation_complete CHECK (
        (status <> 'cancelled')
        OR (cancelled_by_id IS NOT NULL AND cancellation_reason IS NOT NULL
            AND cancelled_at IS NOT NULL)
    ),
    CONSTRAINT shipped_before_confirmed CHECK (
        confirmed_at IS NULL OR shipped_at IS NULL OR confirmed_at >= shipped_at
    ),
    CONSTRAINT auto_confirm_needs_confirmation CHECK (
        NOT auto_confirmed OR confirmed_at IS NOT NULL
    )
);

CREATE INDEX idx_order_buyer ON work_order (buyer_id, status);
CREATE INDEX idx_order_subcon ON work_order (subcontractor_id, status);
CREATE INDEX idx_order_deadline_active ON work_order (deadline)
    WHERE status IN ('accepted', 'production', 'completed', 'shipped');
CREATE INDEX idx_order_auto_confirm ON work_order (shipped_at) WHERE status = 'shipped';
```

**`readiness_week_start` adalah kolom baru untuk FR-087.** Ia dihitung sekali saat kesepakatan terbentuk, dari tanggal kesepakatan ditambah `readiness_lead_days` listing, lalu dibulatkan ke Senin minggu yang memuatnya. Disimpan alih-alih dihitung ulang karena `readiness_lead_days` pada listing dapat berubah kemudian, sementara alokasi pesanan yang sudah terbentuk tidak boleh bergeser. Ini juga yang menutup salah satu edge case spec: subkontraktor mengubah jeda kesiapan setelah punya alokasi berjalan.

`readiness_not_past_deadline` menegakkan FR-090 pada tingkat data: pesanan yang produksinya baru dapat dimulai setelah deadline tidak dapat terbentuk.

`cancelled_by_id` adalah dasar FR-072: pembatalan masuk pembagi tingkat penyelesaian hanya bagi pihak yang membatalkan. Tanpa kolom ini, rumus itu tidak dapat dihitung.

`two_distinct_parties` adalah lapisan kedua atas larangan request ke diri sendiri: bahkan bila trigger dilewati, pesanan berdua pihak sama tidak dapat terbentuk.

Dua indeks parsial terakhir adalah jalur penjadwal R-07: `idx_order_auto_confirm` untuk FR-068 dan FR-069, `idx_order_deadline_active` untuk FR-045.

**Transisi status yang sah** (FR-044). Semua transisi lain ditolak beserta penjelasan urutan yang diizinkan:

```text
accepted ──▶ production ──▶ completed ──▶ shipped ──▶ confirmed
    │            │             │            │
    │            └─────────────┴────────────┴──▶ in_mediation ──▶ cancelled
    │                                                          └──▶ confirmed
    └──▶ cancelled            (pembatalan sendiri, FR-065: hanya dari 'accepted')
```

| Transisi | Pelaku | Aturan |
|----------|--------|--------|
| `accepted → production` | Subkontraktor | Menutup jalur pembatalan sendiri (FR-066) |
| `production → completed → shipped` | Subkontraktor | Berurutan, tidak boleh melompat |
| `shipped → confirmed` | Pemberi order, atau sistem setelah 7 hari | FR-068; `auto_confirmed` menandai yang mana |
| `accepted → cancelled` | Kedua pihak | Wajib beralasan; seluruh alokasi dibalik (FR-020, FR-065) |
| `* → in_mediation` | Kedua pihak melapor | Menghentikan hitungan konfirmasi otomatis (FR-070) |
| `in_mediation → cancelled` | Admin | Admin menentukan pengembalian alokasi dan pihak penanggung (FR-067) |

```sql
CREATE TABLE work_order_status_history (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id uuid NOT NULL REFERENCES work_order(id) ON DELETE CASCADE,
    old_status   work_order_status,
    new_status   work_order_status NOT NULL,
    changed_by   uuid REFERENCES user_account(id),
    by_system    boolean NOT NULL DEFAULT false,
    note         text,
    created_at   timestamptz NOT NULL,

    CONSTRAINT actor_clear CHECK (by_system OR changed_by IS NOT NULL)
);

CREATE INDEX idx_status_history_order ON work_order_status_history (work_order_id, created_at);
```

`actor_clear` memastikan FR-039 terpenuhi: setiap perubahan punya pelaku, dan perubahan oleh penjadwal ditandai `by_system` alih-alih dibiarkan tanpa identitas.

```sql
CREATE TYPE payment_direction AS ENUM ('sent', 'received');

CREATE TABLE payment_record (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id uuid NOT NULL REFERENCES work_order(id) ON DELETE CASCADE,
    profile_id   uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    direction    payment_direction NOT NULL,
    date         date NOT NULL,
    note         text,
    created_at   timestamptz NOT NULL,

    CONSTRAINT one_statement_per_party_per_direction UNIQUE (work_order_id, profile_id, direction)
);

CREATE INDEX idx_payment_order ON payment_record (work_order_id);
```

Tidak ada kolom jumlah uang, dan itu disengaja. Platform tidak memproses dana (FR-040); yang dicatat hanya pernyataan bahwa pembayaran terjadi. FR-043, perbedaan pernyataan antar pihak, dihitung dari ada tidaknya pasangan baris, bukan dari perbandingan angka.

Dokumen sumber menempatkan escrow sebagai penahan dana yang dirilis saat pesanan dikonfirmasi selesai [1]. Versi ini menggantinya dengan pencatatan pernyataan, dan konsekuensinya tercatat di Assumptions spec.

---

## 8. Reputasi dan Sengketa

```sql
CREATE TABLE review (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id    uuid NOT NULL REFERENCES work_order(id) ON DELETE RESTRICT,
    reviewer_id      uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    reviewee_id      uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    rating           smallint NOT NULL,
    text             text,
    hidden           boolean NOT NULL DEFAULT false,
    hidden_by        uuid REFERENCES user_account(id),
    hidden_at        timestamptz,
    hidden_reason    text,
    created_at       timestamptz NOT NULL,

    CONSTRAINT one_review_per_order_per_reviewer UNIQUE (work_order_id, reviewer_id),
    CONSTRAINT rating_one_to_five CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT no_self_review CHECK (reviewer_id <> reviewee_id),
    CONSTRAINT hiding_complete CHECK (
        NOT hidden
        OR (hidden_by IS NOT NULL AND hidden_at IS NOT NULL
            AND hidden_reason IS NOT NULL)
    )
);

CREATE INDEX idx_review_reviewee ON review (reviewee_id) WHERE NOT hidden;
CREATE INDEX idx_review_order ON review (work_order_id);
```

Indeks parsial hanya memuat ulasan yang tampil, karena rata-rata rating harus mengecualikan yang disembunyikan (FR-050). Moderasi ulasan oleh admin adalah sub-fitur yang diminta dokumen sumber [1], dan `hidden_reason` membuat tindakan itu tercatat, bukan hanya berlaku.

Yang tidak dapat ditegakkan basis data: bahwa pesanan sudah berstatus `confirmed` saat ulasan dibuat (FR-047). Ditegakkan aplikasi.

**Nilai turunan, dihitung saat dibaca:**

```sql
-- Rating rata-rata dan jumlah pekerjaan selesai (FR-048)
SELECT
    round(avg(r.rating)::numeric, 2)                             AS average_rating,
    count(*)                                                     AS review_count,
    (SELECT count(*) FROM work_order o
      WHERE o.subcontractor_id = $1 AND o.status = 'confirmed')  AS completed_jobs
FROM review r
WHERE r.reviewee_id = $1 AND NOT r.hidden;

-- Tingkat penyelesaian (FR-071, FR-072, FR-073)
WITH involved AS (
    SELECT status, cancelled_by_id
    FROM work_order
    WHERE buyer_id = $1 OR subcontractor_id = $1
)
SELECT
    count(*) FILTER (WHERE status = 'confirmed') AS completed,
    count(*) FILTER (
        WHERE status <> 'cancelled' OR cancelled_by_id = $1
    ) AS divisor
FROM involved;
```

Pembagi mengecualikan pesanan yang dibatalkan pihak lain, sesuai FR-072. Ambang 3 pesanan (FR-073) diterapkan di lapisan penyajian: bila `divisor < 3`, yang dikirim adalah keterangan bahwa data belum cukup, bukan angka. Dokumen sumber menyebut statistik tingkat penyelesaian sebagai sub-fitur profil reputasi [1] tanpa memberi rumus; rumus di atas adalah keputusan spec kita.

```sql
CREATE TYPE dispute_status AS ENUM ('reported', 'in_mediation', 'resolved');

CREATE TABLE dispute (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id      uuid NOT NULL REFERENCES work_order(id) ON DELETE RESTRICT,
    reporter_id        uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    report_body        text NOT NULL,
    status             dispute_status NOT NULL DEFAULT 'reported',
    admin_note         text,
    allocation_reversed boolean,
    liable_party_id    uuid REFERENCES business_profile(id),
    handled_by         uuid REFERENCES user_account(id),
    resolved_at        timestamptz,
    created_at         timestamptz NOT NULL,

    CONSTRAINT resolution_complete CHECK (
        status <> 'resolved'
        OR (handled_by IS NOT NULL AND resolved_at IS NOT NULL
            AND allocation_reversed IS NOT NULL AND admin_note IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_one_open_dispute
    ON dispute (work_order_id) WHERE status <> 'resolved';
CREATE INDEX idx_dispute_queue ON dispute (created_at) WHERE status <> 'resolved';
```

`resolution_complete` menegakkan FR-067: admin tidak dapat menutup mediasi tanpa memutuskan secara eksplisit apakah alokasi dikembalikan dan siapa yang menanggung. Keduanya `NULL` selama sengketa belum selesai, bukan diberi nilai bawaan yang menyesatkan.

`idx_one_open_dispute` menutup edge case pelaporan berulang untuk menghentikan konfirmasi otomatis berkali-kali. Penanganan sengketa lewat mediasi admin memang jalur yang dipilih dokumen sumber untuk fase awal, karena penanganan legal formal menuntut tim hukum dan asuransi [1].

---

## 9. Notifikasi dan Pembatasan Laju

```sql
CREATE TYPE event_type AS ENUM (
    'request_received', 'offer_received', 'counter_offer', 'agreement_formed',
    'order_status_changed', 'payment_record', 'deadline_approaching', 'deadline_passed',
    'verification_decision', 'rating_request', 'order_cancelled',
    'confirmation_due_approaching', 'order_auto_closed', 'item_proposal_decision',
    'calendar_stale'
);

CREATE TABLE notification (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    event         event_type NOT NULL,
    transactional boolean NOT NULL,
    title         text NOT NULL,
    body          text NOT NULL,
    link          text,
    read_at       timestamptz,
    created_at    timestamptz NOT NULL
);

CREATE INDEX idx_notification_account ON notification (account_id, created_at DESC);
CREATE INDEX idx_notification_unread ON notification (account_id) WHERE read_at IS NULL;
```

**`transactional` adalah kolom baru untuk FR-091.** Penggolongannya ditetapkan spec dan disimpan per baris agar pengirim dapat menghormati preferensi kanal tanpa memelihara pemetaan terpisah di kode:

| Kejadian | Golongan |
|----------|----------|
| `request_received`, `offer_received`, `counter_offer`, `agreement_formed` | Transaksional |
| `order_status_changed`, `order_cancelled`, `payment_record` | Transaksional |
| `deadline_passed`, `confirmation_due_approaching`, `order_auto_closed` | Transaksional |
| `verification_decision`, `item_proposal_decision` | Transaksional |
| `calendar_stale`, `deadline_approaching`, `rating_request` | Non-transaksional |

Hanya yang non-transaksional dapat dimatikan pengguna (FR-053). Perhatikan bahwa `deadline_approaching` non-transaksional sementara `confirmation_due_approaching` transaksional, karena yang kedua berujung pada penutupan pesanan otomatis, sehingga tidak boleh dapat dimatikan.

```sql
CREATE TYPE notification_channel_type AS ENUM ('email', 'whatsapp');
CREATE TYPE delivery_status AS ENUM ('pending', 'sent', 'failed_permanent');

CREATE TABLE notification_channel (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id uuid NOT NULL REFERENCES notification(id) ON DELETE CASCADE,
    channel         notification_channel_type NOT NULL,
    status          delivery_status NOT NULL DEFAULT 'pending',
    attempts        smallint NOT NULL DEFAULT 0,
    last_error      text,
    attempted_at    timestamptz,
    sent_at         timestamptz,

    CONSTRAINT one_channel_per_notification UNIQUE (notification_id, channel),
    CONSTRAINT attempts_max_three CHECK (attempts >= 0 AND attempts <= 3),
    CONSTRAINT failed_after_three_attempts CHECK (
        status <> 'failed_permanent' OR attempts = 3
    )
);

CREATE INDEX idx_channel_queue ON notification_channel (attempted_at NULLS FIRST)
    WHERE status = 'pending';
```

`attempts_max_three` dan `failed_after_three_attempts` menegakkan FR-085.

Baris `notification` ditulis di dalam transaksi kejadiannya, sedangkan `notification_channel` diproses goroutine pengirim setelah transaksi berhasil. Ini memenuhi FR-054 dan FR-086 sekaligus: notifikasi di dalam platform tetap ada meskipun email dan WhatsApp gagal seluruhnya, dan kegagalan kirim tidak pernah menggagalkan pesanan. Dokumen sumber meminta ketiga kanal (email, WhatsApp, dan in-app [1]), dan pemisahan tabel inilah yang membuat kanal ketiga tidak bergantung pada dua yang pertama.

```sql
CREATE TYPE rate_limit_target AS ENUM ('login_account', 'otp_phone', 'otp_address', 'quota_request');

CREATE TABLE rate_limit (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target       rate_limit_target NOT NULL,
    key          text NOT NULL,
    window_start timestamptz NOT NULL,
    count        integer NOT NULL DEFAULT 1,

    CONSTRAINT one_row_per_key_window UNIQUE (target, key, window_start),
    CONSTRAINT count_positive CHECK (count > 0)
);

CREATE INDEX idx_rate_limit_cleanup ON rate_limit (window_start);
```

Tabel, bukan penyimpanan dalam memori (R-10), agar batas tetap berlaku setelah proses dijalankan ulang.

---

## 10. Kueri Pencarian dan Rentang Kapasitas

Bentuk kueri FR-023 sampai FR-025, FR-080, FR-087, dan FR-088. Ini acuan bagi `contracts/` dan T035.

**Rentang kapasitas** setiap kandidat berbeda karena bergantung pada `readiness_lead_days` masing-masing:

```text
readiness_week(candidate) = monday_of(search_date + readiness_lead_days)
deadline_week             = monday_of(requested_deadline)
capacity range            = [readiness_week(candidate) .. deadline_week]
```

Kandidat yang `readiness_week`-nya melampaui `deadline_week` memiliki rentang kosong, sehingga kapasitas tersisanya nol dan kriteria (d) tidak terpenuhi.

```sql
WITH param AS (
    SELECT
        $1::date  AS search_date,
        $2::date  AS deadline_week,       -- sudah dibulatkan ke Senin oleh aplikasi
        $3::int   AS quantity,
        $4::uuid  AS product_item,        -- boleh NULL (FR-023: kriteria tak dievaluasi)
        $5::uuid  AS machine_item,        -- boleh NULL
        $6::int   AS max_lead,            -- boleh NULL
        $7::uuid  AS searcher_profile,
        $8::text  AS city_code,           -- NULL bila tingkat provinsi atau nasional
        $9::text  AS province_code        -- NULL bila tingkat kota atau nasional
),
base_candidate AS (
    SELECT
        l.id AS listing_id,
        l.profile_id,
        pr.business_name,
        l.weekly_capacity,
        l.readiness_lead_days,
        l.horizon_until,
        -- FR-087: minggu paling awal yang boleh dihitung
        date_trunc('week', p.search_date + (l.readiness_lead_days || ' days')::interval)::date
            AS readiness_week
    FROM capacity_listing l
    JOIN business_profile pr ON pr.id = l.profile_id
    CROSS JOIN param p
    WHERE l.published
      AND l.profile_id <> p.searcher_profile                     -- FR-081
      AND (p.city_code  IS NULL OR pr.city_code = p.city_code)
      AND (p.province_code IS NULL OR pr.city_code IN (
              SELECT code FROM city WHERE province_code = p.province_code))
),
capacity AS (
    SELECT
        c.listing_id,
        -- Periode yang sudah ada di dalam rentang kapasitas
        coalesce(sum(pk.total_capacity - pk.used_capacity), 0)
            AS recorded_remaining,
        -- FR-088: minggu di dalam rentang yang belum pernah dibuat
        greatest(0, (
            (p.deadline_week - greatest(c.readiness_week, c.horizon_until + 7)) / 7 + 1
        )) * c.weekly_capacity AS uncreated_remaining
    FROM base_candidate c
    CROSS JOIN param p
    LEFT JOIN availability_period pk
           ON pk.listing_id = c.listing_id
          AND NOT pk.marked_full
          AND pk.week_start BETWEEN c.readiness_week AND p.deadline_week
    GROUP BY c.listing_id, c.readiness_week, c.horizon_until,
             c.weekly_capacity, p.deadline_week
),
scored AS (
    SELECT
        c.*,
        (cap.recorded_remaining + cap.uncreated_remaining) AS remaining_capacity,
        -- FR-023: kriteria yang filternya tidak diisi dihitung terpenuhi
        (p.product_item IS NULL OR EXISTS (
            SELECT 1 FROM listing_product lp
             WHERE lp.listing_id = c.listing_id AND lp.item_id = p.product_item))::int
            AS product_match,
        (p.machine_item IS NULL OR EXISTS (
            SELECT 1 FROM listing_machine lm
             WHERE lm.listing_id = c.listing_id AND lm.item_id = p.machine_item))::int
            AS machine_match,
        (p.max_lead IS NULL OR c.readiness_lead_days <= p.max_lead)::int
            AS lead_match,
        ((cap.recorded_remaining + cap.uncreated_remaining) >= p.quantity)::int
            AS capacity_enough
    FROM base_candidate c
    JOIN capacity cap ON cap.listing_id = c.listing_id
    CROSS JOIN param p
),
ranked AS (
    SELECT *,
           (product_match + machine_match + lead_match + capacity_enough) AS score
    FROM scored
)
SELECT *
FROM ranked
WHERE (score, remaining_capacity, -readiness_lead_days, business_name, listing_id)
      < ($10, $11, -$12, $13, $14)                                -- keyset, R-05
ORDER BY score DESC, remaining_capacity DESC, readiness_lead_days ASC,
         business_name ASC, listing_id ASC
LIMIT $15;
```

Lima hal yang perlu diperhatikan tentang kueri ini.

**`score` dihitung di CTE `ranked`, bukan langsung di `SELECT` terluar.** Postgres tidak mengizinkan alias kolom keluaran dipakai di `WHERE` pada level yang sama, sedangkan keyset R-05 harus menyaring dengan `score`. Membungkus perhitungan skor di satu CTE lalu menyaring di atasnya membuat `score` dan `remaining_capacity` tersedia sebagai kolom biasa di `WHERE` maupun `ORDER BY`.

**Perhitungan minggu memakai `date_trunc('week', …)`** yang di PostgreSQL memulai minggu pada Senin, sama dengan Prinsip V. Aplikasi tetap yang membulatkan `deadline` ke Senin sebelum mengirimnya, agar satu-satunya sumber kebenaran pembulatan ada di kode Go yang dapat diuji.

**`uncreated_remaining` adalah perkiraan optimis** atas periode yang belum ada: setiap minggu di luar `horizon_until` dihitung berkapasitas penuh, karena periode baru memang dibuat dengan `weekly_capacity` sebagai kapasitas total (FR-088). Ini yang mencegah kandidat dinilai tidak memenuhi kriteria hanya karena periodenya belum pernah dibuat.

**Periode benar-benar dibuat saat kesepakatan terbentuk**, bukan di dalam kueri ini. Ketika sebuah penawaran diterima (`/offers/{offerId}/accept`), aplikasi memeriksa `horizon_until < deadline_week` lalu membuat periode yang kurang di dalam transaksi kesepakatan itu (FR-088). Menempatkannya di sana, bukan di jalur pencarian, membuat pencarian tetap operasi baca dan tidak memicu penulisan pada setiap permintaan.

**Kriteria yang filternya `NULL` dihitung terpenuhi** (FR-023, keputusan C-4). Skor tetap 0–4 tanpa normalisasi, sehingga kriteria yang tidak dievaluasi menaikkan skor semua kandidat secara seragam dan tidak membedakan siapa pun. Respons menyertakan nilai per kriteria agar klien dapat menyebutkan mana yang tidak dievaluasi (FR-026).

**Batas yang perlu diketahui**: `score` dan `remaining_capacity` adalah hasil perhitungan sehingga tidak dapat diindeks. Indeks yang ada mempercepat penyaringan dan penggabungan (`idx_listing_published`, `idx_profile_city`, `idx_period_available`, `idx_listing_product_item`), tetapi pengurutan dan agregasi dikerjakan saat kueri berjalan. Pada 50 usaha demo maupun 200 usaha SC-003 ini tidak masalah. Bila SC-010 terancam pada data lebih besar, langkah berikutnya adalah tabel ringkasan kapasitas per listing per minggu yang diperbarui saat alokasi berubah, bukan menambah indeks, karena tidak ada indeks yang menolong kolom hasil hitungan. Ini kandidat optimasi, bukan pekerjaan sekarang, sesuai Standar Performa konstitusi.

---

## 11. Ringkasan Penegakan Aturan

Yang hanya bergantung pada aplikasi adalah kandidat utama pengujian otomatis yang diwajibkan konstitusi.

| Aturan | Basis Data | Aplikasi |
|--------|-----------|----------|
| Kapasitas terpakai ≤ total (FR-079, SC-018) | `CHECK` | Penguncian baris terurut (R-04) |
| Alokasi tidak mendahului minggu kesiapan (FR-087, SC-020) | Trigger | Perhitungan minggu kesiapan dari `Clock` |
| Kesiapan tidak melewati deadline (FR-090) | `CHECK` | Penolakan penawaran beserta penjelasan |
| Horizon diperpanjang sampai deadline (FR-088, SC-021) | Tidak ada | Pembuatan periode saat kesepakatan terbentuk |
| Propagasi perubahan kapasitas (FR-089) | Tidak ada | Menyaring periode tanpa alokasi aktif |
| Satu kesepakatan per request (FR-034) | Indeks unik parsial | Penutupan kandidat lain |
| Larangan request ke diri sendiri (FR-081, FR-083) | Trigger + `two_distinct_parties` | Penyaringan pencarian, pesan pengguna |
| Batas balasan 72 jam (FR-082) | `CHECK` urutan waktu saja | Perhitungan 72 jam dari `Clock` |
| Minggu dimulai Senin (Prinsip V) | `CHECK` pada tiga tabel | Perhitungan batas minggu di WIB |
| Seluruh waktu dari `Clock` (Prinsip V) | Tidak ada `DEFAULT now()` | Setiap `INSERT` mengirim waktunya |
| Satu ulasan per pesanan per pihak (FR-047) | Unik | Pemeriksaan status pesanan dikonfirmasi |
| Rating 1–5 (FR-047) | `CHECK` | Tidak ada |
| Percobaan kirim maksimal 3 (FR-085) | `CHECK` | Logika percobaan ulang |
| Golongan notifikasi (FR-091) | Kolom `transactional` | Penetapan golongan saat penulisan |
| Mediasi wajib memutuskan alokasi dan penanggung (FR-067) | `CHECK` | Antarmuka admin |
| Batas ukuran berkas 5MB | `CHECK` | Magic bytes, pembuangan EXIF, kuota total 500MB |
| Item produk tidak tertukar dengan mesin | Trigger | Validasi form |
| Transisi status sah (FR-044) | Tidak ada | Mesin keadaan di `internal/order` |
| Urutan hasil deterministik (SC-013) | Tidak ada | `ORDER BY` lengkap + keyset (R-05) |
| Alokasi mengisi minggu terawal (FR-018) | Tidak ada | Algoritma alokasi |
| Ambang 3 pesanan (FR-073) | Tidak ada | Lapisan penyajian |

---

## 12. Urutan Migrasi

15 migrasi berurutan, mengikuti arah ketergantungan kunci asing. Daftar ini adalah sumber tunggal jumlah migrasi; dokumen lain merujuk ke sini, tidak menyebut angkanya sendiri.

```text
001_extensions            citext, pgcrypto
002_region                province, city
003_account               user_account, session
004_master_data           catalog_item
005_profile               business_profile, item_proposal
006_file_verification     uploaded_file, verification_request
007_listing               capacity_listing (+ horizon_until), listing_product,
                          listing_machine, item type triggers
008_period                availability_period
009_request               quota_request, request_candidate, self-request trigger, offer
010_work_order            work_order (+ readiness_week_start), work_order_status_history,
                          payment_record
011_allocation            capacity_allocation, reject-allocation-before-readiness trigger
012_reputation            review, dispute
013_notification          notification (+ transactional), notification_channel
014_rate_limit            rate_limit
015_verification_code     verification_code
```

`item_proposal` menunjuk `business_profile` lewat kunci asing, sehingga ia harus dibuat setelah tabel itu ada. Karena itu ia masuk `005_profile`, bukan `004_master_data` bersama `catalog_item`, meski keduanya sama-sama tergolong daftar baku secara domain. `catalog_item` sendiri tidak bergantung pada `business_profile`, jadi tetap di `004`.

`011_allocation` harus setelah `010_work_order` karena merujuk keduanya, dan triggernya membaca `work_order.readiness_week_start`. Migrasi dijalankan otomatis saat startup dengan `pg_try_advisory_lock`, agar dua kontainer yang sempat hidup bersamaan saat penerapan versi baru tidak menjalankannya serentak.
