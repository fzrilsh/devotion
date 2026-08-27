import { IoStar } from "react-icons/io5";
import { FaCheck } from "react-icons/fa6";
import { motion } from "motion/react";

const topCardInitial = {
    flex: "1 1 0%",
};

const topCardWhileInView = {
    flex: "0.3 1 0%",
};

const bottomCardInitial = {
    flex: "0.3 1 0%",
};

const bottomCardWhileInView = {
    flex: "1 1 0%",
};

const swapTransition = {
    duration: 2,
    ease: "easeInOut" as const,
};

export default function HeroSection() {
    return (
        <section id="beranda" className="relative w-full min-h-screen h-full flex items-center justify-center mt-24 scroll-mt-24">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-16 items-center *:content-center">
                    <div className="flex flex-col gap-4">
                        <p className="text-sm sm:text-base font-bold tracking-wide text-industrial-blue-500 uppercase">Marketplace Subkontrak Konveksi B2B</p>
                        <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold leading-tight">
                            Temukan kapasitas produksi yang siap membantu bisnis Anda <span className="text-industrial-blue-500">tumbuh.</span>
                        </h1>
                        <p className="text-base sm:text-lg text-slate-500 max-w-lg">Devotion menghubungkan UMKM konveksi yang membutuhkan kapasitas produksi tambahan dengan subkontraktor yang memiliki kapasitas tersedia. Temukan, bandingkan, dan sepakati dalam satu platform.</p>
                        <div className="flex flex-col sm:flex-row gap-3 mt-2">
                            <a href="/cari-tukang" className="inline-flex items-center justify-center rounded-xl bg-industrial-blue-500 px-6 py-3.5 text-sm font-semibold text-white hover:bg-industrial-blue-600 hover:-translate-y-0.5 transition-all duration-200">
                                Cari SubKontraktor
                            </a>

                            <a href="/gabung-tukang" className="inline-flex items-center justify-center rounded-xl border-2 border-deep-navy-500 px-6 py-3.5 text-sm font-semibold text-deep-navy-500 hover:bg-deep-navy-500/5 transition-all duration-200">
                                Tawarkan Kapasitas
                            </a>
                        </div>
                        <div className="flex items-center justify-center lg:justify-start gap-6 mt-4 pt-6 border-t border-slate-300">
                            <div className="flex flex-col">
                                <p className="text-2xl font-bold text-black flex items-center gap-1">
                                    4.9 <IoStar className="text-yellow-400" />
                                </p>
                                <p className="text-xs text-slate-400">Rating rata-rata</p>
                            </div>
                            <div className="h-8 w-px bg-text-slate-300" />
                            <div className="flex flex-col">
                                <p className="text-2xl font-bold text-black">2.500+</p>
                                <p className="text-xs text-slate-400">Tukang aktif</p>
                            </div>
                            <div className="h-8 w-px bg-text-slate-300" />
                            <div className="flex flex-col">
                                <p className="text-2xl font-bold text-black">15rb+</p>
                                <p className="text-xs text-slate-400">Pekerjaan selesai</p>
                            </div>
                        </div>
                    </div>
                    <div className="grid grid-cols-2 gap-4 h-full">
                        <div className="flex flex-col gap-4 min-h-0">
                            <motion.div initial={bottomCardInitial} whileInView={bottomCardWhileInView} transition={swapTransition} className="rounded-md overflow-hidden relative bg-deep-navy-500 min-h-12">
                                <img src="https://images.unsplash.com/photo-1741183392804-a37864e6a6d9?w=720&h=800&fit=crop&auto=format" alt="Pekerja konveksi menjahit pakaian di pabrik garmen" className="w-full h-full object-cover" />
                                <div className="absolute bottom-3 left-3 right-3 px-3 py-2 rounded-md backdrop-blur-xl bg-deep-navy-500/75">
                                    <p className="text-white text-sm font-semibold">Konveksi Garmen</p>
                                    <p className="text-xs text-slate-200 flex items-center gap-1">
                                        <IoStar className="text-yellow-400" /> 4.9 · 180 pekerjaan
                                    </p>
                                </div>
                            </motion.div>
                            <motion.div initial={topCardInitial} whileInView={topCardWhileInView} transition={swapTransition} className="rounded-md overflow-hidden relative bg-deep-navy-500 min-h-12">
                                <img src="https://images.unsplash.com/photo-1741183395212-5237420ca763?w=720&h=800&fit=crop&auto=format" alt="Penjahit wanita bekerja di pabrik garmen" className="w-full h-full object-cover" />
                            </motion.div>
                        </div>
                        <div className="flex flex-col gap-4 min-h-full">
                            <motion.div initial={topCardInitial} whileInView={topCardWhileInView} transition={swapTransition} className="rounded-md overflow-hidden relative bg-deep-navy-500 min-h-12">
                                <img src="https://images.unsplash.com/photo-1741275270798-d26e71555d0c?w=720&h=800&fit=crop&auto=format" alt="Penjahit menjahit pakaian di konveksi" className="w-full h-full object-cover" />
                            </motion.div>
                            <motion.div initial={bottomCardInitial} whileInView={bottomCardWhileInView} transition={swapTransition} className="rounded-md overflow-hidden relative bg-deep-navy-500 min-h-12">
                                <img src="https://images.unsplash.com/photo-1741275271362-bb17c416dca9?w=720&h=800&fit=crop&auto=format" alt="Pekerja konveksi menjahit di pabrik tekstil" className="w-full h-full object-cover" />
                                <div className="absolute top-3 right-3 px-3 py-1 rounded-xl text-xs font-bold bg-industrial-blue-500 text-white flex items-center gap-1">
                                    <FaCheck /> Terverifikasi
                                </div>
                            </motion.div>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    );
}
