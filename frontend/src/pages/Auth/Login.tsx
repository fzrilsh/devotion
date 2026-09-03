import { zodResolver } from "@hookform/resolvers/zod";
import { useAuth, useLogin } from "@hooks/useAuth";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { loginSchema, type LoginForm } from "@schemas/auth";
import { forwardRef, useEffect, useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import logo from "@assets/logo.png";
import { LuEye, LuEyeOff, LuMail } from "react-icons/lu";
import { Link, useNavigate } from "react-router-dom";
import { getDefaultRedirectPath } from "@lib/roles";
import { HiOutlineLockClosed } from "react-icons/hi2";

const inputClassName = "w-full rounded-xl border py-3 pl-11 pr-4 border-slate-300 bg-white text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "sr-only";

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

export default function Login() {
    const navigate = useNavigate();
    const [showPassword, setShowPassword] = useState(false);
    const { isAuthenticated, user } = useAuth();
    const loginMutation = useLogin();

    const {
        register,
        handleSubmit,
        setError,
        formState: { errors },
    } = useForm<LoginForm>({
        resolver: zodResolver(loginSchema),
        defaultValues: {
            email: "",
            password: "",
        },
    });

    useEffect(() => {
        if (isAuthenticated && user) {
            const redirectPath = getDefaultRedirectPath({
                subcontractor: user.roles?.subcontractor,
                buyer: user.roles?.buyer,
                is_admin: user.is_admin,
            });
            navigate(redirectPath, { replace: true });
        }
    }, [isAuthenticated, user, navigate]);

    async function onSubmit(values: LoginForm) {
        try {
            const account = await loginMutation.mutateAsync(values);
            const redirectPath = getDefaultRedirectPath({
                subcontractor: account.roles?.subcontractor,
                buyer: account.roles?.buyer,
                is_admin: account.is_admin,
            });
            navigate(redirectPath, { replace: true });
        } catch (error) {
            setError("root", { message: getProblemMessage(error, "Terjadi kesalahan. Silakan coba lagi.", { 401: "Email atau kata sandi salah.", 429: "Terlalu banyak percobaan masuk, coba lagi beberapa saat." }) });
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
                    <Link to="/" className="mb-8 flex items-center justify-center lg:hidden">
                        <img src={logo} alt="Devotion" className="h-10" />
                    </Link>

                    <h2 className="text-2xl font-extrabold tracking-tight text-slate-900 sm:text-3xl">Masuk ke Devotion</h2>
                    <p className="mt-2 text-sm text-slate-500">Masuk untuk melanjutkan dan mengelola aktivitas bisnis Anda.</p>

                    <form onSubmit={handleSubmit(onSubmit)} className="mt-8 space-y-4" noValidate>
                        {errors.root?.message ? (
                            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                                {errors.root.message}
                            </div>
                        ) : null}

                        <div>
                            <label htmlFor="email" className={labelClassName}>
                                Email
                            </label>

                            <Field icon={LuMail} id="email" type="email" placeholder="Alamat email" autoComplete="email" disabled={loginMutation.isPending} className={cn(inputClassName, errors.email && "border-red-400 focus:border-red-500 focus:ring-red-500/20")} {...register("email")} />

                            {errors.email?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.email.message}</p> : null}
                        </div>

                        <div>
                            <label htmlFor="password" className={labelClassName}>
                                Kata Sandi
                            </label>

                            <div className="relative">
                                <Field
                                    icon={HiOutlineLockClosed}
                                    id="password"
                                    type={showPassword ? "text" : "password"}
                                    placeholder="Kata sandi"
                                    autoComplete="current-password"
                                    disabled={loginMutation.isPending}
                                    {...register("password")}
                                    endAdornment={
                                        <button
                                            type="button"
                                            disabled={loginMutation.isPending}
                                            onClick={() => setShowPassword((prev) => !prev)}
                                            className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-slate-400 transition-colors hover:text-slate-600 disabled:cursor-not-allowed"
                                            aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                                        >
                                            {showPassword ? <LuEyeOff className="h-5 w-5" /> : <LuEye className="h-5 w-5" />}
                                        </button>
                                    }
                                />
                            </div>

                            {errors.password?.message ? <p className="mt-1.5 text-sm text-red-600">{errors.password.message}</p> : null}
                        </div>

                        <div className="flex justify-end">
                            <Link to="/auth/forgot-password" className="text-sm font-semibold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                                Lupa kata sandi?
                            </Link>
                        </div>

                        <button
                            type="submit"
                            disabled={loginMutation.isPending}
                            className="w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3.5 text-sm font-bold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                            {loginMutation.isPending ? "Memproses..." : "Masuk"}
                        </button>
                    </form>

                    <p className="mt-6 text-center text-sm text-slate-500">
                        Belum punya akun?{" "}
                        <Link to="/auth/register" className="font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            Daftar
                        </Link>
                    </p>
                </div>
            </main>
        </div>
    );
}
