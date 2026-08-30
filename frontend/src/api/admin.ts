import { apiClient } from "./client";
import type { components, paths } from "./types";

export type VerificationRequest = components["schemas"]["VerificationRequest"];
export type VerificationRequestList = components["schemas"]["VerificationRequestList"];
export type VerificationStatus = components["schemas"]["VerificationStatus"];
export type ItemProposal = components["schemas"]["ItemProposal"];
export type Dispute = components["schemas"]["Dispute"];
export type DisputeStatus = components["schemas"]["DisputeStatus"];
export type LateOrderSummary = components["schemas"]["LateOrderSummary"];
export type LateOrderList = components["schemas"]["LateOrderList"];
export type CatalogItem = components["schemas"]["CatalogItem"];
export type Review = components["schemas"]["Review"];
export type ReviewList = components["schemas"]["ReviewList"];
export type WhatsAppStatus = components["schemas"]["WhatsAppStatus"];
export type DisputeResult = Exclude<components["schemas"]["Dispute"]["result"], null | undefined>;

type VerificationDecision = NonNullable<paths["/admin/verification/{requestId}/decision"]["post"]>["requestBody"];
type ProposalDecision = NonNullable<paths["/admin/proposals/{proposalId}/decision"]["post"]>["requestBody"];
type MasterItemCreate = NonNullable<paths["/admin/master/items"]["post"]>["requestBody"];
type MasterItemUpdate = NonNullable<paths["/admin/master/items/{itemId}"]["patch"]>["requestBody"];
type DisputeResolution = NonNullable<paths["/admin/disputes/{disputeId}/resolve"]["post"]>["requestBody"];

type JsonBody<T> = T extends { content: { "application/json": infer Body } } ? Body : never;

export type VerificationDecisionRequest = JsonBody<VerificationDecision>;
export type ProposalDecisionRequest = JsonBody<ProposalDecision>;
export type MasterItemCreateRequest = JsonBody<MasterItemCreate>;
export type MasterItemUpdateRequest = JsonBody<MasterItemUpdate>;
export type DisputeResolutionRequest = JsonBody<DisputeResolution>;

function extractItems<T>(response: T[] | { items?: T[]; data?: T[] }): T[] {
    if (Array.isArray(response)) {
        return response;
    }

    return response.items ?? response.data ?? [];
}

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

    const response = await apiClient<ItemProposal[] | { items?: ItemProposal[]; data?: ItemProposal[] }>(`/admin/proposals${query ? `?${query}` : ""}`);

    return extractItems(response);
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

    const response = await apiClient<Dispute[] | { items?: Dispute[]; data?: Dispute[] }>(`/admin/disputes${query ? `?${query}` : ""}`);

    return extractItems(response);
}

export async function getLateOrders(params?: { cursor?: string }): Promise<LateOrderList> {
    const searchParams = new URLSearchParams();

    if (params?.cursor) {
        searchParams.set("cursor", params.cursor);
    }

    const query = searchParams.toString();

    return apiClient<LateOrderList>(`/admin/late-orders${query ? `?${query}` : ""}`);
}

export async function decideVerification(requestId: string, data: VerificationDecisionRequest): Promise<VerificationRequest> {
    return apiClient<VerificationRequest>(`/admin/verification/${encodeURIComponent(requestId)}/decision`, { method: "POST", body: JSON.stringify(data) });
}

export async function getMasterItems(kind?: "product" | "machine"): Promise<CatalogItem[]> {
    const searchParams = new URLSearchParams();
    if (kind) searchParams.set("kind", kind);
    const query = searchParams.toString();

    return apiClient<CatalogItem[]>(`/admin/master/items${query ? `?${query}` : ""}`);
}

export async function createMasterItem(data: MasterItemCreateRequest): Promise<CatalogItem> {
    return apiClient<CatalogItem>("/admin/master/items", { method: "POST", body: JSON.stringify(data) });
}

export async function updateMasterItem(itemId: string, data: MasterItemUpdateRequest): Promise<CatalogItem> {
    return apiClient<CatalogItem>(`/admin/master/items/${encodeURIComponent(itemId)}`, { method: "PATCH", body: JSON.stringify(data) });
}

export async function decideProposal(proposalId: string, data: ProposalDecisionRequest): Promise<ItemProposal> {
    return apiClient<ItemProposal>(`/admin/proposals/${encodeURIComponent(proposalId)}/decision`, { method: "POST", body: JSON.stringify(data) });
}

export async function hideReview(reviewId: string, reason: string): Promise<Review> {
    return apiClient<Review>(`/admin/reviews/${encodeURIComponent(reviewId)}/hide`, { method: "POST", body: JSON.stringify({ reason }) });
}

export async function mediateDispute(disputeId: string): Promise<Dispute> {
    return apiClient<Dispute>(`/admin/disputes/${encodeURIComponent(disputeId)}/mediate`, { method: "POST" });
}

export async function resolveDispute(disputeId: string, data: DisputeResolutionRequest): Promise<Dispute> {
    return apiClient<Dispute>(`/admin/disputes/${encodeURIComponent(disputeId)}/resolve`, { method: "POST", body: JSON.stringify(data) });
}

export async function getWhatsAppStatus(): Promise<WhatsAppStatus> {
    return apiClient<WhatsAppStatus>("/admin/whatsapp");
}

// Membuang siklus pemasangan yang berjalan lalu memulai yang baru, sehingga admin
// mendapat kode QR segar tanpa akses server. Bentuk balasannya sama dengan GET,
// jadi hasilnya langsung ditulis ke cache status.
export async function reconnectWhatsApp(): Promise<WhatsAppStatus> {
    return apiClient<WhatsAppStatus>("/admin/whatsapp/reconnect", { method: "POST" });
}
