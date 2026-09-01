import Blob from "@components/common/Blob";
import AuthLayout from "@components/layout/AuthLayout";
import { ApiError } from "@api/client";
import { useAuth, useResendCode, useVerifyEmail } from "@hooks/useAuth";
import { getProblemMessage } from "@lib/problem";
import { useEffect, useRef, useState } from "react";
import { LuMail, LuRefreshCw, LuShieldCheck } from "react-icons/lu";
import { Link, useLocation, useNavigate } from "react-router-dom";

type VerifyEmailLocationState = {
    email?: string;
};

export default function VerifyEmail() {
    const navigate = useNavigate();
    const location = useLocation();
    const { email: emailFromState } = (location.state as VerifyEmailLocationState | null) ?? {};
    const { user } = useAuth();
    const email = emailFromState ?? user?.email;
    const verifyMutation = useVerifyEmail();
    const resendMutation = useResendCode();
    const [otp, setOtp] = useState(["", "", "", "", "", ""]);
    const [countdown, setCountdown] = useState(60);
    const [errorMessage, setErrorMessage] = useState("");
    const [sessionLost, setSessionLost] = useState(false);
    const [successMessage, setSuccessMessage] = useState("");
    const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

    useEffect(() => {
        if (countdown === 0) return;

        const timer = window.setTimeout(() => setCountdown((value) => value - 1), 1000);
        return () => window.clearTimeout(timer);
    }, [countdown]);

    useEffect(() => {
        if (!email) {
            navigate(user ? "/profile/me" : "/auth/register", { replace: true });
        }
    }, [email, user, navigate]);

    const handleChange = (value: string, index: number) => {
        const digit = value.replace(/\D/g, "").slice(-1);

        const newOtp = [...otp];
        newOtp[index] = digit;

        setOtp(newOtp);

        if (digit && index < otp.length - 1) {
            inputRefs.current[index + 1]?.focus();
        }
    };

    const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>, index: number) => {
        if (event.key === "Backspace" && !otp[index] && index > 0) {
            inputRefs.current[index - 1]?.focus();
        }
    };

    async function handleVerify() {
        const code = otp.join("");
        setErrorMessage("");
        setSessionLost(false);

        if (code.length !== 6) {
            setErrorMessage("Masukkan kode verifikasi 6 digit.");
            return;
        }

        try {
            await verifyMutation.mutateAsync({ code });

            if (user) {
                if (user.phone_verified) {
                    navigate("/profile/me", { replace: true });
                } else {
                    navigate("/auth/verify-phone", { replace: true, state: { email } });
                }
            } else {
                navigate("/auth/login", { replace: true });
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 401) {
                setSessionLost(true);
            }
            setErrorMessage(getProblemMessage(error, "Permintaan tidak dapat diproses. Silakan coba lagi.", { 401: "Sesi Anda tidak terbaca. Silakan masuk kembali, lalu ulangi verifikasi.", 410: "Kode verifikasi kedaluwarsa atau sudah dipakai. Kirim ulang kode baru." }));
        }
    }

    async function handleResend() {
        if (!email) return;

        setErrorMessage("");
        setSuccessMessage("");

        try {
            await resendMutation.mutateAsync({ target: email, channel: "email" });
            setCountdown(60);
            setSuccessMessage("Kode verifikasi baru sudah dikirim ke email Anda.");
        } catch (error) {
            setErrorMessage(getProblemMessage(error, "Permintaan tidak dapat diproses. Silakan coba lagi.", { 401: "Sesi Anda tidak terbaca. Silakan masuk kembali, lalu ulangi verifikasi.", 410: "Kode verifikasi kedaluwarsa atau sudah dipakai. Kirim ulang kode baru." }));
        }
    }

    return (
        <AuthLayout>
            <section className="col-span-1 lg:col-span-2 relative flex min-h-full w-full items-center justify-center overflow-hidden px-6 py-10">
                <Blob size="lg" className="-top-24 -right-24 bg-industrial-blue-500/20" />
                <Blob size="md" className="-bottom-32 -left-24 bg-deep-navy-500/30" animate={true} />

                <div className="relative w-full max-w-xl">
                    <div className="mb-8 flex items-center flex-col gap-3">
                        <p className="mb-2 text-sm font-semibold uppercase tracking-wider text-industrial-blue-500">Verifikasi Email</p>
                        <h1 className="text-3xl font-bold text-slate-900">Cek inbox Anda</h1>
                        <p className="mt-2 leading-6 text-slate-500">Kami telah mengirimkan kode verifikasi 6 digit ke alamat email:</p>

                        <div className="mt-4 inline-flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2">
                            <LuMail className="h-4 w-4 text-industrial-blue-500" />
                            <p className="font-semibold text-slate-800">{email}</p>
                        </div>
                    </div>

                    {errorMessage ? (
                        <div className="mb-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                            <p>{errorMessage}</p>

                            {sessionLost ? (
                                <Link to="/auth/login" className="mt-2 inline-block font-bold text-red-800 underline underline-offset-2 hover:text-red-900">
                                    Masuk kembali
                                </Link>
                            ) : null}
                        </div>
                    ) : null}

                    {successMessage ? (
                        <div className="mb-6 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700" role="status" aria-live="polite">
                            {successMessage}
                        </div>
                    ) : null}

                    <div className="mb-6">
                        <label className="mb-3 block text-sm font-semibold text-slate-500">Kode Verifikasi</label>

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
                                    className="h-14 w-full rounded-xl border border-slate-300 bg-white text-center text-xl font-bold text-slate-800 outline-none transition-all focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10"
                                />
                            ))}
                        </div>
                    </div>

                    <button type="button" onClick={handleVerify} disabled={verifyMutation.isPending || !email} className="w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3 font-semibold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900 disabled:cursor-not-allowed disabled:opacity-70">
                        {verifyMutation.isPending ? "Memverifikasi..." : "Verifikasi Email"}
                    </button>

                    <div className="mt-6 text-center">
                        {countdown > 0 ? (
                            <p className="text-sm text-slate-400">
                                Kirim ulang kode dalam <span className="font-semibold text-slate-600">00:{String(countdown).padStart(2, "0")}</span>
                            </p>
                        ) : (
                            <button type="button" onClick={handleResend} disabled={resendMutation.isPending} className="inline-flex cursor-pointer items-center gap-2 text-sm font-semibold text-industrial-blue-500 hover:text-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60">
                                <LuRefreshCw className="h-4 w-4" />
                                {resendMutation.isPending ? "Mengirim..." : "Kirim ulang kode"}
                            </button>
                        )}
                    </div>

                    <div className="mt-8 flex items-center gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4">
                        <LuShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-industrial-blue-500" />

                        <p className="text-xs leading-5 text-slate-500">Jangan bagikan kode verifikasi kepada siapa pun. Devotion tidak akan pernah meminta kode ini melalui telepon atau pesan pribadi.</p>
                    </div>
                </div>
            </section>
        </AuthLayout>
    );
}
