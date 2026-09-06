import { parseRetryAfter } from "@lib/problem";

const API_BASE_URL = import.meta.env.VITE_API_URL ?? "/api";

export function apiUrl(path: string): string {
    return `${API_BASE_URL}${path}`;
}

export class ApiError extends Error {
    status: number;
    data: unknown;
    // Value of the Retry-After response header in seconds, when the server sent
    // one (currently only on 429 responses).
    retryAfterSeconds?: number;

    constructor(status: number, data: unknown, retryAfterSeconds?: number) {
        super(`API Error: ${status}`);
        this.name = "ApiError";
        this.status = status;
        this.data = data;
        this.retryAfterSeconds = retryAfterSeconds;
    }
}

export async function apiClient<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers: Record<string, string> = { ...(options.headers as Record<string, string> | undefined) };

    if (!(options.body instanceof FormData)) {
        headers["Content-Type"] = "application/json";
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        credentials: "include",
        headers,
    });

    if (!response.ok) {
        let data: unknown;

        try {
            data = await response.json();
        } catch {
            data = { title: response.statusText };
        }

        const retryAfterSeconds = parseRetryAfter(response.headers.get("Retry-After"));
        throw new ApiError(response.status, data, retryAfterSeconds);
    }

    if (response.status === 204 || response.status === 202) {
        return undefined as T;
    }

    const contentType = response.headers.get("content-type");
    if (!contentType?.includes("application/json")) {
        return undefined as T;
    }

    const text = await response.text();

    return (text ? JSON.parse(text) : undefined) as T;
}
