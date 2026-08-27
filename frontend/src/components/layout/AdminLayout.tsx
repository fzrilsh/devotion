import DashboardLayout, { type DashboardNavItem } from "@components/layout/DashboardLayout";
import { LuClock, LuFileBox, LuLayoutDashboard, LuMessageSquare, LuMessagesSquare, LuPhone, LuShieldCheck, LuStar } from "react-icons/lu";

const navItems: DashboardNavItem[] = [
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
];

export default function AdminLayout() {
    return <DashboardLayout title="Panel Admin" navItems={navItems} />;
}
