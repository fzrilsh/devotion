import { apiClient } from "./client";
import type { components } from "./types";

export type VerificationRequest = components["schemas"]["VerificationRequest"];
export type VerificationRequestList = components["schemas"]["VerificationRequestList"];
export type VerificationStatus = components["schemas"]["VerificationStatus"];
export type ItemProposal = components["schemas"]["ItemProposal"];
export type Dispute = components["schemas"]["Dispute"];
export type DisputeStatus = components["schemas"]["DisputeStatus"];
export type WorkOrderList = components["schemas"]["WorkOrderList"];
export type CatalogItem = components["schemas"]["CatalogItem"];
export type Review = components["schemas"]["Review"];
export type ReviewList = components["schemas"]["ReviewList"];
export type WhatsAppStatus = components["schemas"]["WhatsAppStatus"];
export type DisputeResult = "cancelled" | "continued" | "confirmed";

export async function getVerificationQueue(params?: { status?: VerificationStatus; cursor?: string }): Promise<VerificationRequestList> {
    const searchParams = new URLSearchParams();

    if (params?.status) {
        searchParams.set("status", params.status);
    }

    if (params?.cursor) {
        searchParams.set("cursor", params.cursor);
    }

    const query = searchParams.toString();

    return apiClient<VerificationRequestList>(`/admin/verification${query ? `?${query}` : ""}`);
}

export async function getItemProposals(params?: { cursor?: string }): Promise<ItemProposal[]> {
    const searchParams = new URLSearchParams();

    if (params?.cursor) {
        searchParams.set("cursor", params.cursor);
    }

    const query = searchParams.toString();

    return apiClient<ItemProposal[]>(`/admin/proposals${query ? `?${query}` : ""}`);
}

export async function getDisputes(params?: { status?: DisputeStatus; cursor?: string }): Promise<Dispute[]> {
    const searchParams = new URLSearchParams();

    if (params?.status) {
        searchParams.set("status", params.status);
    }

    if (params?.cursor) {
        searchParams.set("cursor", params.cursor);
    }

    const query = searchParams.toString();

    return apiClient<Dispute[]>(`/admin/disputes${query ? `?${query}` : ""}`);
}

export async function getLateOrders(params?: { cursor?: string }): Promise<WorkOrderList> {
    const searchParams = new URLSearchParams();

    if (params?.cursor) {
        searchParams.set("cursor", params.cursor);
    }

    const query = searchParams.toString();

    return apiClient<WorkOrderList>(`/admin/late-orders${query ? `?${query}` : ""}`);
}

export async function decideVerification(requestId: string, data: { decision: "approved" | "rejected"; reason?: string }): Promise<VerificationRequest> {
    return apiClient<VerificationRequest>(`/admin/verification/${requestId}/decision`, { method: "POST", body: JSON.stringify(data) });
}

export async function getMasterItems(kind?: "product" | "machine"): Promise<CatalogItem[]> {
    const query = kind ? `?kind=${kind}` : "";

    return apiClient<CatalogItem[]>(`/admin/master/items${query}`);
}

export async function createMasterItem(data: { kind: "product" | "machine"; name: string }): Promise<CatalogItem> {
    return apiClient<CatalogItem>("/admin/master/items", { method: "POST", body: JSON.stringify(data) });
}

export async function updateMasterItem(itemId: string, data: { name?: string; active?: boolean }): Promise<CatalogItem> {
    return apiClient<CatalogItem>(`/admin/master/items/${itemId}`, { method: "PATCH", body: JSON.stringify(data) });
}

export async function decideProposal(proposalId: string, data: { decision: "approved" | "rejected"; reason?: string }): Promise<ItemProposal> {
    return apiClient<ItemProposal>(`/admin/proposals/${proposalId}/decision`, { method: "POST", body: JSON.stringify(data) });
}

export async function getProfileReviews(profileId: string, cursor?: string): Promise<ReviewList> {
    const searchParams = new URLSearchParams();

    if (cursor) {
        searchParams.set("cursor", cursor);
    }

    const query = searchParams.toString();

    return apiClient<ReviewList>(`/profile/${profileId}/reviews${query ? `?${query}` : ""}`);
}

export async function hideReview(reviewId: string, reason: string): Promise<Review> {
    return apiClient<Review>(`/admin/reviews/${reviewId}/hide`, { method: "POST", body: JSON.stringify({ reason }) });
}

export async function mediateDispute(disputeId: string): Promise<Dispute> {
    return apiClient<Dispute>(`/admin/disputes/${disputeId}/mediate`, { method: "POST" });
}

export async function resolveDispute(disputeId: string, data: { result: DisputeResult; allocation_reversed?: boolean; liable_profile_id?: string | null; note?: string }): Promise<Dispute> {
    return apiClient<Dispute>(`/admin/disputes/${disputeId}/resolve`, { method: "POST", body: JSON.stringify(data) });
}

export async function getWhatsAppStatus(): Promise<WhatsAppStatus> {
    return apiClient<WhatsAppStatus>("/admin/whatsapp");
}
