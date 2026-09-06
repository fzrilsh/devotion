import type { CandidateStatus, Offer } from "@api/search";

type OfferCarrier = { latest_offer?: Offer; offers?: Offer[] };

const TERMINAL_STATUSES: CandidateStatus[] = ["rejected", "expired", "not_continued", "agreed"];

export function isTerminalCandidate(status: CandidateStatus): boolean {
    return TERMINAL_STATUSES.includes(status);
}

export function resolveOfferChain(candidate: OfferCarrier, sessionOffers: Offer[] = []): Offer[] {
    // Server selalu menjadi sumber kebenaran untuk urutan asli. Ronde dari sesi
    // hanya menambahkan penawaran yang lebih baru dan belum ada dari server,
    // sehingga rantai tetap konsisten.
    const fromServer = candidate.offers?.length ? candidate.offers : candidate.latest_offer ? [candidate.latest_offer] : [];

    if (fromServer.length === 0) return sessionOffers;

    const lastSequence = fromServer[fromServer.length - 1].sequence;
    const seen = new Set(fromServer.map((offer) => offer.offer_id));
    const newer = sessionOffers.filter((offer) => offer.sequence > lastSequence && !seen.has(offer.offer_id));

    if (newer.length === 0) return fromServer;

    return [...fromServer, ...newer].sort((a, b) => a.sequence - b.sequence);
}

export function latestOffer(offers: Offer[]): Offer | undefined {
    return offers.at(-1);
}

export function canCounterOffer(status: CandidateStatus, latest: Offer | undefined): boolean {
    // Counter-offer hanya valid ketika server sudah mengirim ronde penawaran asli.
    // Tanpa offer_id, frontend tidak memiliki target yang aman untuk aksi berikutnya.
    return status === "offered" && latest?.party === "buyer" && Boolean(latest.offer_id);
}

export function canAcceptOffer(status: CandidateStatus, latest: Offer | undefined, side: Offer["party"]): boolean {
    // Penerimaan penawaran hanya merupakan filter tampilan, bukan mesin keadaan
    // kedua. Gunakan status "offered" secara langsung karena "awaiting_reply"
    // tidak memiliki penawaran yang dapat diterima. Backend tetap menjadi sumber
    // kebenaran dan akan menolak penawaran yang tidak valid atau dibuat oleh pihak
    // sendiri dengan FORBIDDEN/own_offer.
    return status === "offered" && Boolean(latest?.offer_id) && latest?.party !== side;
}

export function canSendFirstOffer(status: CandidateStatus, offers: Offer[]): boolean {
    return status === "awaiting_reply" && offers.length === 0;
}

export function isWaitingBuyer(status: CandidateStatus, latest: Offer | undefined): boolean {
    return !isTerminalCandidate(status) && latest?.party === "subcontractor" && Boolean(latest.offer_id);
}

export function isChainFromSessionOnly(candidate: OfferCarrier, offers: Offer[]): boolean {
    // Ketika server belum mengirim riwayat penawaran, kita menandai rantai yang
    // hanya berasal dari mutasi sesi agar UI tidak menganggap data tersebut
    // sudah benar-benar tersedia di server.
    return offers.length > 0 && !candidate.offers?.length && !candidate.latest_offer;
}

export function isOfferChainMissing(status: CandidateStatus, offers: Offer[]): boolean {
    return status === "offered" && offers.length === 0;
}