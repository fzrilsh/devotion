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

function formatCountdown(totalSeconds: number): string {
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;

    return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

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
            navigate("/profile/me", { replace: true });
        }
    }, [email, navigate]);

    // Kirim ulang kode sekali saat halaman dibuka, jadi pengguna tidak perlu
    // menekan tombol hanya untuk menerima kode yang belum tentu sampai. Bila
    // server menolak karena rate limit, waktu tunggunya diambil dari header
    // Retry-After dan ditampilkan sebagai countdown, bukan pesan galat.
    const autoResent = useRef(false);

    useEffect(() => {
        if (!email || autoResent.current) return;

        autoResent.current = true;
        handleResend();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [email]);

    const handleChange = (value: string, index: number) => {
        const digits = value.replace(/\D/g, "");

        if (!digits) {
            // Backspace mengosongkan kotak: nilai kosong tetap harus disimpan,
            // kalau tidak state lama kembali dan digit tidak bisa dihapus.
            const cleared = [...otp];
            cleared[index] = "";
            setOtp(cleared);
            return;
        }

        if (digits.length > 1) {
            const merged = [...otp];
            let cursor = index;

            for (const digit of digits) {
                if (cursor >= otp.length) break;
                merged[cursor] = digit;
                cursor += 1;
            }

            setOtp(merged);
            inputRefs.current[Math.min(cursor, otp.length - 1)]?.focus();
            return;
        }

        const newOtp = [...otp];
        newOtp[index] = digits;

        setOtp(newOtp);

        if (index < otp.length - 1) {
            inputRefs.current[index + 1]?.focus();
        }
    };

    const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>, index: number) => {
        if (event.key === "Backspace" && !otp[index] && index > 0) {
            inputRefs.current[index - 1]?.focus();
        }

        if (event.key === "Enter" && otp.join("").length === 6) {
            handleVerify();
        }
    };

    const handlePaste = (event: React.ClipboardEvent<HTMLInputElement>, index: number) => {
        event.preventDefault();
        handleChange(event.clipboardData.getData("text"), index);
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

            if (user?.phone_verified) {
                navigate("/profile/me", { replace: true });
            } else {
                navigate("/auth/verify-phone", { replace: true });
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
            if (error instanceof ApiError && error.status === 429 && error.retryAfterSeconds) {
                setCountdown(error.retryAfterSeconds);
                return;
            }

            setErrorMessage(getProblemMessage(error, "Permintaan tidak dapat diproses. Silakan coba lagi."));
        }
    }

    return (
        <AuthLayout>
            <section className="relative col-span-1 flex min-h-screen w-full items-center justify-center overflow-hidden px-4 py-8 sm:px-6 sm:py-10 lg:col-span-2">
                <Blob size="lg" className="-top-24 -right-24 bg-industrial-blue-500/20" />
                <Blob size="md" className="-bottom-32 -left-24 bg-deep-navy-500/30" animate={true} />

                <div className="relative min-w-0 w-full max-w-xl">
                    <div className="mb-8 flex flex-col items-center gap-3 text-center">
                        <p className="mb-2 text-sm font-semibold uppercase tracking-wider text-industrial-blue-500">Verifikasi Email</p>
                        <h1 className="text-2xl font-bold text-slate-900 sm:text-3xl">Cek inbox Anda</h1>
                        <p className="mt-2 max-w-md leading-6 text-slate-500">Kami telah mengirimkan kode verifikasi 6 digit ke alamat email:</p>

                        <div className="mt-4 flex max-w-full items-center justify-center gap-2 rounded-lg bg-slate-50 px-3 py-2 text-center">
                            <LuMail className="h-4 w-4 text-industrial-blue-500" />
                            <p className="min-w-0 break-all font-semibold text-slate-800">{email}</p>
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

                        <div className="grid grid-cols-6 gap-1.5 sm:gap-3">
                            {otp.map((digit, index) => (
                                <input
                                    key={index}
                                    ref={(element) => {
                                        inputRefs.current[index] = element;
                                    }}
                                    value={digit}
                                    onChange={(event) => handleChange(event.target.value, index)}
                                    onKeyDown={(event) => handleKeyDown(event, index)}
                                    onPaste={(event) => handlePaste(event, index)}
                                    inputMode="numeric"
                                    maxLength={1}
                                    className="h-12 w-full rounded-lg border border-slate-300 bg-white text-center text-lg font-bold text-slate-800 outline-none transition-all focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10 sm:h-14 sm:rounded-xl sm:text-xl"
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
                                Kirim ulang kode dalam <span className="font-semibold text-slate-600">{formatCountdown(countdown)}</span>
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
