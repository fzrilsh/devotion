import InfoPage from "./InfoPage";

export default function About() {
    return (
        <InfoPage
            title="Tentang Devotion"
            intro="Platform pertukaran kapasitas produksi untuk UMKM konveksi Indonesia."
            sections={[
                {
                    heading: "Masalah yang Kami Jawab",
                    body: [
                        "Banyak UMKM konveksi menolak order karena kapasitasnya penuh, sementara di saat yang sama UMKM lain memiliki mesin dan pekerja yang menganggur. Selama ini pencarian subkontraktor hanya mengandalkan relasi personal, sehingga jangkauannya terbatas dan tidak ada cara sistematis untuk menemukan mitra yang cocok.",
                    ],
                },
                {
                    heading: "Apa itu Devotion",
                    body: [
                        "Devotion, yang berarti kesetiaan, adalah marketplace pertukaran kapasitas yang mempertemukan UMKM dengan order berlebih dan UMKM dengan kapasitas menganggur. Pemberi order mencari kandidat berdasarkan jenis produk, jumlah, dan wilayah; pemilik kapasitas menawarkan kapasitasnya lewat listing dan kalender ketersediaan.",
                        "Negosiasi harga berjalan terstruktur lewat request kuota dan penawaran berbatas waktu, lalu pesanan dipantau dari satu tempat sampai selesai.",
                    ],
                },
                {
                    heading: "Prinsip Kami",
                    body: [
                        "Kepercayaan dibangun dari data. Verifikasi identitas, ulasan, dan tingkat penyelesaian ditampilkan apa adanya agar setiap pihak bisa menilai mitra sebelum bersepakat.",
                        "Devotion tidak menyentuh uang pengguna. Pembayaran terjadi langsung antar pihak, dan platform cukup mencatat kesepakatan serta riwayatnya.",
                    ],
                },
                {
                    heading: "Untuk Siapa",
                    body: [
                        "Devotion dibuat untuk UMKM konveksi di seluruh Indonesia, baik yang membutuhkan kapasitas produksi tambahan maupun yang ingin mengisi kapasitas menganggurnya dengan order dari mitra baru.",
                    ],
                },
            ]}
        />
    );
}
