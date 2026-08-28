import logoColored from "@assets/logo.png";
import logoWhite from "@assets/logo_white.png";
import { useAuth, useLogout } from "@hooks/useAuth";
import { useUnreadCount } from "@hooks/useNotifications";
import { useProfile } from "@hooks/useProfile";
import { cn } from "@lib/utils";
import { useState, type ReactNode } from "react";
import { LuBell, LuChevronDown, LuLogOut, LuMenu, LuShieldCheck, LuX } from "react-icons/lu";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

export type DashboardNavItem = {
    to: string;
    label: string;
    icon: React.ElementType;
    end?: boolean;
    children?: DashboardNavItem[];
};

type DashboardLayoutProps = {
    title: string;
    navItems: DashboardNavItem[];
    header?: ReactNode;
};

type Theme = {
    isAdmin: boolean;
    logo: string;
    logoAlt: string;
    subtitle: string;
    subtitleClass: string;
    navLink: (isActive: boolean) => string;
    navIcon: (isActive: boolean) => string;
    childLink: (isActive: boolean) => string;
    chevron: (isOpen: boolean) => string;
    dividerClass: string;
    profileCardClass: string;
    profileNameClass: string;
    profileEmailClass: string;
    verifiedClass: string;
    logoutClass: string;
    logoutIconClass: string;
    avatarClass: string;
};

const adminTheme: Theme = {
    isAdmin: true,
    logo: logoWhite,
    logoAlt: "Devotion",
    subtitle: "Panel Admin",
    subtitleClass: "text-white/50",
    navLink: (isActive) => cn("group flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition-all duration-200", isActive ? "bg-white/15 text-white shadow-sm" : "text-white/60 hover:bg-white/10 hover:text-white"),
    navIcon: (isActive) => cn("size-5 shrink-0 transition-colors", isActive ? "text-white" : "text-white/40 group-hover:text-white"),
    childLink: (isActive) => cn("flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors", isActive ? "bg-white/15 font-semibold text-white" : "text-white/60 hover:bg-white/10 hover:text-white"),
    chevron: (isOpen) => cn("size-4 transition-transform", isOpen ? "rotate-180 text-white" : "text-white/40"),
    dividerClass: "border-white/15",
    profileCardClass: "border-white/15 bg-white/10 hover:border-white/25 hover:bg-white/15",
    profileNameClass: "text-white",
    profileEmailClass: "text-white/50",
    verifiedClass: "text-white/70",
    logoutClass: "text-white/60 hover:bg-white/10 hover:text-white",
    logoutIconClass: "text-white/40 group-hover:text-white",
    avatarClass: "border border-white/25 bg-white/15 text-white",
};

const userTheme: Theme = {
    isAdmin: false,
    logo: logoColored,
    logoAlt: "Devotion",
    subtitle: "Dasbor",
    subtitleClass: "text-slate-400",
    navLink: (isActive) => cn("group flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition-all duration-200", isActive ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "text-slate-600 hover:bg-slate-50 hover:text-industrial-blue-600"),
    navIcon: (isActive) => cn("size-5 shrink-0 transition-colors", isActive ? "text-industrial-blue-600" : "text-slate-400 group-hover:text-industrial-blue-600"),
    childLink: (isActive) => cn("flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors", isActive ? "bg-slate-100 font-semibold text-slate-900" : "text-slate-600 hover:bg-slate-100"),
    chevron: (isOpen) => cn("size-4 transition-transform", isOpen ? "rotate-180 text-industrial-blue-600" : "text-slate-400"),
    dividerClass: "border-slate-200",
    profileCardClass: "border-slate-200 bg-slate-50/60 hover:border-industrial-blue-500/30 hover:bg-industrial-blue-500/5",
    profileNameClass: "text-slate-800 group-hover:text-industrial-blue-600",
    profileEmailClass: "text-slate-400",
    verifiedClass: "text-industrial-blue-500",
    logoutClass: "text-slate-600 hover:bg-red-50 hover:text-red-600",
    logoutIconClass: "text-slate-400 group-hover:text-red-600",
    avatarClass: "bg-linear-to-br from-industrial-blue-500 to-deep-navy-500 text-white",
};

function NavItemLink({ item, theme, onNavigate }: { item: DashboardNavItem; theme: Theme; onNavigate: () => void }) {
    const Icon = item.icon;

    return (
        <NavLink to={item.to} end={item.end} onClick={onNavigate} className={({ isActive }) => theme.navLink(isActive)}>
            {({ isActive }) => (
                <>
                    <Icon aria-hidden className={theme.navIcon(isActive)} />
                    <span>{item.label}</span>
                </>
            )}
        </NavLink>
    );
}

function NavItemWithChildren({ item, theme, onNavigate }: { item: DashboardNavItem; theme: Theme; onNavigate: () => void }) {
    const location = useLocation();
    const [isOpen, setIsOpen] = useState(() => location.pathname.startsWith(`${item.to}/`));
    const Icon = item.icon;
    const isActive = location.pathname === item.to || location.pathname.startsWith(`${item.to}/`);

    return (
        <div className="w-full">
            <div className={cn("group flex w-full items-center rounded-xl text-sm font-semibold transition-all duration-200", theme.navLink(isActive))}>
                <NavLink to={item.to} end={item.end} onClick={onNavigate} className="flex min-w-0 flex-1 items-center gap-3">
                    <Icon aria-hidden className={theme.navIcon(isActive)} />
                    <span>{item.label}</span>
                </NavLink>

                <button type="button" onClick={() => setIsOpen((prev) => !prev)} aria-expanded={isOpen} aria-label={`${isOpen ? "Tutup" : "Buka"} submenu ${item.label}`} className="cursor-pointer">
                    <LuChevronDown aria-hidden className={theme.chevron(isOpen)} />
                </button>
            </div>

            {isOpen ? (
                <div className="ml-8 mt-1 flex flex-col gap-1">
                    {(item.children ?? []).map((child) => {
                        const ChildIcon = child.icon;

                        return (
                            <NavLink key={child.to} to={child.to} end={child.end} onClick={onNavigate} className={({ isActive }) => theme.childLink(isActive)}>
                                <ChildIcon aria-hidden className="size-4 shrink-0" />
                                {child.label}
                            </NavLink>
                        );
                    })}
                </div>
            ) : null}
        </div>
    );
}

function getInitials(name: string): string {
    return name
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
        .map((word) => word[0]?.toUpperCase() ?? "")
        .join("");
}

export default function DashboardLayout({ title, navItems, header }: DashboardLayoutProps) {
    const navigate = useNavigate();
    const { user } = useAuth();
    const { profile } = useProfile();
    const { unreadCount } = useUnreadCount();
    const logoutMutation = useLogout();

    const [sidebarOpen, setSidebarOpen] = useState(false);

    const theme = user?.is_admin ? adminTheme : userTheme;
    const displayName = user?.is_admin ? "Administrator" : profile?.business_name || user?.email || "Pengguna";
    const initials = getInitials(displayName);

    function closeSidebar() {
        setSidebarOpen(false);
    }

    function handleLogout() {
        logoutMutation.mutate(undefined, {
            onSettled: () => navigate("/auth/login", { replace: true }),
        });
    }

    const sidebarContent = (
        <>
            <div className="flex items-center gap-2.5 px-2 pt-4">
                <Link to="/" className="flex min-w-0 items-center gap-2.5" onClick={closeSidebar}>
                    <img src={theme.logo} alt={theme.logoAlt} className="w-32 shrink-0 object-cover" />
                </Link>
            </div>

            <nav aria-label={`Navigasi ${title}`} className="mt-8 flex flex-1 flex-col gap-1 overflow-y-auto">
                {navItems.map((item) => (item.children && item.children.length > 0 ? <NavItemWithChildren key={item.to} item={item} theme={theme} onNavigate={closeSidebar} /> : <NavItemLink key={item.to} item={item} theme={theme} onNavigate={closeSidebar} />))}
            </nav>

            <div className={cn("mt-4 border-t pt-4", theme.dividerClass)}>
                <Link to="/profile/me" onClick={closeSidebar} className={cn("group flex items-center gap-3 rounded-xl border p-3 transition-all duration-200", theme.profileCardClass)}>
                    <span className={cn("grid size-10 shrink-0 place-items-center rounded-full text-sm font-extrabold", theme.avatarClass)}>{initials || "D"}</span>

                    <span className="min-w-0 flex-1 text-left">
                        <span className={cn("block truncate text-sm font-bold transition-colors", theme.profileNameClass)}>{displayName}</span>
                        <span className={cn("block truncate text-xs", theme.profileEmailClass)}>{user?.email ?? ""}</span>
                    </span>

                    {profile?.identity_verified ? <LuShieldCheck aria-label="Identitas terverifikasi" className={cn("size-5 shrink-0", theme.verifiedClass)} /> : null}
                </Link>

                <button type="button" onClick={handleLogout} disabled={logoutMutation.isPending} className={cn("group mt-2 flex w-full cursor-pointer items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60", theme.logoutClass)}>
                    <LuLogOut aria-hidden className={cn("size-5 transition-colors", theme.logoutIconClass)} />
                    <span>{logoutMutation.isPending ? "Memproses..." : "Keluar"}</span>
                </button>
            </div>
        </>
    );

    return (
        <div className="flex min-h-screen bg-slate-50">
            <aside className="fixed inset-y-0 left-0 z-30 hidden w-72 p-4 lg:flex">
                <div className={cn("flex h-full w-full flex-col rounded-2xl p-4 shadow-sm", theme.isAdmin ? "bg-linear-to-b from-deep-navy-500 to-industrial-blue-500 shadow-deep-navy-500/20" : "border border-slate-200 bg-white")}>{sidebarContent}</div>
            </aside>

            {/* Sidebar mobile */}
            {sidebarOpen ? (
                <div className="fixed inset-0 z-40 lg:hidden">
                    <div className="absolute inset-0 bg-deep-navy-800/50" onClick={closeSidebar} aria-hidden />

                    <aside className={cn("absolute inset-y-0 left-0 flex w-72 flex-col p-5 shadow-xl", theme.isAdmin ? "bg-linear-to-b from-deep-navy-500 to-industrial-blue-500" : "bg-white")}>
                        <button type="button" onClick={closeSidebar} className={cn("absolute right-4 top-5 cursor-pointer transition-colors", theme.isAdmin ? "text-white/50 hover:text-white" : "text-slate-400 hover:text-slate-600")} aria-label="Tutup menu navigasi">
                            <LuX className="size-5" />
                        </button>

                        {sidebarContent}
                    </aside>
                </div>
            ) : null}

            {/* Konten utama */}
            <div className="flex min-h-screen flex-1 flex-col lg:pl-64">
                <header className="sticky top-0 z-20 flex h-16 items-center gap-4 border-b border-slate-200 bg-white/80 px-5 backdrop-blur-xl sm:px-8">
                    <button type="button" onClick={() => setSidebarOpen(true)} className="cursor-pointer text-slate-500 transition-colors hover:text-slate-700 lg:hidden" aria-label="Buka menu navigasi">
                        <LuMenu className="size-6" />
                    </button>

                    <div className="min-w-0 flex-1">{header ?? <h1 className="truncate text-lg font-bold text-slate-900">{title}</h1>}</div>

                    <div className="flex items-center gap-3">
                        <Link
                            to="/notifications"
                            className="relative grid size-10 cursor-pointer place-items-center rounded-xl border border-slate-200 bg-white text-slate-500 transition-all duration-200 hover:border-industrial-blue-500/30 hover:text-industrial-blue-600"
                            aria-label={unreadCount > 0 ? `Notifikasi, ${unreadCount} belum dibaca` : "Notifikasi"}
                        >
                            <LuBell className="size-5" aria-hidden />

                            {unreadCount > 0 ? <span className="absolute -right-1.5 -top-1.5 grid min-w-5 place-items-center rounded-full bg-red-500 px-1 text-[10px] font-bold leading-5 text-white">{unreadCount > 99 ? "99+" : unreadCount}</span> : null}
                        </Link>
                    </div>
                </header>

                <main className="flex-1 px-5 py-8 sm:px-8">
                    <Outlet />
                </main>
            </div>
        </div>
    );
}
