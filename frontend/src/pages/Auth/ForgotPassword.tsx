import { ApiError } from "@api/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRequestPasswordRecovery } from "@hooks/useAuth";
import { cn } from "@lib/utils";
import { recoverRequestSchema, type RecoverRequestForm } from "@schemas/auth";
import { useForm } from "react-hook-form";
import { LuKeyRound, LuMail } from "react-icons/lu";
import { Link, useNavigate } from "react-router-dom";

const inputClassName = "w-full rounded-xl border py-3 pl-11 pr-4 border-slate-300 bg-white text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "sr-only";

function getProblemMessage(error: unknown): string {
    if (error instanceof ApiError) {
        const data = error.data;

        if (typeof data === "object" && data !== null && "detail" in data && typeof data.detail === "string") {
            return data.detail;
        }

        if (typeof data === "object" && data !== null && "title" in data && typeof data.title === "string") {
            return data.title;
        }

        if (error.status === 429) {
            return "Terlalu banyak permintaan. Coba lagi beberapa saat.";
        }
    }

    return "Permintaan tidak dapat diproses. Silakan coba lagi.";
}

export default function ForgotPassword() {
    const navigate = useNavigate();
    const recoverMutation = useRequestPasswordRecovery();

    const {
        register,
        handleSubmit,
        setError,
        formState: { errors },
    } = useForm<RecoverRequestForm>({
        resolver: zodResolver(recoverRequestSchema),
        defaultValues: {
            email: "",
        },
    });

    async function onSubmit(values: RecoverRequestForm) {
        try {
            await recoverMutation.mutateAsync({ email: values.email });
            navigate("/auth/reset-password", { state: { email: values.email } });
        } catch (error) {
            setError("root", { message: getProblemMessage(error) });
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

                    <h2 className="text-2xl font-extrabold tracking-tight text-slate-900 sm:text-3xl">Lupa kata sandi?</h2>
                    <p className="mt-2 text-sm text-slate-500">Masukkan alamat email yang terdaftar dan kami akan mengirimkan kode enam digit untuk membuat kata sandi baru.</p>

                    <form onSubmit={handleSubmit(onSubmit)} className="mt-8 space-y-4" noValidate>
                        {errors.root?.message ? (
                            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                                {errors.root.message}
                            </div>
                        ) : null}

                        <div>
                            <label htmlFor="email" className={labelClassName}>
                                Alamat Email
                            </label>

                            <div className="relative">
                                <LuMail aria-hidden className="pointer-events-none absolute left-3.5 top-1/2 size-5 -translate-y-1/2 text-slate-400/35" />
                                <input
                                    id="email"
                                    type="email"
                                    placeholder="Alamat email"
                                    autoComplete="email"
                                    disabled={recoverMutation.isPending}
                                    className={cn(inputClassName, errors.email && "border-red-400 focus:border-red-500 focus:ring-red-500/20")}
                                    {...register("email")}
                                />
                            </div>

                            {errors.email?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.email.message}</p> : <p className="mt-1.5 text-xs leading-5 text-slate-400">Gunakan email yang Anda pakai saat mendaftarkan akun Devotion.</p>}
                        </div>

                        <button
                            type="submit"
                            disabled={recoverMutation.isPending}
                            className="w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3.5 text-sm font-bold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                            {recoverMutation.isPending ? "Mengirim..." : "Kirim Kode Reset"}
                        </button>
                    </form>

                    <div className="mt-6 rounded-xl border border-slate-200 bg-slate-50 p-4">
                        <div className="flex items-center gap-3">
                            <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white text-industrial-blue-500 shadow-sm">
                                <LuKeyRound className="h-4 w-4" />
                            </div>

                            <div>
                                <p className="text-sm font-semibold text-slate-700">Tetap aman</p>
                                <p className="mt-1 text-xs leading-5 text-slate-500">Kode reset kata sandi hanya dapat digunakan untuk akun yang terdaftar pada email tersebut.</p>
                            </div>
                        </div>
                    </div>

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
