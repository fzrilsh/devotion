import { apiUrl } from "@api/client";
import type { VerificationRequest, VerificationStatus } from "@api/admin";
import Loading from "@components/common/Loading";
import { useDecideVerification, useVerificationQueue } from "@hooks/useAdmin";
import { statusFilters } from "@lib/statusFilters";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { useState } from "react";
import { LuCircleCheck, LuFileImage, LuInbox, LuShieldCheck, LuX } from "react-icons/lu";
import { formatDateTimeId } from "@lib/datetime";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

const statusMeta: Record<VerificationStatus, { label: string; className: string }> = {
    pending: { label: "Menunggu", className: "bg-amber-500/10 text-amber-600" },
    approved: { label: "Disetujui", className: "bg-emerald-500/10 text-emerald-600" },
    rejected: { label: "Ditolak", className: "bg-red-500/10 text-red-600" },
};

const filterOptions = statusFilters(statusMeta, { allLast: true });

function RequestCard({ request }: { request: VerificationRequest }) {
    const decideMutation = useDecideVerification();
    const [rejectOpen, setRejectOpen] = useState(false);
    const [reason, setReason] = useState("");
    const [error, setError] = useState("");

    const meta = statusMeta[request.status];

    async function decide(decision: "approved" | "rejected") {
        setError("");

        if (decision === "rejected" && reason.trim().length < 5) {
            setError("Tuliskan alasan penolakan minimal 5 karakter dan maksimal 1000 karakter. Alasan ini tampil ke pemohon.");
            return;
        }

        if (decision === "rejected" && reason.trim().length > 1000) {
            setError("Alasan penolakan maksimal 1000 karakter.");
            return;
        }

        try {
            await decideMutation.mutateAsync({ requestId: request.request_id, data: { decision, reason: reason.trim() || undefined } });
            setRejectOpen(false);
            setReason("");
        } catch (err) {
            setError(getProblemMessage(err, "Keputusan tidak dapat disimpan. Silakan coba lagi."));
        }
    }

    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                        <p className="text-sm font-bold text-slate-900">{request.business_name || "Usaha tanpa nama"}</p>
                        <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                    </div>

                    <p className="mt-1 text-xs text-slate-400">
                        Diajukan {formatDateTimeId(request.submitted_at)}
                        {request.decided_at ? ` · Diputuskan ${formatDateTimeId(request.decided_at)}` : ""}
                    </p>

                    {request.identity_number ? <p className="mt-1 text-xs text-slate-500">Nomor identitas: {request.identity_number}</p> : null}
                </div>

                <div className="flex shrink-0 gap-2">
                    {request.identity_file_id ? (
                        <a href={apiUrl(`/files/${request.identity_file_id}`)} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                            <LuFileImage className="size-3.5" aria-hidden />
                            Dokumen
                        </a>
                    ) : null}

                    {request.location_file_id ? (
                        <a href={apiUrl(`/files/${request.location_file_id}`)} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                            <LuFileImage className="size-3.5" aria-hidden />
                            Foto Lokasi
                        </a>
                    ) : null}
                </div>
            </div>

            {request.status === "rejected" && request.reason ? <p className="mt-3 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-xs text-red-700">Alasan penolakan: {request.reason}</p> : null}

            {error ? (
                <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                    {error}
                </div>
            ) : null}

            {request.status === "pending" ? (
                <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                    {rejectOpen ? (
                        <div className="space-y-3">
                            <label htmlFor={`reason-${request.request_id}`} className="block text-sm font-semibold text-slate-500">
                                Alasan Penolakan <span className="text-red-500">*</span>
                            </label>
                            <textarea id={`reason-${request.request_id}`} rows={3} value={reason} onChange={(event) => setReason(event.target.value)} className={inputClassName} placeholder="Misalnya: dokumen identitas tidak terbaca" minLength={5} maxLength={1000} />

                            <div className="flex gap-2">
                                <button
                                    type="button"
                                    onClick={() => decide("rejected")}
                                    disabled={decideMutation.isPending}
                                    className="flex-1 cursor-pointer rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60"
                                >
                                    {decideMutation.isPending ? "Menyimpan..." : "Tolak Pengajuan"}
                                </button>
                                <button type="button" onClick={() => setRejectOpen(false)} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                    Batal
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className="flex gap-2">
                            <button
                                type="button"
                                onClick={() => decide("approved")}
                                disabled={decideMutation.isPending}
                                className="inline-flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60"
                            >
                                <LuCircleCheck className="size-4" aria-hidden />
                                {decideMutation.isPending ? "Menyimpan..." : "Setujui"}
                            </button>
                            <button type="button" onClick={() => setRejectOpen(true)} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-red-300 bg-white px-4 py-2.5 text-sm font-semibold text-red-600 transition hover:bg-red-50">
                                <LuX className="size-4" aria-hidden />
                                Tolak
                            </button>
                        </div>
                    )}
                </div>
            ) : null}
        </div>
    );
}

export default function AdminVerificationQueue() {
    const [status, setStatus] = useState<VerificationStatus | "all">("pending");
    const queueQuery = useVerificationQueue(status === "all" ? undefined : status);
    const items = queueQuery.data?.pages.flatMap((page) => page.items) ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Antrean Verifikasi</h1>
                <p className="mt-1 text-sm text-slate-500">Tinjau pengajuan verifikasi identitas usaha. Penolakan wajib menyertakan alasan yang tampil ke pemohon.</p>
            </div>

            <div className="flex flex-wrap gap-2" role="group" aria-label="Saring status pengajuan">
                {filterOptions.map((option) => (
                    <button
                        key={option.value}
                        type="button"
                        onClick={() => setStatus(option.value)}
                        className={cn("cursor-pointer rounded-full border px-4 py-2 text-xs font-semibold transition", status === option.value ? "border-industrial-blue-500 bg-industrial-blue-500 text-white" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50")}
                    >
                        {option.label}
                    </button>
                ))}
            </div>

            {queueQuery.isLoading ? (
                <Loading />
            ) : queueQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Antrean verifikasi tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : items.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Tidak ada pengajuan dengan status ini.</p>
                </div>
            ) : (
                <>
                    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
                        {items.map((request) => (
                            <RequestCard key={request.request_id} request={request} />
                        ))}
                    </div>

                    {queueQuery.hasNextPage ? (
                        <div className="text-center">
                            <button type="button" onClick={() => queueQuery.fetchNextPage()} disabled={queueQuery.isFetchingNextPage} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                                {queueQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                            </button>
                        </div>
                    ) : null}
                </>
            )}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuShieldCheck className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Dokumen identitas hanya dapat dibuka oleh pemilik dan admin. Verifikasi menambah lencana pada profil dan tidak memengaruhi urutan hasil pencarian.</p>
            </div>
        </div>
    );
}
