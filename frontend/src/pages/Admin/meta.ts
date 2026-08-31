import type { DisputeStatus } from "@api/admin";

// Peta label status sengketa dipakai halaman Sengketa dan Dasbor Admin. Satu
// definisi supaya satu sengketa tidak berlabel berbeda tergantung halaman
// pembukanya, dan supaya nilai enum baru hanya perlu ditambah sekali.
export const disputeStatusMeta: Record<DisputeStatus, { label: string; className: string }> = {
    reported: { label: "Dilaporkan", className: "bg-red-500/10 text-red-600" },
    in_mediation: { label: "Dalam Mediasi", className: "bg-amber-500/10 text-amber-600" },
    resolved: { label: "Selesai", className: "bg-emerald-500/10 text-emerald-600" },
};
