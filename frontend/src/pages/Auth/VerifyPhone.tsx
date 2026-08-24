import Blob from "@components/common/Blob";
import AuthLayout from "@components/layout/AuthLayout";
import { useRef, useState } from "react";
import { LuPhone, LuRefreshCw, LuShieldCheck } from "react-icons/lu";

export default function VerifyEmail() {
    const [otp, setOtp] = useState(["", "", "", "", "", ""]);
    const [countdown, setCountdown] = useState(60);
    const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

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

    return (
        <AuthLayout>
            <section className="col-span-1 lg:col-span-2 relative flex min-h-full w-full items-center justify-center overflow-hidden px-6 py-10">
                <Blob size="lg" className="-top-24 -right-24 bg-industrial-orange-500/20" />
                <Blob size="md" className="-bottom-32 -left-24 bg-deep-navy-500/30" animate={true} />

                <div className="relative w-full max-w-xl">
                    <div className="mb-8 flex items-center flex-col gap-3">
                        <p className="mb-2 text-sm font-semibold uppercase tracking-wider text-industrial-orange-500">Verifikasi Nomor HP</p>

                        <h1 className="text-3xl font-bold text-slate-900">Satu langkah lagi</h1>

                        <p className="mt-2 leading-6 text-slate-500">Masukkan kode verifikasi yang kami kirimkan melalui WhatsApp ke nomor:</p>

                        <div className="mt-4 inline-flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2">
                            <LuPhone className="h-4 w-4 text-industrial-orange-500" />
                            <span className="text-sm font-bold text-slate-800">+62 812-****-7890</span>
                        </div>
                    </div>

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
                                    className="h-14 w-full rounded-xl border border-slate-300 bg-white text-center text-xl font-bold text-slate-800 outline-none transition-all focus:border-industrial-orange-500 focus:ring-2 focus:ring-industrial-orange-500/10"
                                />
                            ))}
                        </div>
                    </div>

                    <button type="button" className="w-full cursor-pointer rounded-xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 px-4 py-3 font-semibold text-white transition-all duration-200 hover:from-deep-navy-600 hover:to-deep-navy-900">
                        Verifikasi Nomor HP
                    </button>

                    <div className="mt-6 text-center">
                        <p className="text-sm text-slate-400">Tidak menerima Pesan?</p>

                        <button type="button" className="mt-2 inline-flex cursor-pointer items-center gap-2 text-sm font-semibold text-industrial-orange-500 hover:text-industrial-orange-600">
                            <LuRefreshCw className="h-4 w-4" />
                            Kirim ulang kode
                        </button>
                    </div>

                    <div className="mt-8 flex gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4">
                        <LuShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-industrial-orange-500" />
                        <p className="text-xs leading-5 text-slate-500">Kode verifikasi bersifat rahasia. Jangan berikan kode ini kepada orang lain, termasuk pihak yang mengaku sebagai Devotion.</p>
                    </div>
                </div>
            </section>
        </AuthLayout>
    );
}
