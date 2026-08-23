import AuthLayout from "@components/layout/AuthLayout";
import { cn } from "@lib/utils";
import { useState } from "react";
import { HiOutlineWrenchScrewdriver, HiOutlineClipboardDocumentCheck, HiOutlineArrowsRightLeft, HiCheck } from "react-icons/hi2";
import { LuEyeOff, LuEye } from "react-icons/lu";
import { useNavigate } from "react-router-dom";

const roles = [
    {
        id: "subkontraktor",
        title: "Subkontraktor",
        description: "Saya mengerjakan pekerjaan dari pemberi order.",
        icon: HiOutlineWrenchScrewdriver,
    },
    {
        id: "pemberi-order",
        title: "Pemberi Order",
        description: "Saya mencari dan memberikan pekerjaan kepada subkontraktor.",
        icon: HiOutlineClipboardDocumentCheck,
    },
    {
        id: "keduanya",
        title: "Keduanya",
        description: "Saya dapat menjadi subkontraktor sekaligus pemberi order.",
        icon: HiOutlineArrowsRightLeft,
    },
];

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-orange-500 focus:ring-2 focus:ring-industrial-orange-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

export default function Register() {
    const navigate = useNavigate();
    const [selectedRole, setSelectedRole] = useState("subkontraktor");
    const [showPassword, setShowPassword] = useState(false);
    const [showPasswordConfirmation, setShowPasswordConfirmation] = useState(false);

    function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        navigate("/auth/verify-email");
    }

    return (
        <AuthLayout>
            <aside className="px-6 py-6 relative overflow-y-hidden">
                <div className="bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 h-full w-full justify-between overflow-hidden flex-col gap-4 flex px-12 py-12 rounded-xl">
                    <svg className="absolute inset-0 w-full h-full opacity-5 pointer-events-none flex items-center justify-center" xmlns="http://www.w3.org/2000/svg">
                        <defs>
                            <pattern id="grid" width="64" height="64" patternUnits="userSpaceOnUse">
                                <path className="stroke-slate-300" d="M 64 0 L 0 0 0 64" fill="none" strokeWidth="0.5" />
                            </pattern>
                        </defs>
                        <rect width="100%" height="100%" fill="url(#grid)" />
                    </svg>

                    <div className="relative">
                        <span className="text-sm font-semibold uppercase tracking-[0.2em] text-industrial-orange-400">Account Register</span>
                    </div>

                    <div className="relative">
                        <h1 className="text-4xl text-white font-bold">
                            Ketika kapasitas bertemu <br />
                            kebutuhan, <span className="text-industrial-orange-500">peluang tumbuh bersama.</span>
                        </h1>
                        <p className="mt-5 text-justify max-w-xl text-slate-300">Devotion menghubungkan UMKM konveksi dengan kapasitas produksi menganggur bersama bisnis yang membutuhkan mitra terpercaya.</p>

                        <div className="mt-8 flex items-center gap-3 text-sm text-slate-400">
                            <div className="h-px w-10 bg-slate-600" />
                            <span>Pendaftaran cepat dan mudah</span>
                        </div>
                    </div>
                </div>
            </aside>
            <section className="flex h-full w-full items-center justify-center px-6 py-6 overflow-y-auto">
                <form onSubmit={handleSubmit} className="h-full w-full justify-center flex flex-col gap-4">
                    <div>
                        <h1 className="text-4xl font-bold">Buat akun Devotion</h1>
                        <p className="text-slate-500">Bergabung dengan jaringan bisnis yang membantu kapasitas produksi bertemu peluang baru.</p>
                    </div>

                    <div>
                        <p className="mb-3 text-sm font-semibold text-slate-500">Saya mendaftar sebagai:</p>

                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                            {roles.map((role) => {
                                const Icon = role.icon;
                                const isSelected = selectedRole === role.id;

                                return (
                                    <button key={role.id} className={cn("border rounded-xl p-4 relative transition-all duration-200", isSelected ? "bg-industrial-orange-100 border-industrial-orange-500" : "border-slate-300")} type="button" onClick={() => setSelectedRole(role.id)}>
                                        {isSelected && (
                                            <span className="absolute right-3 top-3 flex h-5 w-5 items-center justify-center rounded-full transition-all duration-200 bg-industrial-orange-500 text-white">
                                                <HiCheck className="h-3 w-3" strokeWidth={2.5} />
                                            </span>
                                        )}

                                        <div className={cn("mb-4 flex h-11 w-11 items-center justify-center rounded-lg transition-all duration-200", isSelected ? "bg-industrial-orange-500 text-white" : "bg-slate-100 text-slate-500")}>
                                            <Icon className="h-5 w-5" />
                                        </div>

                                        <h3 className={cn("font-bold text-left transition-all duration-200", isSelected ? "text-slate-900" : "text-slate-700")}>{role.title}</h3>
                                        <p className="mt-1 text-sm leading-5 text-slate-500 text-justify">{role.description}</p>
                                    </button>
                                );
                            })}
                        </div>
                    </div>

                    <div>
                        <label htmlFor="business-name" className={labelClassName}>
                            Nama Usaha
                        </label>

                        <input id="business-name" name="business_name" type="text" placeholder="Masukkan nama usaha" autoComplete="organization" className={inputClassName} />
                    </div>

                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <div>
                            <label htmlFor="email" className={labelClassName}>
                                Email
                            </label>

                            <input id="email" name="email" type="email" placeholder="contoh@email.com" autoComplete="email" className={inputClassName} />
                        </div>

                        <div>
                            <label htmlFor="phone" className={labelClassName}>
                                Nomor HP
                            </label>

                            <input id="phone" name="phone" type="tel" placeholder="08xxxxxxxxxx" autoComplete="tel" className={inputClassName} />
                            <p className="mt-1.5 text-xs text-slate-400">Nomor ini akan digunakan untuk verifikasi.</p>
                        </div>
                    </div>
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <div>
                            <label htmlFor="province" className={labelClassName}>
                                Provinsi
                            </label>

                            <select id="province" name="province" defaultValue="" className={cn(inputClassName, "cursor-pointer")}>
                                <option value="" disabled>
                                    Pilih provinsi
                                </option>
                                <option value="riau">Riau</option>
                                <option value="sumatera-barat">Sumatera Barat</option>
                                <option value="sumatera-utara">Sumatera Utara</option>
                                <option value="jambi">Jambi</option>
                            </select>
                        </div>

                        <div>
                            <label htmlFor="city" className={labelClassName}>
                                Kota / Kabupaten
                            </label>

                            <select id="city" name="city" defaultValue="" className={cn(inputClassName, "cursor-pointer")}>
                                <option value="" disabled>
                                    Pilih kota / kabupaten
                                </option>
                                <option value="pekanbaru">Kota Pekanbaru</option>
                                <option value="dumai">Kota Dumai</option>
                                <option value="kampar">Kabupaten Kampar</option>
                                <option value="siak">Kabupaten Siak</option>
                            </select>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <div>
                            <label htmlFor="password" className={labelClassName}>
                                Kata Sandi
                            </label>

                            <div className="relative">
                                <input id="password" name="password" type={showPassword ? "text" : "password"} placeholder="Minimal 8 karakter" autoComplete="new-password" className={cn(inputClassName, "pr-11")} />

                                <button
                                    type="button"
                                    onClick={() => setShowPassword((prev) => !prev)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600"
                                    aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                                >
                                    {showPassword ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                </button>
                            </div>
                        </div>

                        <div>
                            <label htmlFor="password-confirmation" className={labelClassName}>
                                Konfirmasi Kata Sandi
                            </label>

                            <div className="relative">
                                <input id="password-confirmation" name="password_confirmation" type={showPasswordConfirmation ? "text" : "password"} placeholder="Ulangi kata sandi" autoComplete="new-password" className={cn(inputClassName, "pr-11")} />

                                <button
                                    type="button"
                                    onClick={() => setShowPasswordConfirmation((prev) => !prev)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600"
                                    aria-label={showPasswordConfirmation ? "Sembunyikan konfirmasi kata sandi" : "Tampilkan konfirmasi kata sandi"}
                                >
                                    {showPasswordConfirmation ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                </button>
                            </div>
                        </div>
                    </div>

                    <button type="submit" className="w-full bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-2 rounded-xl text-white font-semibold hover:bg-linear-to-b hover: cursor-pointer">
                        Daftar Sekarang
                    </button>

                    <p className="text-sm text-slate-500 text-center mt-2 leading-6">
                        Dengan mendaftar, Anda menyetujui <span className="text-industrial-orange-500 font-bold">Syarat & Ketentuan</span> serta <span className="text-industrial-orange-500 font-bold">Kebijakan Privasi</span> Devotion.
                    </p>

                    <p className="text-slate-500 text-center mt-2">
                        Sudah punya akun?{" "}
                        <a href="/auth/login" className="text-industrial-orange-500 font-bold cursor-pointer">
                            Masuk
                        </a>
                    </p>
                </form>
            </section>
        </AuthLayout>
    );
}
