# Data Model: Capacity Exchange, Devotion

**Feature**: `docs/001-capacity-exchange-marketplace/`
**Date**: 2026-08-21
**Last Revised**: 2026-08-22
**Input**: `spec.md` (91 FR, 16 entitas), `research.md` (R-03, R-04, R-05), `docs/memory/constitution.md` v2.1.0

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

**Larangan `DEFAULT now()` adalah perubahan dari versi sebelumnya**, dan alasannya bukan kerapian. Prinsip V mewajibkan seluruh hitungan bertenggat dapat diuji dengan waktu yang digantikan. `DEFAULT now()` melewati `Clock` sepenuhnya, sehingga pengujian yang menggeser waktu akan menghasilkan baris yang waktunya tidak konsisten dengan waktu uji, padahal FR-068, FR-037, serta FR-021 semuanya bergantung pada perbandingan waktu. Satu-satunya cara menutup celah itu adalah melarangnya di seluruh model, bukan sebagian.

Konsekuensi yang harus dijaga: setiap `INSERT` menyertakan kolom waktunya secara eksplisit. Bila sebuah kolom waktu ternyata lupa diisi, `NOT NULL` yang gagal akan menangkapnya saat pengujian, bukan menghasilkan waktu yang salah secara senyap.

Nilai turunan sengaja tidak dimaterialisasi. Kolom yang harus diperbarui setiap kali ulasan disembunyikan atau pesanan dibatalkan adalah sumber ketidaksesuaian yang paling sering muncul, dan pada 50 usaha demo maupun 200 usaha target SC-003 biaya menghitungnya saat dibaca tidak terasa.

---

## Peta Entitas ke Tabel

| # | Entitas Spec | Tabel | Catatan |
|---|--------------|-------|---------|
| 1 | Akun Pengguna | `akun_pengguna` | |
| 2 | Profil Usaha | `profil_usaha` | |
| 3 | Wilayah | `wilayah_provinsi`, `wilayah_kota` | Dua tabel, lihat alasan di bawah |
| 4 | Item Daftar Baku | `item_daftar_baku` | Satu tabel untuk produk dan mesin, dibedakan kolom `jenis` |
| 5 | Usulan Item | `usulan_item` | |
| 6 | Pengajuan Verifikasi Identitas | `pengajuan_verifikasi` | |
| 7 | Listing Kapasitas | `listing_kapasitas` | + `listing_produk`, `listing_mesin` |
| 8 | Periode Ketersediaan | `periode_ketersediaan` | Diperpanjang saat dibutuhkan (FR-088) |
| 9 | Alokasi Kapasitas | `alokasi_kapasitas` | |
| 10 | Request Kuota | `request_kuota` | + `request_kandidat` |
| 11 | Penawaran | `penawaran` | |
| 12 | Pesanan | `pesanan` | + `riwayat_status_pesanan` |
| 13 | Catatan Pembayaran | `catatan_pembayaran` | |
| 14 | Ulasan | `ulasan` | |
| 15 | Sengketa | `sengketa` | |
| 16 | Notifikasi | `notifikasi` | + `notifikasi_kanal` |

**Tabel penopang** yang tidak berdiri sebagai entitas domain:

| Tabel | Dituntut oleh | Alasan tidak jadi entitas |
|-------|---------------|---------------------------|
| `sesi` | FR-003 | Mekanisme autentikasi, bukan konsep bisnis |
| `berkas_unggahan` | FR-006, FR-009 | Metadata penyimpanan; entitas pemiliknya Pengajuan Verifikasi |
| `listing_produk`, `listing_mesin` | FR-012, FR-076 | Relasi banyak-ke-banyak; `listing_mesin` menyimpan jumlah mesin per jenis |
| `request_kandidat` | FR-030 | Satu status per kandidat pada satu request; tidak dapat jadi kolom |
| `riwayat_status_pesanan` | FR-039 | Waktu dan pelaku setiap perubahan status; tabel, bukan kolom |
| `notifikasi_kanal` | FR-085 | Jumlah percobaan dan status per kanal, terpisah per kanal |
| `batas_laju` | R-10 | Pembatasan laju yang harus bertahan setelah proses dijalankan ulang |

**Wilayah menjadi dua tabel, bukan satu tabel berhierarki.** Alasannya integritas: `profil_usaha` harus menunjuk kota/kabupaten, bukan provinsi. Dengan satu tabel berhierarki, tidak ada kunci asing yang dapat mencegah profil menunjuk provinsi. Karena tingkatnya tepat dua dan tetap (FR-062), memisahkannya tidak menimbulkan biaya.

---

## 1. Akun, Sesi, dan Profil

```sql
CREATE TABLE akun_pengguna (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  citext NOT NULL,
    nomor_hp               text   NOT NULL,
    kata_sandi_hash        text   NOT NULL,
    email_terverifikasi    boolean NOT NULL DEFAULT false,
    nomor_hp_terverifikasi boolean NOT NULL DEFAULT false,
    peran_subkontraktor    boolean NOT NULL DEFAULT false,
    peran_pemberi_order    boolean NOT NULL DEFAULT false,
    peran_admin            boolean NOT NULL DEFAULT false,
    notif_nontx_email      boolean NOT NULL DEFAULT true,
    notif_nontx_whatsapp   boolean NOT NULL DEFAULT true,
    dibuat_pada            timestamptz NOT NULL,
    diperbarui_pada        timestamptz NOT NULL,

    CONSTRAINT email_unik    UNIQUE (email),
    CONSTRAINT nomor_hp_unik UNIQUE (nomor_hp),
    CONSTRAINT nomor_hp_format CHECK (nomor_hp ~ '^62[0-9]{8,13}$'),
    CONSTRAINT punya_minimal_satu_peran CHECK (
        peran_subkontraktor OR peran_pemberi_order OR peran_admin
    ),
    CONSTRAINT admin_tidak_berperan_usaha CHECK (
        NOT peran_admin OR (NOT peran_subkontraktor AND NOT peran_pemberi_order)
    )
);
```

Peran sebagai tiga kolom boolean, bukan enum, karena FR-001 mengizinkan satu akun memegang dua peran usaha sekaligus.

`admin_tidak_berperan_usaha` memisahkan admin dari pengguna usaha: admin memutuskan verifikasi dan mediasi (FR-005), dan membiarkannya juga bertransaksi menciptakan konflik kepentingan. **Constraint ini tidak diminta FR mana pun**, ia keputusan model. Bila kasus admin yang juga punya konveksi perlu didukung, lepaskan dan naikkan ke spec.

Nomor HP dinormalkan ke `62…` tanpa tanda plus agar keunikannya bermakna: `08…` dan `+628…` yang sama tidak boleh jadi dua akun.

`notif_nontx_email` dan `notif_nontx_whatsapp` menyimpan preferensi kanal untuk notifikasi non-transaksional (FR-053, FR-091). Keduanya default `true`; mematikannya hanya memengaruhi notifikasi yang bergolongan non-transaksional pada tabel `notifikasi`. Notifikasi transaksional selalu terkirim ke kanal yang tersedia dan tidak membaca dua kolom ini. Preferensi disimpan di akun, bukan tabel terpisah, karena hanya dua tombol per akun dan tidak berkembang jadi pemetaan per kejadian.

```sql
CREATE TABLE sesi (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    akun_id          uuid NOT NULL REFERENCES akun_pengguna(id) ON DELETE CASCADE,
    token_hash       bytea NOT NULL,
    alamat_asal      inet,
    kedaluwarsa_pada timestamptz NOT NULL,
    dibuat_pada      timestamptz NOT NULL,
    diakses_pada     timestamptz NOT NULL,

    CONSTRAINT token_hash_unik UNIQUE (token_hash)
);

CREATE INDEX idx_sesi_akun ON sesi (akun_id);
CREATE INDEX idx_sesi_kedaluwarsa ON sesi (kedaluwarsa_pada);
```

Yang disimpan adalah hash token, bukan token mentah (R-10).

```sql
CREATE TABLE profil_usaha (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    akun_id         uuid NOT NULL REFERENCES akun_pengguna(id) ON DELETE RESTRICT,
    nama_usaha      text NOT NULL,
    kota_kode       text NOT NULL REFERENCES wilayah_kota(kode) ON DELETE RESTRICT,
    lintang         numeric(9,6),
    bujur           numeric(9,6),
    deskripsi       text,
    terverifikasi   boolean NOT NULL DEFAULT false,
    dibuat_pada     timestamptz NOT NULL,
    diperbarui_pada timestamptz NOT NULL,

    CONSTRAINT satu_profil_per_akun UNIQUE (akun_id),
    CONSTRAINT nama_usaha_tidak_kosong CHECK (length(trim(nama_usaha)) >= 3),
    CONSTRAINT koordinat_lengkap_atau_kosong CHECK (
        (lintang IS NULL AND bujur IS NULL) OR (lintang IS NOT NULL AND bujur IS NOT NULL)
    ),
    CONSTRAINT koordinat_dalam_indonesia CHECK (
        lintang IS NULL OR (lintang BETWEEN -11.5 AND 6.5 AND bujur BETWEEN 94.5 AND 141.5)
    )
);

CREATE INDEX idx_profil_kota ON profil_usaha (kota_kode);
CREATE INDEX idx_profil_nama ON profil_usaha (nama_usaha);
```

`terverifikasi` adalah cache keputusan admin terakhir, disimpan karena dibaca pada setiap hasil pencarian (FR-008, FR-027). Ia tidak pernah mempengaruhi ketayangan maupun urutan (FR-010, FR-024). Ini menyimpang dari kriteria penerimaan dokumen sumber yang menempatkan validasi manual admin sebelum listing aktif [1]; penyimpangannya tercatat di Assumptions spec.

Yang tidak dapat ditegakkan basis data: titik koordinat yang berada jauh dari kota yang dipilih. Itu pemeriksaan aplikasi dengan peringatan, bukan penolakan, karena batas kota tidak tersedia di data yang kita simpan.

---

## 2. Wilayah dan Daftar Baku

```sql
CREATE TABLE wilayah_provinsi (
    kode text PRIMARY KEY,
    nama text NOT NULL,
    CONSTRAINT provinsi_kode_format CHECK (kode ~ '^[0-9]{2}$')
);

CREATE TABLE wilayah_kota (
    kode          text PRIMARY KEY,
    provinsi_kode text NOT NULL REFERENCES wilayah_provinsi(kode) ON DELETE RESTRICT,
    nama          text NOT NULL,
    CONSTRAINT kota_kode_format CHECK (kode ~ '^[0-9]{4}$'),
    CONSTRAINT kota_milik_provinsinya CHECK (left(kode, 2) = provinsi_kode)
);

CREATE INDEX idx_kota_provinsi ON wilayah_kota (provinsi_kode);
```

`kota_milik_provinsinya` memanfaatkan sifat kode wilayah resmi: dua digit pertama kode kabupaten/kota adalah kode provinsinya. Ini menangkap kesalahan pemetaan saat seed, dan gagal keras di sana alih-alih senyap saat pencarian. Bila R-02 menemukan kode dari sumber tidak mengikuti pola itu, constraint ini yang pertama gagal.

Tingkat ketiga perluasan pencarian, seluruh Indonesia (FR-063), tidak memerlukan tabel: ia berarti tidak ada penyaringan wilayah.

```sql
CREATE TYPE jenis_item AS ENUM ('produk', 'mesin');

CREATE TABLE item_daftar_baku (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    jenis       jenis_item NOT NULL,
    nama        text NOT NULL,
    aktif       boolean NOT NULL DEFAULT true,
    urutan      integer NOT NULL DEFAULT 0,
    dibuat_pada timestamptz NOT NULL,

    CONSTRAINT nama_item_unik_per_jenis UNIQUE (jenis, nama)
);

CREATE INDEX idx_item_aktif ON item_daftar_baku (jenis, aktif) WHERE aktif;
```

Menonaktifkan item tidak menghapusnya (FR-060), sehingga listing yang sudah memakainya tetap utuh dan tetap dapat ditemukan.

```sql
CREATE TYPE status_usulan AS ENUM ('menunggu', 'disetujui', 'ditolak');

CREATE TABLE usulan_item (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profil_id       uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE CASCADE,
    jenis           jenis_item NOT NULL,
    nama_diusulkan  text NOT NULL,
    status          status_usulan NOT NULL DEFAULT 'menunggu',
    catatan_admin   text,
    diputuskan_oleh uuid REFERENCES akun_pengguna(id),
    diputuskan_pada timestamptz,
    item_id         uuid REFERENCES item_daftar_baku(id),
    dibuat_pada     timestamptz NOT NULL,

    CONSTRAINT keputusan_lengkap CHECK (
        (status = 'menunggu' AND diputuskan_pada IS NULL AND diputuskan_oleh IS NULL)
        OR (status <> 'menunggu' AND diputuskan_pada IS NOT NULL AND diputuskan_oleh IS NOT NULL)
    ),
    CONSTRAINT disetujui_menghasilkan_item CHECK (status <> 'disetujui' OR item_id IS NOT NULL)
);

CREATE INDEX idx_usulan_menunggu ON usulan_item (dibuat_pada) WHERE status = 'menunggu';
```

---

## 3. Berkas Unggahan dan Verifikasi Identitas

```sql
CREATE TYPE jenis_berkas AS ENUM ('dokumen_identitas', 'foto_lokasi');

CREATE TABLE berkas_unggahan (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pemilik_profil_id uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    jenis             jenis_berkas NOT NULL,
    nama_asli         text NOT NULL,
    tipe_mime         text NOT NULL,
    ukuran_byte       integer NOT NULL,
    path_penyimpanan  text NOT NULL,
    dibuat_pada       timestamptz NOT NULL,

    CONSTRAINT ukuran_maksimal CHECK (ukuran_byte > 0 AND ukuran_byte <= 5 * 1024 * 1024),
    CONSTRAINT tipe_diizinkan CHECK (tipe_mime IN ('image/jpeg', 'image/png', 'application/pdf')),
    CONSTRAINT path_unik UNIQUE (path_penyimpanan)
);

CREATE INDEX idx_berkas_pemilik ON berkas_unggahan (pemilik_profil_id);
```

`nama_asli` hanya metadata tampilan; `path_penyimpanan` memakai UUID yang dibuat sistem. Batas 5MB ditegakkan di basis data selain di aplikasi, agar tidak ada jalur tulis yang melewatinya.

Yang wajib di aplikasi: pemeriksaan tipe dari magic bytes (bukan dari header yang dikirim), pembuangan metadata lokasi gambar, dan batas total penyimpanan 500MB.

```sql
CREATE TYPE status_verifikasi AS ENUM ('menunggu', 'disetujui', 'ditolak');

CREATE TABLE pengajuan_verifikasi (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profil_id           uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE CASCADE,
    nomor_identitas     text NOT NULL,
    berkas_identitas_id uuid NOT NULL REFERENCES berkas_unggahan(id) ON DELETE RESTRICT,
    berkas_lokasi_id    uuid NOT NULL REFERENCES berkas_unggahan(id) ON DELETE RESTRICT,
    status              status_verifikasi NOT NULL DEFAULT 'menunggu',
    catatan_admin       text,
    diputuskan_oleh     uuid REFERENCES akun_pengguna(id),
    diputuskan_pada     timestamptz,
    alamat_asal_pengaju inet,
    dibuat_pada         timestamptz NOT NULL,

    CONSTRAINT keputusan_verifikasi_lengkap CHECK (
        (status = 'menunggu' AND diputuskan_pada IS NULL)
        OR (status <> 'menunggu' AND diputuskan_pada IS NOT NULL AND diputuskan_oleh IS NOT NULL)
    ),
    CONSTRAINT penolakan_beralasan CHECK (status <> 'ditolak' OR catatan_admin IS NOT NULL)
);

CREATE UNIQUE INDEX idx_satu_pengajuan_menunggu
    ON pengajuan_verifikasi (profil_id) WHERE status = 'menunggu';
CREATE INDEX idx_pengajuan_antrean
    ON pengajuan_verifikasi (dibuat_pada) WHERE status = 'menunggu';
```

Indeks unik parsial mengizinkan pengajuan ulang setelah penolakan (FR-011) sekaligus mencegah dua pengajuan menunggu sekaligus. Dokumen sumber menuntut unggah NIB/NIK dan foto lokasi usaha sebagai bagian verifikasi identitas [1]; keduanya diwakili dua kunci asing wajib di sini.

---

## 4. Listing Kapasitas

```sql
CREATE TABLE listing_kapasitas (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profil_id                uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE CASCADE,
    kapasitas_mingguan       integer NOT NULL,
    jeda_kesiapan_hari       integer NOT NULL,
    tayang                   boolean NOT NULL DEFAULT true,
    kalender_diperbarui_pada timestamptz NOT NULL,
    horizon_sampai           date NOT NULL,
    dibuat_pada              timestamptz NOT NULL,
    diperbarui_pada          timestamptz NOT NULL,

    CONSTRAINT satu_listing_per_profil UNIQUE (profil_id),
    CONSTRAINT kapasitas_positif CHECK (kapasitas_mingguan > 0),
    CONSTRAINT jeda_tidak_negatif CHECK (jeda_kesiapan_hari >= 0 AND jeda_kesiapan_hari <= 365),
    CONSTRAINT horizon_hari_senin CHECK (EXTRACT(ISODOW FROM horizon_sampai) = 1)
);

CREATE INDEX idx_listing_tayang ON listing_kapasitas (id) WHERE tayang;
CREATE INDEX idx_listing_kalender_basi ON listing_kapasitas (kalender_diperbarui_pada) WHERE tayang;
CREATE INDEX idx_listing_horizon ON listing_kapasitas (horizon_sampai) WHERE tayang;
```

**`horizon_sampai` adalah kolom baru untuk FR-088.** Ia menyimpan periode mingguan terjauh yang sudah pernah dibuat, sehingga pencarian dapat memeriksa satu kolom alih-alih menghitung `MAX(minggu_mulai)` dari `periode_ketersediaan` pada setiap permintaan. Ketika deadline yang diminta melampaui nilai ini, aplikasi membuat periode yang kurang lalu memperbarui kolomnya.

Satu angka `kapasitas_mingguan` untuk seluruh listing, tanpa kolom kapasitas per jenis produk (FR-076). Dokumen sumber memang meminta input kapasitas harian/mingguan dan jenis produk sebagai dua hal terpisah [1]; angka terpisah per produk ditolak karena mesin dan tenaga kerjanya berbagi.

`satu_listing_per_profil` adalah penyederhanaan model: spec tidak pernah menyebut satu usaha punya beberapa listing, dan seluruh Acceptance Scenario memakai bentuk tunggal. Melepasnya nanti tidak mengubah tabel lain, tetapi kueri pencarian dan alokasi perlu disesuaikan.

`kalender_diperbarui_pada` terpisah dari `diperbarui_pada` karena FR-021 mengukur kebaruan kalender: mengubah harga atau deskripsi tidak boleh menghapus penanda "Data Belum Diperbarui".

```sql
CREATE TABLE listing_produk (
    listing_id uuid NOT NULL REFERENCES listing_kapasitas(id) ON DELETE CASCADE,
    item_id    uuid NOT NULL REFERENCES item_daftar_baku(id) ON DELETE RESTRICT,
    PRIMARY KEY (listing_id, item_id)
);

CREATE TABLE listing_mesin (
    listing_id   uuid NOT NULL REFERENCES listing_kapasitas(id) ON DELETE CASCADE,
    item_id      uuid NOT NULL REFERENCES item_daftar_baku(id) ON DELETE RESTRICT,
    jumlah_mesin integer NOT NULL,
    PRIMARY KEY (listing_id, item_id),
    CONSTRAINT jumlah_mesin_positif CHECK (jumlah_mesin > 0)
);

CREATE INDEX idx_listing_produk_item ON listing_produk (item_id);
CREATE INDEX idx_listing_mesin_item  ON listing_mesin (item_id);
```

Indeks pada `item_id` adalah arah yang dipakai pencarian: dari jenis produk yang dicari menuju listing yang menyatakannya (FR-023 kriteria a dan b).

Yang tidak dapat ditegakkan kunci asing: bahwa `item_id` pada `listing_produk` berjenis `produk` dan bukan `mesin`. Ditegakkan trigger, karena `CHECK` tidak boleh merujuk tabel lain.

---

## 5. Periode Ketersediaan dan Alokasi Kapasitas

Inti FR-018 sampai FR-020, FR-077 sampai FR-079, FR-087, FR-088, dan FR-089, sekaligus tempat paling mungkin data rusak diam-diam.

```sql
CREATE TABLE periode_ketersediaan (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id         uuid NOT NULL REFERENCES listing_kapasitas(id) ON DELETE CASCADE,
    minggu_mulai       date NOT NULL,
    kapasitas_total    integer NOT NULL,
    kapasitas_terpakai integer NOT NULL DEFAULT 0,
    ditandai_penuh     boolean NOT NULL DEFAULT false,
    diperbarui_pada    timestamptz NOT NULL,

    CONSTRAINT satu_periode_per_minggu UNIQUE (listing_id, minggu_mulai),
    CONSTRAINT minggu_mulai_hari_senin CHECK (EXTRACT(ISODOW FROM minggu_mulai) = 1),
    CONSTRAINT kapasitas_total_tidak_negatif CHECK (kapasitas_total >= 0),
    CONSTRAINT kapasitas_terpakai_tidak_melebihi_total CHECK (
        kapasitas_terpakai >= 0 AND kapasitas_terpakai <= kapasitas_total
    )
);

CREATE INDEX idx_periode_listing_minggu ON periode_ketersediaan (listing_id, minggu_mulai);
CREATE INDEX idx_periode_tersedia ON periode_ketersediaan (listing_id, minggu_mulai)
    WHERE NOT ditandai_penuh AND kapasitas_terpakai < kapasitas_total;
```

Tiga constraint yang menanggung beban terbesar:

`minggu_mulai_hari_senin` menegakkan Prinsip V pada tingkat data. Tanpa ini, satu galat perhitungan batas minggu menghasilkan periode tumpang tindih, dan penjumlahan kapasitas jadi salah tanpa gejala.

`kapasitas_terpakai_tidak_melebihi_total` adalah penegakan FR-079 dan gerbang SC-018. Bila logika alokasi keliru, transaksi gagal keras alih-alih menghasilkan kapasitas minus. Spec sengaja tidak menyebut cara penegakannya. Itu keputusan model, dan alasannya ada di `research.md` R-04.

`satu_periode_per_minggu` membuat penguncian baris pada R-04 bermakna: satu minggu satu baris.

**Dua keadaan yang tidak dilarang basis data dan wajib ditangani aplikasi**, keduanya edge case yang sudah tercatat di spec:

FR-089: menurunkan `kapasitas_mingguan` listing memperbarui `kapasitas_total` seluruh periode mendatang **yang belum memiliki alokasi**. Periode yang sudah punya alokasi tidak diubah, karena menurunkannya di bawah `kapasitas_terpakai` akan ditolak constraint. Aplikasi harus menyaring periode berdasarkan ada tidaknya baris alokasi aktif, bukan mencoba memperbarui semuanya lalu menangkap galat.

Penandaan penuh atas periode yang sudah teralokasi harus ditolak aplikasi dengan pesan yang menyebut minggu mana beserta jumlah terpakainya, bukan meneruskan galat basis data mentah.

```sql
CREATE TABLE alokasi_kapasitas (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pesanan_id   uuid NOT NULL REFERENCES pesanan(id) ON DELETE RESTRICT,
    periode_id   uuid NOT NULL REFERENCES periode_ketersediaan(id) ON DELETE RESTRICT,
    jumlah       integer NOT NULL,
    dibuat_pada  timestamptz NOT NULL,
    dibalik_pada timestamptz,

    CONSTRAINT satu_alokasi_per_pesanan_periode UNIQUE (pesanan_id, periode_id),
    CONSTRAINT jumlah_alokasi_positif CHECK (jumlah > 0)
);

CREATE INDEX idx_alokasi_pesanan ON alokasi_kapasitas (pesanan_id);
CREATE INDEX idx_alokasi_periode ON alokasi_kapasitas (periode_id) WHERE dibalik_pada IS NULL;
```

Satu pesanan memiliki beberapa baris alokasi pada minggu berurutan mulai dari minggu kesiapan mulai (FR-077, FR-087). Pesanan 3.000 potong pada kapasitas 500 per minggu menghasilkan enam baris.

`dibalik_pada` menyimpan jejak pembatalan alih-alih menghapus baris, sehingga riwayat mediasi tetap dapat dibaca admin (FR-046).

**Trigger untuk FR-087.** Alokasi tidak boleh menyentuh periode sebelum minggu kesiapan mulai pesanan. Perbandingannya melintasi tiga tabel, jadi `CHECK` tidak dapat dipakai:

```sql
CREATE FUNCTION cegah_alokasi_sebelum_kesiapan() RETURNS trigger AS $$
DECLARE
    v_minggu_periode  date;
    v_minggu_kesiapan date;
BEGIN
    SELECT p.minggu_mulai INTO v_minggu_periode
      FROM periode_ketersediaan p WHERE p.id = NEW.periode_id;

    SELECT o.minggu_kesiapan_mulai INTO v_minggu_kesiapan
      FROM pesanan o WHERE o.id = NEW.pesanan_id;

    IF v_minggu_periode < v_minggu_kesiapan THEN
        RAISE EXCEPTION
            'FR-087: alokasi pada minggu % mendahului minggu kesiapan mulai %',
            v_minggu_periode, v_minggu_kesiapan;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_cegah_alokasi_sebelum_kesiapan
    BEFORE INSERT OR UPDATE ON alokasi_kapasitas
    FOR EACH ROW EXECUTE FUNCTION cegah_alokasi_sebelum_kesiapan();
```

Aturan ini mudah dilanggar tanpa disadari: alokasi yang naif akan mulai dari minggu berjalan, dan itu berarti menjadwalkan pekerjaan pada minggu yang menurut pernyataan subkontraktor sendiri belum dapat dipakai. Bug seperti itu tidak akan terlihat pada pengujian manual karena angka totalnya tetap benar.

---

## 6. Request Kuota dan Penawaran

```sql
CREATE TABLE request_kuota (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pemberi_order_id   uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE CASCADE,
    item_produk_id     uuid NOT NULL REFERENCES item_daftar_baku(id) ON DELETE RESTRICT,
    jumlah             integer NOT NULL,
    bahan              text NOT NULL,
    deadline           date NOT NULL,
    catatan            text,
    batas_balasan_pada timestamptz NOT NULL,
    dibuat_pada        timestamptz NOT NULL,

    CONSTRAINT jumlah_request_positif CHECK (jumlah > 0),
    CONSTRAINT batas_balasan_setelah_dibuat CHECK (batas_balasan_pada > dibuat_pada)
);

CREATE INDEX idx_request_pemberi ON request_kuota (pemberi_order_id, dibuat_pada DESC);
CREATE INDEX idx_request_batas ON request_kuota (batas_balasan_pada);
```

**Perbaikan C-3.** Versi sebelumnya memasang `CHECK (batas_balasan_pada = dibuat_pada + interval '72 hours')` bersama `dibuat_pada DEFAULT now()`. Kombinasi itu **selalu gagal**: aplikasi menghitung `batas_balasan_pada` dari `Clock.Now()` sementara basis data mengisi `dibuat_pada` dari `now()`, dan keduanya berbeda mikrodetik. `DEFAULT now()` juga melewati `Clock`, sehingga pengujian kedaluwarsa dengan waktu digeser tidak dapat menghasilkan baris yang konsisten.

Sekarang aplikasi mengirim kedua nilai dari `Clock`, dan constraint hanya menjaga urutannya. Angka 72 jam sendiri ditegakkan aplikasi dan diuji, bukan oleh basis data, karena basis data tidak boleh punya sumber waktu sendiri.

```sql
CREATE TYPE status_kandidat AS ENUM (
    'menunggu_balasan', 'ditawar', 'ditolak', 'kedaluwarsa', 'tidak_dilanjutkan', 'disepakati'
);

CREATE TABLE request_kandidat (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id       uuid NOT NULL REFERENCES request_kuota(id) ON DELETE CASCADE,
    listing_id       uuid NOT NULL REFERENCES listing_kapasitas(id) ON DELETE RESTRICT,
    subkontraktor_id uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    status           status_kandidat NOT NULL DEFAULT 'menunggu_balasan',
    alasan_penolakan text,
    diperbarui_pada  timestamptz NOT NULL,

    CONSTRAINT satu_kandidat_per_request UNIQUE (request_id, listing_id)
);

CREATE INDEX idx_kandidat_subkon ON request_kandidat (subkontraktor_id, status);
CREATE INDEX idx_kandidat_request ON request_kandidat (request_id);
CREATE UNIQUE INDEX idx_satu_kesepakatan_per_request
    ON request_kandidat (request_id) WHERE status = 'disepakati';
```

`idx_satu_kesepakatan_per_request` menegakkan FR-034: menerima satu penawaran menutup kandidat lain, sehingga tidak mungkin ada dua kesepakatan dari satu request.

FR-081 dan FR-083, larangan request ke listing sendiri, tidak dapat dinyatakan `CHECK` karena membandingkan `subkontraktor_id` di tabel ini dengan `pemberi_order_id` di tabel lain:

```sql
CREATE FUNCTION cegah_request_ke_diri_sendiri() RETURNS trigger AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM request_kuota r
        WHERE r.id = NEW.request_id AND r.pemberi_order_id = NEW.subkontraktor_id
    ) THEN
        RAISE EXCEPTION 'FR-083: request kuota tidak boleh dikirim ke listing milik sendiri';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_cegah_request_diri_sendiri
    BEFORE INSERT OR UPDATE ON request_kandidat
    FOR EACH ROW EXECUTE FUNCTION cegah_request_ke_diri_sendiri();
```

Aplikasi menolaknya lebih awal dengan pesan yang dapat dibaca pengguna; trigger adalah jaring pengaman untuk jalur yang dikirim tanpa melalui hasil pencarian, yang disebut eksplisit di FR-083.

```sql
CREATE TYPE pengaju_penawaran AS ENUM ('subkontraktor', 'pemberi_order');

CREATE TABLE penawaran (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kandidat_id        uuid NOT NULL REFERENCES request_kandidat(id) ON DELETE CASCADE,
    urutan             integer NOT NULL,
    diajukan_oleh      pengaju_penawaran NOT NULL,
    harga_total        bigint NOT NULL,
    jeda_kesiapan_hari integer NOT NULL,
    catatan            text,
    dibuat_pada        timestamptz NOT NULL,

    CONSTRAINT urutan_unik_per_kandidat UNIQUE (kandidat_id, urutan),
    CONSTRAINT harga_positif CHECK (harga_total > 0),
    CONSTRAINT jeda_penawaran_wajar CHECK (jeda_kesiapan_hari >= 0 AND jeda_kesiapan_hari <= 365)
);

CREATE INDEX idx_penawaran_kandidat ON penawaran (kandidat_id, urutan);
```

Setiap counter-offer adalah baris baru dengan `urutan` bertambah, sehingga seluruh riwayat negosiasi tersimpan (FR-033). Dokumen sumber menempatkan negosiasi harga sebagai kirim estimasi, lalu terima, tolak, atau ajukan counter-offer [1], dan rangkaian itulah yang direkam kolom `urutan` dan `diajukan_oleh`.

`harga_total` bertipe `bigint` dalam rupiah bulat.

---

## 7. Pesanan, Riwayat Status, dan Pembayaran

```sql
CREATE TYPE status_pesanan AS ENUM (
    'diterima', 'produksi', 'selesai', 'dikirim', 'dikonfirmasi', 'dibatalkan', 'dalam_mediasi'
);

CREATE TABLE pesanan (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kandidat_id           uuid NOT NULL REFERENCES request_kandidat(id) ON DELETE RESTRICT,
    penawaran_id          uuid NOT NULL REFERENCES penawaran(id) ON DELETE RESTRICT,
    pemberi_order_id      uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    subkontraktor_id      uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    jumlah                integer NOT NULL,
    harga_total           bigint NOT NULL,
    deadline              date NOT NULL,
    minggu_kesiapan_mulai date NOT NULL,
    status                status_pesanan NOT NULL DEFAULT 'diterima',
    dikirim_pada          timestamptz,
    dikonfirmasi_pada     timestamptz,
    dikonfirmasi_otomatis boolean NOT NULL DEFAULT false,
    dibatalkan_oleh_id    uuid REFERENCES profil_usaha(id),
    alasan_pembatalan     text,
    dibatalkan_pada       timestamptz,
    dibuat_pada           timestamptz NOT NULL,

    CONSTRAINT satu_pesanan_per_kandidat UNIQUE (kandidat_id),
    CONSTRAINT dua_pihak_berbeda CHECK (pemberi_order_id <> subkontraktor_id),
    CONSTRAINT jumlah_pesanan_positif CHECK (jumlah > 0),
    CONSTRAINT harga_pesanan_positif CHECK (harga_total > 0),
    CONSTRAINT kesiapan_hari_senin CHECK (EXTRACT(ISODOW FROM minggu_kesiapan_mulai) = 1),
    CONSTRAINT kesiapan_tidak_melewati_deadline CHECK (minggu_kesiapan_mulai <= deadline),
    CONSTRAINT pembatalan_lengkap CHECK (
        (status <> 'dibatalkan')
        OR (dibatalkan_oleh_id IS NOT NULL AND alasan_pembatalan IS NOT NULL
            AND dibatalkan_pada IS NOT NULL)
    ),
    CONSTRAINT dikirim_sebelum_dikonfirmasi CHECK (
        dikonfirmasi_pada IS NULL OR dikirim_pada IS NULL OR dikonfirmasi_pada >= dikirim_pada
    ),
    CONSTRAINT konfirmasi_otomatis_perlu_konfirmasi CHECK (
        NOT dikonfirmasi_otomatis OR dikonfirmasi_pada IS NOT NULL
    )
);

CREATE INDEX idx_pesanan_pemberi ON pesanan (pemberi_order_id, status);
CREATE INDEX idx_pesanan_subkon ON pesanan (subkontraktor_id, status);
CREATE INDEX idx_pesanan_deadline_aktif ON pesanan (deadline)
    WHERE status IN ('diterima', 'produksi', 'selesai', 'dikirim');
CREATE INDEX idx_pesanan_tenggat_otomatis ON pesanan (dikirim_pada) WHERE status = 'dikirim';
```

**`minggu_kesiapan_mulai` adalah kolom baru untuk FR-087.** Ia dihitung sekali saat kesepakatan terbentuk, dari tanggal kesepakatan ditambah `jeda_kesiapan_hari` listing, lalu dibulatkan ke Senin minggu yang memuatnya. Disimpan alih-alih dihitung ulang karena `jeda_kesiapan_hari` pada listing dapat berubah kemudian, sementara alokasi pesanan yang sudah terbentuk tidak boleh bergeser. Ini juga yang menutup salah satu edge case spec: subkontraktor mengubah jeda kesiapan setelah punya alokasi berjalan.

`kesiapan_tidak_melewati_deadline` menegakkan FR-090 pada tingkat data: pesanan yang produksinya baru dapat dimulai setelah deadline tidak dapat terbentuk.

`dibatalkan_oleh_id` adalah dasar FR-072: pembatalan masuk pembagi tingkat penyelesaian hanya bagi pihak yang membatalkan. Tanpa kolom ini, rumus itu tidak dapat dihitung.

`dua_pihak_berbeda` adalah lapisan kedua atas larangan request ke diri sendiri: bahkan bila trigger dilewati, pesanan berdua pihak sama tidak dapat terbentuk.

Dua indeks parsial terakhir adalah jalur penjadwal R-07: `idx_pesanan_tenggat_otomatis` untuk FR-068 dan FR-069, `idx_pesanan_deadline_aktif` untuk FR-045.

**Transisi status yang sah** (FR-044). Semua transisi lain ditolak beserta penjelasan urutan yang diizinkan:

```text
diterima ──▶ produksi ──▶ selesai ──▶ dikirim ──▶ dikonfirmasi
    │            │            │           │
    │            └────────────┴───────────┴──▶ dalam_mediasi ──▶ dibatalkan
    │                                                        └──▶ dikonfirmasi
    └──▶ dibatalkan            (pembatalan sendiri, FR-065: hanya dari 'diterima')
```

| Transisi | Pelaku | Aturan |
|----------|--------|--------|
| `diterima → produksi` | Subkontraktor | Menutup jalur pembatalan sendiri (FR-066) |
| `produksi → selesai → dikirim` | Subkontraktor | Berurutan, tidak boleh melompat |
| `dikirim → dikonfirmasi` | Pemberi order, atau sistem setelah 7 hari | FR-068; `dikonfirmasi_otomatis` menandai yang mana |
| `diterima → dibatalkan` | Kedua pihak | Wajib beralasan; seluruh alokasi dibalik (FR-020, FR-065) |
| `* → dalam_mediasi` | Kedua pihak melapor | Menghentikan hitungan konfirmasi otomatis (FR-070) |
| `dalam_mediasi → dibatalkan` | Admin | Admin menentukan pengembalian alokasi dan pihak penanggung (FR-067) |

```sql
CREATE TABLE riwayat_status_pesanan (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pesanan_id  uuid NOT NULL REFERENCES pesanan(id) ON DELETE CASCADE,
    status_lama status_pesanan,
    status_baru status_pesanan NOT NULL,
    diubah_oleh uuid REFERENCES akun_pengguna(id),
    oleh_sistem boolean NOT NULL DEFAULT false,
    catatan     text,
    dibuat_pada timestamptz NOT NULL,

    CONSTRAINT pelaku_jelas CHECK (oleh_sistem OR diubah_oleh IS NOT NULL)
);

CREATE INDEX idx_riwayat_pesanan ON riwayat_status_pesanan (pesanan_id, dibuat_pada);
```

`pelaku_jelas` memastikan FR-039 terpenuhi: setiap perubahan punya pelaku, dan perubahan oleh penjadwal ditandai `oleh_sistem` alih-alih dibiarkan tanpa identitas.

```sql
CREATE TYPE arah_pembayaran AS ENUM ('terkirim', 'diterima');

CREATE TABLE catatan_pembayaran (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pesanan_id  uuid NOT NULL REFERENCES pesanan(id) ON DELETE CASCADE,
    profil_id   uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    arah        arah_pembayaran NOT NULL,
    tanggal     date NOT NULL,
    catatan     text,
    dibuat_pada timestamptz NOT NULL,

    CONSTRAINT satu_pernyataan_per_pihak_per_arah UNIQUE (pesanan_id, profil_id, arah)
);

CREATE INDEX idx_pembayaran_pesanan ON catatan_pembayaran (pesanan_id);
```

Tidak ada kolom jumlah uang, dan itu disengaja. Platform tidak memproses dana (FR-040); yang dicatat hanya pernyataan bahwa pembayaran terjadi. FR-043, perbedaan pernyataan antar pihak, dihitung dari ada tidaknya pasangan baris, bukan dari perbandingan angka.

Dokumen sumber menempatkan escrow sebagai penahan dana yang dirilis saat pesanan dikonfirmasi selesai [1]. Versi ini menggantinya dengan pencatatan pernyataan, dan konsekuensinya tercatat di Assumptions spec.

---

## 8. Reputasi dan Sengketa

```sql
CREATE TABLE ulasan (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pesanan_id         uuid NOT NULL REFERENCES pesanan(id) ON DELETE RESTRICT,
    penilai_id         uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    dinilai_id         uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    rating             smallint NOT NULL,
    teks               text,
    disembunyikan      boolean NOT NULL DEFAULT false,
    disembunyikan_oleh uuid REFERENCES akun_pengguna(id),
    disembunyikan_pada timestamptz,
    alasan_penyembunyian text,
    dibuat_pada        timestamptz NOT NULL,

    CONSTRAINT satu_ulasan_per_pesanan_per_penilai UNIQUE (pesanan_id, penilai_id),
    CONSTRAINT rating_satu_sampai_lima CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT tidak_menilai_diri_sendiri CHECK (penilai_id <> dinilai_id),
    CONSTRAINT penyembunyian_lengkap CHECK (
        NOT disembunyikan
        OR (disembunyikan_oleh IS NOT NULL AND disembunyikan_pada IS NOT NULL
            AND alasan_penyembunyian IS NOT NULL)
    )
);

CREATE INDEX idx_ulasan_dinilai ON ulasan (dinilai_id) WHERE NOT disembunyikan;
CREATE INDEX idx_ulasan_pesanan ON ulasan (pesanan_id);
```

Indeks parsial hanya memuat ulasan yang tampil, karena rata-rata rating harus mengecualikan yang disembunyikan (FR-050). Moderasi ulasan oleh admin adalah sub-fitur yang diminta dokumen sumber [1], dan `alasan_penyembunyian` membuat tindakan itu tercatat, bukan hanya berlaku.

Yang tidak dapat ditegakkan basis data: bahwa pesanan sudah berstatus `dikonfirmasi` saat ulasan dibuat (FR-047). Ditegakkan aplikasi.

**Nilai turunan, dihitung saat dibaca:**

```sql
-- Rating rata-rata dan jumlah pekerjaan selesai (FR-048)
SELECT
    round(avg(u.rating)::numeric, 2)                             AS rating_rata_rata,
    count(*)                                                     AS jumlah_ulasan,
    (SELECT count(*) FROM pesanan p
      WHERE p.subkontraktor_id = $1 AND p.status = 'dikonfirmasi') AS pekerjaan_selesai
FROM ulasan u
WHERE u.dinilai_id = $1 AND NOT u.disembunyikan;

-- Tingkat penyelesaian (FR-071, FR-072, FR-073)
WITH terlibat AS (
    SELECT status, dibatalkan_oleh_id
    FROM pesanan
    WHERE pemberi_order_id = $1 OR subkontraktor_id = $1
)
SELECT
    count(*) FILTER (WHERE status = 'dikonfirmasi') AS selesai,
    count(*) FILTER (
        WHERE status <> 'dibatalkan' OR dibatalkan_oleh_id = $1
    ) AS pembagi
FROM terlibat;
```

Pembagi mengecualikan pesanan yang dibatalkan pihak lain, sesuai FR-072. Ambang 3 pesanan (FR-073) diterapkan di lapisan penyajian: bila `pembagi < 3`, yang dikirim adalah keterangan bahwa data belum cukup, bukan angka. Dokumen sumber menyebut statistik tingkat penyelesaian sebagai sub-fitur profil reputasi [1] tanpa memberi rumus; rumus di atas adalah keputusan spec kita.

```sql
CREATE TYPE status_sengketa AS ENUM ('dilaporkan', 'dalam_mediasi', 'selesai');

CREATE TABLE sengketa (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    pesanan_id           uuid NOT NULL REFERENCES pesanan(id) ON DELETE RESTRICT,
    pelapor_id           uuid NOT NULL REFERENCES profil_usaha(id) ON DELETE RESTRICT,
    isi_laporan          text NOT NULL,
    status               status_sengketa NOT NULL DEFAULT 'dilaporkan',
    catatan_admin        text,
    alokasi_dikembalikan boolean,
    penanggung_id        uuid REFERENCES profil_usaha(id),
    ditangani_oleh       uuid REFERENCES akun_pengguna(id),
    diselesaikan_pada    timestamptz,
    dibuat_pada          timestamptz NOT NULL,

    CONSTRAINT penyelesaian_lengkap CHECK (
        status <> 'selesai'
        OR (ditangani_oleh IS NOT NULL AND diselesaikan_pada IS NOT NULL
            AND alokasi_dikembalikan IS NOT NULL AND catatan_admin IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_satu_sengketa_terbuka
    ON sengketa (pesanan_id) WHERE status <> 'selesai';
CREATE INDEX idx_sengketa_antrean ON sengketa (dibuat_pada) WHERE status <> 'selesai';
```

`penyelesaian_lengkap` menegakkan FR-067: admin tidak dapat menutup mediasi tanpa memutuskan secara eksplisit apakah alokasi dikembalikan dan siapa yang menanggung. Keduanya `NULL` selama sengketa belum selesai, bukan diberi nilai bawaan yang menyesatkan.

`idx_satu_sengketa_terbuka` menutup edge case pelaporan berulang untuk menghentikan konfirmasi otomatis berkali-kali. Penanganan sengketa lewat mediasi admin memang jalur yang dipilih dokumen sumber untuk fase awal, karena penanganan legal formal menuntut tim hukum dan asuransi [1].

---

## 9. Notifikasi dan Pembatasan Laju

```sql
CREATE TYPE jenis_kejadian AS ENUM (
    'request_diterima', 'penawaran_masuk', 'counter_offer', 'kesepakatan_terbentuk',
    'status_pesanan_berubah', 'catatan_pembayaran', 'deadline_mendekat', 'deadline_terlampaui',
    'keputusan_verifikasi', 'permintaan_rating', 'pesanan_dibatalkan',
    'tenggat_konfirmasi_mendekat', 'pesanan_tertutup_otomatis', 'keputusan_usulan_item',
    'kalender_basi'
);

CREATE TABLE notifikasi (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    akun_id       uuid NOT NULL REFERENCES akun_pengguna(id) ON DELETE CASCADE,
    kejadian      jenis_kejadian NOT NULL,
    transaksional boolean NOT NULL,
    judul         text NOT NULL,
    isi           text NOT NULL,
    tautan        text,
    dibaca_pada   timestamptz,
    dibuat_pada   timestamptz NOT NULL
);

CREATE INDEX idx_notifikasi_akun ON notifikasi (akun_id, dibuat_pada DESC);
CREATE INDEX idx_notifikasi_belum_dibaca ON notifikasi (akun_id) WHERE dibaca_pada IS NULL;
```

**`transaksional` adalah kolom baru untuk FR-091.** Penggolongannya ditetapkan spec dan disimpan per baris agar pengirim dapat menghormati preferensi kanal tanpa memelihara pemetaan terpisah di kode:

| Kejadian | Golongan |
|----------|----------|
| `request_diterima`, `penawaran_masuk`, `counter_offer`, `kesepakatan_terbentuk` | Transaksional |
| `status_pesanan_berubah`, `pesanan_dibatalkan`, `catatan_pembayaran` | Transaksional |
| `deadline_terlampaui`, `tenggat_konfirmasi_mendekat`, `pesanan_tertutup_otomatis` | Transaksional |
| `keputusan_verifikasi`, `keputusan_usulan_item` | Transaksional |
| `kalender_basi`, `deadline_mendekat`, `permintaan_rating` | Non-transaksional |

Hanya yang non-transaksional dapat dimatikan pengguna (FR-053). Perhatikan bahwa `deadline_mendekat` non-transaksional sementara `tenggat_konfirmasi_mendekat` transaksional, karena yang kedua berujung pada penutupan pesanan otomatis, sehingga tidak boleh dapat dimatikan.

```sql
CREATE TYPE kanal_notifikasi AS ENUM ('email', 'whatsapp');
CREATE TYPE status_kirim AS ENUM ('menunggu', 'terkirim', 'gagal_permanen');

CREATE TABLE notifikasi_kanal (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notifikasi_id  uuid NOT NULL REFERENCES notifikasi(id) ON DELETE CASCADE,
    kanal          kanal_notifikasi NOT NULL,
    status         status_kirim NOT NULL DEFAULT 'menunggu',
    percobaan      smallint NOT NULL DEFAULT 0,
    galat_terakhir text,
    dicoba_pada    timestamptz,
    terkirim_pada  timestamptz,

    CONSTRAINT satu_kanal_per_notifikasi UNIQUE (notifikasi_id, kanal),
    CONSTRAINT percobaan_maksimal_tiga CHECK (percobaan >= 0 AND percobaan <= 3),
    CONSTRAINT gagal_setelah_tiga_percobaan CHECK (
        status <> 'gagal_permanen' OR percobaan = 3
    )
);

CREATE INDEX idx_kanal_antrean ON notifikasi_kanal (dicoba_pada NULLS FIRST)
    WHERE status = 'menunggu';
```

`percobaan_maksimal_tiga` dan `gagal_setelah_tiga_percobaan` menegakkan FR-085.

Baris `notifikasi` ditulis di dalam transaksi kejadiannya, sedangkan `notifikasi_kanal` diproses goroutine pengirim setelah transaksi berhasil. Ini memenuhi FR-054 dan FR-086 sekaligus: notifikasi di dalam platform tetap ada meskipun email dan WhatsApp gagal seluruhnya, dan kegagalan kirim tidak pernah menggagalkan pesanan. Dokumen sumber meminta ketiga kanal (email, WhatsApp, dan in-app [1]), dan pemisahan tabel inilah yang membuat kanal ketiga tidak bergantung pada dua yang pertama.

```sql
CREATE TYPE sasaran_batas AS ENUM ('login_akun', 'otp_nomor', 'otp_alamat', 'request_kuota');

CREATE TABLE batas_laju (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sasaran       sasaran_batas NOT NULL,
    kunci         text NOT NULL,
    jendela_mulai timestamptz NOT NULL,
    hitungan      integer NOT NULL DEFAULT 1,

    CONSTRAINT satu_baris_per_kunci_jendela UNIQUE (sasaran, kunci, jendela_mulai),
    CONSTRAINT hitungan_positif CHECK (hitungan > 0)
);

CREATE INDEX idx_batas_pembersihan ON batas_laju (jendela_mulai);
```

Tabel, bukan penyimpanan dalam memori (R-10), agar batas tetap berlaku setelah proses dijalankan ulang.

---

## 10. Kueri Pencarian dan Rentang Kapasitas

Bentuk kueri FR-023 sampai FR-025, FR-080, FR-087, dan FR-088. Ini acuan bagi `contracts/` dan T035.

**Rentang kapasitas** setiap kandidat berbeda karena bergantung pada `jeda_kesiapan_hari` masing-masing:

```text
minggu_kesiapan(kandidat) = senin_dari(tanggal_pencarian + jeda_kesiapan_hari)
minggu_deadline           = senin_dari(deadline_diminta)
rentang kapasitas         = [minggu_kesiapan(kandidat) .. minggu_deadline]
```

Kandidat yang `minggu_kesiapan`-nya melampaui `minggu_deadline` memiliki rentang kosong, sehingga kapasitas tersisanya nol dan kriteria (d) tidak terpenuhi.

```sql
WITH param AS (
    SELECT
        $1::date  AS tanggal_pencarian,
        $2::date  AS minggu_deadline,     -- sudah dibulatkan ke Senin oleh aplikasi
        $3::int   AS jumlah,
        $4::uuid  AS item_produk,         -- boleh NULL (FR-023: kriteria tak dievaluasi)
        $5::uuid  AS item_mesin,          -- boleh NULL
        $6::int   AS jeda_maksimal,       -- boleh NULL
        $7::uuid  AS profil_pencari,
        $8::text  AS kota_kode,           -- NULL bila tingkat provinsi atau nasional
        $9::text  AS provinsi_kode        -- NULL bila tingkat kota atau nasional
),
kandidat_dasar AS (
    SELECT
        l.id AS listing_id,
        l.profil_id,
        pr.nama_usaha,
        l.kapasitas_mingguan,
        l.jeda_kesiapan_hari,
        l.horizon_sampai,
        -- FR-087: minggu paling awal yang boleh dihitung
        date_trunc('week', p.tanggal_pencarian + (l.jeda_kesiapan_hari || ' days')::interval)::date
            AS minggu_kesiapan
    FROM listing_kapasitas l
    JOIN profil_usaha pr ON pr.id = l.profil_id
    CROSS JOIN param p
    WHERE l.tayang
      AND l.profil_id <> p.profil_pencari                        -- FR-081
      AND (p.kota_kode  IS NULL OR pr.kota_kode = p.kota_kode)
      AND (p.provinsi_kode IS NULL OR pr.kota_kode IN (
              SELECT kode FROM wilayah_kota WHERE provinsi_kode = p.provinsi_kode))
),
kapasitas AS (
    SELECT
        k.listing_id,
        -- Periode yang sudah ada di dalam rentang kapasitas
        coalesce(sum(pk.kapasitas_total - pk.kapasitas_terpakai), 0)
            AS tersisa_tercatat,
        -- FR-088: minggu di dalam rentang yang belum pernah dibuat
        greatest(0, (
            (p.minggu_deadline - greatest(k.minggu_kesiapan, k.horizon_sampai + 7)) / 7 + 1
        )) * k.kapasitas_mingguan AS tersisa_belum_dibuat
    FROM kandidat_dasar k
    CROSS JOIN param p
    LEFT JOIN periode_ketersediaan pk
           ON pk.listing_id = k.listing_id
          AND NOT pk.ditandai_penuh
          AND pk.minggu_mulai BETWEEN k.minggu_kesiapan AND p.minggu_deadline
    GROUP BY k.listing_id, k.minggu_kesiapan, k.horizon_sampai,
             k.kapasitas_mingguan, p.minggu_deadline
),
dinilai AS (
    SELECT
        k.*,
        (c.tersisa_tercatat + c.tersisa_belum_dibuat) AS kapasitas_tersisa,
        -- FR-023: kriteria yang filternya tidak diisi dihitung terpenuhi
        (p.item_produk IS NULL OR EXISTS (
            SELECT 1 FROM listing_produk lp
             WHERE lp.listing_id = k.listing_id AND lp.item_id = p.item_produk))::int
            AS produk_cocok,
        (p.item_mesin IS NULL OR EXISTS (
            SELECT 1 FROM listing_mesin lm
             WHERE lm.listing_id = k.listing_id AND lm.item_id = p.item_mesin))::int
            AS mesin_cocok,
        (p.jeda_maksimal IS NULL OR k.jeda_kesiapan_hari <= p.jeda_maksimal)::int
            AS jeda_cocok,
        ((c.tersisa_tercatat + c.tersisa_belum_dibuat) >= p.jumlah)::int
            AS kapasitas_cukup
    FROM kandidat_dasar k
    JOIN kapasitas c ON c.listing_id = k.listing_id
    CROSS JOIN param p
)
SELECT *,
       (produk_cocok + mesin_cocok + jeda_cocok + kapasitas_cukup) AS skor
FROM dinilai
WHERE (skor, kapasitas_tersisa, -jeda_kesiapan_hari, nama_usaha, listing_id)
      < ($10, $11, -$12, $13, $14)                                -- keyset, R-05
ORDER BY skor DESC, kapasitas_tersisa DESC, jeda_kesiapan_hari ASC,
         nama_usaha ASC, listing_id ASC
LIMIT $15;
```

Empat hal yang perlu diperhatikan tentang kueri ini.

**Perhitungan minggu memakai `date_trunc('week', …)`** yang di PostgreSQL memulai minggu pada Senin, sama dengan Prinsip V. Aplikasi tetap yang membulatkan `deadline` ke Senin sebelum mengirimnya, agar satu-satunya sumber kebenaran pembulatan ada di kode Go yang dapat diuji.

**`tersisa_belum_dibuat` adalah perkiraan optimis** atas periode yang belum ada: setiap minggu di luar `horizon_sampai` dihitung berkapasitas penuh, karena periode baru memang dibuat dengan `kapasitas_mingguan` sebagai kapasitas total (FR-088). Ini yang mencegah kandidat dinilai tidak memenuhi kriteria hanya karena periodenya belum pernah dibuat.

**Periode benar-benar dibuat sebagai efek samping pencarian**, bukan di dalam kueri ini. Aplikasi memeriksa `horizon_sampai < minggu_deadline` pada kandidat yang lolos, lalu membuat periode yang kurang di dalam transaksi tersendiri sebelum mengembalikan hasil. Menempatkannya di luar kueri pencarian membuat pencarian tetap operasi baca dan tidak memicu penulisan pada setiap permintaan.

**Kriteria yang filternya `NULL` dihitung terpenuhi** (FR-023, keputusan C-4). Skor tetap 0–4 tanpa normalisasi, sehingga kriteria yang tidak dievaluasi menaikkan skor semua kandidat secara seragam dan tidak membedakan siapa pun. Respons menyertakan nilai per kriteria agar klien dapat menyebutkan mana yang tidak dievaluasi (FR-026).

**Batas yang perlu diketahui**: `skor` dan `kapasitas_tersisa` adalah hasil perhitungan sehingga tidak dapat diindeks. Indeks yang ada mempercepat penyaringan dan penggabungan (`idx_listing_tayang`, `idx_profil_kota`, `idx_periode_tersedia`, `idx_listing_produk_item`), tetapi pengurutan dan agregasi dikerjakan saat kueri berjalan. Pada 50 usaha demo maupun 200 usaha SC-003 ini tidak masalah. Bila SC-010 terancam pada data lebih besar, langkah berikutnya adalah tabel ringkasan kapasitas per listing per minggu yang diperbarui saat alokasi berubah, bukan menambah indeks, karena tidak ada indeks yang menolong kolom hasil hitungan. Ini kandidat optimasi, bukan pekerjaan sekarang, sesuai Standar Performa konstitusi.

---

## 11. Ringkasan Penegakan Aturan

Yang hanya bergantung pada aplikasi adalah kandidat utama pengujian otomatis yang diwajibkan konstitusi.

| Aturan | Basis Data | Aplikasi |
|--------|-----------|----------|
| Kapasitas terpakai ≤ total (FR-079, SC-018) | `CHECK` | Penguncian baris terurut (R-04) |
| Alokasi tidak mendahului minggu kesiapan (FR-087, SC-020) | Trigger | Perhitungan minggu kesiapan dari `Clock` |
| Kesiapan tidak melewati deadline (FR-090) | `CHECK` | Penolakan penawaran beserta penjelasan |
| Horizon diperpanjang sampai deadline (FR-088, SC-021) | Tidak ada | Pembuatan periode sebagai efek samping pencarian |
| Propagasi perubahan kapasitas (FR-089) | Tidak ada | Menyaring periode tanpa alokasi aktif |
| Satu kesepakatan per request (FR-034) | Indeks unik parsial | Penutupan kandidat lain |
| Larangan request ke diri sendiri (FR-081, FR-083) | Trigger + `dua_pihak_berbeda` | Penyaringan pencarian, pesan pengguna |
| Batas balasan 72 jam (FR-082) | `CHECK` urutan waktu saja | Perhitungan 72 jam dari `Clock` |
| Minggu dimulai Senin (Prinsip V) | `CHECK` pada tiga tabel | Perhitungan batas minggu di WIB |
| Seluruh waktu dari `Clock` (Prinsip V) | Tidak ada `DEFAULT now()` | Setiap `INSERT` mengirim waktunya |
| Satu ulasan per pesanan per pihak (FR-047) | Unik | Pemeriksaan status pesanan dikonfirmasi |
| Rating 1–5 (FR-047) | `CHECK` | Tidak ada |
| Percobaan kirim maksimal 3 (FR-085) | `CHECK` | Logika percobaan ulang |
| Golongan notifikasi (FR-091) | Kolom `transaksional` | Penetapan golongan saat penulisan |
| Mediasi wajib memutuskan alokasi dan penanggung (FR-067) | `CHECK` | Antarmuka admin |
| Batas ukuran berkas 5MB | `CHECK` | Magic bytes, pembuangan EXIF, kuota total 500MB |
| Item produk tidak tertukar dengan mesin | Trigger | Validasi form |
| Transisi status sah (FR-044) | Tidak ada | Mesin keadaan di `internal/order` |
| Urutan hasil deterministik (SC-013) | Tidak ada | `ORDER BY` lengkap + keyset (R-05) |
| Alokasi mengisi minggu terawal (FR-018) | Tidak ada | Algoritma alokasi |
| Ambang 3 pesanan (FR-073) | Tidak ada | Lapisan penyajian |

---

## 12. Urutan Migrasi

Mengikuti arah ketergantungan kunci asing:

```text
001_extensions            citext, pgcrypto
002_wilayah               wilayah_provinsi, wilayah_kota
003_akun                  akun_pengguna, sesi
004_master_data           item_daftar_baku, usulan_item
005_profil                profil_usaha
006_berkas_verifikasi     berkas_unggahan, pengajuan_verifikasi
007_listing               listing_kapasitas (+ horizon_sampai), listing_produk,
                          listing_mesin, trigger jenis item
008_periode               periode_ketersediaan
009_request               request_kuota, request_kandidat, trigger diri sendiri, penawaran
010_pesanan               pesanan (+ minggu_kesiapan_mulai), riwayat_status_pesanan,
                          catatan_pembayaran
011_alokasi               alokasi_kapasitas, trigger cegah alokasi sebelum kesiapan
012_reputasi              ulasan, sengketa
013_notifikasi            notifikasi (+ transaksional), notifikasi_kanal
014_batas_laju            batas_laju
```

`011_alokasi` harus setelah `010_pesanan` karena merujuk keduanya, dan triggernya membaca `pesanan.minggu_kesiapan_mulai`. Migrasi dijalankan otomatis saat startup dengan `pg_try_advisory_lock`, agar dua kontainer yang sempat hidup bersamaan saat penerapan versi baru tidak menjalankannya serentak.