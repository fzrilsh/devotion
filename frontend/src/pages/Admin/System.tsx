import { ApiError } from "@api/client";
import type { Health } from "@api/system";
import Loading from "@components/common/Loading";
import { useHealth } from "@hooks/useSystem";
import { cn } from "@lib/utils";
import { LuActivity, LuDatabase, LuHardDrive, LuMessageCircle, LuRefreshCw, LuTriangleAlert } from "react-icons/lu";
import { Link } from "react-router-dom";

const overallMeta: Record<Health["status"], { label: string; note: string; card: string; badge: string }> = {
    ok: { label: "Sehat", note: "Seluruh ketergantungan berjalan normal.", card: "border-emerald-200 bg-emerald-50", badge: "bg-emerald-500/15 text-emerald-700" },
    degraded: { label: "Terganggu", note: "Ada ketergantungan yang tidak sehat. Rincian per komponen ada di bawah.", card: "border-amber-200 bg-amber-50", badge: "bg-amber-500/15 text-amber-700" },
};

const databaseMeta = {
    ok: { label: "Normal", detail: "Basis data menerima kueri.", tone: "ok" as const },
    fail: { label: "Gagal", detail: "Basis data tidak dapat dihubungi. Instance dibalas 503 dan layak ditarik dari rotasi.", tone: "bad" as const },
};

const whatsappMeta = {
    connected: { label: "Terhubung", detail: "Kode verifikasi dan notifikasi WhatsApp terkirim.", tone: "ok" as const },
    disconnected: { label: "Terputus", detail: "Kanal WhatsApp mati sampai sesi ditautkan ulang lewat halaman WhatsApp. Notifikasi email tetap jalan.", tone: "warn" as const },
};

const storageMeta = {
    ok: { label: "Longgar", detail: "Ruang unggahan masih cukup.", tone: "ok" as const },
    near_full: { label: "Hampir penuh", detail: "Ruang unggahan mendekati batas. Bersihkan berkas lama sebelum unggahan mulai ditolak.", tone: "warn" as const },
    full: { label: "Penuh", detail: "Ruang unggahan habis. Unggahan ditolak dan instance dibalas 503.", tone: "bad" as const },
};

const toneClass = {
    ok: { card: "border-slate-200 bg-white", icon: "bg-emerald-500/10 text-emerald-600", badge: "bg-emerald-500/10 text-emerald-700" },
    warn: { card: "border-amber-200 bg-amber-50/60", icon: "bg-amber-500/10 text-amber-600", badge: "bg-amber-500/10 text-amber-700" },
    bad: { card: "border-red-200 bg-red-50/60", icon: "bg-red-500/10 text-red-600", badge: "bg-red-500/10 text-red-700" },
};

function formatDateTimeId(isoDate?: string | null): string {
    if (!isoDate) return "-";

    return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Jakarta" }).format(new Date(isoDate));
}

function DependencyCard({ title, icon: Icon, label, detail, tone, children }: { title: string; icon: React.ElementType; label: string; detail: string; tone: "ok" | "warn" | "bad"; children?: React.ReactNode }) {
    const classes = toneClass[tone];

    return (
        <div className={cn("rounded-2xl border p-5", classes.card)}>
            <div className="flex items-start justify-between gap-3">
                <span className={cn("grid size-10 shrink-0 place-items-center rounded-xl", classes.icon)}>
                    <Icon className="size-5" aria-hidden />
                </span>

                <span className={cn("shrink-0 rounded-full px-3 py-1 text-[11px] font-bold", classes.badge)}>{label}</span>
            </div>

            <h3 className="mt-4 text-sm font-bold text-slate-800">{title}</h3>
            <p className="mt-1 text-xs leading-5 text-slate-500">{detail}</p>

            {children}
        </div>
    );
}

// 503 justru keadaan yang paling perlu tampil, tetapi apiClient melemparkan galat
// pada respons non-2xx sehingga badannya tidak terbaca. Kegagalan permintaan karena
// itu dibaca sebagai instance tidak sehat, bukan sekadar galat jaringan.
function UnhealthyCard({ error }: { error: unknown }) {
    const status = error instanceof ApiError ? error.status : null;

    return (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6">
            <div className="flex items-start gap-3">
                <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-red-600" aria-hidden />

                <div>
                    <p className="text-sm font-bold text-red-800">{status === 503 ? "Instance tidak dapat melayani" : "Status sistem tidak dapat dibaca"}</p>
                    <p className="mt-1 text-xs leading-5 text-red-700">
                        {status === 503
                            ? "Balasan 503 hanya dipicu basis data gagal atau penyimpanan penuh. Periksa log kontainer dan sisa disk VPS sebelum menyalakan ulang."
                            : "Permintaan health gagal, jadi keadaan ketergantungan tidak diketahui. Bila ini berulang, periksa apakah proses backend masih hidup."}
                    </p>
                </div>
            </div>
        </div>
    );
}

export default function AdminSystem() {
    const healthQuery = useHealth();
    const health = healthQuery.data;
    const meta = health ? overallMeta[health.status] : null;

    const storage = health?.dependencies.storage;
    const storagePercent = storage && storage.limit_mb > 0 ? Math.min(100, Math.round((storage.used_mb / storage.limit_mb) * 100)) : null;

    return (
        <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">Status Sistem</h1>
                    <p className="mt-1 text-sm text-slate-500">Kesehatan instance beserta ketergantungannya: basis data, kanal WhatsApp, dan ruang unggahan.</p>
                </div>

                <button type="button" onClick={() => healthQuery.refetch()} disabled={healthQuery.isFetching} className="inline-flex shrink-0 cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" aria-label="Muat ulang status sistem">
                    <LuRefreshCw className={healthQuery.isFetching ? "size-4 animate-spin" : "size-4"} aria-hidden />
                    Muat Ulang
                </button>
            </div>

            {healthQuery.isLoading ? (
                <Loading />
            ) : healthQuery.isError ? (
                <UnhealthyCard error={healthQuery.error} />
            ) : health && meta ? (
                <div className="space-y-4">
                    <div className={cn("flex flex-wrap items-center justify-between gap-3 rounded-2xl border p-6", meta.card)}>
                        <div className="flex items-start gap-4">
                            <span className={cn("grid size-12 shrink-0 place-items-center rounded-xl", meta.badge)}>
                                <LuActivity className="size-6" aria-hidden />
                            </span>

                            <div>
                                <p className="text-sm font-bold text-slate-800">Instance {meta.label}</p>
                                <p className="mt-1 text-sm leading-6 text-slate-600">{meta.note}</p>
                            </div>
                        </div>

                        <div className="text-right text-xs text-slate-500">
                            <p>Diperiksa {formatDateTimeId(health.time)}</p>
                            {health.version ? <p className="mt-0.5">Versi {health.version}</p> : null}
                        </div>
                    </div>

                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                        <DependencyCard title="Basis Data" icon={LuDatabase} label={databaseMeta[health.dependencies.database].label} detail={databaseMeta[health.dependencies.database].detail} tone={databaseMeta[health.dependencies.database].tone} />

                        <DependencyCard title="Kanal WhatsApp" icon={LuMessageCircle} label={whatsappMeta[health.dependencies.whatsapp].label} detail={whatsappMeta[health.dependencies.whatsapp].detail} tone={whatsappMeta[health.dependencies.whatsapp].tone}>
                            {health.dependencies.whatsapp === "disconnected" ? (
                                <Link to="/admin/whatsapp" className="mt-3 inline-flex items-center gap-1.5 text-xs font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                                    Buka halaman WhatsApp
                                </Link>
                            ) : null}
                        </DependencyCard>

                        <DependencyCard title="Ruang Unggahan" icon={LuHardDrive} label={storageMeta[storage?.status ?? "ok"].label} detail={storageMeta[storage?.status ?? "ok"].detail} tone={storageMeta[storage?.status ?? "ok"].tone}>
                            {storage ? (
                                <div className="mt-3">
                                    <div className="flex items-center justify-between text-[11px] font-semibold text-slate-500">
                                        <span>
                                            {storage.used_mb.toLocaleString("id-ID")} MB dari {storage.limit_mb.toLocaleString("id-ID")} MB
                                        </span>
                                        {storagePercent != null ? <span>{storagePercent}%</span> : null}
                                    </div>

                                    {storagePercent != null ? (
                                        <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-slate-200" role="img" aria-label={`Ruang unggahan terpakai ${storagePercent} persen`}>
                                            <div className={cn("h-full rounded-full", storage.status === "full" ? "bg-red-500" : storage.status === "near_full" ? "bg-amber-500" : "bg-emerald-500")} style={{ width: `${storagePercent}%` }} />
                                        </div>
                                    ) : null}
                                </div>
                            ) : null}
                        </DependencyCard>
                    </div>
                </div>
            ) : null}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuActivity className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">
                    WhatsApp terputus tidak menjatuhkan instance: pemulihannya menuntut pemindaian kode QR manual, jadi menyalakan ulang kontainer tidak menolong. Hanya basis data gagal atau penyimpanan penuh yang membuat instance layak ditarik dari rotasi.
                </p>
            </div>
        </div>
    );
}
