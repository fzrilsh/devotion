import { apiClient } from "./client";
import type { components } from "./types";

export type WorkOrderList = components["schemas"]["WorkOrderList"];
export type WorkOrderDetail = components["schemas"]["WorkOrderDetail"];
export type WorkOrderStatus = components["schemas"]["WorkOrderStatus"];
export type PaymentRecord = components["schemas"]["PaymentRecord"];
export type Review = components["schemas"]["Review"];
export type Dispute = components["schemas"]["Dispute"];

// work_order_id kadang tiba sebagai path penuh ("/work-orders/<id>").
// Ambil segmen terakhir supaya request tidak pernah membawa prefix dobel.
function normalizeWorkOrderId(workOrderId: string): string {
    return workOrderId.split("/").filter(Boolean).pop() ?? workOrderId;
}

export async function getWorkOrders(params?: { status?: WorkOrderStatus[]; role?: "as_buyer" | "as_subcontractor"; cursor?: string }): Promise<WorkOrderList> {
    const searchParams = new URLSearchParams();

    for (const status of params?.status ?? []) {
        searchParams.append("status", status);
    }

    if (params?.role) searchParams.set("role", params.role);
    if (params?.cursor) searchParams.set("cursor", params.cursor);

    const query = searchParams.toString();

    return apiClient<WorkOrderList>(`/work-orders${query ? `?${query}` : ""}`);
}

export async function getWorkOrder(workOrderId: string): Promise<WorkOrderDetail> {
    return apiClient<WorkOrderDetail>(`/work-orders/${normalizeWorkOrderId(workOrderId)}`);
}

export async function changeWorkOrderStatus(workOrderId: string, newStatus: "production" | "completed" | "shipped", note?: string): Promise<WorkOrderDetail> {
    return apiClient<WorkOrderDetail>(`/work-orders/${normalizeWorkOrderId(workOrderId)}/status`, { method: "POST", body: JSON.stringify({ new_status: newStatus, note }) });
}

export async function confirmWorkOrder(workOrderId: string): Promise<WorkOrderDetail> {
    return apiClient<WorkOrderDetail>(`/work-orders/${normalizeWorkOrderId(workOrderId)}/confirm`, { method: "POST" });
}

export async function cancelWorkOrder(workOrderId: string, reason: string): Promise<WorkOrderDetail> {
    return apiClient<WorkOrderDetail>(`/work-orders/${normalizeWorkOrderId(workOrderId)}/cancel`, { method: "POST", body: JSON.stringify({ reason }) });
}

export async function recordPayment(workOrderId: string, data: { direction: "sent" | "received"; date: string; note?: string }): Promise<PaymentRecord> {
    return apiClient<PaymentRecord>(`/work-orders/${normalizeWorkOrderId(workOrderId)}/payments`, { method: "POST", body: JSON.stringify(data) });
}

export async function reportDispute(workOrderId: string, reportBody: string): Promise<Dispute> {
    return apiClient<Dispute>(`/work-orders/${normalizeWorkOrderId(workOrderId)}/disputes`, { method: "POST", body: JSON.stringify({ report_body: reportBody }) });
}

export async function submitReview(workOrderId: string, data: { rating: number; text?: string }): Promise<Review> {
    return apiClient<Review>(`/work-orders/${normalizeWorkOrderId(workOrderId)}/reviews`, { method: "POST", body: JSON.stringify(data) });
}
