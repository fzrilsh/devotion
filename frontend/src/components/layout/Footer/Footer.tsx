import { mainNavigation } from "@data/navigation";
import { Link } from "react-router-dom";

const platformLinks = [
    { label: "Cari Subkontraktor", to: "/search" },
    { label: "Daftar Sebagai Mitra", to: "/auth/register" },
    { label: "Masuk", to: "/auth/login" },
];

const companyLinks = [
    { label: "Tentang Devotion", to: "/tentang" },
    { label: "Bantuan", to: "/bantuan" },
];

const legalLinks = [
    { label: "Syarat & Ketentuan", to: "/syarat-ketentuan" },
    { label: "Kebijakan Privasi", to: "/kebijakan-privasi" },
];

function FooterLinkGroup({ title, links }: { title: string; links: { label: string; to: string }[] }) {
    return (
        <div>
            <h3 className="text-sm font-bold uppercase tracking-wider text-white">{title}</h3>

            <ul className="mt-4 space-y-2.5">
                {links.map((link) => (
                    <li key={link.label}>
                        <Link to={link.to} className="text-sm text-slate-400 transition-colors hover:text-industrial-blue-400">
                            {link.label}
                        </Link>
                    </li>
                ))}
            </ul>
        </div>
    );
}

export default function Footer() {
    return (
        <footer className="border-t border-deep-navy-950 bg-linear-to-b from-deep-navy-500 to-deep-navy-800">
            <div className="mx-auto grid w-full max-w-7xl grid-cols-1 gap-10 px-4 py-14 sm:grid-cols-2 sm:px-6 lg:grid-cols-4 lg:px-8">
                                <div>
                    <Link to="/" className="flex items-center gap-2.5">
                        <span className="text-lg font-bold tracking-tight text-white">Devotion</span>
                    </Link>

                    <p className="mt-4 max-w-xs text-sm leading-6 text-slate-400">Devotion menghubungkan UMKM konveksi dengan subkontraktor terpercaya yang siap memenuhi kebutuhan produksi Anda. Temukan, bandingkan, dan sepakati dalam satu platform.</p>
                </div>

                <div>
                    <h3 className="text-sm font-bold uppercase tracking-wider text-white">Jelajah</h3>

                    <ul className="mt-4 space-y-2.5">
                        {mainNavigation.map((item) => (
                            <li key={item.label}>
                                <a href={item.href} className="text-sm text-slate-400 transition-colors hover:text-industrial-blue-400">
                                    {item.label}
                                </a>
                            </li>
                        ))}
                    </ul>
                </div>

                <FooterLinkGroup title="Platform" links={platformLinks} />
                <FooterLinkGroup title="Perusahaan" links={companyLinks} />
            </div>

            <div className="border-t border-white/10">
                <div className="mx-auto flex w-full max-w-7xl flex-col items-center justify-between gap-4 px-4 py-6 sm:flex-row sm:px-6 lg:px-8">
                    <p className="text-xs text-slate-500">&copy; {new Date().getFullYear()} Devotion. Hak cipta dilindungi undang-undang.</p>

                    <ul className="flex items-center gap-6">
                        {legalLinks.map((link) => (
                            <li key={link.label}>
                                <Link to={link.to} className="text-xs text-slate-400 transition-colors hover:text-industrial-blue-400">
                                    {link.label}
                                </Link>
                            </li>
                        ))}
                    </ul>
                </div>
            </div>
        </footer>
    );
}
