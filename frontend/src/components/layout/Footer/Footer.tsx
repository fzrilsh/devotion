import { mainNavigation } from "@data/navigation";
import { Link } from "react-router-dom";

export default function Footer() {
    return (
        <footer className="bg-linear-90 from-deep-navy-800 via-deep-navy-500 to-deep-navy-800 border-t border-deep-navy-800 py-10">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row justify-between gap-6">
                <div>
                    <div className="flex items-center gap-2 mb-2">
                        <span className="font-extrabold text-white text-xl">
                            Devo<span className="text-industrial-orange-500">tion</span>
                        </span>
                    </div>
                    <p className="text-slate-400 max-w-xs">Delegate your overload production.</p>
                </div>

                <div className="flex gap-8 text-sm text-slate-400">
                    <div className="flex flex-col gap-2">
                        {mainNavigation.map((l) => (
                            <Link key={l.label} to={l.href} className="hover:text-industrial-orange-500 transition-colors">
                                {l.label}
                            </Link>
                        ))}
                    </div>
                    <div className="flex flex-col gap-2">
                        <Link to="/about" className="hover:text-industrial-orange-500 transition-colors">
                            Tentang Devotion
                        </Link>
                        <Link to="/help" className="hover:text-industrial-orange-500 transition-colors">
                            Bantuan
                        </Link>
                        <Link to="/privacy" className="hover:text-industrial-orange-500 transition-colors">
                            Kebijakan Privasi
                        </Link>
                    </div>
                </div>
            </div>

            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-8 pt-6 border-t border-deep-navy-950">
                <p className="text-slate-300 text-sm text-center">&copy; {new Date().getFullYear()} Devotion. Hak cipta dilindungi undang-undang.</p>
            </div>
        </footer>
    );
}
