import { apiClient, ApiError } from "./client";
import type { components, paths } from "./types";

export type MyAccount = components["schemas"]["MyAccount"];
export type RegisterRequest = components["schemas"]["RegisterRequest"];
export type RegisterResponse = components["schemas"]["RegisterResponse"];
export type VerificationCodeRequest = components["schemas"]["VerificationCodeRequest"];

type LoginBody = NonNullable<paths["/auth/login"]["post"]>["requestBody"];
type RecoverBody = NonNullable<paths["/auth/recover/request"]["post"]>["requestBody"];
type RecoverConfirmBody = NonNullable<paths["/auth/recover/confirm"]["post"]>["requestBody"];
type ResendCodeBody = NonNullable<paths["/auth/resend-code"]["post"]>["requestBody"];
type RolesUpdateBody = NonNullable<paths["/me/roles"]["patch"]>["requestBody"];

type JsonBody<T> = T extends { content: { "application/json": infer Body } } ? Body : never;

export type LoginRequest = JsonBody<LoginBody>;
export type RecoverRequest = JsonBody<RecoverBody>;
export type RecoverConfirmRequest = JsonBody<RecoverConfirmBody>;
export type ResendCodeRequest = JsonBody<ResendCodeBody>;
export type RolesUpdateRequest = JsonBody<RolesUpdateBody>;

export async function register(data: RegisterRequest): Promise<RegisterResponse> {
    return apiClient<RegisterResponse>("/auth/register", {
        method: "POST",
        body: JSON.stringify(data),
    });
}

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
