# Setup VPS

Ringkasan penyiapan server dari fresh install. Prosedur lengkap dengan perintah
ada di `docs/001-capacity-exchange-marketplace/quickstart.md` bagian A dan B;
dokumen ini merangkum urutannya agar mudah dirujuk.

## A. Prasyarat

- VPS Linux 2GB RAM, 50GB disk, akses root awal.
- Domain dengan nameserver diarahkan ke Cloudflare.
- Akun Cloudflare (domain aktif), Mailjet (kunci API dan secret), Sentry (DSN).
- Nomor WhatsApp khusus, bukan nomor pribadi, ponselnya tersedia untuk memindai QR.
- GitHub dengan izin menulis ke GitHub Container Registry.
- Mesin lokal dengan Docker, Go 1.25+, Node 20+ untuk pengembangan.

Sengaja tidak dipakai: payment gateway, object storage eksternal, reverse proxy
sebagai proses tersendiri, mail server sendiri.

## B. Runbook VPS

Enam belas langkah berurutan; jangan melompat.

1. **Masuk pertama dan pengguna non-root.** Buat user `devotion`, salin kunci
   SSH, uji dari terminal baru, lalu matikan login kata sandi dan root.
2. **Pembaruan sistem dan zona waktu.** `apt upgrade`, set timezone
   `Asia/Jakarta`. Perhitungan batas minggu tetap eksplisit di kode (Prinsip V).
3. **Swap 2GB.** Agar lonjakan sesaat tidak membuat kernel membunuh Postgres.
4. **Firewall, dengan gerbang verifikasi.** `ufw` default deny incoming, izinkan
   22, dan 443 hanya dari rentang Cloudflare yang diambil dari sumber resmi. Port
   80 tidak dibuka. Rentang yang diverifikasi dipatok ke `docs/cloudflare-ips.md`
   dan konstanta Go.

   Ini gerbang, bukan saran. Jangan lanjut ke langkah 5 sebelum dua hal terbukti
   dari luar VPS:

   ```bash
   # dari mesin lain, bukan dari dalam VPS
   curl -sS --max-time 5 https://<IP-VPS-LANGSUNG>/api/health   # harus GAGAL / timeout
   curl -sS https://devotion.web.id/api/health                  # harus berhasil lewat Cloudflare
   ```

   Port yang dipublikasikan Docker melewati `ufw`. Karena itu kepatuhan firewall
   diperiksa dari luar dengan `curl`, bukan disimpulkan dari `ufw status`. Bila
   `curl` ke IP langsung berhasil, port 443 masih terbuka ke publik dan langkah
   berikutnya ditunda sampai itu tertutup.
5. **Docker.** Pasang lewat skrip resmi, tambahkan `devotion` ke grup docker.
6. **Cloudflare dasbor.** DNS `A` proxied, SSL Full (strict), Origin Certificate,
   Authenticated Origin Pulls, Always Use HTTPS, bypass cache untuk `/api/*`.
   Simpan sertifikat di `/opt/devotion/tls` dengan izin ketat.
7. **Struktur direktori dan volume.** `/opt/devotion/{tls,uploads,backups}`.
   `uploads` terpisah dari image agar penerapan baru tidak menghapus unggahan.
8. **Variabel lingkungan.** Ambil `.env.example`, salin ke `.env`, isi nilainya,
   `chmod 600`. Nilai tidak ada di dokumen mana pun. `.env` tidak pernah di-commit.
   `UPLOAD_PATH` diisi `/opt/devotion/uploads` agar cocok dengan bind di compose.
9. **Konfigurasi email di Cloudflare DNS.** SPF, DKIM, DMARC `p=none`. Lakukan
   sedini mungkin; reputasi domain baru butuh waktu dan kegagalan email senyap.
10. **Pasang `docker-compose.yml`.** Salin `docker-compose.yml` dari repo ke
    `/opt/devotion/`. Tepat dua layanan, `postgres` dan `backend`. Port Postgres
    hanya ke `127.0.0.1`, unggahan mengikat `${UPLOAD_PATH}`. Verifikasi jumlah
    layanan sebelum lanjut:

    ```bash
    cd /opt/devotion
    docker compose config --services | wc -l    # harus 2
    ```
11. **Menyalakan dan memverifikasi migrasi.**

    ```bash
    echo "$GITHUB_TOKEN" | docker login ghcr.io -u <username> --password-stdin
    docker compose pull
    docker compose up -d
    docker compose logs -f backend
    ```

    Di log harus terlihat berurutan: migrasi berjalan sampai versi terakhir,
    koneksi basis data berhasil, penjadwal menyala, server mendengarkan di 443.
    Migrasi jalan otomatis saat startup dengan advisory lock, jadi aman meski dua
    kontainer sempat hidup bersamaan.
12. **Mengisi data acuan dan membuat admin.**

    ```bash
    docker compose exec backend /devotion seed:regions
    docker compose exec backend /devotion seed:master-data
    docker compose exec backend /devotion admin:create
    ```

    `seed:test-data` **tidak** dijalankan di produksi; perintah itu menolak jalan
    saat `APP_ENV=production`. Kata sandi admin di sini tidak boleh sama dengan
    kredensial akun uji di `quickstart.md` §E.
13. **Menyambungkan WhatsApp.** Buka `/admin/whatsapp`, pindai QR dengan ponsel
    nomor khusus. Sesi tersimpan agar tidak perlu memindai ulang tiap restart.
14. **Health check dan pemantau uptime.**

    ```bash
    curl -sS https://devotion.web.id/api/health    # seluruh ketergantungan sehat
    ```

    Daftarkan pemantau uptime eksternal ke URL itu agar downtime ketahuan tanpa
    menunggu laporan pengguna.
15. **Cadangan basis data.** Cron di tingkat host, bukan di dalam kontainer,
    karena `pg_dump` harus tetap jalan ketika aplikasi mati, justru saat cadangan
    paling dibutuhkan. Penyimpangan ini tercatat di Complexity Tracking `plan.md`
    dan `docs/utang-teknis.md`.

    ```bash
    cat > /opt/devotion/backup.sh <<'EOF'
    #!/bin/bash
    set -euo pipefail
    cd /opt/devotion
    set -a; . ./.env; set +a
    STAMP=$(date +%Y%m%d-%H%M)
    docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" devotion \
      | gzip > "/opt/devotion/backups/devotion-$STAMP.sql.gz"
    ls -1t /opt/devotion/backups/devotion-*.sql.gz | tail -n +4 | xargs -r rm --
    EOF

    chmod +x /opt/devotion/backup.sh
    /opt/devotion/backup.sh          # uji sekarang, jangan tunggu besok
    ls -lh /opt/devotion/backups/

    ( crontab -l 2>/dev/null; echo "15 2 * * * /opt/devotion/backup.sh >> /opt/devotion/backups/cron.log 2>&1" ) | crontab -
    ```

    Disalurkan langsung ke gzip agar tidak menulis berkas mentah besar, hanya
    tiga salinan terakhir yang disimpan. **Salin keluar VPS**, karena cadangan di
    disk yang sama tidak menolong ketika VPS itu yang bermasalah. Dari mesin lokal:

    ```bash
    rsync -avz devotion@devotion.web.id:/opt/devotion/backups/ ./backups-devotion/
    ```
16. **Snapshot VPS.** Setelah semua langkah di atas terverifikasi, ambil snapshot
    dari panel penyedia. Ini yang paling penting dari seluruh bagian B: satu
    server tanpa cadangan berarti satu kesalahan dapat menghapus seluruh submission.

## C. Urutan eksekusi menuju penjurian

Jalankan berurutan, jangan melompat: runbook langkah 1 sampai 16, lalu seed
wilayah dan daftar baku, buat admin, sambungkan WhatsApp, pasang cron cadangan,
uji cadangan sekali dan salin keluar VPS, daftarkan pemantau uptime, baru ambil
snapshot. Snapshot selalu terakhir, setelah semuanya terverifikasi.

## D. Checklist sebelum penjurian

```text
[ ] Cadangan manual: /opt/devotion/backup.sh dijalankan, hasilnya disalin keluar VPS
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
[ ] curl ke IP VPS langsung gagal; akses lewat domain berhasil
[ ] Tidak ada kredensial sungguhan di repository; .env hanya ada di server
```

Prosedur lengkap dengan seluruh perintah ada di
`docs/001-capacity-exchange-marketplace/quickstart.md` bagian B sampai H; dokumen
ini adalah runbook yang berdiri sendiri untuk dieksekusi langsung di server.
