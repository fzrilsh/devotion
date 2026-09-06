import DashboardLayout, { type DashboardNavItem } from "@components/layout/DashboardLayout";
import { useAuth } from "@hooks/useAuth";
import { LuActivity, LuClipboardList, LuClock, LuFileBox, LuFileInput, LuFileOutput, LuLayoutDashboard, LuMessageSquare, LuMessagesSquare, LuPhone, LuSearch, LuShieldCheck, LuStar } from "react-icons/lu";

const adminNavItems: DashboardNavItem[] = [
    { to: "/admin", label: "Dasbor", icon: LuLayoutDashboard, end: true },
    { to: "/admin/verification", label: "Antrean Verifikasi", icon: LuShieldCheck },
    {
        to: "/admin/master",
        label: "Data Master",
        icon: LuFileBox,
        children: [
            { to: "/admin/master/items", label: "Item Baku", icon: LuFileBox },
            { to: "/admin/proposals", label: "Usulan Item", icon: LuMessageSquare },
        ],
    },
    {
        to: "/admin/orders",
        label: "Pesanan",
        icon: LuClock,
        children: [
            { to: "/admin/late-orders", label: "Pesanan Terlambat", icon: LuClock },
            { to: "/admin/disputes", label: "Sengketa", icon: LuMessagesSquare },
        ],
    },
    { to: "/admin/reviews", label: "Moderasi Ulasan", icon: LuStar },
    { to: "/admin/whatsapp", label: "WhatsApp", icon: LuPhone },
    { to: "/admin/system", label: "Status Sistem", icon: LuActivity },
];

export default function AppLayout() {
    const { user } = useAuth();

    const navItems: DashboardNavItem[] = [];

    if (user?.is_admin) {
        navItems.push(...adminNavItems);
    }

    if (user?.roles?.subcontractor) {
        navItems.push(
            { to: "/listing", label: "Listing Kapasitas", icon: LuLayoutDashboard, end: true },
            { to: "/requests/incoming", label: "Request Masuk", icon: LuFileInput },
        );
    }

    if (user?.roles?.buyer) {
        navItems.push(
            { to: "/search", label: "Cari Subkontraktor", icon: LuSearch, end: true },
            { to: "/quota-requests", label: "Request Terkirim", icon: LuFileOutput },
        );
    }

    if (user?.roles?.subcontractor || user?.roles?.buyer) {
        navItems.push({ to: "/orders", label: "Pesanan", icon: LuClipboardList });
    }

    return <DashboardLayout title={user?.is_admin ? "Panel Admin" : "Dasbor"} navItems={navItems} />;
}
