import { apiClient } from "./client";
import type { components } from "./types";

export type Notification = components["schemas"]["Notification"];
export type NotificationList = components["schemas"]["NotificationList"];
export type NotificationPreferences = components["schemas"]["NotificationPreferences"];

export async function getNotifications(params?: { unread?: boolean; cursor?: string }): Promise<NotificationList> {
    const searchParams = new URLSearchParams();

    if (params?.unread) {
        searchParams.set("unread", "true");
    }

    if (params?.cursor) {
        searchParams.set("cursor", params.cursor);
    }

    const query = searchParams.toString();

    return apiClient<NotificationList>(`/notifications${query ? `?${query}` : ""}`);
}

export async function markNotificationRead(notificationId: string): Promise<void> {
    await apiClient<void>(`/notifications/${notificationId}/read`, {
        method: "POST",
    });
}

export async function getNotificationPreferences(): Promise<NotificationPreferences> {
    return apiClient<NotificationPreferences>("/notifications/preferences");
}

export async function updateNotificationPreferences(data: NotificationPreferences): Promise<NotificationPreferences> {
    return apiClient<NotificationPreferences>("/notifications/preferences", {
        method: "PUT",
        body: JSON.stringify(data),
    });
}
