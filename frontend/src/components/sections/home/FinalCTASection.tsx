import Blob from "@components/common/Blob";
import { LuSearch } from "react-icons/lu";

export default function FinalCTASection() {
    return (
        <section className="h-full flex items-center justify-center scroll-mt-24 relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
            <div className="px-4 py-8 sm:px-6 lg:px-8 bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 rounded-xl relative overflow-hidden w-full">
                <Blob animate={false} size="lg" className="right-0 top-0 bg-deep-navy-800/50" />
                <Blob animate={false} size="md" className="right-16 bottom-0 bg-deep-navy-300/30" />

                <div className="relative max-w-xl">
                    <p className="text-industrial-orange-500 text-xs font-semibold tracking-widest uppercase mb-4">Mulai Sekarang</p>
                    <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">Mulai temukan peluang produksi baru hari ini.</h2>
                    <p className="text-white/60 text-sm leading-relaxed mb-8">Gratis untuk mendaftar. Tidak perlu komitmen di muka. Mulai eksplorasi kapasitas yang tersedia atau pasang profil usaha Anda.</p>
                </div>

                <div className="flex flex-col sm:flex-row gap-3">
                    <a
                        href="/search"
                        className="inline-flex items-center justify-center gap-2 bg-industrial-orange-500 hover:bg-industrial-orange-600 text-white font-semibold px-6 py-3.5 rounded transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-industrial-orange-500"
                    >
                        <LuSearch className="w-4 h-4" />
                        Mulai mencari
                    </a>
                    <a href="/auth/register" className="inline-flex items-center justify-center gap-2 border border-slate-300 hover:border-slate-400 text-slate-300 font-semibold px-6 py-3.5 rounded transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-300">
                        Buat akun
                    </a>
                </div>
            </div>
        </section>
    );
}
