import { apiClient, ApiError } from "./client";
import type { components } from "./types";

export type Listing = components["schemas"]["Listing"];
export type ListingRequest = components["schemas"]["ListingRequest"];
export type AvailabilityPeriod = components["schemas"]["AvailabilityPeriod"];
export type PeriodUpdateItem = components["schemas"]["PeriodUpdateItem"];
export type CatalogItem = components["schemas"]["CatalogItem"];

// 404 di sini berarti "belum punya listing" (LISTING_NOT_FOUND), bukan galat.
// Kembalikan null supaya TanStack Query tidak memperlakukannya sebagai error.
export async function getMyListing(): Promise<Listing | null> {
    try {
        return await apiClient<Listing>("/listing/me");
    } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
    }
}

export async function createListing(data: ListingRequest): Promise<Listing> {
    return apiClient<Listing>("/listing/me", { method: "POST", body: JSON.stringify(data) });
}

export async function updateListing(data: ListingRequest): Promise<Listing> {
    return apiClient<Listing>("/listing/me", { method: "PUT", body: JSON.stringify(data) });
}

export async function setListingVisibility(published: boolean): Promise<Listing> {
    return apiClient<Listing>("/listing/me/visibility", { method: "PUT", body: JSON.stringify({ published }) });
}

export async function getListingPeriods(params?: { from?: string; to?: string }): Promise<AvailabilityPeriod[]> {
    const searchParams = new URLSearchParams();

    if (params?.from) searchParams.set("from", params.from);
    if (params?.to) searchParams.set("to", params.to);

    const query = searchParams.toString();

    return apiClient<AvailabilityPeriod[]>(`/listing/me/periods${query ? `?${query}` : ""}`);
}

export async function updateListingPeriods(periods: PeriodUpdateItem[]): Promise<AvailabilityPeriod[]> {
    return apiClient<AvailabilityPeriod[]>("/listing/me/periods", { method: "PUT", body: JSON.stringify({ periods }) });
}

export async function getMasterProducts(): Promise<CatalogItem[]> {
    return apiClient<CatalogItem[]>("/master/products");
}

export async function getMasterMachines(): Promise<CatalogItem[]> {
    return apiClient<CatalogItem[]>("/master/machines");
}
