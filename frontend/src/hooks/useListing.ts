import { createListing, getListingPeriods, getMasterMachines, getMasterProducts, getMyListing, setListingVisibility, updateListing, updateListingPeriods, type ListingRequest, type PeriodUpdateItem } from "@api/listing";
import { proposeCatalogItem } from "@api/master";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const listingKeys = {
    me: ["listing", "me"] as const,
    periods: (from?: string, to?: string) => ["listing", "periods", from ?? "", to ?? ""] as const,
    products: ["master", "products"] as const,
    machines: ["master", "machines"] as const,
};

export function useMyListing() {
    return useQuery({
        queryKey: listingKeys.me,
        queryFn: getMyListing,
        staleTime: 60 * 1000,
        retry: false,
    });
}

export function useCreateListing() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: ListingRequest) => createListing(data),
        onSuccess: (listing) => {
            queryClient.setQueryData(listingKeys.me, listing);
        },
    });
}

export function useUpdateListing() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: ListingRequest) => updateListing(data),
        onSuccess: (listing) => {
            queryClient.setQueryData(listingKeys.me, listing);
        },
    });
}

export function useListingVisibility() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (published: boolean) => setListingVisibility(published),
        onSuccess: (listing) => {
            queryClient.setQueryData(listingKeys.me, listing);
        },
    });
}

export function useListingPeriods(from?: string, to?: string) {
    return useQuery({
        queryKey: listingKeys.periods(from, to),
        queryFn: () => getListingPeriods({ from, to }),
        staleTime: 30 * 1000,
        retry: false,
    });
}

export function useUpdateListingPeriods() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (periods: PeriodUpdateItem[]) => updateListingPeriods(periods),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["listing", "periods"] });
            queryClient.invalidateQueries({ queryKey: listingKeys.me });
        },
    });
}

export function useMasterProducts() {
    return useQuery({
        queryKey: listingKeys.products,
        queryFn: getMasterProducts,
        staleTime: 10 * 60 * 1000,
    });
}

export function useMasterMachines() {
    return useQuery({
        queryKey: listingKeys.machines,
        queryFn: getMasterMachines,
        staleTime: 10 * 60 * 1000,
    });
}

export function useProposeMasterItem() {
    return useMutation({
        mutationFn: ({ kind, proposedName }: { kind: "product" | "machine"; proposedName: string }) => proposeCatalogItem(kind, proposedName),
    });
}
