import { zodResolver } from "@hookform/resolvers/zod";
import { useRegister } from "@hooks/useAuth";
import { useWilayah } from "@hooks/useWilayah";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { registerSchema, type RegisterForm } from "@schemas/auth";
import { forwardRef, useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import { HiOutlineArrowsRightLeft, HiOutlineClipboardDocumentCheck, HiOutlineWrenchScrewdriver, HiCheck } from "react-icons/hi2";
import { LuBuilding2, LuEye, LuEyeOff, LuMail, LuPhone } from "react-icons/lu";
import { Link, useNavigate } from "react-router-dom";

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

const inputClassName = "w-full rounded-xl border py-3 pl-11 pr-4 border-slate-300 bg-white text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "sr-only";

function normalizePhone(value: string): string {
    const digits = value.replace(/\D/g, "");

    if (digits.startsWith("0")) {
        return `+62${digits.slice(1)}`;
    }

    if (digits.startsWith("62")) {
        return digits;
    }

    return digits;
}

const Field = forwardRef<
    HTMLInputElement,
    {
        icon: React.ElementType;
        id: string;
        endAdornment?: ReactNode;
    } & React.InputHTMLAttributes<HTMLInputElement>
>(({ icon: Icon, id, endAdornment, className, ...props }, ref) => {
    return (
        <div className="relative">
            <Icon aria-hidden className="pointer-events-none absolute left-3.5 top-1/2 size-5 -translate-y-1/2 text-slate-400/35" />

            <input ref={ref} id={id} {...props} className={cn(inputClassName, endAdornment ? "pr-11" : "", className ?? "")} />

            {endAdornment}
        </div>
    );
});

export default function Register() {
    const navigate = useNavigate();
    const registerMutation = useRegister();

    const [selectedRole, setSelectedRole] = useState("subkontraktor");
    const [showPassword, setShowPassword] = useState(false);
    const [showPasswordConfirmation, setShowPasswordConfirmation] = useState(false);

    const [provinceCode, setProvinceCode] = useState("");

    const { provinces, cities } = useWilayah(provinceCode);

    const {
        register,
        handleSubmit,
        setError,
        formState: { errors },
    } = useForm<RegisterForm>({
        resolver: zodResolver(registerSchema),
        defaultValues: {
            email: "",
            phone: "",
            business_name: "",
            city_code: "",
            password: "",
            password_confirmation: "",
            roles: {
                subcontractor: true,
                buyer: false,
            },
        },
    });

    function getRoles(role: string) {
        return {
            subcontractor: role === "subkontraktor" || role === "keduanya",
            buyer: role === "pemberi-order" || role === "keduanya",
        };
    }

    async function onSubmit(values: RegisterForm) {
        try {
            await registerMutation.mutateAsync({
                email: values.email,
                phone: normalizePhone(values.phone),
                password: values.password,
                business_name: values.business_name,
                city_code: values.city_code,
                roles: getRoles(selectedRole),
            });

            navigate("/auth/verify-email", { replace: true, state: { email: values.email } });
        } catch (error) {
            setError("root", {
                message: getProblemMessage(error, "Pendaftaran gagal. Silakan coba lagi.", { 409: "Email atau data akun tersebut sudah terdaftar.", 429: "Terlalu banyak percobaan pendaftaran. Coba lagi beberapa saat." }),
            });
        }
    }

    return (
        <div className="grid min-h-screen bg-white lg:grid-cols-2">
            <aside className="relative hidden overflow-hidden bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 lg:flex lg:flex-col lg:justify-center lg:p-12">
                <div aria-hidden className="pointer-events-none absolute inset-0">
                    <div className="absolute -right-32 -top-32 size-112 rounded-full bg-industrial-blue-500/20 blur-3xl" />
                    <div className="absolute -bottom-40 -left-24 size-96 rounded-full bg-industrial-blue-500/15 blur-3xl" />
                    <div className="absolute right-16 top-1/3 size-40 rounded-3xl border border-white/10" />
                    <div className="absolute bottom-24 right-40 size-24 rounded-full border border-white/10" />
                    <div className="absolute left-1/3 top-16 size-16 rounded-2xl bg-white/5" />
                </div>

                <div className="relative max-w-md">
                    <h1 className="text-4xl font-extrabold leading-tight tracking-tight text-white">Satu platform untuk kapasitas produksi konveksi.</h1>

                    <p className="mt-4 text-base leading-relaxed text-white/70">Devotion menghubungkan UMKM konveksi dengan kapasitas produksi menganggur bersama bisnis yang membutuhkan mitra terpercaya, dari pencarian hingga pesanan selesai.</p>
                </div>
            </aside>

            <main className="flex items-center justify-center overflow-y-auto px-5 py-10 sm:px-8">
                <div className="w-full max-w-md">
                    <h2 className="text-2xl font-extrabold tracking-tight text-slate-900 sm:text-3xl">Buat akun Devotion</h2>
                    <p className="mt-2 text-sm text-slate-500">Satu akun untuk mencari subkontraktor terpercaya maupun menawarkan kapasitas produksi Anda kepada pemberi order.</p>

                    <form onSubmit={handleSubmit(onSubmit)} className="mt-8 space-y-4" noValidate>
                        {errors.root?.message ? (
                            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                                {errors.root.message}
                            </div>
                        ) : null}

                        <div>
                            <p className="mb-2 text-sm font-semibold text-slate-700">Saya mendaftar sebagai:</p>

                            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                                {roles.map((role) => {
                                    const Icon = role.icon;
                                    const isSelected = selectedRole === role.id;

                                    return (
                                        <button
                                            key={role.id}
                                            type="button"
                                            disabled={registerMutation.isPending}
                                            onClick={() => setSelectedRole(role.id)}
                                            className={cn(
                                                "relative rounded-xl border p-3.5 text-left transition-all duration-200",
                                                isSelected ? "border-industrial-blue-500 bg-industrial-blue-100" : "border-slate-300 bg-white hover:border-slate-400",
                                                registerMutation.isPending && "cursor-not-allowed opacity-70",
                                            )}
                                        >
                                            {isSelected ? (
                                                <span className="absolute right-3 top-3 flex h-5 w-5 items-center justify-center rounded-full bg-industrial-blue-500 text-white">
                                                    <HiCheck className="h-3 w-3" strokeWidth={2.5} />
                                                </span>
                                            ) : null}

                                            <div className={cn("mb-3 flex h-9 w-9 items-center justify-center rounded-lg transition-all duration-200", isSelected ? "bg-industrial-blue-500 text-white" : "bg-slate-100 text-slate-500")}>
                                                <Icon className="h-5 w-5" />
                                            </div>

                                            <h3 className={cn("text-sm font-bold", isSelected ? "text-slate-900" : "text-slate-700")}>{role.title}</h3>

                                            <p className="mt-1 text-xs leading-5 text-slate-500">{role.description}</p>
                                        </button>
                                    );
                                })}
                            </div>
                        </div>

                        <div>
                            <label htmlFor="business-name" className={labelClassName}>
                                Nama Usaha
                            </label>

                            <Field
                                icon={LuBuilding2}
                                id="business-name"
                                type="text"
                                placeholder="Nama usaha"
                                autoComplete="organization"
                                disabled={registerMutation.isPending}
                                className={cn(inputClassName, errors.business_name && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                {...register("business_name")}
                            />

                            {errors.business_name?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.business_name.message}</p> : null}
                        </div>

                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            <div>
                                <label htmlFor="email" className={labelClassName}>
                                    Email
                                </label>

                                <Field icon={LuMail} id="email" type="email" placeholder="Alamat email" autoComplete="email" disabled={registerMutation.isPending} className={cn(inputClassName, errors.email && "border-red-400 focus:border-red-500 focus:ring-red-500/20")} {...register("email")} />

                                {errors.email?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.email.message}</p> : null}
                            </div>

                            <div>
                                <label htmlFor="phone" className={labelClassName}>
                                    Nomor HP
                                </label>

                                <Field
                                    icon={LuPhone}
                                    id="phone"
                                    type="tel"
                                    inputMode="tel"
                                    placeholder="Nomor HP"
                                    autoComplete="tel"
                                    disabled={registerMutation.isPending}
                                    className={cn(inputClassName, errors.phone && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                    {...register("phone")}
                                />

                                {errors.phone?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.phone.message}</p> : null}
                            </div>
                        </div>

                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            <div>
                                <label htmlFor="province" className={labelClassName}>
                                    Provinsi
                                </label>

                                <select
                                    id="province"
                                    value={provinceCode}
                                    disabled={registerMutation.isPending}
                                    onChange={(event) => {
                                        setProvinceCode(event.target.value);
                                    }}
                                    className={cn(inputClassName, "cursor-pointer pl-4")}
                                >
                                    <option value="">Pilih provinsi</option>

                                    {provinces.map((province) => (
                                        <option key={province.code} value={province.code}>
                                            {province.name}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            <div>
                                <label htmlFor="city" className={labelClassName}>
                                    Kota / Kabupaten
                                </label>

                                <select id="city" disabled={registerMutation.isPending || !provinceCode || cities.length === 0} className={cn(inputClassName, "cursor-pointer pl-4", errors.city_code && "border-red-400 focus:border-red-500 focus:ring-red-500/20")} {...register("city_code")}>
                                    <option value="">Pilih kota / kabupaten</option>

                                    {cities.map((city) => (
                                        <option key={city.code} value={city.code}>
                                            {city.name}
                                        </option>
                                    ))}
                                </select>

                                {errors.city_code?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.city_code.message}</p> : null}
                            </div>
                        </div>

                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            <div>
                                <label htmlFor="password" className={labelClassName}>
                                    Kata Sandi
                                </label>

                                <div className="relative">
                                    <input
                                        id="password"
                                        type={showPassword ? "text" : "password"}
                                        placeholder="Kata sandi"
                                        autoComplete="new-password"
                                        disabled={registerMutation.isPending}
                                        className={cn(inputClassName, "pl-4 pr-11", errors.password && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                        {...register("password")}
                                    />

                                    <button
                                        type="button"
                                        disabled={registerMutation.isPending}
                                        onClick={() => setShowPassword((prev) => !prev)}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600 disabled:cursor-not-allowed"
                                        aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                                    >
                                        {showPassword ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                    </button>
                                </div>

                                {errors.password?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.password.message}</p> : null}
                            </div>

                            <div>
                                <label htmlFor="password-confirmation" className={labelClassName}>
                                    Konfirmasi Kata Sandi
                                </label>

                                <div className="relative">
                                    <input
                                        id="password-confirmation"
                                        type={showPasswordConfirmation ? "text" : "password"}
                                        placeholder="Ulangi kata sandi"
                                        autoComplete="new-password"
                                        disabled={registerMutation.isPending}
                                        className={cn(inputClassName, "pl-4 pr-11", errors.password_confirmation && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                        {...register("password_confirmation")}
                                    />

                                    <button
                                        type="button"
                                        disabled={registerMutation.isPending}
                                        onClick={() => setShowPasswordConfirmation((prev) => !prev)}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600 disabled:cursor-not-allowed"
                                        aria-label={showPasswordConfirmation ? "Sembunyikan konfirmasi kata sandi" : "Tampilkan konfirmasi kata sandi"}
                                    >
                                        {showPasswordConfirmation ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                    </button>
                                </div>

                                {errors.password_confirmation?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.password_confirmation.message}</p> : null}
                            </div>
                        </div>

                        <button
                            type="submit"
                            disabled={registerMutation.isPending}
                            className="w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3.5 text-sm font-bold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                            {registerMutation.isPending ? "Memproses..." : "Daftar Sekarang"}
                        </button>
                    </form>

                    <p className="mt-2 text-center text-sm text-slate-500">
                        Sudah punya akun?{" "}
                        <Link to="/auth/login" className="font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            Masuk
                        </Link>
                    </p>

                    <p className="mt-6 text-center text-sm leading-6 text-slate-500">
                        Dengan mendaftar, Anda menyetujui <Link to="/syarat-ketentuan" className="font-bold text-industrial-blue-500">Syarat & Ketentuan</Link> serta <Link to="/kebijakan-privasi" className="font-bold text-industrial-blue-500">Kebijakan Privasi</Link> Devotion.
                    </p>
                </div>
            </main>
        </div>
    );
}
