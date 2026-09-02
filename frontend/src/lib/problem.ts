import { ApiError } from "@api/client";

export type ProblemDetails = { code?: string; detail: string; meta?: Record<string, unknown> };

export function getProblem(error: unknown): ProblemDetails | null {
    if (!(error instanceof ApiError)) return null;

    if (typeof error.data === "object" && error.data !== null && "detail" in error.data && typeof error.data.detail === "string") {
        const data = error.data as { code?: string; detail: string; meta?: Record<string, unknown> };
        return { code: data.code, detail: data.detail, meta: data.meta };
    }

    if (error.status === 401) return { detail: "Sesi Anda habis, silakan masuk kembali." };
    if (error.status === 403) return { detail: "Anda tidak berwenang melakukan aksi ini." };
    if (error.status === 410) return { detail: "Batas waktu balasan request ini sudah terlampaui." };

    return null;
}

export function getProblemMessage(error: unknown, fallback = "Aksi tidak dapat diproses. Silakan coba lagi.", statusMessages: Record<number, string> = {}): string {
    if (error instanceof ApiError && statusMessages[error.status]) return statusMessages[error.status];

    const problem = getProblem(error);
    if (problem) {
        if (error instanceof ApiError && typeof error.data === "object" && error.data !== null && "errors" in error.data && Array.isArray(error.data.errors)) {
            const fieldMessages = error.data.errors
                .filter((item): item is { message?: string } => typeof item === "object" && item !== null)
                .map((item) => item.message)
                .filter((message): message is string => Boolean(message));

            if (fieldMessages.length > 0) return `${problem.detail} ${fieldMessages.join(" ")}`;
        }

        return problem.detail;
    }

    if (error instanceof ApiError && typeof error.data === "object" && error.data !== null && "title" in error.data && typeof error.data.title === "string") {
        return error.data.title;
    }

    return fallback;
}