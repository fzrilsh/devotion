import AuthLayout from "@components/layout/AuthLayout";
import { LuKeyRound, LuMail } from "react-icons/lu";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-orange-500 focus:ring-2 focus:ring-industrial-orange-500/10";

const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

export default function ForgotPassword() {
    return (
        <AuthLayout>
            <aside className="relative px-6 py-6">
                <div className="relative flex h-full w-full flex-col justify-between overflow-hidden rounded-xl bg-linear-to-br from-deep-navy-800 via-deep-navy-700 to-deep-navy-500 px-12 py-12">
                    <svg className="pointer-events-none absolute inset-0 h-full w-full opacity-[0.04]" xmlns="http://www.w3.org/2000/svg">
                        <defs>
                            <pattern id="forgot-grid" width="64" height="64" patternUnits="userSpaceOnUse">
                                <path d="M 64 0 L 0 0 0 64" fill="none" stroke="currentColor" strokeWidth="0.5" />
                            </pattern>
                        </defs>

                        <rect width="100%" height="100%" fill="url(#forgot-grid)" />
                    </svg>

                    <div className="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full border border-white/5" />
                    <div className="pointer-events-none absolute -right-12 -top-12 h-48 w-48 rounded-full border border-white/5" />

                    <div className="relative">
                        <span className="text-sm font-semibold uppercase tracking-[0.2em] text-industrial-orange-400">Account Recovery</span>
                    </div>

                    <div className="relative">
                        <h1 className="max-w-lg text-4xl font-bold leading-tight text-white">
                            Jangan khawatir,
                            <br />
                            <span className="text-industrial-orange-500">kami bantu kembali.</span>
                        </h1>

                        <p className="mt-5 max-w-xl text-justify leading-7 text-slate-300">Kehilangan akses ke akun bukan berarti kehilangan kesempatan. Masukkan email yang terdaftar dan kami akan membantu Anda mendapatkan kembali akses ke akun Devotion.</p>

                        <div className="mt-8 flex items-center gap-3 text-sm text-slate-400">
                            <div className="h-px w-10 bg-slate-600" />
                            <span>Proses aman dan terpercaya</span>
                        </div>
                    </div>
                </div>
            </aside>

            <section className="flex h-full w-full items-center justify-center px-6 py-6">
                <div className="w-full">
                    <div className="mb-8">
                        <h1 className="text-4xl font-bold text-slate-900">Lupa kata sandi?</h1>

                        <p className="mt-2 max-w-md leading-6 text-slate-500">Tidak masalah. Masukkan alamat email yang terdaftar dan kami akan mengirimkan link untuk membuat kata sandi baru.</p>
                    </div>

                    <form className="flex flex-col gap-5">
                        <div>
                            <label htmlFor="email" className={labelClassName}>
                                Alamat Email
                            </label>

                            <div className="relative">
                                <LuMail className="pointer-events-none absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-slate-400" />

                                <input id="email" name="email" type="email" placeholder="contoh@email.com" autoComplete="email" required className={`${inputClassName} pl-12`} />
                            </div>

                            <p className="mt-2 text-xs leading-5 text-slate-400">Gunakan email yang Anda gunakan saat mendaftarkan akun Devotion.</p>
                        </div>

                        <button type="submit" className="mt-2 flex w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3 font-semibold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900">
                            Kirim Link Reset
                        </button>
                    </form>

                    <div className="mt-8 rounded-xl border border-slate-200 bg-slate-50 p-4">
                        <div className="flex gap-3 items-center">
                            <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white text-industrial-orange-500 shadow-sm">
                                <LuKeyRound className="h-4 w-4" />
                            </div>

                            <div>
                                <p className="text-sm font-semibold text-slate-700">Tetap aman</p>
                                <p className="mt-1 text-xs leading-5 text-slate-500">Link reset kata sandi hanya dapat digunakan untuk akun yang terdaftar pada email tersebut.</p>
                            </div>
                        </div>
                    </div>

                    <p className="mt-8 text-center text-sm text-slate-500">
                        Ingat kata sandi Anda?{" "}
                        <a href="/auth/login" className="font-bold text-industrial-orange-500 transition-colors hover:text-industrial-orange-600">
                            Masuk
                        </a>
                    </p>
                </div>
            </section>
        </AuthLayout>
    );
}
