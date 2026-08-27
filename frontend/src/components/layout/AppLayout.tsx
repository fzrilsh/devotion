import DashboardLayout, { type DashboardNavItem } from "@components/layout/DashboardLayout";
import { useAuth } from "@hooks/useAuth";
import { LuClipboardList, LuFileInput, LuFileOutput, LuLayoutDashboard, LuSearch } from "react-icons/lu";

export default function AppLayout() {
    const { user } = useAuth();

    const navItems: DashboardNavItem[] = [];

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

    return <DashboardLayout title="Dasbor" navItems={navItems} />;
}
