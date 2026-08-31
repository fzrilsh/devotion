import { acceptOffer, counterOffer, createQuotaRequest, getIncomingCandidates, getQuotaRequest, getSentQuotaRequests, rejectCandidate, searchSubcontractors, sendOffer, type CandidateStatus, type IncomingCandidate, type QuotaRequestCreate, type SearchParams } from "@api/search";
import { appendSessionOffer, getSessionOffers, subscribeOfferSession } from "@lib/offerSession";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useSyncExternalStore } from "react";

export const quotaKeys = {
    search: (params: SearchParams | null) => ["quota", "search", params] as const,
    sent: ["quota", "sent"] as const,
    detail: (requestId: string) => ["quota", "detail", requestId] as const,
    incoming: (status?: CandidateStatus) => ["quota", "incoming", status ?? "all"] as const,
};

export function useSearch(params: SearchParams | null) {
    return useInfiniteQuery({
        queryKey: quotaKeys.search(params),
        queryFn: ({ pageParam }) => searchSubcontractors(params!, pageParam),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        enabled: Boolean(params),
        staleTime: 30 * 1000,
    });
}

export function useCreateQuotaRequest() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: QuotaRequestCreate) => createQuotaRequest(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: quotaKeys.sent });
        },
    });
}

export function useSentQuotaRequests() {
    return useInfiniteQuery({
        queryKey: quotaKeys.sent,
        queryFn: ({ pageParam }) => getSentQuotaRequests(pageParam),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        staleTime: 30 * 1000,
    });
}

export function useQuotaRequest(requestId: string) {
    return useQuery({
        queryKey: quotaKeys.detail(requestId),
        queryFn: () => getQuotaRequest(requestId),
        enabled: Boolean(requestId),
        staleTime: 30 * 1000,
        retry: false,
    });
}

export function useIncomingCandidates(status?: CandidateStatus) {
    return useInfiniteQuery({
        queryKey: quotaKeys.incoming(status),
        queryFn: ({ pageParam }) => getIncomingCandidates({ status, cursor: pageParam }),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        staleTime: 30 * 1000,
    });
}

// Kandidat request masuk diidentifikasi candidate_id. Kontrak belum punya
// GET /candidates/{candidateId}, dan endpoint detail request (/quota-requests/{id})
// khusus pembanding milik pemberi order, jadi halaman subkontraktor menyusuri
// daftar incoming sampai kandidatnya ketemu.
//
// Penyusuran mengikuti has_next sampai habis, bukan berhenti pada jumlah halaman
// tetap: kandidat lama tetap dapat dibuka dari notifikasi meski sudah terdorong
// jauh ke belakang daftar. Batas 200 halaman hanya jaring pengaman terhadap
// kursor yang tidak maju, bukan batas jangkauan yang diharapkan tercapai.
const MAX_INCOMING_SCAN_PAGES = 200;

export function useIncomingCandidate(candidateId: string) {
    const queryClient = useQueryClient();

    return useQuery({
        queryKey: ["quota", "incoming", "detail", candidateId] as const,
        queryFn: async () => {
            const seen = new Set<string>();
            let cursor: string | undefined;

            for (let page = 0; page < MAX_INCOMING_SCAN_PAGES; page += 1) {
                const result = await getIncomingCandidates({ cursor });
                const found = result.items.find((item) => item.candidate_id === candidateId);

                if (found) return found;
                if (!result.pagination.has_next || !result.pagination.next_cursor) break;

                // Kursor bersifat opaque, jadi kemajuannya tidak bisa diperiksa selain
                // dengan membandingkan nilainya. Kursor yang berulang berarti berputar.
                if (seen.has(result.pagination.next_cursor)) break;

                seen.add(result.pagination.next_cursor);
                cursor = result.pagination.next_cursor;
            }

            // Cache daftar bisa saja belum termuat halaman yang memuat kandidat ini.
            // Lempar agar select jatuh ke cache daftar, atau memicu refetch.
            throw new Error("CANDIDATE_NOT_LOADED");
        },
        select: (fresh) => {
            const pages = queryClient.getQueryData<{ pages: { items: IncomingCandidate[] }[] }>(quotaKeys.incoming());

            return pages?.pages.flatMap((page) => page.items).find((item) => item.candidate_id === candidateId) ?? fresh;
        },
        enabled: Boolean(candidateId),
        staleTime: 30 * 1000,
        retry: false,
    });
}

function useQuotaInvalidator() {
    const queryClient = useQueryClient();

    return () => {
        queryClient.invalidateQueries({ queryKey: ["quota"] });
    };
}

// Rantai penawaran yang sudah diterima dari backend selama sesi ini, dibaca dari
// penampung di luar React supaya nilainya bertahan saat halaman detail dilepas
// dan dipasang ulang (misalnya setelah menutup panel atau berpindah kandidat).
export function useSessionOffers(candidateId: string) {
    const getSnapshot = useCallback(() => getSessionOffers(candidateId), [candidateId]);

    return useSyncExternalStore(subscribeOfferSession, getSnapshot, getSnapshot);
}

export function useSendOffer(candidateId: string) {
    const invalidate = useQuotaInvalidator();

    return useMutation({
        mutationFn: (data: { total_price: number; readiness_lead_days: number; note?: string }) => sendOffer(candidateId, data),
        // Respons POST /candidates/{id}/offers adalah Offer utuh. Daftar incoming
        // tidak mengirim rantai penawaran, jadi Offer ini disimpan apa adanya supaya
        // ronde yang baru terkirim tetap terlihat setelah daftar dimuat ulang.
        onSuccess: (offer) => {
            appendSessionOffer(candidateId, offer);
            invalidate();
        },
    });
}

export function useRejectCandidate(candidateId: string) {
    const invalidate = useQuotaInvalidator();

    return useMutation({
        mutationFn: (reason: string) => rejectCandidate(candidateId, reason),
        onSuccess: () => invalidate(),
    });
}

// candidateId opsional: sisi pemberi order sudah menerima rantai penawaran utuh
// dari GET /quota-requests/{requestId}, jadi hanya sisi subkontraktor yang perlu
// menampung Offer hasil counter.
export function useCounterOffer(candidateId?: string) {
    const invalidate = useQuotaInvalidator();

    return useMutation({
        mutationFn: ({ offerId, data }: { offerId: string; data: { total_price: number; note?: string } }) => counterOffer(offerId, data),
        onSuccess: (offer) => {
            if (candidateId) appendSessionOffer(candidateId, offer);
            invalidate();
        },
    });
}

export function useAcceptOffer() {
    const invalidate = useQuotaInvalidator();

    return useMutation({
        mutationFn: (offerId: string) => acceptOffer(offerId),
        onSuccess: () => invalidate(),
    });
}
