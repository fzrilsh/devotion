import { AnimatePresence, motion } from "motion/react";
import { LuLayoutDashboard, LuLogIn, LuLogOut, LuUserPlus } from "react-icons/lu";
import { useHeader } from "./useHeader";
import HeaderLink from "./HeaderLink";
import { Link } from "react-router-dom";

interface HeaderMobileProps {
    isOpen: boolean;
    navItems: Array<{ label: string; href: string }>;
    isAuthenticated: boolean;
    dashboardPath: string;
    onLogout: () => void;
    logoutPending: boolean;
}

export default function HeaderMobile({ isOpen, navItems, isAuthenticated, dashboardPath, onLogout, logoutPending }: HeaderMobileProps) {
    const { activeLink, handleClick } = useHeader(navItems, 120);

    return (
        <AnimatePresence>
            {isOpen && (
                <>
                    <motion.div initial={{ y: "-300%" }} animate={{ y: "30%" }} exit={{ y: "-300%" }} transition={{ duration: 0.5, ease: "easeInOut" }} className="fixed top-0 right-0 left-0 z-50 block w-full pb-6 bg-white transition-transform duration-300 ease-in-out lg:hidden">
                        <ul className="flex w-full flex-col items-center gap-4 px-8 py-8">
                            {navItems.map((item, idx) => (
                                <HeaderLink key={idx} href={item.href} active={item.href === activeLink} onClick={(e: React.MouseEvent<HTMLAnchorElement>) => handleClick(e, item.href)}>
                                    {item.label}
                                </HeaderLink>
                            ))}
                        </ul>

                        <div className="flex flex-col items-center gap-3 border-t border-slate-100 px-8 pt-6">
                            {isAuthenticated ? (
                                <>
                                    <Link to={dashboardPath} className="inline-flex w-full max-w-xs items-center justify-center gap-2 rounded-xl bg-industrial-blue-500 px-6 py-3 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                                        <LuLayoutDashboard className="size-4" aria-hidden />
                                        Dasbor
                                    </Link>

                                    <button type="button" onClick={onLogout} disabled={logoutPending} className="inline-flex w-full max-w-xs cursor-pointer items-center justify-center gap-2 rounded-xl border border-slate-300 bg-white px-6 py-3 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                                        <LuLogOut className="size-4" aria-hidden />
                                        {logoutPending ? "Keluar..." : "Keluar"}
                                    </button>
                                </>
                            ) : (
                                <>
                                    <Link to="/auth/login" className="inline-flex w-full max-w-xs items-center justify-center gap-2 rounded-xl border border-industrial-blue-500/40 bg-white px-6 py-3 text-sm font-semibold text-industrial-blue-600 transition hover:bg-industrial-blue-500/5">
                                        <LuLogIn className="size-4" aria-hidden />
                                        Masuk
                                    </Link>

                                    <Link to="/auth/register" className="inline-flex w-full max-w-xs items-center justify-center gap-2 rounded-xl bg-industrial-blue-500 px-6 py-3 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                                        <LuUserPlus className="size-4" aria-hidden />
                                        Daftar
                                    </Link>
                                </>
                            )}
                        </div>
                    </motion.div>
                </>
            )}
        </AnimatePresence>
    );
}
