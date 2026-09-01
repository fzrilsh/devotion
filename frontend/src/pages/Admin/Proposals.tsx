import type { ItemProposal } from "@api/admin";
import Loading from "@components/common/Loading";
import { useDecideProposal, useItemProposals } from "@hooks/useAdmin";
import { statusFilters } from "@lib/statusFilters";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { useState } from "react";
import { LuCircleCheck, LuCog, LuInbox, LuPackage, LuX } from "react-icons/lu";
import { formatDateTimeId } from "@lib/datetime";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

const proposalStatusMeta: Record<ItemProposal["status"], { label: string; className: string }> = {
    pending: { label: "Menunggu", className: "bg-amber-500/10 text-amber-600" },
    approved: { label: "Disetujui", className: "bg-emerald-500/10 text-emerald-600" },
    rejected: { label: "Ditolak", className: "bg-red-500/10 text-red-600" },
};

const filterOptions = statusFilters(proposalStatusMeta, { allLast: true });

function ProposalCard({ proposal }: { proposal: ItemProposal }) {
    const decideMutation = useDecideProposal();
    const [rejectOpen, setRejectOpen] = useState(false);
    const [reason, setReason] = useState("");
    const [error, setError] = useState("");

    const meta = proposalStatusMeta[proposal.status];

    async function decide(decision: "approved" | "rejected") {
        setError("");

        if (decision === "rejected" && reason.trim().length < 5) {
            setError("Tuliskan alasan penolakan, minimal 5 karakter.");
            return;
        }

        try {
            await decideMutation.mutateAsync({ proposalId: proposal.proposal_id, data: { decision, reason: reason.trim() || undefined } });
            setRejectOpen(false);
            setReason("");
        } catch (err) {
            setError(getProblemMessage(err, "Keputusan tidak dapat disimpan. Silakan coba lagi."));
        }
    }

    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="flex items-start gap-4">
                <span className={cn("grid size-11 shrink-0 place-items-center rounded-xl", proposal.kind === "product" ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "bg-violet-500/10 text-violet-600")}>
                    {proposal.kind === "product" ? <LuPackage className="size-5" aria-hidden /> : <LuCog className="size-5" aria-hidden />}
                </span>

                <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                        <p className="text-sm font-bold text-slate-900">{proposal.proposed_name}</p>
                        <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                        <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-[11px] font-bold text-slate-500">{proposal.kind === "product" ? "Produk" : "Mesin"}</span>
                    </div>

                    <p className="mt-1 text-xs text-slate-400">Diusulkan {formatDateTimeId(proposal.created_at)}</p>
                </div>
            </div>

            {proposal.status !== "pending" && proposal.reason ? (
                <p className="mt-3 rounded-lg border border-slate-100 bg-slate-50 px-3 py-2 text-xs text-slate-600">Catatan: {proposal.reason}</p>
            ) : null}

            {error ? (
                <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                    {error}
                </div>
            ) : null}

            {proposal.status === "pending" ? (
                <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                    {rejectOpen ? (
                        <div className="space-y-3">
                            <label htmlFor={`reason-${proposal.proposal_id}`} className="block text-sm font-semibold text-slate-500">
                                Alasan Penolakan <span className="text-red-500">*</span>
                            </label>
                            <textarea id={`reason-${proposal.proposal_id}`} rows={2} value={reason} onChange={(event) => setReason(event.target.value)} className={inputClassName} placeholder="Misalnya: item sudah ada di daftar baku" minLength={5} maxLength={1000} />

                            <div className="flex gap-2">
                                <button type="button" onClick={() => decide("rejected")} disabled={decideMutation.isPending} className="flex-1 cursor-pointer rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60">
                                    {decideMutation.isPending ? "Menyimpan..." : "Tolak Usulan"}
                                </button>
                                <button type="button" onClick={() => setRejectOpen(false)} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                    Batal
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className="flex gap-2">
                            <button type="button" onClick={() => decide("approved")} disabled={decideMutation.isPending} className="inline-flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60">
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

export default function AdminProposals() {
    const [filter, setFilter] = useState<(typeof filterOptions)[number]["value"]>("pending");
    const proposalsQuery = useItemProposals();

    const proposals = proposalsQuery.data ?? [];
    const filtered = filter === "all" ? proposals : proposals.filter((proposal) => proposal.status === filter);

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Usulan Item</h1>
                <p className="mt-1 text-sm text-slate-500">Usulan produk dan mesin baru dari pengguna. Persetujuan langsung memasukkan item ke daftar baku.</p>
            </div>

            <div className="flex flex-wrap gap-2" role="group" aria-label="Saring status usulan">
                {filterOptions.map((option) => (
                    <button
                        key={option.value}
                        type="button"
                        onClick={() => setFilter(option.value)}
                        className={cn("cursor-pointer rounded-full border px-4 py-2 text-xs font-semibold transition", filter === option.value ? "border-industrial-blue-500 bg-industrial-blue-500 text-white" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50")}
                    >
                        {option.label}
                    </button>
                ))}
            </div>

            {proposalsQuery.isLoading ? (
                <Loading />
            ) : proposalsQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Daftar usulan tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : filtered.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Tidak ada usulan dengan status ini.</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                    {filtered.map((proposal) => (
                        <ProposalCard key={proposal.proposal_id} proposal={proposal} />
                    ))}
                </div>
            )}
        </div>
    );
}