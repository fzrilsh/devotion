import type { WorkOrderDetail, WorkOrderStatus } from "@api/workOrders";

export const workOrderStatusMeta: Record<WorkOrderStatus, { label: string; className: string }> = {
    accepted: { label: "Diterima", className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    production: { label: "Produksi", className: "bg-violet-500/10 text-violet-600" },
    completed: { label: "Selesai", className: "bg-emerald-500/10 text-emerald-600" },
    shipped: { label: "Dikirim", className: "bg-sky-500/10 text-sky-600" },
    confirmed: { label: "Terkonfirmasi", className: "bg-emerald-600/10 text-emerald-700" },
    cancelled: { label: "Dibatalkan", className: "bg-red-500/10 text-red-600" },
    in_mediation: { label: "Mediasi", className: "bg-amber-500/10 text-amber-600" },
};

function parseSafeDate(value?: string | null): Date | null {
    if (!value) return null;

    const raw = String(value).trim();
    if (!raw) return null;

    const normalized = raw.includes("T") ? raw : raw.replace(" ", "T");
    const date = new Date(normalized);

    return Number.isNaN(date.getTime()) ? null : date;
}

export function formatDateId(isoDate?: string | null): string {
    const date = parseSafeDate(isoDate);
    if (!date) return "-";

    return new Intl.DateTimeFormat("id-ID", {
        day: "numeric",
        month: "short",
        year: "numeric",
        timeZone: "Asia/Jakarta",
    }).format(date);
}

export function formatDateTimeId(isoDate?: string | null): string {
    const date = parseSafeDate(isoDate);
    if (!date) return "-";

    return new Intl.DateTimeFormat("id-ID", {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        timeZone: "Asia/Jakarta",
    }).format(date);
}

export function formatRupiah(amount?: number | null): string {
    if (amount == null) return "-";

    return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(amount);
}

export type WorkOrderSide = "buyer" | "subcontractor" | null;

// Posisi pemanggil di dalam pesanan, diturunkan dari profile_id akun sendiri.
// Bukan mesin keadaan: tombol aksi tetap dirender dari allowed_transitions.
export function getWorkOrderSide(order: Pick<WorkOrderDetail, "buyer_profile_id" | "subcontractor_profile_id">, myProfileId?: string | null): WorkOrderSide {
    if (!myProfileId) return null;
    if (order.buyer_profile_id === myProfileId) return "buyer";
    if (order.subcontractor_profile_id === myProfileId) return "subcontractor";
    return null;
}

export const workOrderSideMeta: Record<Exclude<WorkOrderSide, null>, { label: string; className: string }> = {
    buyer: { label: "Anda pemberi order", className: "bg-violet-500/10 text-violet-600" },
    subcontractor: { label: "Anda subkontraktor", className: "bg-sky-500/10 text-sky-600" },
};
