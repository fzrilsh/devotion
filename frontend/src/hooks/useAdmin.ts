import { createMasterItem, decideProposal, decideVerification, getDisputes, getItemProposals, getLateOrders, getMasterItems, getProfileReviews, getVerificationQueue, getWhatsAppStatus, hideReview, mediateDispute, resolveDispute, updateMasterItem, type DisputeResult, type DisputeStatus, type VerificationStatus } from "@api/admin";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const adminKeys = {
    verification: (status?: VerificationStatus) => ["admin", "verification", status ?? "all"] as const,
    proposals: ["admin", "proposals"] as const,
    disputes: (status?: DisputeStatus) => ["admin", "disputes", status ?? "all"] as const,
    lateOrders: ["admin", "late-orders"] as const,
    masterItems: (kind?: "product" | "machine") => ["admin", "master-items", kind ?? "all"] as const,
    reviews: (profileId: string) => ["admin", "reviews", profileId] as const,
    whatsapp: ["admin", "whatsapp"] as const,
};

const queueQueryOptions = {
    staleTime: 30 * 1000,
    refetchOnWindowFocus: true,
    retry: false,
} as const;

export function useVerificationQueue(status?: VerificationStatus) {
    return useQuery({
        queryKey: adminKeys.verification(status),
        queryFn: () => getVerificationQueue({ status }),
        ...queueQueryOptions,
    });
}

export function useDecideVerification() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ requestId, decision, reason }: { requestId: string; decision: "approved" | "rejected"; reason?: string }) => decideVerification(requestId, { decision, reason }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "verification"] });
        },
    });
}

export function useItemProposals() {
    return useQuery({
        queryKey: adminKeys.proposals,
        queryFn: () => getItemProposals(),
        ...queueQueryOptions,
    });
}

export function useDecideProposal() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ proposalId, decision, reason }: { proposalId: string; decision: "approved" | "rejected"; reason?: string }) => decideProposal(proposalId, { decision, reason }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: adminKeys.proposals });
            queryClient.invalidateQueries({ queryKey: ["admin", "master-items"] });
        },
    });
}

export function useDisputes(status?: DisputeStatus) {
    return useQuery({
        queryKey: adminKeys.disputes(status),
        queryFn: () => getDisputes({ status }),
        ...queueQueryOptions,
    });
}

export function useMediateDispute() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (disputeId: string) => mediateDispute(disputeId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "disputes"] });
        },
    });
}

export function useResolveDispute() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ disputeId, data }: { disputeId: string; data: { result: DisputeResult; allocation_reversed?: boolean; liable_profile_id?: string | null; note?: string } }) => resolveDispute(disputeId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "disputes"] });
            queryClient.invalidateQueries({ queryKey: adminKeys.lateOrders });
        },
    });
}

export function useLateOrders() {
    return useQuery({
        queryKey: adminKeys.lateOrders,
        queryFn: () => getLateOrders(),
        ...queueQueryOptions,
    });
}

export function useMasterItems(kind?: "product" | "machine") {
    return useQuery({
        queryKey: adminKeys.masterItems(kind),
        queryFn: () => getMasterItems(kind),
        staleTime: 30 * 1000,
        retry: false,
    });
}

export function useCreateMasterItem() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: { kind: "product" | "machine"; name: string }) => createMasterItem(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "master-items"] });
        },
    });
}

export function useUpdateMasterItem() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ itemId, data }: { itemId: string; data: { name?: string; active?: boolean } }) => updateMasterItem(itemId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "master-items"] });
        },
    });
}

export function useProfileReviews(profileId: string) {
    return useInfiniteQuery({
        queryKey: adminKeys.reviews(profileId),
        queryFn: ({ pageParam }) => getProfileReviews(profileId, pageParam),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        enabled: Boolean(profileId),
        staleTime: 30 * 1000,
        retry: false,
    });
}

export function useHideReview() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ reviewId, reason }: { reviewId: string; reason: string }) => hideReview(reviewId, reason),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "reviews"] });
        },
    });
}

export function useWhatsAppStatus() {
    return useQuery({
        queryKey: adminKeys.whatsapp,
        queryFn: getWhatsAppStatus,
        staleTime: 10 * 1000,
        refetchInterval: 30 * 1000,
        retry: false,
    });
}
