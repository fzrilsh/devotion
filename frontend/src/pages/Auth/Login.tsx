import AuthLayout from "@components/layout/AuthLayout";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuEye, LuEyeOff } from "react-icons/lu";
import { useNavigate } from "react-router-dom";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-orange-500 focus:ring-2 focus:ring-industrial-orange-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

export default function Login() {
    const navigate = useNavigate();
    const [showPassword, setShowPassword] = useState(false);

    function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        navigate("/dashboard");
    }

    return (
        <AuthLayout>
            <aside className="relative px-6 py-6">
                <div className="bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 relative flex h-full w-full flex-col justify-between gap-4 overflow-hidden rounded-xl px-12 py-12">
                    <svg className="pointer-events-none absolute inset-0 h-full w-full opacity-5" xmlns="http://www.w3.org/2000/svg">
                        <defs>
                            <pattern id="grid" width="64" height="64" patternUnits="userSpaceOnUse">
                                <path className="stroke-slate-300" d="M 64 0 L 0 0 0 64" fill="none" strokeWidth="0.5" />
                            </pattern>
                        </defs>

                        <rect width="100%" height="100%" fill="url(#grid)" />
                    </svg>

                    <div className="relative">
                        <span className="text-sm font-semibold uppercase tracking-[0.2em] text-industrial-orange-400">Account Login</span>
                    </div>

                    <div className="relative">
                        <h1 className="relative text-4xl font-bold text-white">
                            Ketika kapasitas bertemu <br />
                            kebutuhan, <span className="text-industrial-orange-500">peluang tumbuh bersama.</span>
                        </h1>

                        <p className="mt-5 relative max-w-xl text-justify text-slate-300">Devotion menghubungkan UMKM konveksi dengan kapasitas produksi menganggur bersama bisnis yang membutuhkan mitra terpercaya.</p>

                        <div className="mt-8 flex items-center gap-3 text-sm text-slate-400">
                            <div className="h-px w-10 bg-slate-600" />
                            <span>Akses kembali ke bisnis Anda</span>
                        </div>
                    </div>
                </div>
            </aside>

            <section className="flex h-full w-full items-center justify-center px-6 py-6">
                <form onSubmit={handleSubmit} className="flex w-full flex-col justify-center gap-5">
                    <div>
                        <h1 className="text-4xl font-bold">Masuk ke Devotion</h1>

                        <p className="mt-1 text-slate-500">Masuk untuk melanjutkan dan mengelola aktivitas bisnis Anda.</p>
                    </div>

                    <div>
                        <label htmlFor="email" className={labelClassName}>
                            Email
                        </label>

                        <input id="email" name="email" type="email" placeholder="Alamat Email" autoComplete="email" required className={inputClassName} />
                    </div>

                    <div>
                        <div className="mb-2 flex items-center justify-between">
                            <label htmlFor="password" className="block text-sm font-semibold text-slate-500">
                                Kata Sandi
                            </label>

                            <a href="/auth/forgot-password" className="text-sm font-semibold text-industrial-orange-500 transition-colors hover:text-industrial-orange-600">
                                Lupa kata sandi?
                            </a>
                        </div>

                        <div className="relative">
                            <input id="password" name="password" type={showPassword ? "text" : "password"} placeholder="Kata Sandi" autoComplete="current-password" required className={cn(inputClassName, "pr-11")} />

                            <button type="button" onClick={() => setShowPassword((prev) => !prev)} className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600" aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}>
                                {showPassword ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                            </button>
                        </div>
                    </div>

                    <button type="submit" className="mt-2 w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3 font-semibold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900">
                        Masuk
                    </button>

                    <p className="mt-2 text-center text-sm text-slate-500">
                        Belum punya akun?{" "}
                        <a href="/auth/register" className="font-bold text-industrial-orange-500 transition-colors hover:text-industrial-orange-600">
                            Daftar
                        </a>
                    </p>
                </form>
            </section>
        </AuthLayout>
    );
}
