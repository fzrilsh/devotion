import type { Notification } from "@api/notifications";

export type NotificationLink = { to: string; label: string };

// Rute yang hanya ada di bawah gerbang peran pemberi order. Subkontraktor yang
// membukanya dipantulkan ke profilnya, jadi tautan ke sana bukan tautan mati
// tapi tautan menyesatkan: labelnya menjanjikan halaman yang tidak pernah tampil.
const buyerOnlyPrefixes = ["/quota-requests", "/search"];

function isBuyerOnlyPath(path: string): boolean {
    return buyerOnlyPrefixes.some((prefix) => path === prefix || path.startsWith(`${prefix}/`) || path.startsWith(`${prefix}?`));
}

// Notification membawa link generik ke sumber kejadiannya. Bila tujuannya tidak
// dapat dipakai peran pemanggil, kembalikan null supaya pemanggil jatuh ke
// tujuan bawaan menurut jenis kejadian, bukan ke halaman yang memantul.
export function normalizeLink(link: string, isBuyer: boolean): string | null {
    const trimmed = link.trim();
    if (!trimmed) return null;

    // Backend dapat mengirim path API ("/work-orders/<id>"); rute klien memakai
    // "/orders/<id>". Sisanya diteruskan apa adanya bila sudah path relatif.
    const workOrderMatch = /^\/?(?:api\/)?work-orders\/([^/?#]+)/.exec(trimmed);
    if (workOrderMatch) return `/orders/${workOrderMatch[1]}`;

    // Notifikasi negosiasi dikirim ke kedua pihak dengan link yang sama, yaitu
    // request_id. Pemberi order punya halamannya; subkontraktor tidak, karena
    // detail request masuk dikunci candidate_id, bukan request_id. Kembalikan
    // null supaya tujuannya jatuh ke daftar request masuk yang memang ada.
    const quotaRequestMatch = /^\/?(?:api\/)?quota-requests\/([^/?#]+)/.exec(trimmed);
    if (quotaRequestMatch) return isBuyer ? `/quota-requests/${quotaRequestMatch[1]}` : null;

    if (!trimmed.startsWith("/")) return null;

    return isBuyer || !isBuyerOnlyPath(trimmed) ? trimmed : null;
}

// Tujuan bawaan menurut jenis kejadian dan peran pemanggil, supaya tidak ada
// notifikasi yang berlabel tanpa tujuan.
export function getFallbackLink(event: Notification["event"], isBuyer: boolean): NotificationLink | null {
    switch (event) {
        case "request_received":
            return { to: "/requests/incoming", label: "Lihat request masuk" };

        case "offer_received":
        case "counter_offer":
        case "request_expired":
            return isBuyer ? { to: "/quota-requests", label: "Lihat request terkirim" } : { to: "/requests/incoming", label: "Lihat request masuk" };

        case "agreement_formed":
        case "order_status_changed":
        case "payment_record":
        case "deadline_approaching":
        case "deadline_passed":
        case "order_cancelled":
        case "confirmation_due_approaching":
        case "order_auto_closed":
            return { to: "/orders", label: "Lihat pesanan" };

        // Ulasan diberikan dari detail pesanan yang sudah dikonfirmasi. Tanpa id
        // pesanan, arahkan ke daftar pesanan terkonfirmasi supaya pengguna tetap
        // punya jalan menulis ulasannya (FR-047).
        case "rating_request":
            return { to: "/orders?status=confirmed", label: "Pilih pesanan untuk diulas" };

        case "calendar_stale":
            return { to: "/listing/calendar", label: "Perbarui kalender" };

        case "verification_decision":
            return { to: "/verification", label: "Lihat verifikasi" };

        case "item_proposal_decision":
            return { to: "/listing", label: "Lihat listing" };

        default:
            return null;
    }
}

export function getNotificationLink(notification: Notification, isBuyer: boolean): NotificationLink | null {
    const fallback = getFallbackLink(notification.event, isBuyer);
    const target = notification.link ? normalizeLink(notification.link, isBuyer) : null;

    if (target) {
        return { to: target, label: fallback?.label ?? "Lihat detail" };
    }

    return fallback;
}
