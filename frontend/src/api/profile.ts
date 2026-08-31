import { apiClient } from "./client";
import type { components } from "./types";

export type Profile = components["schemas"]["MyProfile"];
export type PublicProfile = components["schemas"]["PublicProfile"];
export type ProfileUpdateRequest = components["schemas"]["ProfileUpdateRequest"];
export type Reputation = components["schemas"]["Reputation"];

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
