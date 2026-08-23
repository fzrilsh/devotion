import AuthLayout from "@components/layout/AuthLayout";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuCheck, LuEye, LuEyeOff } from "react-icons/lu";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-orange-500 focus:ring-2 focus:ring-industrial-orange-500/10";

const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

export default function ResetPassword() {
    const [showPassword, setShowPassword] = useState(false);
    const [showPasswordConfirmation, setShowPasswordConfirmation] = useState(false);

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
                        <span className="text-sm font-semibold uppercase tracking-[0.2em] text-industrial-orange-400">Secure Your Account</span>
                    </div>

                    <div className="relative">
                        <h1 className="max-w-lg text-4xl font-bold leading-tight text-white">
                            Satu langkah lagi,
                            <br />
                            <span className="text-industrial-orange-500">amankan akun Anda.</span>
                        </h1>

                        <p className="mt-5 max-w-xl text-justify leading-7 text-slate-300">Buat kata sandi baru yang kuat untuk menjaga keamanan akun Devotion dan melanjutkan perjalanan bisnis Anda bersama kami.</p>

                        <div className="mt-8 flex items-center gap-3 text-sm text-slate-400">
                            <div className="h-px w-10 bg-slate-600" />
                            <span>Akun Anda tetap aman bersama Devotion</span>
                        </div>
                    </div>
                </div>
            </aside>

            <section className="flex h-full w-full items-center justify-center px-6 py-6">
                <div className="w-full">
                    <div className="mb-8">
                        <h1 className="text-4xl font-bold text-slate-900">Buat kata sandi baru</h1>

                        <p className="mt-2 max-w-md leading-6 text-slate-500">Buat kata sandi baru yang aman untuk melindungi akun Devotion Anda.</p>
                    </div>

                    <form className="flex flex-col gap-5">
                        <div>
                            <label htmlFor="password" className={labelClassName}>
                                Kata Sandi Baru
                            </label>

                            <div className="relative">
                                <input id="password" name="password" type={showPassword ? "text" : "password"} placeholder="Masukkan kata sandi baru" autoComplete="new-password" required minLength={8} className={cn(inputClassName, "pr-11")} />

                                <button
                                    type="button"
                                    onClick={() => setShowPassword((prev) => !prev)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600"
                                    aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                                >
                                    {showPassword ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                </button>
                            </div>

                            <p className="mt-1.5 text-xs text-slate-400">Gunakan minimal 8 karakter.</p>
                        </div>

                        <div>
                            <label htmlFor="password-confirmation" className={labelClassName}>
                                Konfirmasi Kata Sandi
                            </label>

                            <div className="relative">
                                <input id="password-confirmation" name="password_confirmation" type={showPasswordConfirmation ? "text" : "password"} placeholder="Ulangi kata sandi baru" autoComplete="new-password" required className={cn(inputClassName, "pr-11")} />

                                <button
                                    type="button"
                                    onClick={() => setShowPasswordConfirmation((prev) => !prev)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600"
                                    aria-label={showPasswordConfirmation ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                                >
                                    {showPasswordConfirmation ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                </button>
                            </div>
                        </div>

                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                            <p className="mb-3 text-xs font-semibold text-slate-600">Kata sandi yang baik:</p>

                            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                                <div className="flex items-center gap-2 text-xs text-slate-500">
                                    <LuCheck className="h-3.5 w-3.5 text-industrial-orange-500" />
                                    Minimal 8 karakter
                                </div>

                                <div className="flex items-center gap-2 text-xs text-slate-500">
                                    <LuCheck className="h-3.5 w-3.5 text-industrial-orange-500" />
                                    Tidak mudah ditebak
                                </div>
                            </div>
                        </div>

                        <button type="submit" className="mt-1 w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3 font-semibold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900">
                            Simpan Kata Sandi
                        </button>
                    </form>

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
