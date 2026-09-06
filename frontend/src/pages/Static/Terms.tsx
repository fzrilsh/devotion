import InfoPage from "./InfoPage";

export default function Terms() {
    return (
        <InfoPage
            title="Syarat & Ketentuan"
            intro="Ketentuan penggunaan platform Devotion sebagai marketplace pertukaran kapasitas produksi antar UMKM konveksi."
            sections={[
                {
                    heading: "Tentang Layanan",
                    body: [
                        "Devotion adalah platform yang mempertemukan UMKM konveksi yang membutuhkan kapasitas produksi tambahan dengan UMKM yang memiliki kapasitas menganggur. Devotion bertindak sebagai perantara pencarian dan pencatatan kesepakatan, bukan sebagai pihak dalam transaksi produksi.",
                        "Dengan mendaftar dan menggunakan platform ini, Anda menyetujui syarat dan ketentuan yang berlaku.",
                    ],
                },
                {
                    heading: "Akun dan Verifikasi",
                    body: [
                        "Anda wajib memberikan data yang benar saat mendaftar dan menjaga kerahasiaan kata sandi akun Anda.",
                        "Verifikasi email dan nomor WhatsApp diperlukan sebelum Anda dapat mengisi profil usaha, membuat listing kapasitas, maupun mengirim request kuota. Verifikasi identitas usaha dilakukan oleh admin berdasarkan dokumen yang Anda unggah.",
                    ],
                },
                {
                    heading: "Kesepakatan dan Pembayaran",
                    body: [
                        "Kesepakatan produksi terjadi langsung antara pemberi order dan pemilik kapasitas. Devotion tidak menahan, menyalurkan, maupun memproses dana pihak mana pun.",
                        "Pembayaran dilakukan langsung antar pihak di luar platform. Devotion hanya mencatat pernyataan pembayaran dari masing-masing pihak sebagai bukti riwayat pesanan.",
                    ],
                },
                {
                    heading: "Perilaku yang Dilarang",
                    body: [
                        "Anda dilarang menyalahgunakan platform untuk penipuan, memanipulasi ulasan, mengirim request yang tidak serius secara berulang, atau mengunggah dokumen palsu.",
                        "Pelanggaran dapat berakibat pada penangguhan atau penutupan akun oleh admin.",
                    ],
                },
                {
                    heading: "Pembatalan dan Sengketa",
                    body: [
                        "Pembatalan pesanan sebelum produksi dimulai mengembalikan alokasi kapasitas secara otomatis dan tercatat pada tingkat penyelesaian pihak yang membatalkan.",
                        "Bila terjadi sengketa, kedua pihak dapat mengajukan mediasi dan admin platform akan meninjau bukti dari kedua sisi sebelum mengambil keputusan.",
                    ],
                },
                {
                    heading: "Batasan Tanggung Jawab",
                    body: [
                        "Devotion tidak bertanggung jawab atas kualitas hasil produksi, keterlambatan, maupun kerugian yang timbul dari kesepakatan antar pengguna. Keputusan memilih mitra sepenuhnya berada di tangan Anda berdasarkan informasi profil, reputasi, dan riwayat yang tersedia.",
                    ],
                },
            ]}
        />
    );
}
