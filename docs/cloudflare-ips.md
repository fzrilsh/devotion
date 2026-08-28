# Rentang Alamat Cloudflare

Dipakai untuk memutuskan kapan header `CF-Connecting-IP` boleh dipercaya. Header
alamat asal hanya dipercaya bila koneksi datang dari salah satu rentang di bawah;
koneksi di luar rentang dianggap langsung dan header-nya diabaikan.

Diambil dari sumber resmi Cloudflare, bukan dari ingatan atau dari `research.md`
R-01:

- IPv4: https://www.cloudflare.com/ips-v4
- IPv6: https://www.cloudflare.com/ips-v6

Diambil: 2026-08-22 12:14 UTC

Konstanta `RetrievedAt` di `internal/platform/cloudflare` wajib sama dengan tanggal
di atas; sebuah uji menegakkannya, supaya dokumen ini tidak basi pada penyegaran
daftar berikutnya.

## IPv4

```text
173.245.48.0/20
103.21.244.0/22
103.22.200.0/22
103.31.4.0/22
141.101.64.0/18
108.162.192.0/18
190.93.240.0/20
188.114.96.0/20
197.234.240.0/22
198.41.128.0/17
162.158.0.0/15
104.16.0.0/13
104.24.0.0/14
172.64.0.0/13
131.0.72.0/22
```

## IPv6

```text
2400:cb00::/32
2606:4700::/32
2803:f800::/32
2405:b500::/32
2405:8100::/32
2a06:98c0::/29
2c0f:f248::/32
```
