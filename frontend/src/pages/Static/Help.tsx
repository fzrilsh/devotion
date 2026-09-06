import InfoPage from "./InfoPage";

export default function Help() {
    return (
        <InfoPage
            title="Bantuan"
            intro="Jawaban atas pertanyaan yang paling sering diajukan pengguna Devotion."
            sections={[
                {
                    heading: "Bagaimana cara mulai mencari subkontraktor?",
                    body: [
                        "Daftar akun, verifikasi email dan nomor WhatsApp, lalu lengkapi profil usaha Anda. Setelah itu buka halaman Cari Subkontraktor, pilih jenis produk dan jumlah kebutuhan, dan sistem akan menampilkan kandidat yang kapasitasnya sesuai.",
                        "Anda dapat memperluas cakupan pencarian dari kota ke provinsi hingga nasional bila kandidat di wilayah Anda terbatas.",
                    ],
                },
                {
                    heading: "Bagaimana cara menawarkan kapasitas produksi saya?",
                    body: [
                        "Setelah akun terverifikasi, buka halaman Listing Kapasitas dan isi kapasitas mingguan, jenis produk yang dikerjakan, serta mesin yang dimiliki. Tayangkan listing agar muncul di hasil pencarian pemberi order.",
                        "Perbarui kalender kapasitas secara berkala. Listing dengan kalender yang sudah penuh atau kedaluwarsa tidak akan muncul di pencarian.",
                    ],
                },
                {
                    heading: "Bagaimana proses negosiasi berjalan?",
                    body: [
                        "Pemberi order mengirim satu request kuota ke beberapa kandidat sekaligus. Setiap kandidat punya batas 72 jam untuk membalas dengan penawaran harga dan kesiapan.",
                        "Pemberi order dapat memberi penawaran balik atau langsung menerima salah satu penawaran. Saat satu penawaran diterima, kandidat lain pada request yang sama otomatis tidak dilanjutkan.",
                    ],
                },
                {
                    heading: "Apakah pembayaran dilakukan lewat platform?",
                    body: [
                        "Tidak. Pembayaran terjadi langsung antara pemberi order dan pemilik kapasitas di luar platform. Devotion hanya mencatat pernyataan pembayaran dari kedua pihak sebagai riwayat pesanan.",
                    ],
                },
                {
                    heading: "Apa itu tingkat penyelesaian?",
                    body: [
                        "Tingkat penyelesaian adalah persentase pesanan yang selesai dari total pesanan yang pernah disepakati. Pembatalan hanya membebani pihak yang membatalkan.",
                        "Nilai ini baru ditampilkan setelah ada cukup riwayat pesanan, agar angka yang tampil benar-benar mencerminkan rekam jejak.",
                    ],
                },
                {
                    heading: "Pesanan saya bermasalah, apa yang harus dilakukan?",
                    body: [
                        "Ajukan sengketa dari halaman detail pesanan dan sertakan bukti yang Anda miliki. Admin akan meninjau kronologi dan bukti dari kedua pihak sebelum memutuskan hasil mediasi.",
                        "Mengajukan sengketa juga menghentikan konfirmasi penyelesaian otomatis sehingga pesanan tidak ditutup sebelum masalah ditinjau.",
                    ],
                },
            ]}
        />
    );
}
