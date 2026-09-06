import type { Notification } from "@api/notifications";
import { getNotificationLink } from "./notificationLinks";

function notification(event: Notification["event"], link?: string | null): Notification {
    return {
        notification_id: "11111111-1111-1111-1111-111111111111",
        event,
        read: false,
        created_at: "2026-08-30T03:00:00Z",
        link: link ?? null,
    };
}

describe("getNotificationLink", () => {
    it("FR-051: setiap jenis notifikasi punya tujuan meski link kosong", () => {
        const events: Notification["event"][] = [
            "request_received",
            "offer_received",
            "counter_offer",
            "agreement_formed",
            "order_status_changed",
            "payment_record",
            "deadline_approaching",
            "deadline_passed",
            "verification_decision",
            "rating_request",
            "order_cancelled",
            "confirmation_due_approaching",
            "order_auto_closed",
            "item_proposal_decision",
            "calendar_stale",
            "request_expired",
        ];

        for (const event of events) {
            expect(getNotificationLink(notification(event), true)).not.toBeNull();
            expect(getNotificationLink(notification(event), false)).not.toBeNull();
        }
    });

    it("FR-047: permintaan ulasan tanpa link mengarah ke pesanan terkonfirmasi", () => {
        expect(getNotificationLink(notification("rating_request"), true)?.to).toBe("/orders?status=confirmed");
    });

    it("FR-051: path pesanan dari API dipetakan ke rute klien", () => {
        const target = getNotificationLink(notification("order_status_changed", "/work-orders/9f1c2e77-0000-4000-8000-000000000001"), true);
        expect(target?.to).toBe("/orders/9f1c2e77-0000-4000-8000-000000000001");
    });

    it("FR-051: link server diteruskan apa adanya untuk path yang valid", () => {
        const link = "/quota-requests/ab9890cb-ac01-414e-b08f-f7e6a90b4cd8";

        expect(getNotificationLink(notification("counter_offer", link), true)?.to).toBe(link);
        expect(getNotificationLink(notification("counter_offer", link), false)?.to).toBe(link);
    });

    it("FR-033: tautan detail kandidat masuk dipetakan ke rute request masuk", () => {
        const link = "/requests/incoming/8ae07574-923d-40e4-b170-439bd238a9ed";

        expect(getNotificationLink(notification("counter_offer", link), false)?.to).toBe(link);
        expect(getNotificationLink(notification("counter_offer", link), false)?.label).toBe("Lihat request masuk");
    });

    it("FR-051: path khusus pemberi order lain tetap diteruskan bila server mengirimkan path yang valid", () => {
        expect(getNotificationLink(notification("request_received", "/search?produk=kaos"), false)?.to).toBe("/search?produk=kaos");
        expect(getNotificationLink(notification("offer_received", "/quota-requests"), false)?.to).toBe("/quota-requests");
    });

    it("link absolut ke host lain diabaikan dan jatuh ke tujuan bawaan", () => {
        expect(getNotificationLink(notification("order_auto_closed", "https://contoh.id/orders/1"), true)?.to).toBe("/orders");
    });
});
