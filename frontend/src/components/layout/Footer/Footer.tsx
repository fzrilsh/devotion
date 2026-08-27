import { mainNavigation } from "@data/navigation";
import { Link } from "react-router-dom";

const platformLinks = [
    { label: "Cari Subkontraktor", to: "/search" },
    { label: "Daftar Sebagai Mitra", to: "/auth/register" },
    { label: "Masuk", to: "/auth/login" },
];

const companyLinks = [
    { label: "Tentang Devotion", to: "/" },
    { label: "Bantuan", to: "/" },
];

const legalLinks = [
    { label: "Syarat & Ketentuan", to: "/" },
    { label: "Kebijakan Privasi", to: "/" },
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
                {/* Identitas */}
                <div>
                    <Link to="/" className="flex items-center gap-2.5">
                        <span className="grid size-9 place-items-center rounded-xl bg-industrial-blue-500 text-sm font-extrabold text-white">D</span>
                        <span className="text-lg font-bold tracking-tight text-white">Devotion</span>
                    </Link>

                    <p className="mt-4 max-w-xs text-sm leading-6 text-slate-400">Platform capacity exchange yang mempertemukan UMKM konveksi dengan kapasitas produksi menganggur bersama bisnis yang membutuhkan mitra produksi terpercaya.</p>

                    <p className="mt-4 text-xs leading-5 text-slate-500">Pembayaran dilakukan langsung antar pihak. Devotion tidak menahan maupun memproses dana pengguna.</p>
                </div>

                {/* Jelajah (anchor di beranda) */}
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

            {/* Bar bawah */}
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
