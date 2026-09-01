import { zodResolver } from "@hookform/resolvers/zod";
import { useConfirmPasswordRecovery } from "@hooks/useAuth";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { recoverConfirmSchema, type RecoverConfirmForm } from "@schemas/auth";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { LuCheck, LuEye, LuEyeOff, LuMail } from "react-icons/lu";
import { Link, useLocation, useNavigate } from "react-router-dom";

const inputClassName = "w-full rounded-xl border py-3 pl-4 pr-11 border-slate-300 bg-white text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "sr-only";

type ResetPasswordLocationState = {
    email?: string;
};

export default function ResetPassword() {
    const navigate = useNavigate();
    const location = useLocation();
    const { email } = (location.state as ResetPasswordLocationState | null) ?? {};
    const confirmMutation = useConfirmPasswordRecovery();

    const [otp, setOtp] = useState(["", "", "", "", "", ""]);
    const [showPassword, setShowPassword] = useState(false);
    const [showPasswordConfirmation, setShowPasswordConfirmation] = useState(false);
    const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

    const {
        register,
        handleSubmit,
        setError,
        setValue,
        formState: { errors },
    } = useForm<RecoverConfirmForm>({
        resolver: zodResolver(recoverConfirmSchema),
        defaultValues: {
            email: email ?? "",
            code: "",
            new_password: "",
            password_confirmation: "",
        },
    });

    useEffect(() => {
        if (!email) {
            navigate("/auth/forgot-password", { replace: true });
        }
    }, [email, navigate]);

    const handleChange = (value: string, index: number) => {
        const digit = value.replace(/\D/g, "").slice(-1);

        const newOtp = [...otp];
        newOtp[index] = digit;

        setOtp(newOtp);
        setValue("code", newOtp.join(""), { shouldValidate: true });

        if (digit && index < newOtp.length - 1) {
            inputRefs.current[index + 1]?.focus();
        }
    };

    const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>, index: number) => {
        if (event.key === "Backspace" && !otp[index] && index > 0) {
            inputRefs.current[index - 1]?.focus();
        }
    };

    async function onSubmit(values: RecoverConfirmForm) {
        try {
            await confirmMutation.mutateAsync({
                email: values.email,
                code: values.code,
                new_password: values.new_password,
            });

            navigate("/auth/login", { replace: true });
        } catch (error) {
            setError("root", { message: getProblemMessage(error, "Permintaan tidak dapat diproses. Silakan coba lagi.", { 410: "Kode pemulihan kedaluwarsa atau sudah dipakai. Minta kode baru.", 429: "Terlalu banyak percobaan. Coba lagi beberapa saat." }) });
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
                    <Link to="/" className="mb-8 flex items-center gap-2.5 lg:hidden">
                        <span className="grid size-9 place-items-center rounded-xl bg-industrial-blue-500 text-sm font-extrabold text-white">D</span>
                        <span className="text-lg font-bold tracking-tight text-slate-900">Devotion</span>
                    </Link>

                    <h2 className="text-2xl font-extrabold tracking-tight text-slate-900 sm:text-3xl">Buat kata sandi baru</h2>
                    <p className="mt-2 text-sm text-slate-500">Masukkan kode pemulihan yang kami kirim ke email Anda, lalu buat kata sandi baru.</p>

                    {email ? (
                        <div className="mt-4 inline-flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2">
                            <LuMail className="h-4 w-4 text-industrial-blue-500" />
                            <p className="text-sm font-semibold text-slate-800">{email}</p>
                        </div>
                    ) : null}

                    <form onSubmit={handleSubmit(onSubmit)} className="mt-8 space-y-4" noValidate>
                        {errors.root?.message ? (
                            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                                {errors.root.message}
                            </div>
                        ) : null}

                        <div>
                            <label className="mb-2 block text-sm font-semibold text-slate-700">Kode Pemulihan</label>

                            <div className="grid grid-cols-6 gap-2 sm:gap-3">
                                {otp.map((digit, index) => (
                                    <input
                                        key={index}
                                        ref={(element) => {
                                            inputRefs.current[index] = element;
                                        }}
                                        value={digit}
                                        onChange={(event) => handleChange(event.target.value, index)}
                                        onKeyDown={(event) => handleKeyDown(event, index)}
                                        inputMode="numeric"
                                        maxLength={1}
                                        disabled={confirmMutation.isPending}
                                        aria-label={`Digit kode ke-${index + 1}`}
                                        className="h-14 w-full rounded-xl border border-slate-300 bg-white text-center text-xl font-bold text-slate-800 outline-none transition-all focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10"
                                    />
                                ))}
                            </div>

                            {errors.code?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.code.message}</p> : null}
                        </div>

                        <div>
                            <label htmlFor="password" className={labelClassName}>
                                Kata Sandi Baru
                            </label>

                            <div className="relative">
                                <input
                                    id="password"
                                    type={showPassword ? "text" : "password"}
                                    placeholder="Kata sandi baru (min. 8 karakter)"
                                    autoComplete="new-password"
                                    disabled={confirmMutation.isPending}
                                    className={cn(inputClassName, errors.new_password && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                    {...register("new_password")}
                                />

                                <button
                                    type="button"
                                    disabled={confirmMutation.isPending}
                                    onClick={() => setShowPassword((prev) => !prev)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600"
                                    aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                                >
                                    {showPassword ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                </button>
                            </div>

                            {errors.new_password?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.new_password.message}</p> : null}
                        </div>

                        <div>
                            <label htmlFor="password-confirmation" className={labelClassName}>
                                Konfirmasi Kata Sandi
                            </label>

                            <div className="relative">
                                <input
                                    id="password-confirmation"
                                    type={showPasswordConfirmation ? "text" : "password"}
                                    placeholder="Ulangi kata sandi baru"
                                    autoComplete="new-password"
                                    disabled={confirmMutation.isPending}
                                    className={cn(inputClassName, errors.password_confirmation && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                    {...register("password_confirmation")}
                                />

                                <button
                                    type="button"
                                    disabled={confirmMutation.isPending}
                                    onClick={() => setShowPasswordConfirmation((prev) => !prev)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600"
                                    aria-label={showPasswordConfirmation ? "Sembunyikan konfirmasi kata sandi" : "Tampilkan konfirmasi kata sandi"}
                                >
                                    {showPasswordConfirmation ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                </button>
                            </div>

                            {errors.password_confirmation?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.password_confirmation.message}</p> : null}
                        </div>

                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                            <p className="mb-3 text-xs font-semibold text-slate-600">Kata sandi yang baik:</p>

                            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                                <div className="flex items-center gap-2 text-xs text-slate-500">
                                    <LuCheck className="h-3.5 w-3.5 text-industrial-blue-500" />
                                    Minimal 8 karakter
                                </div>

                                <div className="flex items-center gap-2 text-xs text-slate-500">
                                    <LuCheck className="h-3.5 w-3.5 text-industrial-blue-500" />
                                    Tidak mudah ditebak
                                </div>
                            </div>
                        </div>

                        <button
                            type="submit"
                            disabled={confirmMutation.isPending}
                            className={cn(
                                "w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3.5 text-sm font-bold text-white transition-all duration-200",
                                "hover:from-deep-navy-600 hover:to-deep-navy-900 disabled:cursor-not-allowed disabled:opacity-70",
                            )}
                        >
                            {confirmMutation.isPending ? "Menyimpan..." : "Simpan Kata Sandi"}
                        </button>
                    </form>

                    <p className="mt-6 text-center text-sm text-slate-500">
                        Ingat kata sandi Anda?{" "}
                        <Link to="/auth/login" className="font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            Masuk
                        </Link>
                    </p>
                </div>
            </main>
        </div>
    );
}
