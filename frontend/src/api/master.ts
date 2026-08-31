import { apiClient } from "./client";
import type { components } from "./types";

export type CatalogItem = components["schemas"]["CatalogItem"];
export type Province = components["schemas"]["Province"];
export type City = components["schemas"]["City"];
export type ItemProposal = components["schemas"]["ItemProposal"];

export async function getProducts(): Promise<CatalogItem[]> {
    return apiClient<CatalogItem[]>("/master/products");
}

export async function getMachines(): Promise<CatalogItem[]> {
    return apiClient<CatalogItem[]>("/master/machines");
}

export async function getProvinces(): Promise<Province[]> {
    return apiClient<Province[]>("/regions/provinces");
}

export async function getCities(provinceCode?: string): Promise<City[]> {
    const params = new URLSearchParams();
    if (provinceCode) params.append("province", provinceCode);
    const query = params.toString();
    return apiClient<City[]>(`/regions/cities${query ? `?${query}` : ""}`);
}

export async function proposeCatalogItem(kind: "product" | "machine", proposedName: string): Promise<ItemProposal> {
    return apiClient<ItemProposal>("/master/proposals", {
        method: "POST",
        body: JSON.stringify({ kind, proposed_name: proposedName }),
    });
}
