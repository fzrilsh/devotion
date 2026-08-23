const API_BASE_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8000";

export class ApiError extends Error {
    status: number;
    data: unknown;

    constructor(status: number, data: unknown) {
        super(`API Error: ${status}`);
        this.name = "ApiError";
        this.status = status;
        this.data = data;
    }
}

export async function apiClient<T>(path: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        credentials: "include",
        headers: {
            "Content-Type": "application/json",
            ...options.headers,
        },
    });

    if (!response.ok) {
        let data: unknown;

        try {
            data = await response.json();
        } catch {
            data = { title: response.statusText };
        }

        throw new ApiError(response.status, data);
    }

    if (response.status === 204) return undefined as T;
    return response.json();
}
