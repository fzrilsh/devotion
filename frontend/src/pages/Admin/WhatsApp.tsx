import type { WhatsAppStatus } from "@api/admin";
import Loading from "@components/common/Loading";
import { useReconnectWhatsApp, useWhatsAppStatus } from "@hooks/useAdmin";
import QRCode from "qrcode";
import { useEffect, useState } from "react";
import { LuMessageCircle, LuPlug, LuQrCode, LuRefreshCw, LuTriangleAlert } from "react-icons/lu";
import { getProblemMessage } from "@lib/problem";

type SessionState = "connected" | "pairing" | "disconnected";

function sessionState(status: WhatsAppStatus): SessionState {
    if (status.connected) return "connected";

    return status.qr_code?.trim() ? "pairing" : "disconnected";
}

const statusMeta: Record<SessionState, { title: string; body: string; card: string; icon: string; titleColor: string; bodyColor: string }> = {
    connected: {
        title: "Sesi terhubung",
        body: "Kode verifikasi dan notifikasi WhatsApp terkirim normal.",
        card: "border-emerald-200 bg-emerald-50",
        icon: "bg-emerald-500/15 text-emerald-600",
        titleColor: "text-emerald-800",
        bodyColor: "text-emerald-700",
    },
    pairing: {
        title: "Menunggu pemindaian",
        body: "Siklus pemasangan sedang berjalan. Pindai kode QR di bawah dari perangkat layanan sebelum siklusnya berakhir.",
        card: "border-amber-200 bg-amber-50",
        icon: "bg-amber-500/15 text-amber-600",
        titleColor: "text-amber-800",
        bodyColor: "text-amber-700",
    },
    disconnected: {
        title: "Sesi terputus",
        body: "Kode verifikasi dan notifikasi WhatsApp tidak terkirim sampai sesi ditautkan ulang. Notifikasi email tetap berjalan.",
        card: "border-red-200 bg-red-50",
        icon: "bg-red-500/15 text-red-600",
        titleColor: "text-red-800",
        bodyColor: "text-red-700",
    },
};

export default function AdminWhatsApp() {
    const statusQuery = useWhatsAppStatus();
    const reconnect = useReconnectWhatsApp();
    const status = statusQuery.data;
    const [qrResult, setQrResult] = useState<{ value: string; image: string | null; error: boolean } | null>(null);
    const [reconnectError, setReconnectError] = useState("");

    useEffect(() => {
        let cancelled = false;
        const qrValue = status?.qr_code?.trim();

        if (!qrValue || qrValue.startsWith("data:image/")) {
            return () => {
                cancelled = true;
            };
        }

        QRCode.toDataURL(qrValue, {
            errorCorrectionLevel: "M",
            margin: 2,
            width: 448,
            color: { dark: "#102a43", light: "#ffffff" },
        })
            .then((dataUrl) => {
                if (!cancelled) setQrResult({ value: qrValue, image: dataUrl, error: false });
            })
            .catch(() => {
                if (!cancelled) setQrResult({ value: qrValue, image: null, error: true });
            });

        return () => {
            cancelled = true;
        };
    }, [status?.qr_code]);

    const qrValue = status?.qr_code?.trim();
    const qrImage = qrValue?.startsWith("data:image/") ? qrValue : qrResult?.value === qrValue ? qrResult?.image : null;
    const qrError = qrResult?.value === qrValue && Boolean(qrResult?.error);
    const state = status ? sessionState(status) : null;
    const meta = state ? statusMeta[state] : null;

    async function handleReconnect() {
        setReconnectError("");

        try {
            await reconnect.mutateAsync();
        } catch (err) {
            setReconnectError(getProblemMessage(err, "Sesi tidak dapat disambungkan ulang. Coba lagi beberapa saat lagi."));
        }
    }

    return (
        <div className="mx-auto space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">WhatsApp</h1>
                    <p className="mt-1 text-sm text-slate-500">Status sesi WhatsApp yang mengirim kode verifikasi dan notifikasi ke pengguna.</p>
                </div>

                <div className="flex shrink-0 flex-wrap items-center gap-2">
                    <button type="button" onClick={() => statusQuery.refetch()} disabled={statusQuery.isFetching} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" aria-label="Muat ulang status">
                        <LuRefreshCw className={statusQuery.isFetching ? "size-4 animate-spin" : "size-4"} aria-hidden />
                        Muat Ulang
                    </button>

                    <button type="button" onClick={handleReconnect} disabled={reconnect.isPending} className="inline-flex cursor-pointer items-center gap-2 rounded-xl bg-deep-navy-500 px-4 py-2 text-sm font-semibold text-white transition hover:bg-deep-navy-600 disabled:cursor-not-allowed disabled:opacity-60">
                        <LuPlug className={reconnect.isPending ? "size-4 animate-pulse" : "size-4"} aria-hidden />
                        {reconnect.isPending ? "Menyambungkan..." : "Sambungkan Ulang"}
                    </button>
                </div>
            </div>

            <p className="text-xs leading-5 text-slate-500">Muat Ulang hanya membaca status. Sambungkan Ulang memulai siklus pemasangan baru sehingga kode QR-nya segar; bila sesi masih terpasang, soketnya disambungkan ulang tanpa menghapus tautan.</p>

            {reconnectError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                    {reconnectError}
                </div>
            ) : null}

            {statusQuery.isLoading ? (
                <Loading />
            ) : statusQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Status WhatsApp tidak dapat dimuat. Coba muat ulang.</p>
                </div>
            ) : status && meta ? (
                <div className="space-y-4">
                    <div className={`flex items-start gap-4 rounded-2xl border p-6 ${meta.card}`}>
                        <span className={`grid size-12 shrink-0 place-items-center rounded-xl ${meta.icon}`}>
                            <LuMessageCircle className="size-6" aria-hidden />
                        </span>

                        <div>
                            <p className={`text-sm font-bold ${meta.titleColor}`}>{meta.title}</p>
                            <p className={`mt-1 text-sm leading-6 ${meta.bodyColor}`}>{meta.body}</p>
                        </div>
                    </div>

                    {status.last_error ? (
                        <div className="flex items-start gap-3 rounded-2xl border border-amber-200 bg-amber-50 p-4">
                            <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden />
                            <div>
                                <p className="text-sm font-semibold text-amber-800">Galat terakhir</p>
                                <p className="mt-1 text-xs leading-5 text-amber-700">{status.last_error}</p>
                            </div>
                        </div>
                    ) : null}

                    {status.qr_code ? (
                        <div className="rounded-2xl border border-slate-200 bg-white p-6 text-center">
                            <div className="flex items-center justify-center gap-2">
                                <LuQrCode className="size-5 text-slate-500" aria-hidden />
                                <h2 className="text-sm font-bold text-slate-800">Tautkan ulang sesi</h2>
                            </div>

                            <p className="mt-2 text-xs leading-5 text-slate-500">Pindai kode QR ini dari aplikasi WhatsApp pada perangkat layanan (menu Perangkat Tertaut).</p>

                            {qrImage ? <img src={qrImage} alt="Kode QR penautan sesi WhatsApp" className="mx-auto mt-4 size-56 rounded-xl border border-slate-200" /> : qrError ? <p className="mt-4 text-sm text-red-600">Kode QR tidak dapat dibuat. Muat ulang status untuk mencoba lagi.</p> : <p className="mt-4 text-sm text-slate-500">Menyiapkan kode QR...</p>}

                            <p className="mt-3 text-[11px] text-slate-400">Halaman ini memuat ulang status tiap 30 detik. Bila kode QR-nya kedaluwarsa, tekan Sambungkan Ulang.</p>
                        </div>
                    ) : state === "disconnected" ? (
                        <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-6 text-center">
                            <LuQrCode className="mx-auto size-8 text-slate-300" aria-hidden />
                            <p className="mt-2 text-sm text-slate-500">Belum ada kode QR. Tekan Sambungkan Ulang untuk memulai siklus pemasangan.</p>
                        </div>
                    ) : null}
                </div>
            ) : null}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuMessageCircle className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Nomor layanan tidak pernah ditampilkan di sini demi keamanan. Saat sesi terputus, platform tetap melayani; hanya kanal WhatsApp yang terganggu.</p>
            </div>
        </div>
    );
}
