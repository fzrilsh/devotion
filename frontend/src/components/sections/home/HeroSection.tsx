import { LuSearch } from "react-icons/lu";
import { Link } from "react-router-dom";

export default function HeroSection() {
    return (
        <section id="beranda" className="relative w-full min-h-screen h-full flex items-center justify-center scroll-mt-24 mt-12 md:mt-6">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8 grid grid-cols-1 lg:grid-cols-2 gap-16 items-center *:content-center">
                <div className="flex flex-col gap-4">
                    <h1 className="text-3xl sm:text-4xl lg:text-5xl font-extrabold leading-tight">
                        <span className="text-industrial-blue-500">Delegate</span> Your Overload Production
                    </h1>
                    <p className="text-base text-slate-500 max-w-lg">Devotion menghubungkan UMKM konveksi dengan subkontraktor terpercaya yang siap memenuhi kebutuhan produksi Anda. Temukan, bandingkan, dan sepakati dalam satu platform.</p>
                    <div className="flex flex-col sm:flex-row gap-3 mt-2">
                        <Link to="/search" className="inline-flex items-center justify-center gap-2 rounded-full bg-industrial-blue-500 px-6 py-3.5 text-sm font-semibold text-white hover:bg-industrial-blue-600 hover:-translate-y-0.5 transition-all duration-200">
                            <LuSearch /> Cari SubKontraktor
                        </Link>

                        <Link to="/listing" className="inline-flex items-center justify-center rounded-full border-2 border-deep-navy-500 px-6 py-3.5 text-sm font-semibold text-deep-navy-500 hover:bg-deep-navy-500/5 transition-all duration-200">
                            Tawarkan Kapasitas
                        </Link>
                    </div>
                </div>
                <div className="grid grid-cols-1 gap-4 h-full md:p-16 w-full">
                    <div className="grid grid-cols-1 md:grid-cols-2 relative gap-4 items-center justify-center w-full min-h-full">
                        <div className="w-full h-full bg-slate-100 p-6 rounded-xl justify-between flex flex-col items-start">
                            <h1 className="text-4xl font-extrabold tracking-tight text-industrial-blue-500">
                                120<span className="text-deep-navy-300">+</span>
                            </h1>
                            <p className="text-slate-500 text-sm">Mitra industri terpercaya dengan kapasitas produksi yang siap pakai</p>
                        </div>
                        <div className="w-full h-full rounded-xl overflow-hidden">
                            <img src="https://images.unsplash.com/photo-1741275271362-bb17c416dca9?w=800&h=800&fit=crop&auto=format" alt="Penjahit wanita bekerja di pabrik garmen" className="w-full h-full object-cover" />
                        </div>
                    </div>
                    <div className="flex gap-4 min-h-full w-full rounded-xl overflow-hidden">
                        <img src="https://images.unsplash.com/photo-1741183392804-a37864e6a6d9?w=700&h=350&fit=crop&auto=format" alt="Pekerja konveksi menjahit pakaian di pabrik garmen" className="w-full h-full object-cover" />
                    </div>
                </div>
            </div>
        </section>
    );
}
