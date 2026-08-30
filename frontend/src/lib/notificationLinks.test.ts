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

    it("FR-051: request kedaluwarsa mengikuti peran pemanggil", () => {
        expect(getNotificationLink(notification("request_expired"), true)?.to).toBe("/quota-requests");
        expect(getNotificationLink(notification("request_expired"), false)?.to).toBe("/requests/incoming");
    });

    it("link absolut ke host lain diabaikan dan jatuh ke tujuan bawaan", () => {
        expect(getNotificationLink(notification("order_auto_closed", "https://contoh.id/orders/1"), true)?.to).toBe("/orders");
    });
});
