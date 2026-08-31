import { ApiError } from "@api/client";
import Loading from "@components/common/Loading";
import { useNotificationPreferences, useUpdateNotificationPreferences } from "@hooks/useNotifications";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuInfo, LuMail, LuMessageCircle } from "react-icons/lu";

type ChannelKey = "email" | "whatsapp";

function getProblemMessage(error: unknown): string {
    if (error instanceof ApiError) {
        const data = error.data;

        if (typeof data === "object" && data !== null && "detail" in data && typeof data.detail === "string") {
            return data.detail;
        }

        if (error.status === 401) return "Sesi Anda habis, silakan masuk kembali.";
    }

    return "Preferensi tidak dapat disimpan. Silakan coba lagi.";
}

function ChannelToggle({ icon: Icon, title, description, checked, disabled, onChange }: { icon: React.ElementType; title: string; description: string; checked: boolean; disabled?: boolean; onChange: (value: boolean) => void }) {
    return (
        <div className="flex items-center justify-between gap-4 rounded-2xl border border-slate-200 bg-white p-5">
            <div className="flex items-center gap-4">
                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                    <Icon className="size-5" aria-hidden />
                </span>

                <div>
                    <h3 className="text-sm font-bold text-slate-800">{title}</h3>
                    <p className="mt-0.5 text-xs leading-5 text-slate-500">{description}</p>
                </div>
            </div>

            <button
                type="button"
                role="switch"
                aria-checked={checked}
                aria-label={title}
                disabled={disabled}
                onClick={() => onChange(!checked)}
                className={cn("relative h-7 w-12 shrink-0 cursor-pointer rounded-full transition-colors duration-200 disabled:cursor-not-allowed disabled:opacity-60", checked ? "bg-industrial-blue-500" : "bg-slate-300")}
            >
                <span className={cn("absolute top-1 size-5 rounded-full bg-white shadow transition-all duration-200", checked ? "left-6" : "left-1")} />
            </button>
        </div>
    );
}

export default function Preferences() {
    const preferencesQuery = useNotificationPreferences();
    const updateMutation = useUpdateNotificationPreferences();
    const [errorMessage, setErrorMessage] = useState("");

    const preferences = preferencesQuery.data?.non_transactional;

    function handleToggle(channel: ChannelKey, value: boolean) {
        setErrorMessage("");

        updateMutation.mutate(
            {
                non_transactional: {
                    email: channel === "email" ? value : (preferences?.email ?? true),
                    whatsapp: channel === "whatsapp" ? value : (preferences?.whatsapp ?? true),
                },
            },
            {
                onError: (error) => setErrorMessage(getProblemMessage(error)),
            },
        );
    }

    if (preferencesQuery.isLoading) return <Loading />;

    return (
        <div className="mx-auto space-y-6">
            <div>
                <h1 className="mt-3 text-xl font-bold text-slate-900">Preferensi Notifikasi</h1>
                <p className="mt-1 text-sm text-slate-500">Atur kanal pengiriman untuk notifikasi non-transaksional seperti pengingat tenggat dan permintaan ulasan.</p>
            </div>

            {errorMessage ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                    {errorMessage}
                </div>
            ) : null}

            {preferencesQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Preferensi tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : (
                <div className="space-y-3">
                    <ChannelToggle icon={LuMail} title="Email" description="Pengingat dan notifikasi non-transaksional dikirim ke alamat email akun Anda." checked={preferences?.email ?? true} disabled={updateMutation.isPending} onChange={(value) => handleToggle("email", value)} />

                    <ChannelToggle
                        icon={LuMessageCircle}
                        title="WhatsApp"
                        description="Pengingat dan notifikasi non-transaksional dikirim ke nomor WhatsApp akun Anda."
                        checked={preferences?.whatsapp ?? true}
                        disabled={updateMutation.isPending}
                        onChange={(value) => handleToggle("whatsapp", value)}
                    />
                </div>
            )}

            <div className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4">
                <LuInfo className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Notifikasi transaksional, seperti perubahan status pesanan dan kesepakatan, selalu terkirim dan tidak dapat dimatikan karena bagian dari jalannya transaksi.</p>
            </div>
        </div>
    );
}
