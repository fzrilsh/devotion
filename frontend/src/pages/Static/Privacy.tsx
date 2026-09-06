import InfoPage from "./InfoPage";

export default function Privacy() {
    return (
        <InfoPage
            title="Kebijakan Privasi"
            intro="Bagaimana Devotion mengumpulkan, menggunakan, dan melindungi data Anda."
            sections={[
                {
                    heading: "Data yang Kami Kumpulkan",
                    body: [
                        "Data akun: alamat email, nomor WhatsApp, dan kata sandi yang disimpan dalam bentuk hash.",
                        "Data usaha: nama usaha, deskripsi, lokasi, jenis produk, mesin, dan kapasitas produksi yang Anda isi pada profil dan listing.",
                        "Data verifikasi: dokumen identitas usaha yang Anda unggah untuk keperluan peninjauan admin.",
                        "Data aktivitas: request kuota, penawaran, pesanan, ulasan, dan riwayat notifikasi selama Anda menggunakan platform.",
                    ],
                },
                {
                    heading: "Penggunaan Data",
                    body: [
                        "Data usaha dan reputasi ditampilkan kepada pengguna lain untuk keperluan pencarian dan penilaian mitra.",
                        "Email dan nomor WhatsApp digunakan untuk verifikasi akun dan pengiriman notifikasi terkait aktivitas pesanan Anda.",
                        "Dokumen identitas hanya dapat diakses oleh Anda dan admin untuk keperluan verifikasi, tidak ditampilkan kepada pengguna lain.",
                    ],
                },
                {
                    heading: "Perlindungan Data",
                    body: [
                        "Kata sandi disimpan dengan hashing bcrypt dan tidak pernah disimpan dalam bentuk asli.",
                        "Sesi masuk memakai cookie httpOnly sehingga tidak dapat dibaca oleh skrip di peramban.",
                        "Metadata lokasi pada gambar yang Anda unggah dihapus secara otomatis sebelum disimpan.",
                    ],
                },
                {
                    heading: "Berbagi Data dengan Pihak Ketiga",
                    body: [
                        "Kami tidak menjual data Anda kepada pihak mana pun. Data hanya dibagikan kepada penyedia layanan pengiriman pesan (email dan WhatsApp) sebatas yang diperlukan untuk mengirim notifikasi.",
                    ],
                },
                {
                    heading: "Hak Anda",
                    body: [
                        "Anda dapat memperbarui data profil dan usaha kapan saja melalui halaman profil.",
                        "Untuk pertanyaan atau permintaan penghapusan akun, hubungi kami melalui halaman Bantuan.",
                    ],
                },
            ]}
        />
    );
}
