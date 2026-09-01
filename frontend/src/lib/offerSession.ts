import type { Offer } from "@api/search";

const chains = new Map<string, Offer[]>();
const listeners = new Set<() => void>();

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

export function appendSessionOffer(candidateId: string, offer: Offer): void {
    if (!candidateId || !offer.offer_id) return;

    const current = chains.get(candidateId) ?? EMPTY;
    if (current.some((item) => item.offer_id === offer.offer_id)) return;

    chains.set(candidateId, [...current, offer]);
    emit();
}

export function resetOfferSession(): void {
    chains.clear();
    emit();
}
