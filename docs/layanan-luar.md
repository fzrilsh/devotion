# Layanan Luar

Lima layanan berjalan di luar server dan tidak dihitung sebagai proses runtime
(Gate I tetap dua layanan). Masing-masing dicatat di sini beserta akibat bila
mati, karena kegagalannya tidak akan tampak sebagai container yang berhenti.

## Cloudflare

Proksi tepi, TLS, dan firewall. Origin hanya menerima koneksi dari rentang
alamat Cloudflare pada port 443 (mode Full strict, Authenticated Origin Pulls).

**Bila mati:** aplikasi tidak terjangkau dari internet meski container sehat,
karena port 443 origin tertutup untuk semua alamat di luar Cloudflare. Tidak ada
jalur langsung ke origin yang disengaja.

## Mailjet

Pengiriman email lewat SMTP (`net/smtp`), untuk kode verifikasi dan pemulihan
akun. Email adalah kanal kedua di samping WhatsApp.

**Bila mati:** email verifikasi dan pemulihan tidak terkirim. Pendaftaran dan
pemulihan yang bergantung pada email terhenti; verifikasi lewat WhatsApp masih
jalan.

## Sentry

Pelaporan galat backend. Payload dibersihkan lewat allowlist field sebelum
dikirim (tanpa kata sandi, token, nomor telepon, atau dokumen identitas).

**Bila mati:** galat tidak terlaporkan ke Sentry; aplikasi tetap berjalan dan
tetap mencatat lewat `log/slog`. Bukan ketergantungan jalur kritis.

## Pemantau uptime

Layanan eksternal yang menge-ping `/health` secara berkala.

**Bila mati:** kehilangan pemberitahuan dini saat aplikasi turun. Tidak
memengaruhi jalannya aplikasi.

## wilayah.id

Sumber data wilayah (provinsi dan kota) untuk `seed:regions --refresh`. Salinan
JSON di-commit di `docs/master-data/regions.json` (R-02), sehingga seed default
tidak menyentuh jaringan.

**Bila mati:** hanya `--refresh` yang gagal. Seed dari salinan JSON ter-commit
tetap jalan, jadi ini bukan ketergantungan runtime.
