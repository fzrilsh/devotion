import Blob from "@components/common/Blob";
import { motion } from "motion/react";
import { Link } from "react-router-dom";

export default function NotFound() {
    return (
        <>
            <section className="relative isolate min-h-screen flex items-center justify-center overflow-hidden px-4 py-16 sm:px-6 lg:px-8">
                <Blob size="lg" className="-left-55 -top-35 bg-industrial-blue-300/30" />
                <Blob size="md" className="-right-30 -bottom-30 bg-deep-navy-300/35" />

                <div className="relative mx-auto flex w-full max-w-6xl items-center justify-center">
                    <motion.div
                        initial={{ opacity: 0, y: 24 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.55, ease: "easeOut" }}
                        className="w-full rounded-xl bg-white/85 p-6 shadow-[0_30px_90px_-45px_rgba(15,23,42,0.5)] backdrop-blur-xl sm:p-10 lg:p-14"
                    >
                        <div className="grid gap-10 lg:grid-cols-[1.1fr_0.9fr] lg:items-end">
                            <div className="space-y-5">
                                <p className="inline-flex items-center rounded-full border border-industrial-blue-200 bg-industrial-blue-50 px-4 py-1.5 text-xs font-semibold uppercase tracking-[0.2em] text-industrial-blue-700">Halaman Tidak Ditemukan</p>
                                <h1 className="text-5xl font-extrabold leading-[0.95] text-deep-navy-900 sm:text-6xl lg:text-7xl">
                                    404
                                    <span className="block text-2xl font-bold text-deep-navy-500 sm:text-3xl">Alamat yang Anda tuju tidak tersedia</span>
                                </h1>
                                <p className="max-w-2xl text-base leading-relaxed text-slate-600 sm:text-lg">Tautan mungkin sudah berubah atau halaman telah dipindahkan. Anda bisa kembali ke beranda untuk melanjutkan pencarian mitra konveksi, atau masuk untuk melihat dashboard Anda.</p>

                                <div className="flex flex-col gap-3 pt-2 sm:flex-row">
                                    <Link to="/" className="inline-flex items-center justify-center rounded-xl bg-industrial-blue-500 px-6 py-3.5 text-sm font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:bg-industrial-blue-600">
                                        Kembali ke Beranda
                                    </Link>
                                    <Link to="/auth/login" className="inline-flex items-center justify-center rounded-xl border-2 border-deep-navy-500 px-6 py-3.5 text-sm font-semibold text-deep-navy-500 transition-all duration-200 hover:bg-deep-navy-50">
                                        Masuk ke Akun
                                    </Link>
                                </div>
                            </div>

                            <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} transition={{ delay: 0.15, duration: 0.5, ease: "easeOut" }} className="relative mx-auto w-full max-w-sm">
                                <div className="relative overflow-hidden rounded-2xl border border-deep-navy-100 bg-deep-navy-600 p-6 text-white shadow-2xl">
                                    <div className="absolute -right-10 -top-10 h-28 w-28 rounded-full bg-industrial-blue-400/35 blur-xl" />
                                    <div className="absolute -bottom-8 -left-8 h-24 w-24 rounded-full bg-deep-navy-300/35 blur-xl" />

                                    <p className="relative text-xs font-semibold uppercase tracking-[0.22em] text-industrial-blue-300">Status Halaman</p>
                                    <p className="relative mt-4 text-4xl font-black text-industrial-blue-400">404</p>
                                    <p className="relative mt-2 text-sm leading-relaxed text-deep-navy-100">Rute ini tidak cocok dengan halaman aplikasi yang aktif.</p>

                                    <div className="relative mt-6 space-y-2 rounded-xl bg-white/10 p-3">
                                        <p className="text-xs text-deep-navy-100">Tips cepat</p>
                                        <p className="text-sm font-medium text-white">Periksa kembali URL atau gunakan menu navigasi utama di bagian atas halaman.</p>
                                    </div>
                                </div>
                            </motion.div>
                        </div>
                    </motion.div>
                </div>
            </section>
        </>
    );
}
