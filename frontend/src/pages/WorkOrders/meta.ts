import type { WorkOrderStatus } from "@api/workOrders";

export const workOrderStatusMeta: Record<WorkOrderStatus, { label: string; className: string }> = {
    accepted: { label: "Diterima", className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    production: { label: "Produksi", className: "bg-violet-500/10 text-violet-600" },
    completed: { label: "Selesai", className: "bg-emerald-500/10 text-emerald-600" },
    shipped: { label: "Dikirim", className: "bg-sky-500/10 text-sky-600" },
    confirmed: { label: "Terkonfirmasi", className: "bg-emerald-600/10 text-emerald-700" },
    cancelled: { label: "Dibatalkan", className: "bg-red-500/10 text-red-600" },
    in_mediation: { label: "Mediasi", className: "bg-amber-500/10 text-amber-600" },
};

export function formatDateId(isoDate?: string | null): string {
    if (!isoDate) return "-";

    return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "Asia/Jakarta" }).format(new Date(isoDate));
}

export function formatDateTimeId(isoDate?: string | null): string {
    if (!isoDate) return "-";

    return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Jakarta" }).format(new Date(isoDate));
}

export function formatRupiah(amount?: number | null): string {
    if (amount == null) return "-";

    return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(amount);
}
