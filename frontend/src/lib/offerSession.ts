import type { Offer } from "@api/search";

// Penampung Offer yang benar-benar dikembalikan backend dari POST
// /candidates/{candidateId}/offers dan POST /offers/{offerId}/counter.
//
// Ini tambalan, bukan sumber kebenaran. GET /quota-requests/incoming belum
// mengirim rantai penawaran, jadi tanpa penampung ini penawaran yang baru saja
// dikirim langsung hilang dari layar begitu daftar dimuat ulang. Isinya hanya
// hidup di memori: tidak ada localStorage maupun sessionStorage, sehingga muat
// ulang halaman mengosongkannya dan halaman jatuh ke keadaan "riwayat belum
// tersedia" alih-alih menyusun ulang rantai dari data yang tidak dikirim server.
const chains = new Map<string, Offer[]>();
const listeners = new Set<() => void>();

// Referensi tetap untuk kandidat tanpa penawaran tersimpan. useSyncExternalStore
// membandingkan snapshot dengan Object.is, jadi array baru pada tiap pembacaan
// akan memicu render berulang tanpa henti.
const EMPTY: Offer[] = [];

function emit() {
    for (const listener of listeners) listener();
}

export function subscribeOfferSession(listener: () => void): () => void {
    listeners.add(listener);

    return () => {
        listeners.delete(listener);
    };
}

export function getSessionOffers(candidateId: string): Offer[] {
    return chains.get(candidateId) ?? EMPTY;
}

// Offer yang sudah ada, dikenali dari offer_id, tidak digandakan: mutasi yang
// diulang bisa mengembalikan ronde yang sama dua kali. Offer tanpa offer_id
// dibuang, karena tanpa id itu tidak ada yang bisa di-counter.
export function appendSessionOffer(candidateId: string, offer: Offer): void {
    if (!candidateId || !offer.offer_id) return;

    const current = chains.get(candidateId) ?? EMPTY;
    if (current.some((item) => item.offer_id === offer.offer_id)) return;

    chains.set(candidateId, [...current, offer]);
    emit();
}

// Dipakai pengujian supaya satu kasus tidak mewarisi isi kasus sebelumnya.
export function resetOfferSession(): void {
    chains.clear();
    emit();
}
