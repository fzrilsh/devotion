# Quickstart: Devotion, Capacity Exchange

**Feature**: `docs/001-capacity-exchange-marketplace/`
**Date**: 2026-08-21
**Input**: `plan.md`, `research.md` (R-01, R-03, R-07), `data-model.md`, `contracts/openapi.yaml`
**Constitution**: v2.1.0

Berkas ini adalah sumber bagi tiga dokumen repository yang diwajibkan konstitusi:
`docs/setup-vps.md` (bagian B), `docs/menjalankan.md` (bagian C–D), dan
`docs/skenario-uji-manual.md` (bagian E–F).

Rincian yang tidak diulang di sini: bentuk tabel dan constraint ada di `data-model.md`,
bentuk endpoint dan kode galat ada di `contracts/openapi.yaml`, alasan setiap keputusan
teknis ada di `research.md`.

---

## A. Prasyarat

| Yang dibutuhkan | Keterangan |
|-----------------|------------|
| VPS Linux | 2GB RAM, 50GB disk, fresh install, akses root awal |
| Domain | Sudah dibeli, nameserver diarahkan ke Cloudflare |
| Akun Cloudflare | Domain sudah aktif di dasbor |
| Akun Mailjet | Kunci API dan secret sudah didapat |
| Akun Sentry | DSN untuk backend, opsional untuk frontend |
| Nomor WhatsApp khusus | Bukan nomor pribadi anggota tim; ponselnya tersedia untuk memindai QR |
| GitHub | Repository dan izin menulis ke GitHub Container Registry |
| Mesin lokal | Docker, Go 1.22+, Node 20+ untuk pengembangan |

Yang **tidak** dibutuhkan dan sengaja tidak dipakai: payment gateway (dilarang Batas
Keuangan konstitusi, sehingga mitigasi gagal bayar berupa escrow wajib pada dokumen
sumber [1] tidak berlaku di versi ini), object storage eksternal, reverse proxy sebagai
proses tersendiri, dan mail server sendiri.

---

## B. Runbook VPS dari Fresh Install

Enam belas langkah berurutan. Jangan melompat: langkah 4 bergantung pada data dari
langkah 6, dan langkah 12 tidak akan berhasil sebelum 11 terverifikasi.

### B1. Masuk pertama dan pengguna non-root

```bash
ssh root@<IP_VPS>

adduser devotion
usermod -aG sudo devotion
rsync --archive --chown=devotion:devotion ~/.ssh /home/devotion
```

Uji dari terminal **baru** sebelum menutup sesi root. Kalau kunci belum benar dan sesi
root sudah tertutup, kamu terkunci di luar:

```bash
ssh devotion@<IP_VPS>
```

Setelah berhasil, matikan masuk dengan kata sandi dan masuk sebagai root:

```bash
sudo sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo sed -i 's/^#*PermitRootLogin.*/PermitRootLogin no/'               /etc/ssh/sshd_config
sudo systemctl restart ssh
```

### B2. Pembaruan sistem dan zona waktu

```bash
sudo apt update && sudo apt upgrade -y
sudo timedatectl set-timezone Asia/Jakarta
timedatectl        # verifikasi: Time zone: Asia/Jakarta (WIB, +0700)
```

Zona waktu server disetel ke WIB agar log dan `pg_dump` mudah dibaca. Perhitungan batas
minggu tetap dilakukan eksplisit di kode, tidak diserahkan ke pengaturan ini. Prinsip V
mewajibkannya karena pengaturan server dapat berubah tanpa sepengetahuan aplikasi.

### B3. Swap 2GB

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

echo 'vm.swappiness=10' | sudo tee /etc/sysctl.d/99-devotion.conf
sudo sysctl --system

free -h            # verifikasi: Swap total 2,0Gi
```

Bukan untuk dipakai rutin. Ini agar lonjakan sesaat tidak berakhir dengan proses dibunuh
kernel, dan yang biasanya dibunuh adalah Postgres, bukan penyebab lonjakannya.

### B4. Firewall

Ambil rentang alamat Cloudflare dari sumber resmi. Jangan pakai daftar dari dokumen
mana pun tanpa dicocokkan, termasuk daftar di `research.md` R-01 yang ditulis sebagai
acuan dan wajib diverifikasi:

```bash
curl -s https://www.cloudflare.com/ips-v4 -o /tmp/cf-v4
curl -s https://www.cloudflare.com/ips-v6 -o /tmp/cf-v6
cat /tmp/cf-v4 /tmp/cf-v6      # periksa isinya sebelum dipakai
```

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp comment 'SSH'

while read -r cidr; do
  [ -n "$cidr" ] && sudo ufw allow from "$cidr" to any port 443 proto tcp comment 'Cloudflare'
done < /tmp/cf-v4
while read -r cidr; do
  [ -n "$cidr" ] && sudo ufw allow from "$cidr" to any port 443 proto tcp comment 'Cloudflare v6'
done < /tmp/cf-v6

sudo ufw enable
sudo ufw status numbered
```

Port 80 **tidak** dibuka. Cloudflare menghubungi origin di 443 pada mode Full (strict),
dan pengalihan HTTP ke HTTPS ditangani di tepi.

Salin daftar yang sudah diverifikasi ke `docs/cloudflare-ips.md` beserta tanggal
pengambilannya, dan patok nilainya sebagai konstanta Go. Jangan mengambilnya lewat
jaringan saat aplikasi naik, karena satu kegagalan HTTP akan membuat aplikasi gagal
menyala.

Setelah langkah B6, uji bahwa origin benar-benar tertutup dari luar Cloudflare:

```bash
curl -sk --max-time 5 https://<IP_VPS>/api/health    # harus timeout atau tertolak
curl -s https://devotion.cloud/api/health                  # harus 200
```

Bila perintah pertama berhasil, lapisan tepi bisa dilewati begitu saja beserta seluruh
pembatasan lajunya. Jangan lanjut sebelum ini benar.

### B5. Docker

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker devotion
newgrp docker
docker --version && docker compose version
```

### B6. Cloudflare

Di dasbor, berurutan:

| Bagian | Setelan | Alasan |
|--------|---------|--------|
| DNS | `A` record `@` → IP VPS, **Proxied** (oranye) | Menyembunyikan IP origin |
| SSL/TLS → Overview | **Full (strict)** | Mode Flexible membuat cookie `Secure` tidak terkirim dan login gagal tanpa pesan galat yang jelas |
| SSL/TLS → Origin Server | Buat **Origin Certificate**, simpan sertifikat dan kunci | Berlaku 15 tahun, tanpa perpanjangan |
| SSL/TLS → Origin Server | Aktifkan **Authenticated Origin Pulls** | Lapisan kedua bila aturan firewall salah |
| SSL/TLS → Edge Certificates | **Always Use HTTPS** aktif | Pengalihan di tepi, bukan di origin |
| Caching → Cache Rules | Bypass cache untuk `/api/*` | Hasil pencarian yang ter-cache menampilkan kapasitas basi, persis masalah data tidak aktual yang platform ini dibangun untuk menyelesaikan |

Simpan sertifikat di server dengan izin ketat:

```bash
sudo mkdir -p /opt/devotion/tls
sudo nano /opt/devotion/tls/origin.pem      # tempel sertifikat
sudo nano /opt/devotion/tls/origin.key      # tempel kunci privat
sudo curl -so /opt/devotion/tls/cf-client-ca.pem \
  https://developers.cloudflare.com/ssl/static/authenticated_origin_pull_ca.pem

sudo chown -R devotion:devotion /opt/devotion/tls
sudo chmod 600 /opt/devotion/tls/origin.key
sudo chmod 644 /opt/devotion/tls/origin.pem /opt/devotion/tls/cf-client-ca.pem
```

### B7. Struktur direktori dan volume

```bash
sudo mkdir -p /opt/devotion/{tls,unggahan,cadangan}
sudo chown -R devotion:devotion /opt/devotion
```

`unggahan` adalah volume terpisah dari image, agar penerapan versi baru tidak menghapus
berkas yang sudah diunggah.

### B8. Variabel lingkungan

```bash
cd /opt/devotion
curl -so .env.example https://raw.githubusercontent.com/<org>/devotion/main/.env.example
cp .env.example .env
nano .env
chmod 600 .env
```

Nama variabel yang harus terisi. Nilainya tidak ada di dokumen mana pun, termasuk berkas
ini:

```text
APP_ENV=production
APP_BASE_URL=https://devotion.cloud
TLS_CERT_PATH=/opt/devotion/tls/origin.pem
TLS_KEY_PATH=/opt/devotion/tls/origin.key
CF_CLIENT_CA_PATH=/opt/devotion/tls/cf-client-ca.pem

POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=devotion
DATABASE_URL=

MAILJET_API_KEY=
MAILJET_SECRET_KEY=
MAIL_FROM=noreply@devotion.cloud

WHATSAPP_NOMOR=
SENTRY_DSN=

UNGGAHAN_PATH=/opt/devotion/unggahan
UNGGAHAN_BATAS_TOTAL_MB=500
UNGGAHAN_BATAS_BERKAS_MB=5
```

`.env` tidak pernah masuk repository. `.env.example` hanya memuat nama kunci tanpa nilai.

### B9. Konfigurasi email di Cloudflare DNS

Lakukan **sekarang, bukan menjelang tenggat**. Reputasi domain baru butuh waktu terbentuk
dan tidak dapat dipercepat di hari terakhir.

Ambil nilai SPF dan DKIM dari dasbor Mailjet, lalu tambahkan sebagai record DNS di
Cloudflare dengan proxy **dimatikan** (record TXT tidak diproksikan). Tambahkan juga
DMARC dengan `p=none` pada tahap ini.

Verifikasi di dasbor Mailjet sampai domain berstatus terverifikasi, lalu kirim satu email
uji ke Gmail dan periksa apakah masuk kotak masuk atau spam. Kegagalan email bersifat
senyap (server penerima menjawab sukses lalu membuang pesannya), sehingga dasbor Mailjet
adalah satu-satunya cara mengetahui apa yang sebenarnya terjadi.

Karena FR-002 menjadikan verifikasi email sebagai gerbang, kegagalan di sini berarti tidak
satu pun user story dapat didemokan. Bukan fitur notifikasi yang rusak, melainkan pintu
masuknya.

### B10. docker-compose.yml

Tepat dua layanan. Angka penyetelan Postgres berasal dari `research.md` R-03.

```yaml
services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER:     ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB:       ${POSTGRES_DB}
      TZ: Asia/Jakarta
    command:
      - postgres
      - -c max_connections=20
      - -c shared_buffers=256MB
      - -c effective_cache_size=768MB
      - -c work_mem=4MB
      - -c maintenance_work_mem=64MB
      - -c wal_buffers=8MB
      - -c min_wal_size=128MB
      - -c max_wal_size=512MB
      - -c checkpoint_completion_target=0.9
      - -c random_page_cost=1.1
      - -c effective_io_concurrency=200
      - -c log_min_duration_statement=500ms
      - -c timezone=Asia/Jakarta
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5
    logging:
      driver: json-file
      options: { max-size: "10m", max-file: "3" }

  backend:
    image: ghcr.io/<org>/devotion:latest
    restart: unless-stopped
    depends_on:
      postgres: { condition: service_healthy }
    env_file: .env
    ports:
      - "443:443"
    volumes:
      - /opt/devotion/tls:/tls:ro
      - /opt/devotion/unggahan:/unggahan
    healthcheck:
      test: ["CMD", "/devotion", "health:check"]
      interval: 30s
      timeout: 5s
      retries: 3
    logging:
      driver: json-file
      options: { max-size: "10m", max-file: "3" }

volumes:
  pgdata:
```

Batas log bukan kebersihan, melainkan pencegahan kegagalan total: log yang tumbuh tanpa
batas akan mengisi 50GB, lalu Postgres berhenti menerima tulisan dan aplikasi mati.

Tidak ada layanan frontend karena Go menyajikannya, dan tidak ada layanan proxy karena Go
menghabiskan TLS sendiri. **Cara memeriksa Gate I: hitung entri di bawah `services:`.
Lebih dari dua berarti pelanggaran.**

### B11. Menyalakan dan memverifikasi migrasi

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin
docker compose pull
docker compose up -d
docker compose logs -f backend
```

Yang harus terlihat di log, berurutan: migrasi berjalan sampai versi terakhir, koneksi
basis data berhasil, penjadwal menyala, server mendengarkan di 443. Migrasi dijalankan
otomatis saat startup dengan penguncian, sehingga dua kontainer yang sempat hidup
bersamaan saat penerapan versi baru tidak menjalankannya serentak.

```bash
docker compose exec postgres psql -U ${POSTGRES_USER} -d devotion \
  -c "select version, dirty from schema_migrations;"
```

`dirty = true` berarti migrasi gagal di tengah. Jangan lanjut. Periksa log, perbaiki,
lalu jalankan ulang. Melanjutkan dengan skema setengah jadi akan menghasilkan galat yang
tampak tidak berhubungan di langkah-langkah berikutnya.

Sekarang jalankan uji firewall dari langkah B4.

### B12. Mengisi data acuan dan membuat admin

Urutannya mengikat: daftar baku dan wilayah adalah prasyarat data bagi User Story 1 dan 2,
dan antarmuka pencocokan tidak dapat didemokan tanpa ketiganya.

```bash
# Wilayah: dua tingkat administratif saja (provinsi, kota/kabupaten).
# Di server, baca dari salinan repository, jangan bergantung pada layanan luar.
docker compose exec backend /devotion seed:wilayah

# Daftar baku jenis produk dan jenis mesin.
docker compose exec backend /devotion seed:master-data

# Admin pertama. Kata sandi diminta lewat prompt, tidak lewat argumen,
# agar tidak tersimpan di riwayat shell.
docker compose exec -it backend /devotion admin:create --email admin@devotion.cloud
```

Ketiganya idempoten: menjalankan dua kali tidak menduplikasi data.

Pengambilan pertama data wilayah dari sumber luar dilakukan **sekali di mesin lokal**,
bukan di server, lalu hasilnya di-commit ke `docs/master-data/wilayah.json`:

```bash
./devotion seed:wilayah --refresh    # hanya di mesin lokal, sekali
```

Verifikasi:

```bash
docker compose exec postgres psql -U ${POSTGRES_USER} -d devotion -c "
  select (select count(*) from wilayah_provinsi) as provinsi,
         (select count(*) from wilayah_kota)     as kota,
         (select count(*) from item_daftar_baku where jenis='produk') as produk,
         (select count(*) from item_daftar_baku where jenis='mesin')  as mesin,
         (select count(*) from akun_pengguna where peran_admin)       as admin;"
```

Kelimanya harus lebih dari nol. Bila `kota` nol sementara `provinsi` terisi, pemetaan kode
gagal. Constraint `kota_milik_provinsinya` menolak baris yang dua digit pertama kodenya
tidak cocok dengan kode provinsinya, dan itu memang gunanya: gagal keras saat seed, bukan
senyap saat pencarian.

### B13. Menyambungkan WhatsApp

Buka `https://devotion.cloud/admin/whatsapp`, masuk sebagai admin, pindai QR dengan ponsel yang
memegang nomor khusus lomba.

Sesi dapat lepas kapan saja, termasuk bila ponselnya lama tidak aktif. Halaman ini ada
justru agar penyambungan ulang tidak memerlukan akses SSH, dan itu penting karena FR-002
menjadikan verifikasi nomor HP sebagai gerbang pendaftaran.

Risiko yang diterima secara sadar: whatsmeow memakai protokol WhatsApp Web, bukan API
resmi. Mengirim kode ke nomor yang belum pernah berinteraksi adalah pola yang paling
mungkin dideteksi sebagai spam. Bila nomor terblokir, jalan daruratnya:

```bash
docker compose exec backend /devotion user:verify --phone 62xxxxxxxxxx
```

### B14. Health check dan pemantau uptime

```bash
curl -s https://devotion.cloud/api/health | jq
```

Yang diharapkan: `status: sehat`, basis data sehat, WhatsApp tersambung, penyimpanan
berkas sehat dengan `terpakai_mb` jauh di bawah 500.

Daftarkan URL itu ke layanan pemantau uptime gratis dengan interval 5 menit. Layanannya
berada di luar server sehingga tidak dihitung dalam batas dua layanan, dan wajib dicatat
di `docs/layanan-luar.md` beserta akibat bila mati.

### B15. Cadangan basis data

Cron di tingkat host, bukan di dalam kontainer, karena `pg_dump` harus tetap berjalan
ketika aplikasi sedang mati atau rusak, justru saat itulah cadangan paling dibutuhkan.
Penyimpangan ini tercatat di Complexity Tracking `plan.md`.

```bash
cat > /opt/devotion/cadangan.sh <<'EOF'
#!/bin/bash
set -euo pipefail
cd /opt/devotion
set -a; . ./.env; set +a
STAMP=$(date +%Y%m%d-%H%M)
docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" devotion \
  | gzip > "/opt/devotion/cadangan/devotion-$STAMP.sql.gz"
ls -1t /opt/devotion/cadangan/devotion-*.sql.gz | tail -n +4 | xargs -r rm --
EOF

chmod +x /opt/devotion/cadangan.sh
/opt/devotion/cadangan.sh        # uji sekarang, jangan tunggu besok
ls -lh /opt/devotion/cadangan/

( crontab -l 2>/dev/null; echo "15 2 * * * /opt/devotion/cadangan.sh >> /opt/devotion/cadangan/cron.log 2>&1" ) | crontab -
```

Disalurkan langsung ke gzip agar tidak menulis berkas mentah besar, dan hanya tiga salinan
terakhir disimpan.

**Salin keluar VPS.** Cadangan yang tersimpan di disk yang sama tidak menolong ketika VPS
itu sendiri yang bermasalah. Dari mesin lokal:

```bash
rsync -avz devotion@devotion.cloud:/opt/devotion/cadangan/ ./cadangan-devotion/
```

### B16. Snapshot VPS

Setelah semua langkah di atas terverifikasi, ambil snapshot dari panel penyedia VPS.

Ini yang paling penting dari seluruh bagian B. Satu server tanpa cadangan berarti satu
kesalahan dapat menghapus seluruh submission, dan itu terjadi jauh lebih sering daripada
yang diperkirakan.

---

## C. Alur Penerapan Versi Baru

Membangun artefak dilarang dilakukan di server. Build Vite pada mesin 2GB sambil Postgres
hidup adalah resep kehabisan memori, dan yang dibunuh kernel biasanya Postgres.

```text
push ke main
  └─ GitHub Actions
       ├─ go vet ./...
       ├─ go test ./...            (skema uji terpisah)
       ├─ npm ci && npm test       (Jest)
       ├─ npm run build            → frontend/dist
       ├─ salin dist → backend/webdist
       ├─ docker build (multi-stage, embed.FS)
       └─ push ghcr.io/<org>/devotion:<sha> dan :latest
  └─ SSH ke VPS
       └─ docker compose pull && docker compose up -d
```

Di VPS, satu-satunya perintah:

```bash
cd /opt/devotion && docker compose pull && docker compose up -d
```

Migrasi berjalan otomatis saat kontainer baru naik. Berhenti sebentar lalu hidup lagi
sudah memadai untuk lomba, dan jauh lebih sedikit yang bisa salah dibanding penerapan
tanpa jeda.

Kunci SSH disimpan sebagai secret di GitHub, tidak pernah di dalam repository.

---

## D. Menjalankan Sistem dan Pengujian di Mesin Lokal

```bash
git clone https://github.com/<org>/devotion.git && cd devotion
cp .env.example .env        # isi untuk lokal; APP_ENV=development

docker compose up -d postgres
cd backend && go run ./cmd/devotion serve

cd ../frontend && npm ci && npm run dev
```

Pada `APP_ENV=development`, backend melayani HTTP biasa tanpa TLS dan tanpa pemeriksaan
sertifikat klien Cloudflare, dan Vite dev server memproksikan `/api` ke backend.

Pengujian:

```bash
cd backend  && go test ./...              # seluruh pengujian backend
cd frontend && npm test                   # Jest
```

Pengujian backend memakai **skema terpisah pada layanan Postgres yang sama**, bukan
layanan basis data tambahan. Konstitusi melarang menambah layanan untuk keperluan
pengujian.

Pengujian yang menyangkut tenggat memakai sumber waktu yang digantikan, bukan menunggu
waktu nyata. Tanpa itu, konfirmasi otomatis tujuh hari hanya dapat diverifikasi dengan
menunggu tujuh hari.

Yang wajib lulus sebelum sebuah story dinyatakan selesai ada di Gate konstitusi; daftar
aturan yang wajib diuji secara khusus ada di `contracts/README.md`.

---

## E. Data Uji dan Akun Uji

```bash
docker compose exec backend /devotion seed:test-data      # menyiapkan
docker compose exec backend /devotion reset:test-data     # memulihkan ke keadaan awal
```

`reset:test-data` ada agar penguji kedua tidak mewarisi kekacauan penguji pertama.

### Yang disiapkan

| Kelompok | Isi |
|----------|-----|
| Usaha | 50 profil dengan kota tersebar di lima kota target; sekitar sepertiga terverifikasi |
| Listing | Kapasitas mingguan bervariasi 150–1.500 potong, jeda kesiapan 0–30 hari |
| Kapasitas lintas periode | Satu kandidat berkapasitas 500/minggu untuk menguji pesanan 3.000 potong |
| Kalender basi | Satu listing dengan kalender tidak diperbarui 8 hari |
| Request kedaluwarsa | Satu request yang dikirim lebih dari 72 jam lalu tanpa balasan |
| Tenggat mendekat | Satu pesanan berstatus Dikirim sejak 6 hari lalu |
| Tenggat terlampaui | Satu pesanan berstatus Dikirim sejak 8 hari lalu |
| Pesanan telat | Satu pesanan berstatus Produksi yang melewati deadline |
| Pengajuan verifikasi | Dua menunggu keputusan admin |
| Reputasi | Beberapa usaha dengan ulasan; satu usaha hanya 2 pesanan untuk menguji ambang data |

Alur bertenggat disiapkan sebagai data yang sudah berada pada keadaan itu, **bukan** lewat
kendali geser waktu. Data yang di-seed lebih sederhana dan tidak berisiko ikut terbawa ke
lingkungan sungguhan.

### Akun uji

| Peran | Email | Kata sandi |
|-------|-------|------------|
| Subkontraktor | `budi@contoh.test` | `UjiDevotion123!` |
| Pemberi order | `sari@contoh.test` | `UjiDevotion123!` |
| Peran ganda | `dua@contoh.test` | `UjiDevotion123!` |
| Admin | `admin@contoh.test` | `UjiDevotion123!` |

**Pengecualian yang disengaja terhadap Batasan Keamanan konstitusi.** Konstitusi melarang
kredensial di repository, tetapi juga mewajibkan kredensial akun uji tersedia di `docs/`
bagi penguji eksternal. Keduanya diselesaikan dengan tiga syarat yang mengikat:

1. Akun-akun ini **hanya** ada pada data yang dibuat `seed:test-data`, dan perintah itu
   **menolak berjalan** ketika `APP_ENV=production`.
2. Kata sandi di atas tidak pernah dipakai untuk akun sungguhan mana pun, termasuk admin
   yang dibuat `admin:create`.
3. Domain `.test` tidak dapat diregistrasi, sehingga tidak ada email sungguhan yang
   terlibat. Ini juga memenuhi larangan memakai data pribadi orang sungguhan di data uji.

Bila platform ini kemudian dipakai sungguhan, penghapusan seluruh akun `.test` adalah
langkah pertama.

---

## F. Verifikasi Manual per User Story

Untuk penguji di luar tim. Setiap langkah menyebutkan akun yang dipakai, tindakan, dan
hasil yang diharapkan. Kolom terakhir untuk menuliskan **apa yang benar-benar terjadi**,
bukan hanya lulus atau gagal, karena uraian bebas dari penguji sering menemukan hal yang
tidak ada di skenario ini.

### Yang di luar cakupan pengujian Anda

**Notifikasi WhatsApp dan email tidak perlu Anda periksa.** Keduanya dikirim ke nomor dan
alamat yang tidak Anda miliki. Jalur pengamatan notifikasi adalah **ikon lonceng di dalam
aplikasi**. Notifikasi selalu tampil di sana meskipun pengiriman ke WhatsApp dan email
gagal seluruhnya. Bila WhatsApp atau email tidak sampai, itu bukan temuan.

---

### US1: Subkontraktor mempublikasikan kapasitas produksinya

Menjawab masalah pencarian subkontraktor yang selama ini hanya lewat relasi personal
sehingga jangkauannya terbatas dan tidak ada mekanisme matching sistematis [1].

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 1.1 | publik | Daftar akun baru sebagai subkontraktor dengan email dan nomor HP Anda sendiri | Muncul permintaan memasukkan kode verifikasi untuk email dan nomor HP | |
| 1.2 | akun baru | Coba buka halaman pembuatan listing sebelum verifikasi selesai | Ditolak dengan penjelasan bahwa email dan nomor HP harus diverifikasi lebih dulu | |
| 1.3 | `budi@contoh.test` | Buka halaman listing kapasitas | Terlihat kapasitas mingguan, jeda kesiapan mulai, jenis produk, dan jenis mesin | |
| 1.4 | `budi@contoh.test` | Perhatikan kolom jenis produk | Hanya dapat memilih dari daftar yang tersedia; tidak dapat menaip nama sendiri | |
| 1.5 | `budi@contoh.test` | Perhatikan apakah ada kolom kapasitas terpisah untuk setiap jenis produk | Tidak ada. Hanya satu angka kapasitas mingguan untuk seluruh jenis produk | |
| 1.6 | `budi@contoh.test` | Kosongkan kapasitas mingguan, lalu simpan | Listing tidak tersimpan; pesan menyebut kolom mana yang belum diisi | |
| 1.7 | `budi@contoh.test` | Isi lengkap lalu simpan | Listing tersimpan dan langsung berstatus tayang, tanpa menunggu persetujuan siapa pun | |
| 1.8 | tanpa masuk | Buka halaman publik profil Pak Budi | Seluruh atribut kapasitas tampil benar sesuai yang diisi | |
| 1.9 | `budi@contoh.test` | Usulkan satu jenis produk yang belum ada di daftar | Usulan terkirim, dan listing tetap dapat disimpan dengan pilihan yang tersedia | |
| 1.10 | `budi@contoh.test` | Ubah kapasitas mingguan, lalu buka lagi halaman publiknya | Perubahan langsung terlihat | |

Mengacu Acceptance Scenario US1 nomor 1–6.

---

### US2: Pemberi order menemukan subkontraktor yang cocok

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 2.1 | `sari@contoh.test` | Cari: produk "Kaos Polos", kota "Bandung", jumlah 3.000, deadline 8 minggu dari hari ini | Muncul daftar kandidat; setiap baris menampilkan jeda kesiapan mulai dan total kapasitas sampai deadline | |
| 2.2 | `sari@contoh.test` | Cari kandidat yang kapasitas mingguannya 500 potong | Kandidat itu **ikut muncul** dan ditandai memenuhi kriteria kapasitas, karena totalnya sampai deadline mencukupi | |
| 2.3 | `sari@contoh.test` | Ulangi pencarian yang sama, tetapi deadline hanya 4 minggu | Kandidat 500 potong per minggu kini ditandai **tidak** memenuhi kriteria kapasitas dan turun peringkat | |
| 2.4 | `sari@contoh.test` | Jalankan pencarian yang sama dua kali berturut-turut | Urutan hasil sama persis pada kedua percobaan | |
| 2.5 | `sari@contoh.test` | Buka halaman 2, lalu kembali ke halaman 1 | Tidak ada kandidat yang muncul dua kali dan tidak ada yang hilang | |
| 2.6 | `sari@contoh.test` | Bandingkan dua kandidat yang memenuhi kriteria sama, satu berlencana terverifikasi | Lencana **tidak** mengubah posisi keduanya | |
| 2.7 | `sari@contoh.test` | Perhatikan kandidat berating tinggi dan kandidat berating rendah dengan kriteria sama | Rating **tidak** mengubah urutan | |
| 2.8 | `sari@contoh.test` | Cari dengan filter sangat sempit sampai hasilnya kosong, lalu tekan perluas | Cakupan naik dari kota ke provinsi; tingkat yang dipakai disebutkan pada hasil | |
| 2.9 | `sari@contoh.test` | Perluas sekali lagi | Cakupan menjadi seluruh Indonesia, dan tombol perluas tidak tersedia lagi | |
| 2.10 | `sari@contoh.test` | Bila masih kosong, perhatikan pesan yang muncul | Disebutkan filter mana yang paling membatasi beserta saran pelonggaran yang konkret | |
| 2.11 | `sari@contoh.test` | Buka profil satu kandidat yang mencantumkan titik lokasi | Peta menampilkan posisi usaha beserta perkiraan jarak | |
| 2.12 | `dua@contoh.test` | Sebagai akun berperan ganda, jalankan pencarian | Listing milik sendiri **tidak** muncul di hasil | |

Mengacu Acceptance Scenario US2 nomor 1–10.

---

### US3: Request kuota dan perbandingan penawaran

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 3.1 | `sari@contoh.test` | Pilih 3 kandidat, kirim satu request dengan produk, jumlah, bahan, dan deadline | Ketiganya tercatat "Menunggu Balasan", dan batas waktu balasan 72 jam terlihat | |
| 3.2 | `sari@contoh.test` | Cari kolom untuk mengatur sendiri batas waktu balasan | Tidak ada. Sistem yang menetapkan 72 jam | |
| 3.3 | `dua@contoh.test` | Coba kirim request kuota ke listing milik sendiri | Ditolak dengan penjelasan yang jelas | |
| 3.4 | `budi@contoh.test` | Buka request masuk, kirim estimasi harga dan jeda kesiapan mulai | Penawaran terkirim, status kandidat berubah menjadi ditawar | |
| 3.5 | `budi@contoh.test` | Coba menyanggupi jumlah yang jauh melampaui kapasitas Anda sampai deadline | Ditolak, dan pesannya **menyebutkan angka** total kapasitas yang sebenarnya tersisa | |
| 3.6 | `sari@contoh.test` | Buka detail request setelah dua kandidat membalas | Kedua penawaran tampil berdampingan beserta harga dan jeda kesiapan masing-masing | |
| 3.7 | `sari@contoh.test` | Ajukan counter-offer harga | Penawaran baru tercatat, dan riwayat penawaran sebelumnya tetap terlihat | |
| 3.8 | `budi@contoh.test` | Setujui counter-offer | Pesanan terbentuk dengan harga, jumlah, dan deadline yang disepakati | |
| 3.9 | `sari@contoh.test` | Periksa kandidat lain pada request yang sama | Keduanya berubah menjadi "Tidak Dilanjutkan" | |
| 3.10 | `sari@contoh.test` | Buka request kedaluwarsa yang sudah disiapkan | Berstatus "Kedaluwarsa", dan tidak ada lagi tombol untuk membalas | |
| 3.11 | `sari@contoh.test` | Buka ikon lonceng | Ada notifikasi kesepakatan terbentuk | |

Mengacu Acceptance Scenario US3 nomor 1–7.

---

### US4: Kalender ketersediaan tetap aktual

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 4.1 | `budi@contoh.test` | Buka kalender ketersediaan | Terlihat periode mingguan minimal 3 bulan ke depan, setiap periode dimulai hari Senin | |
| 4.2 | `budi@contoh.test` | Catat kapasitas tersisa minggu depan dan dua minggu setelahnya | Angkanya tercatat untuk dibandingkan nanti | |
| 4.3 | `sari@contoh.test` | Sepakati pesanan yang jumlahnya melebihi kapasitas satu minggu Pak Budi | Pesanan terbentuk | |
| 4.4 | `budi@contoh.test` | Buka kalender lagi | Kapasitas berkurang dari minggu **paling awal** lebih dulu, baru meluber ke minggu berikutnya; minggu jauh tetap utuh | |
| 4.5 | `budi@contoh.test` | Buka detail pesanan itu | Terlihat rincian berapa potong dipakai pada masing-masing minggu | |
| 4.6 | `sari@contoh.test` | Batalkan pesanan itu (masih sebelum produksi) dengan menyebut alasan | Pesanan berstatus Dibatalkan | |
| 4.7 | `budi@contoh.test` | Buka kalender lagi | Kapasitas **seluruh** minggu yang tadi terpakai kembali ke angka pada langkah 4.2 | |
| 4.8 | `budi@contoh.test` | Tandai satu minggu sebagai "Penuh" | Minggu itu tampil penuh | |
| 4.9 | `sari@contoh.test` | Cari dengan deadline yang hanya mencakup minggu tersebut | Pak Budi tidak dihitung memenuhi kriteria kapasitas | |
| 4.10 | `budi@contoh.test` | Coba tandai penuh sebuah minggu yang sudah dipakai pesanan berjalan | Ditolak, dan pesannya menyebut minggu mana beserta jumlah yang sudah terpakai | |
| 4.11 | `sari@contoh.test` | Cari kandidat, temukan listing berkalender basi yang sudah disiapkan | Ditandai "Data Belum Diperbarui", tetapi posisinya di urutan **tidak** berubah karenanya | |

Mengacu Acceptance Scenario US4 nomor 1–6.

---

### US5: Memantau pesanan sampai tuntas

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 5.1 | `budi@contoh.test` | Ubah status pesanan menjadi "Produksi" | Status berubah | |
| 5.2 | `sari@contoh.test` | Buka pesanan yang sama | Status baru terlihat beserta waktu perubahannya | |
| 5.3 | `sari@contoh.test` | Cari tombol batalkan pada pesanan yang sudah Produksi | Tidak tersedia, dan Anda diarahkan untuk melaporkan sengketa | |
| 5.4 | `budi@contoh.test` | Coba ubah status langsung dari Produksi ke Dikirim | Ditolak, dan pesannya **menyebutkan urutan status yang diizinkan** | |
| 5.5 | `budi@contoh.test` | Ubah berurutan: Selesai, lalu Dikirim | Kedua perubahan berhasil | |
| 5.6 | `sari@contoh.test` | Konfirmasi penerimaan | Pesanan pindah ke riwayat, dan muncul ajakan memberi rating | |
| 5.7 | `sari@contoh.test` | Buka pesanan yang statusnya Dikirim sejak 6 hari lalu | Terlihat tanggal pesanan akan dianggap diterima otomatis | |
| 5.8 | `sari@contoh.test` | Buka ikon lonceng | Ada pemberitahuan bahwa tenggat konfirmasi mendekat | |
| 5.9 | `sari@contoh.test` | Buka pesanan yang statusnya Dikirim sejak 8 hari lalu | Sudah dikonfirmasi otomatis, dan ditandai bahwa penutupan terjadi otomatis, bukan oleh Anda | |
| 5.10 | `sari@contoh.test` | Pada pesanan yang tenggatnya mendekat, laporkan sengketa | Hitungan konfirmasi otomatis **berhenti**; pesanan menunggu mediasi admin | |
| 5.11 | `sari@contoh.test` | Catat pembayaran sebagai "terkirim" | Catatan tampil, disertai keterangan bahwa platform tidak menahan maupun menjamin dana | |
| 5.12 | `budi@contoh.test` | Buka pesanan yang sama | Catatan pembayaran dari Bu Sari terlihat | |
| 5.13 | `budi@contoh.test` | Cari kolom untuk memasukkan jumlah uang | Tidak ada. Yang dicatat hanya pernyataan bahwa pembayaran terjadi | |

Mengacu Acceptance Scenario US5 nomor 1–9.

---

### US6: Reputasi dari transaksi nyata

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 6.1 | `sari@contoh.test` | Beri rating dan ulasan pada pesanan yang sudah dikonfirmasi | Ulasan tersimpan | |
| 6.2 | tanpa masuk | Buka profil publik Pak Budi | Ulasan tampil beserta nama pemberi ulasan dan tanggal transaksi; tidak anonim | |
| 6.3 | `sari@contoh.test` | Coba beri ulasan kedua pada pesanan yang sama | Ditolak | |
| 6.4 | `sari@contoh.test` | Coba beri rating pada usaha yang belum pernah bertransaksi dengan Anda | Tidak tersedia atau ditolak | |
| 6.5 | `sari@contoh.test` | Coba beri ulasan pada pesanan yang belum dikonfirmasi | Ditolak dengan penjelasan bahwa pesanan harus dikonfirmasi diterima lebih dulu | |
| 6.6 | tanpa masuk | Buka profil usaha yang baru punya 2 pesanan | Tingkat penyelesaian **tidak** ditampilkan sebagai persentase; ada keterangan bahwa data belum cukup | |
| 6.7 | `sari@contoh.test` | Catat tingkat penyelesaian Anda dan Pak Budi, lalu batalkan satu pesanan sebelum produksi | Pesanan berstatus Dibatalkan | |
| 6.8 | tanpa masuk | Buka kedua profil | Tingkat penyelesaian **Bu Sari** turun; tingkat penyelesaian **Pak Budi** tidak berubah sama sekali | |
| 6.9 | `admin@contoh.test` | Sembunyikan satu ulasan dengan alasan | Ulasan hilang dari profil publik dan rata-rata rating berubah karenanya | |

Mengacu Acceptance Scenario US6 nomor 1–5. Langkah 6.8 adalah yang paling penting: itu
yang membedakan aturan tingkat penyelesaian ini dari perhitungan biasa.

---

### US7: Admin: daftar baku, lencana, mediasi

| # | Akun | Langkah | Hasil yang diharapkan | Apa yang terjadi |
|---|------|---------|-----------------------|------------------|
| 7.1 | `admin@contoh.test` | Tambah satu jenis produk baru | Item tersimpan | |
| 7.2 | `budi@contoh.test` | Buka form listing | Item baru itu sudah dapat dipilih | |
| 7.3 | `sari@contoh.test` | Buka filter pencarian | Item baru itu sudah dapat dipilih | |
| 7.4 | `admin@contoh.test` | Nonaktifkan satu item yang masih dipakai listing tayang | Item tidak dapat dipilih lagi untuk listing baru | |
| 7.5 | `sari@contoh.test` | Cari listing yang memakai item yang baru dinonaktifkan | Listing itu **tetap utuh dan tetap dapat ditemukan** | |
| 7.6 | `admin@contoh.test` | Buka antrean usulan item, setujui satu, tolak satu | Kedua keputusan tersimpan | |
| 7.7 | akun pengusul | Buka ikon lonceng | Ada notifikasi hasil keputusan usulan | |
| 7.8 | `admin@contoh.test` | Buka antrean verifikasi, buka satu pengajuan | Dokumen identitas dan foto lokasi dapat dilihat | |
| 7.9 | `admin@contoh.test` | Setujui satu pengajuan | Usaha itu mendapat lencana terverifikasi di profil dan hasil pencarian | |
| 7.10 | `admin@contoh.test` | Tolak satu pengajuan dengan alasan | Tersimpan; pengaju dapat mengajukan ulang | |
| 7.11 | `sari@contoh.test` | Cari usaha yang pengajuannya baru ditolak | Listing-nya **tetap tayang**, hanya tanpa lencana | |
| 7.12 | `admin@contoh.test` | Buka daftar pesanan yang melewati deadline | Pesanan yang sudah disiapkan muncul di sana | |
| 7.13 | `admin@contoh.test` | Buka sengketa dari langkah 5.10, tandai "Dalam Mediasi" | Status berubah; seluruh riwayat pesanan dapat dibaca, termasuk catatan pembayaran | |
| 7.14 | `admin@contoh.test` | Tutup mediasi sebagai dibatalkan | **Wajib** menentukan apakah kapasitas dikembalikan dan pihak mana yang menanggung; tidak dapat ditutup tanpa keduanya | |
| 7.15 | `budi@contoh.test` | Bila admin memilih mengembalikan kapasitas, buka kalender | Kapasitas periode terkait kembali | |
| 7.16 | `budi@contoh.test` | Coba buka halaman admin mana pun | Ditolak karena peran tidak berwenang | |
| 7.17 | `sari@contoh.test` | Coba buka berkas dokumen identitas milik usaha lain lewat tautan langsung | Ditolak | |

Mengacu Acceptance Scenario US7 nomor 1–9. Langkah 7.17 memverifikasi pembatasan akses
dokumen identitas, yang pada dokumen sumber dimitigasi lewat akses berbasis peran [1].

---

### Melaporkan temuan

Untuk setiap langkah yang hasilnya berbeda dari kolom "hasil yang diharapkan", tuliskan:
nomor langkah, akun yang dipakai, apa yang Anda lakukan, apa yang muncul di layar
(kutip pesannya bila ada), dan apakah dapat diulang.

Temuan dicatat di `docs/temuan-penguji.md` beserta keputusan tim: diperbaiki, atau
diterima sebagai utang dengan alasannya.

---

## G. Bila Ada yang Salah

| Gejala | Kemungkinan penyebab | Yang diperiksa |
|--------|----------------------|----------------|
| Kesalahan TLS di browser | Mode SSL/TLS bukan Full (strict), atau sertifikat origin salah | Dasbor Cloudflare; `docker compose logs backend` |
| Login berhasil lalu langsung keluar | Mode Flexible; cookie `Secure` tidak terkirim | Setel ke Full (strict) |
| `curl https://<IP_VPS>` berhasil | Aturan firewall tidak berlaku; lapisan tepi dapat dilewati | `sudo ufw status numbered`; ulangi B4 |
| Aplikasi tidak naik, log menyebut migrasi | Migrasi gagal di tengah | Periksa `schema_migrations.dirty` |
| Kota kosong setelah seed wilayah | Kode kota tidak cocok dengan kode provinsinya | Periksa `docs/master-data/wilayah.json`; jalankan `--refresh` di lokal |
| Pencarian tidak menghasilkan apa pun | Daftar baku atau wilayah belum terisi | Ulangi B12 dan verifikasi hitungannya |
| Kode verifikasi email tidak sampai | SPF/DKIM belum benar, atau masuk spam | Dasbor Mailjet; periksa folder spam |
| Kode WhatsApp tidak sampai | Sesi lepas atau nomor terblokir | `/admin/whatsapp`; jalur darurat `user:verify` |
| Aplikasi mati mendadak | Memori habis, atau disk penuh | `free -h`, `df -h`, `docker stats`, `dmesg \| grep -i oom` |
| Postgres berhenti menerima tulisan | Disk penuh | `df -h`; periksa ukuran log dan direktori unggahan |
| Halaman dalam menghasilkan 404 saat disegarkan | Fallback SPA tidak aktif | Perutean statis di backend |
| Hasil pencarian menampilkan kapasitas basi | Cloudflare meng-cache `/api/*` | Cache Rules; pastikan bypass |

Untuk diagnosis, setiap baris log memuat pengenal permintaan. Bila penguji melaporkan
sebuah kegagalan, pengenal itu merangkai seluruh kejadian satu permintaan:

```bash
docker compose logs backend | grep '<request_id>'
```

---

## H. Sebelum Penjurian

```text
[ ] Cadangan manual: /opt/devotion/cadangan.sh dijalankan, hasilnya disalin keluar VPS
[ ] Snapshot VPS terbaru diambil
[ ] Sesi WhatsApp tersambung; /admin/whatsapp menunjukkan tersambung
[ ] Satu email uji sampai ke kotak masuk, bukan spam
[ ] /api/health menunjukkan seluruh ketergantungan sehat
[ ] seed:test-data dijalankan; data alur bertenggat sudah pada keadaan yang benar
[ ] Pencarian dibuka sekali agar tidak ada lambat pertama saat demo
[ ] free -h dan df -h diperiksa: memori dan disk masih lega
[ ] Jumlah layanan di docker-compose.yml masih dua
[ ] docs/ lengkap: setup-vps, menjalankan, pengujian, skenario-uji-manual,
    temuan-penguji, layanan-luar, dependencies, utang-teknis
[ ] Tidak ada kredensial sungguhan di repository; .env hanya ada di server
```

---

## Referensi

| Butuh apa | Ada di mana |
|-----------|-------------|
| Perilaku produk dan requirement | `spec.md` |
| Prinsip dan gerbang mutu | `docs/memory/constitution.md` |
| Konteks teknis dan pemeriksaan konstitusi | `plan.md` |
| Alasan keputusan teknis | `research.md` |
| Tabel, constraint, indeks | `data-model.md` |
| Endpoint, skema, kode galat | `contracts/openapi.yaml` |
| Peta endpoint ke requirement | `contracts/README.md` |