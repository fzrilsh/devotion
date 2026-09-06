import { useAccountVerification } from "@hooks/useAccountVerification";
import { useAuth } from "@hooks/useAuth";
import { LuMail, LuPhone, LuShieldAlert } from "react-icons/lu";
import { Link } from "react-router-dom";

export default function VerificationGate({ action }: { action: string }) {
    const { user } = useAuth();
    const { needsEmail, needsPhone } = useAccountVerification();

    const channels = [
        {
            key: "email" as const,
            icon: LuMail,
            label: "Email",
            value: user?.email,
            missing: needsEmail,
            to: "/auth/verify-email",
            state: { email: user?.email },
        },
        {
            key: "phone" as const,
            icon: LuPhone,
            label: "Nomor WhatsApp",
            value: user?.phone,
            missing: needsPhone,
            to: "/auth/verify-phone",
            state: { email: user?.email },
        },
    ];

    return (
        <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6" role="alert">
            <div className="flex items-start gap-3">
                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-amber-100 text-amber-600">
                    <LuShieldAlert className="size-5" aria-hidden />
                </span>

                <div>
                    <h2 className="text-sm font-bold text-amber-800">Verifikasi akun diperlukan</h2>
                    <p className="mt-1 text-sm leading-6 text-amber-700">
                        Sebelum {action}, verifikasi {needsEmail && needsPhone ? "email dan nomor WhatsApp" : needsEmail ? "email" : "nomor WhatsApp"} Anda terlebih dulu. Kode verifikasi dikirim ke masing-masing kanal.
                    </p>
                </div>
            </div>

            <ul className="mt-4 space-y-2">
                {channels.map((channel) => (
                    <li key={channel.key} className="flex items-center justify-between gap-4 rounded-xl border border-amber-200 bg-white px-4 py-3">
                        <span className="flex min-w-0 items-center gap-3">
                            <channel.icon className="size-4.5 shrink-0 text-slate-400" aria-hidden />
                            <span className="min-w-0">
                                <span className="block text-xs font-bold text-slate-700">{channel.label}</span>
                                <span className="block truncate text-xs text-slate-400">{channel.value || "Belum diisi"}</span>
                            </span>
                        </span>

                        {channel.missing ? (
                            <Link to={channel.to} state={channel.state} className="shrink-0 rounded-lg bg-industrial-blue-500 px-4 py-2 text-xs font-bold text-white transition hover:bg-industrial-blue-600">
                                Verifikasi
                            </Link>
                        ) : (
                            <span className="shrink-0 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700">Terverifikasi</span>
                        )}
                    </li>
                ))}
            </ul>
        </div>
    );
}
