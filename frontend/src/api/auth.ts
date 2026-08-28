import { apiClient, ApiError } from "./client";
import type { components } from "./types";

export type MyAccount = components["schemas"]["MyAccount"];
export type RegisterRequest = components["schemas"]["RegisterRequest"];
export type RegisterResponse = components["schemas"]["RegisterResponse"];
export type VerificationCodeRequest = components["schemas"]["VerificationCodeRequest"];

export type LoginRequest = {
    email: string;
    password: string;
};

export type RecoverRequest = {
    email: string;
};

export type RecoverConfirmRequest = {
    email: string;
    code: string;
    new_password: string;
};

export type ResendCodeRequest = {
    target: string;
    channel: "email" | "whatsapp";
};

export async function register(data: RegisterRequest): Promise<RegisterResponse> {
    return apiClient<RegisterResponse>("/auth/register", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export type RolesUpdateRequest = {
    subcontractor?: boolean;
    buyer?: boolean;
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

export async function verifyEmail(data: VerificationCodeRequest): Promise<void> {
    await apiClient<void>("/auth/verify-email", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export async function verifyPhone(data: VerificationCodeRequest): Promise<void> {
    await apiClient<void>("/auth/verify-phone", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export async function resendCode(data: ResendCodeRequest): Promise<void> {
    await apiClient<void>("/auth/resend-code", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export async function requestPasswordRecovery(data: RecoverRequest): Promise<void> {
    await apiClient<void>("/auth/recover/request", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export async function confirmPasswordRecovery(data: RecoverConfirmRequest): Promise<void> {
    await apiClient<void>("/auth/recover/confirm", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

export async function updateMyRoles(data: RolesUpdateRequest): Promise<MyAccount> {
    return apiClient<MyAccount>("/me/roles", {
        method: "PATCH",
        body: JSON.stringify(data),
    });
}
