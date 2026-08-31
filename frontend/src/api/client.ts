const API_BASE_URL = import.meta.env.VITE_API_URL ?? "/api";

// Alamat penuh satu endpoint, untuk hal yang tidak lewat fetch: pratayang berkas
// yang dibuka di tab baru. Jangan menulis prefiks "/api" di halaman, karena
// VITE_API_URL bisa menunjuk host lain.
export function apiUrl(path: string): string {
    return `${API_BASE_URL}${path}`;
}

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
    const headers: Record<string, string> = { ...(options.headers as Record<string, string> | undefined) };

    // JSON bawaan; multipart dibiarkan tanpa Content-Type supaya fetch memasang boundary sendiri.
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

        throw new ApiError(response.status, data);
    }

    if (response.status === 204 || response.status === 202) {
        return undefined as T;
    }

    const contentType = response.headers.get("content-type");
    if (!contentType?.includes("application/json")) {
        return undefined as T;
    }

    // Body sukses bisa kosong; jangan anggap itu kegagalan parse.
    const text = await response.text();

    return (text ? JSON.parse(text) : undefined) as T;
}
