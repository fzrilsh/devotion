import type { Dispute, DisputeResult, DisputeStatus } from "@api/admin";
import Loading from "@components/common/Loading";
import PaymentMismatchNotice from "@components/common/PaymentMismatchNotice";
import { useDisputes, useMediateDispute, useResolveDispute } from "@hooks/useAdmin";
import { useWorkOrder } from "@hooks/useWorkOrders";
import { statusFilters } from "@lib/statusFilters";
import { getProblemMessage } from "@lib/problem";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuArrowRight, LuGavel, LuHandshake, LuInbox, LuMessagesSquare } from "react-icons/lu";
import { Link } from "react-router-dom";
import { disputeStatusMeta } from "./meta";
import { formatDateTimeId } from "@lib/datetime";
import { validateDisputeResolution } from "./disputeValidation";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

const resultMeta: Record<DisputeResult, { label: string; hint: string }> = {
    continued: { label: "Pesanan Dilanjutkan", hint: "Kedua pihak sepakat melanjutkan produksi." },
    confirmed: { label: "Pesanan Dinyatakan Selesai", hint: "Barang dianggap diterima; pesanan ditutup." },
    cancelled: { label: "Pesanan Dibatalkan", hint: "Alokasi kapasitas dibalik dan pihak penanggung ditetapkan." },
};

const resultOptions = (Object.entries(resultMeta) as [DisputeResult, { label: string; hint: string }][]).map(([value, meta]) => ({ value, ...meta }));

const filterOptions = statusFilters(disputeStatusMeta);

function DisputeCard({ dispute }: { dispute: Dispute }) {
    const mediateMutation = useMediateDispute();
    const resolveMutation = useResolveDispute();
    const orderQuery = useWorkOrder(dispute.work_order_id, { enabled: dispute.status !== "resolved" });

    const [resolveOpen, setResolveOpen] = useState(false);
    const [result, setResult] = useState<DisputeResult>("continued");
    const [liableProfileId, setLiableProfileId] = useState<string>("");
    const [note, setNote] = useState("");
    const [error, setError] = useState("");

    const meta = disputeStatusMeta[dispute.status];
    const busy = mediateMutation.isPending || resolveMutation.isPending;

    async function handleMediate() {
        setError("");

        try {
            await mediateMutation.mutateAsync(dispute.dispute_id);
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    async function handleResolve(event: React.FormEvent) {
        event.preventDefault();
        setError("");

        const trimmedNote = note.trim();
        const validationError = validateDisputeResolution({ result, liableProfileId, note: trimmedNote });

        if (validationError) {
            setError(validationError);
            return;
        }

        try {
            await resolveMutation.mutateAsync({
                disputeId: dispute.dispute_id,
                data: {
                    result,
                    allocation_reversed: result === "cancelled",
                    liable_profile_id: result === "cancelled" ? liableProfileId : undefined,
                    note: trimmedNote || undefined,
                },
            });
            setResolveOpen(false);
            setNote("");
            setLiableProfileId("");
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                        <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                        <p className="text-xs text-slate-400">Dilaporkan {formatDateTimeId(dispute.created_at)}</p>
                    </div>

                    <p className="mt-2 text-sm leading-6 text-slate-800">{dispute.report_body}</p>

                    <PaymentMismatchNotice mismatch={orderQuery.data?.payment_mismatch} audience="admin" className="mt-3" />
                </div>

                <Link to={`/admin/orders/${dispute.work_order_id}`} className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                    Lihat Pesanan
                    <LuArrowRight className="size-3.5" aria-hidden />
                </Link>
            </div>

            {dispute.status === "resolved" ? (
                <div className="mt-3 rounded-lg border border-emerald-100 bg-emerald-50 px-3 py-2 text-xs text-emerald-700">
                    Diputuskan {formatDateTimeId(dispute.resolved_at)}
                    {dispute.result ? ` · Hasil: ${resultMeta[dispute.result]?.label ?? dispute.result}` : ""}
                    {dispute.allocation_reversed ? " · Alokasi dibalik" : ""}
                </div>
            ) : null}

            {error ? (
                <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                    {error}
                </div>
            ) : null}

            {dispute.status !== "resolved" ? (
                <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                    {resolveOpen ? (
                        <form onSubmit={handleResolve} className="space-y-3">
                            <div className="space-y-2" role="radiogroup" aria-label="Hasil sengketa">
                                {resultOptions.map((option) => (
                                    <label key={option.value} className={cn("flex cursor-pointer items-start gap-3 rounded-xl border p-3 transition", result === option.value ? "border-industrial-blue-500 bg-industrial-blue-500/5" : "border-slate-200 hover:border-slate-300")}>
                                        <input type="radio" name={`result-${dispute.dispute_id}`} value={option.value} checked={result === option.value} onChange={() => setResult(option.value)} className="mt-1 accent-industrial-blue-500" />
                                        <span>
                                            <span className="block text-sm font-semibold text-slate-800">{option.label}</span>
                                            <span className="block text-xs text-slate-500">{option.hint}</span>
                                        </span>
                                    </label>
                                ))}
                            </div>

                            {result === "cancelled" && orderQuery.data ? (
                                <div>
                                    <label htmlFor={`liable-${dispute.dispute_id}`} className="mb-2 block text-sm font-semibold text-slate-500">
                                        Pihak yang Menanggung <span className="text-red-500">*</span>
                                    </label>
                                    <select
                                        id={`liable-${dispute.dispute_id}`}
                                        value={liableProfileId}
                                        onChange={(event) => setLiableProfileId(event.target.value)}
                                        className={inputClassName}
                                    >
                                        <option value="">-- Pilih pihak --</option>
                                        <option value={orderQuery.data.buyer_profile_id}>Pembeli</option>
                                        <option value={orderQuery.data.subcontractor_profile_id}>Subkontraktor</option>
                                    </select>
                                </div>
                            ) : null}

                            <div>
                                <label htmlFor={`note-${dispute.dispute_id}`} className="mb-2 block text-sm font-semibold text-slate-500">
                                    Catatan Keputusan {result === "cancelled" ? <span className="text-red-500"> (wajib)</span> : "(opsional)"}
                                </label>
                                <textarea
                                    id={`note-${dispute.dispute_id}`}
                                    rows={2}
                                    value={note}
                                    onChange={(event) => setNote(event.target.value)}
                                    className={inputClassName}
                                    placeholder={result === "cancelled" ? "Jelaskan alasan pembatalan dan pihak yang menanggung." : "Ringkasan hasil mediasi"}
                                    maxLength={2000}
                                    required={result === "cancelled"}
                                />
                            </div>

                            <div className="flex gap-2">
                                <button type="submit" disabled={busy || (result === "cancelled" && (!liableProfileId || !note.trim()))} className="flex-1 cursor-pointer rounded-xl bg-deep-navy-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-deep-navy-600 disabled:cursor-not-allowed disabled:opacity-60">
                                    {resolveMutation.isPending ? "Menyimpan..." : "Putuskan Sengketa"}
                                </button>
                                <button type="button" onClick={() => {
                                    setResolveOpen(false);
                                    setLiableProfileId("");
                                }} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                    Batal
                                </button>
                            </div>

                            <p className="text-[11px] leading-4 text-slate-400">Pembatalan lewat keputusan sengketa membalik seluruh alokasi kapasitas. Pilih pihak yang akan menanggung hasil pembatalan (pembeli atau subkontraktor).</p>
                        </form>
                    ) : (
                        <div className="flex flex-wrap gap-2">
                            {dispute.status === "reported" ? (
                                <button type="button" onClick={handleMediate} disabled={busy} className="inline-flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl bg-amber-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-amber-600 disabled:cursor-not-allowed disabled:opacity-60">
                                    <LuHandshake className="size-4" aria-hidden />
                                    {mediateMutation.isPending ? "Memproses..." : "Mulai Mediasi"}
                                </button>
                            ) : null}

                            <button type="button" onClick={() => setResolveOpen(true)} className="inline-flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl bg-deep-navy-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-deep-navy-600">
                                <LuGavel className="size-4" aria-hidden />
                                Putuskan
                            </button>
                        </div>
                    )}
                </div>
            ) : null}
        </div>
    );
}

export default function AdminDisputes() {
    const [status, setStatus] = useState<DisputeStatus | "all">("all");
    const disputesQuery = useDisputes(status === "all" ? undefined : status);
    const disputes = disputesQuery.data?.pages.flatMap((page) => page.items) ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Sengketa</h1>
                <p className="mt-1 text-sm text-slate-500">Mediasi laporan sengketa pesanan dan putuskan hasilnya. Konfirmasi otomatis berhenti selama sengketa berjalan.</p>
            </div>

            <div className="flex flex-wrap gap-2" role="group" aria-label="Saring status sengketa">
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

            {disputesQuery.isLoading ? (
                <Loading />
            ) : disputesQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Antrean sengketa tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : disputes.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Tidak ada sengketa dengan status ini.</p>
                </div>
            ) : (
                <>
                    <div className="space-y-4">
                        {disputes.map((dispute) => (
                            <DisputeCard key={dispute.dispute_id} dispute={dispute} />
                        ))}
                    </div>

                    {disputesQuery.hasNextPage ? (
                        <div className="text-center">
                            <button type="button" onClick={() => disputesQuery.fetchNextPage()} disabled={disputesQuery.isFetchingNextPage} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                                {disputesQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                            </button>
                        </div>
                    ) : null}
                </>
            )}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuMessagesSquare className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Hasil sengketa menentukan nasib pesanan dan tingkat penyelesaian kedua pihak. Pembatalan hanya membebani pihak yang ditetapkan menanggung.</p>
            </div>
        </div>
    );
}
