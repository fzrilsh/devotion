import type { Offer } from "@api/search";
import { canAcceptOffer, canCounterOffer, canSendFirstOffer, isChainFromSessionOnly, isOfferChainMissing, isTerminalCandidate, isWaitingBuyer, latestOffer, resolveOfferChain } from "./offers";

function offer(sequence: number, party: Offer["party"], offerId = `offer-${sequence}`): Offer {
    return {
        offer_id: offerId,
        party,
        total_price: 1_500_000,
        readiness_lead_days: 7,
        sequence,
        note: null,
        created_at: "2026-08-30T03:00:00Z",
    };
}

describe("resolveOfferChain", () => {
    it("FR-032: rantai dari server dipakai apa adanya", () => {
        const offers = [offer(1, "subcontractor"), offer(2, "buyer")];

        expect(resolveOfferChain({ offers }, [offer(1, "subcontractor", "sesi")])).toEqual(offers);
    });

    it("FR-032: latest_offer sendiri menjadi rantai satu elemen", () => {
        const latest = offer(2, "buyer");

        expect(resolveOfferChain({ latest_offer: latest })).toEqual([latest]);
    });

    it("FR-032: tanpa offers dan latest_offer, rantainya kosong", () => {
        expect(resolveOfferChain({})).toEqual([]);
    });

    it("FR-032: penawaran sesi menambal saat server tidak mengirim apa pun", () => {
        const session = [offer(1, "subcontractor", "sesi")];

        expect(resolveOfferChain({}, session)).toEqual(session);
        expect(resolveOfferChain({ offers: [] }, session)).toEqual(session);
    });

    it("FR-032: ronde sesi yang sudah tertinggal dari server dibuang", () => {
        const session = [offer(1, "subcontractor", "sesi")];
        const chain = resolveOfferChain({ latest_offer: offer(3, "buyer") }, session);

        expect(chain).toHaveLength(1);
        expect(chain[0].sequence).toBe(3);
    });

    it("FR-032: ronde sesi yang lebih baru dari server ikut ke belakang rantai", () => {
        const chain = resolveOfferChain({ latest_offer: offer(2, "buyer") }, [offer(3, "subcontractor", "counter-sesi")]);

        expect(chain.map((item) => item.sequence)).toEqual([2, 3]);
        expect(latestOffer(chain)?.offer_id).toBe("counter-sesi");
    });
});

describe("latestOffer", () => {
    it("FR-032: ronde terakhir adalah elemen terakhir rantai", () => {
        expect(latestOffer([offer(1, "subcontractor"), offer(2, "buyer")])?.sequence).toBe(2);
        expect(latestOffer([])).toBeUndefined();
    });
});

describe("canCounterOffer", () => {
    it("FR-032: status offered saja tidak cukup untuk meng-counter", () => {
        expect(canCounterOffer("offered", undefined)).toBe(false);
    });

    it("FR-032: counter butuh penawaran pemberi order dengan offer_id sungguhan", () => {
        expect(canCounterOffer("offered", offer(2, "buyer"))).toBe(true);
        expect(canCounterOffer("offered", offer(2, "subcontractor"))).toBe(false);
        expect(canCounterOffer("offered", { ...offer(2, "buyer"), offer_id: "" })).toBe(false);
    });

    it("FR-032: status yang sudah berakhir tidak bisa di-counter", () => {
        for (const status of ["awaiting_reply", "rejected", "expired", "not_continued", "agreed"] as const) {
            expect(canCounterOffer(status, offer(2, "buyer"))).toBe(false);
        }
    });
});

describe("canSendFirstOffer", () => {
    it("FR-032: penawaran pertama hanya saat belum ada ronde sama sekali", () => {
        expect(canSendFirstOffer("awaiting_reply", [])).toBe(true);
        expect(canSendFirstOffer("awaiting_reply", [offer(1, "subcontractor")])).toBe(false);
        expect(canSendFirstOffer("offered", [])).toBe(false);
    });
});

describe("canAcceptOffer", () => {
    it("FR-032: pihak yang bukan pemilik ronde terakhir dapat menerima", () => {
        expect(canAcceptOffer("offered", offer(2, "buyer"), "subcontractor")).toBe(true);
        expect(canAcceptOffer("offered", offer(2, "subcontractor"), "buyer")).toBe(true);
        expect(canAcceptOffer("offered", offer(2, "buyer"), "buyer")).toBe(false);
        expect(canAcceptOffer("offered", { ...offer(2, "buyer"), offer_id: "" }, "subcontractor")).toBe(false);
    });

    it("FR-032: status terminal atau tanpa offer_id tidak menerima", () => {
        expect(canAcceptOffer("agreed", offer(2, "buyer"), "subcontractor")).toBe(false);
        expect(canAcceptOffer("awaiting_reply", offer(2, "subcontractor"), "buyer")).toBe(false);
        expect(canAcceptOffer("offered", undefined, "subcontractor")).toBe(false);
    });
});

describe("isWaitingBuyer", () => {
    it("FR-032: bola di pemberi order selama ronde terakhir milik subkontraktor", () => {
        expect(isWaitingBuyer("offered", offer(1, "subcontractor"))).toBe(true);
        expect(isWaitingBuyer("offered", offer(2, "buyer"))).toBe(false);
        expect(isWaitingBuyer("offered", undefined)).toBe(false);
    });

    it("FR-032: penawaran yang baru terkirim tetap terbaca menunggu meski status belum berubah", () => {
        expect(isWaitingBuyer("awaiting_reply", offer(1, "subcontractor"))).toBe(true);
    });

    it("FR-032: kandidat yang sudah berakhir tidak menunggu siapa pun", () => {
        for (const status of ["rejected", "expired", "not_continued", "agreed"] as const) {
            expect(isWaitingBuyer(status, offer(1, "subcontractor"))).toBe(false);
        }
    });
});

describe("isOfferChainMissing", () => {
    it("FR-032: status offered tanpa satu pun ronde berarti riwayatnya tidak terkirim", () => {
        expect(isOfferChainMissing("offered", [])).toBe(true);
        expect(isOfferChainMissing("offered", [offer(1, "subcontractor")])).toBe(false);
    });

    it("FR-032: status lain tidak pernah dianggap kehilangan riwayat", () => {
        for (const status of ["awaiting_reply", "rejected", "expired", "not_continued", "agreed"] as const) {
            expect(isOfferChainMissing(status, [])).toBe(false);
        }
    });
});

describe("isTerminalCandidate", () => {
    it("FR-032: empat status berakhir, dua status masih berjalan", () => {
        for (const status of ["rejected", "expired", "not_continued", "agreed"] as const) {
            expect(isTerminalCandidate(status)).toBe(true);
        }

        expect(isTerminalCandidate("awaiting_reply")).toBe(false);
        expect(isTerminalCandidate("offered")).toBe(false);
    });
});

describe("isChainFromSessionOnly", () => {
    it("FR-032: rantai yang hanya berasal dari respons mutasi ditandai", () => {
        const session = [offer(1, "subcontractor", "sesi")];

        expect(isChainFromSessionOnly({}, resolveOfferChain({}, session))).toBe(true);
        expect(isChainFromSessionOnly({ offers: [] }, resolveOfferChain({ offers: [] }, session))).toBe(true);
    });

    it("FR-032: rantai yang dikirim server tidak ditandai", () => {
        const offers = [offer(1, "subcontractor")];

        expect(isChainFromSessionOnly({ offers }, offers)).toBe(false);
        expect(isChainFromSessionOnly({ latest_offer: offers[0] }, offers)).toBe(false);
    });

    it("FR-032: rantai kosong tidak ditandai", () => {
        expect(isChainFromSessionOnly({}, [])).toBe(false);
    });
});
