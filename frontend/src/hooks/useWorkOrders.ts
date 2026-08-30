import { cancelWorkOrder, changeWorkOrderStatus, confirmWorkOrder, getWorkOrder, getWorkOrderContacts, getWorkOrders, recordPayment, reportDispute, submitReview, type WorkOrderStatus } from "@api/workOrders";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const workOrderKeys = {
    list: (status: WorkOrderStatus[], role?: string) => ["work-orders", "list", status.join(","), role ?? "all"] as const,
    detail: (id: string) => ["work-orders", "detail", id] as const,
    contacts: (id: string) => ["work-orders", "contacts", id] as const,
};

export function useWorkOrders(status: WorkOrderStatus[], role?: "as_buyer" | "as_subcontractor") {
    return useInfiniteQuery({
        queryKey: workOrderKeys.list(status, role),
        queryFn: ({ pageParam }) => getWorkOrders({ status: status.length > 0 ? status : undefined, role, cursor: pageParam }),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        staleTime: 30 * 1000,
    });
}

export function useWorkOrder(workOrderId: string) {
    return useQuery({
        queryKey: workOrderKeys.detail(workOrderId),
        queryFn: () => getWorkOrder(workOrderId),
        enabled: Boolean(workOrderId),
        staleTime: 30 * 1000,
        retry: false,
    });
}

// Kontak lawan memuat email dan nomor WhatsApp, jadi hanya diminta ketika
// pemanggil memang pihak pesanan. Bukan-pihak menerima 404 dan tidak diulang.
export function useWorkOrderContacts(workOrderId: string, enabled: boolean) {
    return useQuery({
        queryKey: workOrderKeys.contacts(workOrderId),
        queryFn: () => getWorkOrderContacts(workOrderId),
        enabled: Boolean(workOrderId) && enabled,
        staleTime: 5 * 60 * 1000,
        retry: false,
    });
}

function useDetailInvalidator(workOrderId: string) {
    const queryClient = useQueryClient();

    return (updated?: unknown) => {
        if (updated) {
            queryClient.setQueryData(workOrderKeys.detail(workOrderId), updated);
        }

        queryClient.invalidateQueries({
            queryKey: workOrderKeys.detail(workOrderId),
        });

        queryClient.invalidateQueries({
            queryKey: ["work-orders", "list"],
        });
    };
}

export function useChangeWorkOrderStatus(workOrderId: string) {
    const invalidate = useDetailInvalidator(workOrderId);

    return useMutation({
        mutationFn: ({ newStatus, note }: { newStatus: "production" | "completed" | "shipped"; note?: string }) => changeWorkOrderStatus(workOrderId, newStatus, note),
        onSuccess: (updated) => invalidate(updated),
    });
}

export function useConfirmWorkOrder(workOrderId: string) {
    const invalidate = useDetailInvalidator(workOrderId);

    return useMutation({
        mutationFn: () => confirmWorkOrder(workOrderId),
        onSuccess: (updated) => invalidate(updated),
    });
}

export function useCancelWorkOrder(workOrderId: string) {
    const invalidate = useDetailInvalidator(workOrderId);

    return useMutation({
        mutationFn: (reason: string) => cancelWorkOrder(workOrderId, reason),
        onSuccess: (updated) => invalidate(updated),
    });
}

export function useRecordPayment(workOrderId: string) {
    const invalidate = useDetailInvalidator(workOrderId);

    return useMutation({
        mutationFn: (data: { direction: "sent" | "received"; date: string; note?: string }) => recordPayment(workOrderId, data),
        onSuccess: (updated) => invalidate(updated),
    });
}

export function useReportDispute(workOrderId: string) {
    const invalidate = useDetailInvalidator(workOrderId);

    return useMutation({
        mutationFn: (reportBody: string) => reportDispute(workOrderId, reportBody),
        onSuccess: (updated) => invalidate(updated),
    });
}

export function useSubmitReview(workOrderId: string) {
    const invalidate = useDetailInvalidator(workOrderId);

    return useMutation({
        mutationFn: (data: { rating: number; text?: string }) => submitReview(workOrderId, data),
        onSuccess: () => invalidate(),
    });
}
