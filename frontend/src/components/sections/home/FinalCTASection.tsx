import { LuArrowRight, LuCalendarCheck, LuClipboardCheck, LuSearch } from "react-icons/lu";
import { Link } from "react-router-dom";

const highlights = [
    { icon: LuSearch, label: "Pencarian subkontraktor terverifikasi" },
    { icon: LuCalendarCheck, label: "Kalender kapasitas mingguan" },
    { icon: LuClipboardCheck, label: "Pesanan tercatat sampai selesai" },
];

export default function FinalCTASection() {
    return (
        <section className="relative mx-auto flex w-full max-w-7xl scroll-mt-24 items-center justify-center px-4 py-16 sm:px-6 lg:px-8">
            <div className="relative w-full overflow-hidden rounded-3xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800">
                {/* Bentuk abstrak */}
                <div aria-hidden className="pointer-events-none absolute inset-0">
                    <div className="absolute -right-32 -top-32 size-112 rounded-full bg-industrial-blue-500/20 blur-3xl" />
                    <div className="absolute -bottom-40 -left-24 size-96 rounded-full bg-industrial-blue-500/15 blur-3xl" />
                    <div className="absolute right-16 top-1/3 hidden size-40 rounded-3xl border border-white/10 lg:block" />
                    <div className="absolute bottom-24 right-40 hidden size-24 rounded-full border border-white/10 lg:block" />
                    <div className="absolute left-1/3 top-16 size-16 rounded-2xl bg-white/5" />
                </div>

                <div className="relative grid gap-10 px-6 py-12 sm:px-12 lg:grid-cols-2 lg:items-center lg:py-16">
                    <div>
                        <h2 className="text-3xl font-extrabold leading-tight tracking-tight text-white sm:text-4xl">Mulai temukan peluang produksi baru hari ini.</h2>

                        <p className="mt-4 max-w-md text-sm leading-relaxed text-white/70">Gratis untuk mendaftar. Tidak perlu komitmen di muka. Mulai eksplorasi kapasitas yang tersedia atau pasang profil usaha Anda.</p>

                        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
                            <Link
                                to="/search"
                                className="inline-flex items-center justify-center gap-2 rounded-xl bg-industrial-blue-500 px-6 py-3.5 text-sm font-bold text-white transition-all duration-200 hover:bg-industrial-blue-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-industrial-blue-400"
                            >
                                <LuSearch className="size-4" aria-hidden />
                                Mulai mencari
                            </Link>

                            <Link
                                to="/auth/register"
                                className="group inline-flex items-center justify-center gap-2 rounded-xl border border-white/25 px-6 py-3.5 text-sm font-bold text-white transition-all duration-200 hover:border-white/50 hover:bg-white/5 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
                            >
                                Buat akun
                                <LuArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-1" aria-hidden />
                            </Link>
                        </div>
                    </div>

                    {/* Panel sorotan fitur */}
                    <div className="lg:justify-self-end">
                        <ul className="flex w-full max-w-sm flex-col gap-3">
                            {highlights.map((item) => {
                                const Icon = item.icon;

                                return (
                                    <li key={item.label} className="flex items-center gap-4 rounded-2xl border border-white/10 bg-white/5 px-5 py-4 backdrop-blur-sm">
                                        <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/20 text-white">
                                            <Icon className="size-5" aria-hidden />
                                        </span>
                                        <span className="text-sm font-semibold text-white">{item.label}</span>
                                    </li>
                                );
                            })}
                        </ul>
                    </div>
                </div>
            </div>
        </section>
    );
}
