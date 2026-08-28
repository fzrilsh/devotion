import { apiClient } from "./client";
import type { components } from "./types";

export type SearchResult = components["schemas"]["SearchResult"];
export type SearchCandidate = components["schemas"]["SearchCandidate"];
export type QuotaRequestCreate = components["schemas"]["QuotaRequestCreate"];
export type QuotaRequestDetail = components["schemas"]["QuotaRequestDetail"];
export type QuotaRequestList = components["schemas"]["QuotaRequestList"];
export type RequestCandidate = components["schemas"]["RequestCandidate"];
export type CandidateStatus = components["schemas"]["CandidateStatus"];
export type Offer = components["schemas"]["Offer"];
export type IncomingCandidateList = components["schemas"]["IncomingCandidateList"];

export type SearchParams = {
    product_item_id: string;
    machine_item_id?: string;
    quantity: number;
    deadline: string;
    max_lead_days?: number;
    region_level: "city" | "province" | "national";
    city_code?: string;
    province_code?: string;
};

export async function searchSubcontractors(params: SearchParams, cursor?: string): Promise<SearchResult> {
    const searchParams = new URLSearchParams();

    searchParams.set("product_item_id", params.product_item_id);
    searchParams.set("quantity", String(params.quantity));
    searchParams.set("deadline", params.deadline);
    searchParams.set("region_level", params.region_level);

    if (params.machine_item_id) searchParams.set("machine_item_id", params.machine_item_id);
    if (params.max_lead_days != null) searchParams.set("max_lead_days", String(params.max_lead_days));
    if (params.city_code) searchParams.set("city_code", params.city_code);
    if (params.province_code) searchParams.set("province_code", params.province_code);
    if (cursor) searchParams.set("cursor", cursor);

    return apiClient<SearchResult>(`/search?${searchParams.toString()}`);
}

export async function createQuotaRequest(data: QuotaRequestCreate): Promise<QuotaRequestDetail> {
    return apiClient<QuotaRequestDetail>("/quota-requests", { method: "POST", body: JSON.stringify(data) });
}

export async function getSentQuotaRequests(cursor?: string): Promise<QuotaRequestList> {
    const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";

    return apiClient<QuotaRequestList>(`/quota-requests${query}`);
}

export async function getQuotaRequest(requestId: string): Promise<QuotaRequestDetail> {
    return apiClient<QuotaRequestDetail>(`/quota-requests/${requestId}`);
}

export async function getIncomingCandidates(params?: { status?: CandidateStatus; cursor?: string }): Promise<IncomingCandidateList> {
    const searchParams = new URLSearchParams();

    if (params?.status) searchParams.set("status", params.status);
    if (params?.cursor) searchParams.set("cursor", params.cursor);

    const query = searchParams.toString();

    return apiClient<IncomingCandidateList>(`/quota-requests/incoming${query ? `?${query}` : ""}`);
}

export async function sendOffer(candidateId: string, data: { total_price: number; readiness_lead_days: number; note?: string }): Promise<Offer> {
    return apiClient<Offer>(`/candidates/${candidateId}/offers`, { method: "POST", body: JSON.stringify(data) });
}

export async function rejectCandidate(candidateId: string, reason: string): Promise<void> {
    return apiClient<void>(`/candidates/${candidateId}/reject`, { method: "POST", body: JSON.stringify({ reason }) });
}

export async function counterOffer(offerId: string, data: { total_price: number; note?: string }): Promise<Offer> {
    return apiClient<Offer>(`/offers/${offerId}/counter`, { method: "POST", body: JSON.stringify(data) });
}

export async function acceptOffer(offerId: string): Promise<components["schemas"]["WorkOrderDetail"]> {
    return apiClient<components["schemas"]["WorkOrderDetail"]>(`/offers/${offerId}/accept`, { method: "POST" });
}
