import type { DisputeStatus } from "@api/admin";

export const disputeStatusMeta: Record<DisputeStatus, { label: string; className: string }> = {
    reported: { label: "Dilaporkan", className: "bg-red-500/10 text-red-600" },
    in_mediation: { label: "Dalam Mediasi", className: "bg-amber-500/10 text-amber-600" },
    resolved: { label: "Selesai", className: "bg-emerald-500/10 text-emerald-600" },
};
