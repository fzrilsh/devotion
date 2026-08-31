import type { Offer } from "@api/search";
import { appendSessionOffer, getSessionOffers, resetOfferSession, subscribeOfferSession } from "./offerSession";

function offer(sequence: number, offerId: string): Offer {
    return {
        offer_id: offerId,
        party: "subcontractor",
        total_price: 1_500_000,
        readiness_lead_days: 7,
        sequence,
        note: null,
        created_at: "2026-08-30T03:00:00Z",
    };
}

describe("offerSession", () => {
    beforeEach(() => {
        resetOfferSession();
    });

    it("FR-032: penawaran dari respons backend tersimpan per kandidat", () => {
        appendSessionOffer("kandidat-a", offer(1, "offer-1"));
        appendSessionOffer("kandidat-b", offer(1, "offer-9"));

        expect(getSessionOffers("kandidat-a").map((item) => item.offer_id)).toEqual(["offer-1"]);
        expect(getSessionOffers("kandidat-b").map((item) => item.offer_id)).toEqual(["offer-9"]);
        expect(getSessionOffers("kandidat-c")).toEqual([]);
    });

    it("FR-032: counter menambah ronde di belakang, urutannya terjaga", () => {
        appendSessionOffer("kandidat-a", offer(1, "offer-1"));
        appendSessionOffer("kandidat-a", offer(2, "offer-2"));

        expect(getSessionOffers("kandidat-a").map((item) => item.sequence)).toEqual([1, 2]);
    });

    it("FR-032: offer_id yang sama tidak digandakan", () => {
        appendSessionOffer("kandidat-a", offer(1, "offer-1"));
        appendSessionOffer("kandidat-a", offer(1, "offer-1"));

        expect(getSessionOffers("kandidat-a")).toHaveLength(1);
    });

    it("penawaran tanpa offer_id diabaikan, tidak ada offer_id yang dikarang", () => {
        appendSessionOffer("kandidat-a", offer(1, ""));

        expect(getSessionOffers("kandidat-a")).toEqual([]);
    });

    it("snapshot kandidat tanpa penawaran bereferensi tetap, supaya useSyncExternalStore tidak berulang", () => {
        expect(getSessionOffers("kandidat-a")).toBe(getSessionOffers("kandidat-b"));
    });

    it("pelanggan diberi tahu setiap penawaran baru, dan berhenti setelah berhenti berlangganan", () => {
        let calls = 0;
        const unsubscribe = subscribeOfferSession(() => {
            calls += 1;
        });

        appendSessionOffer("kandidat-a", offer(1, "offer-1"));
        expect(calls).toBe(1);

        unsubscribe();
        appendSessionOffer("kandidat-a", offer(2, "offer-2"));
        expect(calls).toBe(1);
    });
});
