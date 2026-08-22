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
- Mesin lokal dengan Docker, Go 1.22+, Node 20+ untuk pengembangan.

Sengaja tidak dipakai: payment gateway, object storage eksternal, reverse proxy
sebagai proses tersendiri, mail server sendiri.

## B. Runbook VPS

Enam belas langkah berurutan; jangan melompat.

1. **Masuk pertama dan pengguna non-root.** Buat user `devotion`, salin kunci
   SSH, uji dari terminal baru, lalu matikan login kata sandi dan root.
2. **Pembaruan sistem dan zona waktu.** `apt upgrade`, set timezone
   `Asia/Jakarta`. Perhitungan batas minggu tetap eksplisit di kode (Prinsip V).
3. **Swap 2GB.** Agar lonjakan sesaat tidak membuat kernel membunuh Postgres.
4. **Firewall.** `ufw` default deny incoming, izinkan 22, dan 443 hanya dari
   rentang Cloudflare yang diambil dari sumber resmi. Port 80 tidak dibuka.
   Rentang yang diverifikasi dipatok ke `docs/cloudflare-ips.md` dan konstanta Go.
5. **Docker.** Pasang lewat skrip resmi, tambahkan `devotion` ke grup docker.
6. **Cloudflare dasbor.** DNS `A` proxied, SSL Full (strict), Origin Certificate,
   Authenticated Origin Pulls, Always Use HTTPS, bypass cache untuk `/api/*`.
   Simpan sertifikat di `/opt/devotion/tls` dengan izin ketat.
7. **Struktur direktori dan volume.** `/opt/devotion/{tls,unggahan,cadangan}`.
   `unggahan` terpisah dari image agar penerapan baru tidak menghapus unggahan.
8. **Variabel lingkungan.** Ambil `.env.example`, salin ke `.env`, isi nilainya,
   `chmod 600`. Nilai tidak ada di dokumen mana pun. `.env` tidak pernah di-commit.
9. **Konfigurasi email di Cloudflare DNS.** SPF, DKIM, DMARC `p=none`. Lakukan
   sedini mungkin; reputasi domain baru butuh waktu dan kegagalan email senyap.

Langkah 10 dan seterusnya (compose, deploy, cadangan) ada di quickstart.md.
