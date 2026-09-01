import { useEffect } from "react";
import { useLocation } from "react-router-dom";

const titles: Record<string, string> = {
    "/": "Home",
    "/tentang": "Tentang Kami",
    "/bantuan": "Bantuan",
    "/syarat-ketentuan": "Syarat & Ketentuan",
    "/kebijakan-privasi": "Kebijakan Privasi",

    "/auth/login": "Masuk",
    "/auth/register": "Daftar",
    "/auth/forgot-password": "Lupa Password",
    "/auth/reset-password": "Reset Password",
    "/auth/verify-email": "Verifikasi Email",
    "/auth/verify-phone": "Verifikasi Nomor Telepon",

    "/profile/me": "Profil Saya",

    "/verification": "Verifikasi Akun",

    "/notifications": "Notifikasi",
    "/notifications/preferences": "Preferensi Notifikasi",

    "/orders": "Pesanan",

    "/listing": "Listing Jasa",
    "/listing/calendar": "Kalender Listing",

    "/requests/incoming": "Permintaan Masuk",

    "/search": "Cari Jasa",
    "/quota-requests/new": "Buat Permintaan Kuota",
    "/quota-requests": "Permintaan Saya",

    "/admin": "Dashboard Admin",
    "/admin/verification": "Verifikasi Pengguna",
    "/admin/master/items": "Master Item",
    "/admin/proposals": "Proposal",
    "/admin/late-orders": "Pesanan Terlambat",
    "/admin/disputes": "Sengketa",
    "/admin/reviews": "Moderasi Review",
    "/admin/whatsapp": "WhatsApp",
    "/admin/system": "Pengaturan Sistem",
};

function PageTitle() {
    const location = useLocation();

    useEffect(() => {
        const pathname = location.pathname;
        let title = titles[pathname];

        if (!title) {
            if (pathname.startsWith("/profile/")) {
                title = "Profil Pengguna";
            } else if (pathname.startsWith("/orders/")) {
                title = "Detail Pesanan";
            } else if (pathname.startsWith("/requests/incoming/")) {
                title = "Detail Permintaan Masuk";
            } else if (pathname.startsWith("/quota-requests/")) {
                title = "Detail Permintaan";
            } else if (pathname.startsWith("/admin/orders/")) {
                title = "Detail Pesanan Admin";
            }
        }

        document.title = `${title || "Halaman Tidak Ditemukan"} | Devotion`;
    }, [location.pathname]);

    return null;
}

export default PageTitle;
