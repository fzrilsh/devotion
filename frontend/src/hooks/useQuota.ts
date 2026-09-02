import { acceptOffer, counterOffer, createQuotaRequest, getIncomingCandidates, getQuotaRequest, getSentQuotaRequests, rejectCandidate, searchSubcontractors, sendOffer, type CandidateStatus, type IncomingCandidate, type QuotaRequestCreate, type SearchParams } from "@api/search";
import { appendSessionOffer, getSessionOffers, subscribeOfferSession } from "@lib/offerSession";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useCallback, useSyncExternalStore } from "react";

export const quotaKeys = {
    search: (params: SearchParams | null) => ["quota", "search", params] as const,
    sent: ["quota", "sent"] as const,
    detail: (requestId: string) => ["quota", "detail", requestId] as const,
    incoming: (status?: CandidateStatus) => ["quota", "incoming", status ?? "all"] as const,
    // Shared prefix of every incoming list, one cache entry per status filter.
    incomingLists: ["quota", "incoming"] as const,
    // Deliberately not nested under incomingLists: a prefix read of the lists must
    // return list pages only, never this single-candidate entry.
    incomingDetail: (candidateId: string) => ["quota", "incoming-detail", candidateId] as const,
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

type IncomingListPages = { pages: { items: IncomingCandidate[] }[] };

// There is no GET for a single candidate in the contract, so the detail page
// reads the lists the user has already loaded (issue #30). These two failures
// look the same to the cache but mean different things to the user.
export const CANDIDATE_LISTS_NOT_LOADED = "CANDIDATE_LISTS_NOT_LOADED";
export const CANDIDATE_NOT_IN_LOADED_LISTS = "CANDIDATE_NOT_IN_LOADED_LISTS";

/** Every incoming list currently in the cache, one entry per status filter. */
function readIncomingLists(queryClient: QueryClient): IncomingListPages[] {
    return queryClient
        .getQueriesData<IncomingListPages>({ queryKey: quotaKeys.incomingLists })
        .map(([, data]) => data)
        .filter((data): data is IncomingListPages => Boolean(data));
}

export function useIncomingCandidate(candidateId: string) {
    const queryClient = useQueryClient();

    return useQuery({
        queryKey: quotaKeys.incomingDetail(candidateId),
        queryFn: async () => {
            // Reading one hardcoded key would miss every candidate opened from a
            // filtered list, because the filter is part of the writer's key
            // (FR-030, FR-031).
            const lists = readIncomingLists(queryClient);

            if (lists.length === 0) {
                throw new Error(CANDIDATE_LISTS_NOT_LOADED);
            }

            const candidate = lists
                .flatMap((list) => list.pages)
                .flatMap((page) => page.items)
                .find((item) => item.candidate_id === candidateId);

            if (!candidate) {
                throw new Error(CANDIDATE_NOT_IN_LOADED_LISTS);
            }

            return candidate;
        },
        enabled: Boolean(candidateId),
        staleTime: 30 * 1000,
        retry: false,
    });
}

/**
 * Reload for the detail page. Refetching the detail entry alone can never reach
 * the server, because its queryFn only reads the cache: the lists have to be
 * refetched first, then the detail recomputed from them.
 */
export function useReloadIncomingCandidate(candidateId: string) {
    const queryClient = useQueryClient();

    return useCallback(async () => {
        await queryClient.refetchQueries({ queryKey: quotaKeys.incomingLists });
        await queryClient.refetchQueries({ queryKey: quotaKeys.incomingDetail(candidateId) });
    }, [candidateId, queryClient]);
}

function useQuotaInvalidator() {
    const queryClient = useQueryClient();

    return () => {
        queryClient.invalidateQueries({ queryKey: ["quota"] });
    };
}

export function useSessionOffers(candidateId: string) {
    const getSnapshot = useCallback(() => getSessionOffers(candidateId), [candidateId]);

    return useSyncExternalStore(subscribeOfferSession, getSnapshot, getSnapshot);
}

export function useSendOffer(candidateId: string) {
    const invalidate = useQuotaInvalidator();

    return useMutation({
        mutationFn: (data: { total_price: number; readiness_lead_days: number; note?: string }) => sendOffer(candidateId, data),
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
