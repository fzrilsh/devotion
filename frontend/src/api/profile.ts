import { apiClient } from "./client";
import type { components } from "./types";

export type Profile = components["schemas"]["MyProfile"];
export type PublicProfile = components["schemas"]["PublicProfile"];
export type ProfileUpdateRequest = components["schemas"]["ProfileUpdateRequest"];
export type Reputation = components["schemas"]["Reputation"];
export type Review = components["schemas"]["Review"];
export type ReviewList = components["schemas"]["ReviewList"];

export async function getMyProfile(): Promise<Profile> {
    return apiClient<Profile>("/profile/me");
}

export async function updateMyProfile(data: ProfileUpdateRequest): Promise<Profile> {
    return apiClient<Profile>("/profile/me", {
        method: "PUT",
        body: JSON.stringify(data),
    });
}

export async function getPublicProfile(profileId: string): Promise<PublicProfile> {
    return apiClient<PublicProfile>(`/profile/${profileId}`);
}

// Endpoint publik: ulasan tampil di profil publik tanpa sesi. Ulasan yang
// disembunyikan admin tidak dikembalikan (FR-049, FR-050).
export async function getProfileReviews(profileId: string, cursor?: string): Promise<ReviewList> {
    const searchParams = new URLSearchParams();

    if (cursor) {
        searchParams.set("cursor", cursor);
    }

    const query = searchParams.toString();

    return apiClient<ReviewList>(`/profile/${encodeURIComponent(profileId)}/reviews${query ? `?${query}` : ""}`);
}
