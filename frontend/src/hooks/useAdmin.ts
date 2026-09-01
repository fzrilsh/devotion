import { createMasterItem, decideProposal, decideVerification, getDisputes, getItemProposals, getLateOrders, getMasterItems, getVerificationQueue, getWhatsAppStatus, hideReview, mediateDispute, reconnectWhatsApp, resolveDispute, updateMasterItem, type DisputeResolutionRequest, type MasterItemCreateRequest, type MasterItemUpdateRequest, type ProposalDecisionRequest, type VerificationDecisionRequest, type DisputeStatus, type VerificationStatus } from "@api/admin";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const adminKeys = {
    verification: (status?: VerificationStatus) => ["admin", "verification", status ?? "all"] as const,
    proposals: ["admin", "proposals"] as const,
    disputes: (status?: DisputeStatus) => ["admin", "disputes", status ?? "all"] as const,
    lateOrders: ["admin", "late-orders"] as const,
    masterItems: (kind?: "product" | "machine") => ["admin", "master-items", kind ?? "all"] as const,
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
        mutationFn: ({ requestId, data }: { requestId: string; data: VerificationDecisionRequest }) => decideVerification(requestId, data),
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
        mutationFn: ({ proposalId, data }: { proposalId: string; data: ProposalDecisionRequest }) => decideProposal(proposalId, data),
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
        mutationFn: ({ disputeId, data }: { disputeId: string; data: DisputeResolutionRequest }) => resolveDispute(disputeId, data),
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
        mutationFn: (data: MasterItemCreateRequest) => createMasterItem(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "master-items"] });
        },
    });
}

export function useUpdateMasterItem() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ itemId, data }: { itemId: string; data: MasterItemUpdateRequest }) => updateMasterItem(itemId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["admin", "master-items"] });
        },
    });
}

export function useHideReview() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ reviewId, reason }: { reviewId: string; reason: string }) => hideReview(reviewId, reason),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["profile", "reviews"] });
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

export function useReconnectWhatsApp() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: reconnectWhatsApp,
        onSuccess: (status) => {
            queryClient.setQueryData(adminKeys.whatsapp, status);
        },
    });
}
