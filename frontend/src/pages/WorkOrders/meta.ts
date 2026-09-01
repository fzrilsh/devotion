import type { WorkOrderDetail, WorkOrderStatus } from "@api/workOrders";

// Pemformat tanggal dan rupiah tinggal di @lib/datetime; direkspor di sini supaya
// halaman yang sudah mengimpor dari modul meta ini tidak perlu diubah.
export { formatDateId, formatDateTimeId, formatRupiah } from "@lib/datetime";

export const workOrderStatusMeta: Record<WorkOrderStatus, { label: string; className: string }> = {
    accepted: { label: "Diterima", className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    production: { label: "Produksi", className: "bg-violet-500/10 text-violet-600" },
    completed: { label: "Selesai", className: "bg-emerald-500/10 text-emerald-600" },
    shipped: { label: "Dikirim", className: "bg-sky-500/10 text-sky-600" },
    confirmed: { label: "Terkonfirmasi", className: "bg-emerald-600/10 text-emerald-700" },
    cancelled: { label: "Dibatalkan", className: "bg-red-500/10 text-red-600" },
    in_mediation: { label: "Mediasi", className: "bg-amber-500/10 text-amber-600" },
};

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

// Transisi yang masuk akal untuk masing-masing pihak. Produksi, selesai, dan
// dikirim dilaporkan pihak yang mengerjakan; konfirmasi penerimaan hanya milik
// pihak yang menerima barang. Pembatalan dan sengketa terbuka untuk keduanya.
//
// Ini penyaring tampilan di atas allowed_transitions, bukan mesin keadaan kedua:
// daftar transisi tetap datang dari backend, di sini hanya dibuang yang bukan
// urusan pihak yang sedang melihat halaman. Kalau backend mempersempit lebih
// jauh, hasil penyaringan menyempit ikut, dan tidak ada tombol yang muncul
// hanya karena tabel ini menyebutnya.
const sideTransitions: Record<Exclude<WorkOrderSide, null>, readonly WorkOrderStatus[]> = {
    buyer: ["confirmed", "cancelled", "in_mediation"],
    subcontractor: ["production", "completed", "shipped", "cancelled", "in_mediation"],
};

export function transitionsForSide(transitions: readonly WorkOrderStatus[], side: WorkOrderSide): WorkOrderStatus[] {
    if (!side) return [];

    return transitions.filter((status) => sideTransitions[side].includes(status));
}

// Arah pernyataan pembayaran mengikuti posisi pihak: pemberi order menyatakan
// sudah membayar, subkontraktor menyatakan sudah menerima. Deteksi
// ketidakcocokan (FR-043) memang membandingkan pernyataan kedua pihak, jadi satu
// pihak tidak perlu memilih arah lawannya.
export const paymentDirectionForSide: Record<Exclude<WorkOrderSide, null>, "sent" | "received"> = {
    buyer: "sent",
    subcontractor: "received",
};

export const paymentDirectionLabel = { sent: "Saya sudah membayar", received: "Saya sudah menerima pembayaran" } as const;
