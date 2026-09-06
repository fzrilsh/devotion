import type { CandidateStatus } from "@api/search";

export const candidateStatusMeta: Record<CandidateStatus, { label: string; className: string }> = {
    awaiting_reply: { label: "Menunggu Balasan", className: "bg-amber-500/10 text-amber-600" },
    offered: { label: "Ada Penawaran", className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    rejected: { label: "Ditolak", className: "bg-red-500/10 text-red-600" },
    expired: { label: "Kedaluwarsa", className: "bg-slate-200 text-slate-500" },
    not_continued: { label: "Tidak Dilanjutkan", className: "bg-slate-200 text-slate-500" },
    agreed: { label: "Sepakat", className: "bg-emerald-500/10 text-emerald-600" },
};

export { formatDateId as formatDateShort, formatRupiah } from "@lib/datetime";
