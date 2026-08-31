import type { CandidateStatus, Offer } from "@api/search";

type OfferCarrier = { latest_offer?: Offer; offers?: Offer[] };

const TERMINAL_STATUSES: CandidateStatus[] = ["rejected", "expired", "not_continued", "agreed"];

export function isTerminalCandidate(status: CandidateStatus): boolean {
    return TERMINAL_STATUSES.includes(status);
}

// GET /quota-requests/incoming tidak mengirim rantai penawaran, sedangkan
// GET /quota-requests/{requestId} yang mengirimnya hanya boleh dibaca pemberi
// order. Jadi kedua field ini diperlakukan opsional, bukan dianggap selalu ada.
//
// Yang dikirim server selalu menang: offers dipakai lebih dulu, lalu latest_offer
// sebagai rantai satu elemen. Rantai kosong dihitung sebagai tidak mengirim.
// sessionOffers hanya menambal, dan isinya pun Offer yang dikembalikan backend
// dari mutasi di sesi ini, bukan susunan sendiri.
export function resolveOfferChain(candidate: OfferCarrier, sessionOffers: Offer[] = []): Offer[] {
    const fromServer = candidate.offers?.length ? candidate.offers : candidate.latest_offer ? [candidate.latest_offer] : [];

    if (fromServer.length === 0) return sessionOffers;

    // Server bisa mengirim satu ronde yang sudah tertinggal, misalnya latest_offer
    // saja pada respons yang dicache sementara sesi ini sudah meng-counter. Ronde
    // sesi yang nomornya di atas ronde terakhir server ditambahkan supaya tombol
    // counter tidak menunjuk offer_id yang sudah dilewati.
    const lastSequence = fromServer[fromServer.length - 1].sequence;
    const seen = new Set(fromServer.map((offer) => offer.offer_id));
    const newer = sessionOffers.filter((offer) => offer.sequence > lastSequence && !seen.has(offer.offer_id));

    if (newer.length === 0) return fromServer;

    return [...fromServer, ...newer].sort((a, b) => a.sequence - b.sequence);
}

// Rantai selalu terurut sequence naik, jadi ronde terakhir adalah elemen
// terakhir. Klien tidak menghitung ulang mana yang terbaru dari sequence.
export function latestOffer(offers: Offer[]): Offer | undefined {
    return offers.at(-1);
}

// Counter selalu menunjuk offer_id yang benar-benar diterima dari backend.
// Status offered saja tidak cukup: statusnya bisa benar sementara rantainya
// tidak ikut terkirim, dan tombol counter tanpa offer_id pasti gagal.
export function canCounterOffer(status: CandidateStatus, latest: Offer | undefined): boolean {
    return status === "offered" && latest?.party === "buyer" && Boolean(latest.offer_id);
}

// Penawaran pertama hanya masuk akal bila belum ada satu pun ronde di rantai.
export function canSendFirstOffer(status: CandidateStatus, offers: Offer[]): boolean {
    return status === "awaiting_reply" && offers.length === 0;
}

// Bola ada di pemberi order selama ronde terakhir datang dari subkontraktor.
// Ini dibaca dari rantai, bukan dari status saja, supaya sesaat setelah penawaran
// terkirim (daftar incoming belum selesai dimuat ulang, statusnya masih terbaca
// awaiting_reply) layarnya tidak berbalik mengaku riwayatnya hilang.
export function isWaitingBuyer(status: CandidateStatus, latest: Offer | undefined): boolean {
    return !isTerminalCandidate(status) && latest?.party === "subcontractor" && Boolean(latest.offer_id);
}

// Benar bila satu-satunya ronde yang diketahui halaman ini datang dari respons
// mutasi sesi ini, bukan dari respons daftar. Dalam keadaan itu ronde balasan
// pihak lain bisa sudah ada di basis data tanpa pernah sampai ke layar, jadi
// layarnya harus mengakui keterbatasan itu, bukan mengaku sedang menunggu.
export function isChainFromSessionOnly(candidate: OfferCarrier, offers: Offer[]): boolean {
    return offers.length > 0 && !candidate.offers?.length && !candidate.latest_offer;
}

// Status offered berarti rantai penawarannya ada di basis data. Bila rantai itu
// tidak ikut terkirim dan sesi ini juga belum menerima satu pun Offer, tidak ada
// offer_id untuk meng-counter: keadaan ini dinyatakan apa adanya, bukan ditutup
// dengan tombol yang pasti gagal atau kartu kosong tanpa penjelasan.
export function isOfferChainMissing(status: CandidateStatus, offers: Offer[]): boolean {
    return status === "offered" && offers.length === 0;
}
