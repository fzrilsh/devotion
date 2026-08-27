import { useAuth, useLogout } from "@hooks/useAuth";
import { useProfile } from "@hooks/useProfile";
import { useUnreadCount } from "@hooks/useNotifications";
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

function NavItemLink({ item, onNavigate }: { item: DashboardNavItem; onNavigate: () => void }) {
    const Icon = item.icon;

    return (
        <NavLink
            to={item.to}
            end={item.end}
            onClick={onNavigate}
            className={({ isActive }) =>
                cn(
                    "group flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition-all duration-200",
                    isActive ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "text-slate-600 hover:bg-slate-50 hover:text-industrial-blue-600",
                )
            }
        >
            {({ isActive }) => (
                <>
                    <Icon aria-hidden className={cn("size-5 shrink-0 transition-colors", isActive ? "text-industrial-blue-600" : "text-slate-400 group-hover:text-industrial-blue-600")} />
                    <span>{item.label}</span>
                </>
            )}
        </NavLink>
    );
}

function NavItemWithChildren({ item, onNavigate }: { item: DashboardNavItem; onNavigate: () => void }) {
    const location = useLocation();
    const [isOpen, setIsOpen] = useState(() => location.pathname.startsWith(`${item.to}/`));
    const Icon = item.icon;
    const isActive = location.pathname === item.to || location.pathname.startsWith(`${item.to}/`);

    return (
        <div className="w-full">
            <button
                type="button"
                onClick={() => setIsOpen((prev) => !prev)}
                aria-expanded={isOpen}
                className={cn(
                    "group flex w-full cursor-pointer items-center justify-between rounded-xl px-3 py-2.5 text-sm font-semibold transition-all duration-200",
                    isActive ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "text-slate-600 hover:bg-slate-50 hover:text-industrial-blue-600",
                )}
            >
                <span className="flex items-center gap-3">
                    <Icon aria-hidden className={cn("size-5 shrink-0 transition-colors", isActive ? "text-industrial-blue-600" : "text-slate-400 group-hover:text-industrial-blue-600")} />
                    {item.label}
                </span>

                <LuChevronDown aria-hidden className={cn("size-4 transition-transform", isOpen ? "rotate-180 text-industrial-blue-600" : "text-slate-400")} />
            </button>

            {isOpen ? (
                <div className="ml-8 mt-1 flex flex-col gap-1">
                    {(item.children ?? []).map((child) => {
                        const ChildIcon = child.icon;

                        return (
                            <NavLink
                                key={child.to}
                                to={child.to}
                                end={child.end}
                                onClick={onNavigate}
                                className={({ isActive }) =>
                                    cn("flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors", isActive ? "bg-slate-100 font-semibold text-slate-900" : "text-slate-600 hover:bg-slate-100")
                                }
                            >
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

    const displayName = profile?.business_name || user?.email || "Pengguna";
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
            <Link to="/" className="flex items-center gap-2.5 px-2" onClick={closeSidebar}>
                <span className="grid size-9 place-items-center rounded-xl bg-industrial-blue-500 text-sm font-extrabold text-white">D</span>
                <div>
                    <p className="text-sm font-bold tracking-tight text-slate-900">Devotion</p>
                    <p className="text-xs text-slate-400">{title}</p>
                </div>
            </Link>

            <nav aria-label={`Navigasi ${title}`} className="mt-8 flex flex-1 flex-col gap-1 overflow-y-auto">
                {navItems.map((item) => (item.children && item.children.length > 0 ? <NavItemWithChildren key={item.to} item={item} onNavigate={closeSidebar} /> : <NavItemLink key={item.to} item={item} onNavigate={closeSidebar} />))}
            </nav>

            <div className="mt-4 border-t border-slate-200 pt-4">
                <Link to="/profile/me" onClick={closeSidebar} className="group flex items-center gap-3 rounded-xl border border-slate-200 bg-slate-50/60 p-3 transition-all duration-200 hover:border-industrial-blue-500/30 hover:bg-industrial-blue-500/5">
                    <span className="grid size-10 shrink-0 place-items-center rounded-full bg-linear-to-br from-industrial-blue-500 to-deep-navy-500 text-sm font-extrabold text-white">{initials || "D"}</span>

                    <span className="min-w-0 flex-1 text-left">
                        <span className="block truncate text-sm font-bold text-slate-800 transition-colors group-hover:text-industrial-blue-600">{displayName}</span>
                        <span className="block truncate text-xs text-slate-400">{user?.email ?? ""}</span>
                    </span>

                    {profile?.identity_verified ? <LuShieldCheck aria-label="Identitas terverifikasi" className="size-5 shrink-0 text-industrial-blue-500" /> : null}
                </Link>

                <button
                    type="button"
                    onClick={handleLogout}
                    disabled={logoutMutation.isPending}
                    className="group mt-2 flex w-full cursor-pointer items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold text-slate-600 transition-all duration-200 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    <LuLogOut aria-hidden className="size-5 text-slate-400 transition-colors group-hover:text-red-600" />
                    <span>{logoutMutation.isPending ? "Memproses..." : "Keluar"}</span>
                </button>
            </div>
        </>
    );

    return (
        <div className="flex min-h-screen bg-slate-50">
            {/* Sidebar desktop */}
            <aside className="fixed inset-y-0 left-0 z-30 hidden w-64 flex-col border-r border-slate-200 bg-white/80 p-5 backdrop-blur-xl lg:flex">{sidebarContent}</aside>

            {/* Sidebar mobile */}
            {sidebarOpen ? (
                <div className="fixed inset-0 z-40 lg:hidden">
                    <div className="absolute inset-0 bg-deep-navy-800/50" onClick={closeSidebar} aria-hidden />

                    <aside className="absolute inset-y-0 left-0 flex w-64 flex-col bg-white p-5 shadow-xl">
                        <button type="button" onClick={closeSidebar} className="absolute right-4 top-5 cursor-pointer text-slate-400 transition-colors hover:text-slate-600" aria-label="Tutup menu navigasi">
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

                            {unreadCount > 0 ? (
                                <span className="absolute -right-1.5 -top-1.5 grid min-w-5 place-items-center rounded-full bg-red-500 px-1 text-[10px] font-bold leading-5 text-white">{unreadCount > 99 ? "99+" : unreadCount}</span>
                            ) : null}
                        </Link>

                        {/* <span className="hidden h-8 w-px bg-slate-200 sm:block" aria-hidden />

                        <Link to="/profile/me" className="group hidden items-center gap-2.5 sm:flex">
                            <span className="grid size-9 shrink-0 place-items-center rounded-full bg-linear-to-br from-industrial-blue-500 to-deep-navy-500 text-xs font-extrabold text-white">{initials || "D"}</span>

                            <span className="text-left">
                                <span className="block max-w-40 truncate text-sm font-semibold text-slate-800 transition-colors group-hover:text-industrial-blue-600">{displayName}</span>
                                <span className="block text-xs text-slate-400">{title}</span>
                            </span>
                        </Link> */}
                    </div>
                </header>

                <main className="flex-1 px-5 py-8 sm:px-8">
                    <Outlet />
                </main>
            </div>
        </div>
    );
}
