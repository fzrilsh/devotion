import { getNotificationPreferences, getNotifications, markNotificationRead, updateNotificationPreferences, type NotificationPreferences } from "@api/notifications";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const notificationKeys = {
    list: ["notifications", "list"] as const,
    preferences: ["notifications", "preferences"] as const,
};

export function useUnreadCount() {
    const query = useQuery({
        queryKey: notificationKeys.list,
        queryFn: () => getNotifications(),
        staleTime: 60 * 1000,
        refetchOnWindowFocus: true,
        retry: false,
    });

    return {
        unreadCount: query.data?.unread_count ?? 0,
        isLoading: query.isLoading,
    };
}

export function useNotifications(params?: { unread?: boolean }) {
    return useInfiniteQuery({
        queryKey: [...notificationKeys.list, { unread: params?.unread ?? false }],
        queryFn: ({ pageParam }) => getNotifications({ unread: params?.unread, cursor: pageParam }),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        staleTime: 30 * 1000,
        refetchOnWindowFocus: true,
        retry: false,
    });
}

export function useMarkNotificationRead() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (notificationId: string) => markNotificationRead(notificationId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: notificationKeys.list });
        },
    });
}

export function useNotificationPreferences() {
    return useQuery({
        queryKey: notificationKeys.preferences,
        queryFn: getNotificationPreferences,
        staleTime: 5 * 60 * 1000,
        retry: false,
    });
}

export function useUpdateNotificationPreferences() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: NotificationPreferences) => updateNotificationPreferences(data),
        onSuccess: (updated) => {
            queryClient.setQueryData(notificationKeys.preferences, updated);
        },
    });
}
