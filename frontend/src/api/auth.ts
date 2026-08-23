import { apiClient, ApiError } from "./client";
import type { components } from "./types";

export type MyAccount = components["schemas"]["MyAccount"];
export type LoginRequest = {
    email: string;
    password: string;
};

export async function getMe(): Promise<MyAccount | null> {
    try {
        return await apiClient<MyAccount>("/me");
    } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
            return null;
        }
        throw error;
    }
}

export async function login(data: LoginRequest): Promise<MyAccount> {
    return apiClient<MyAccount>("/auth/login", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export async function logout(): Promise<void> {
    await apiClient<void>("/auth/logout", {
        method: "POST",
    });
}
