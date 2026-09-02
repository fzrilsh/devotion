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

export function useIncomingCandidate(candidateId: string) {
    const queryClient = useQueryClient();

    return useQuery({
        queryKey: ["quota", "incoming", "detail", candidateId] as const,
        queryFn: async () => {
            const pages = queryClient.getQueryData<{ pages: { items: IncomingCandidate[] }[] }>(quotaKeys.incoming());
            const candidate = pages?.pages.flatMap((page) => page.items).find((item) => item.candidate_id === candidateId);

            if (candidate) {
                return candidate;
            }

            throw new Error("CANDIDATE_NOT_VISIBLE_IN_CACHE");
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
