import { ApiError } from "./client";
import type { components } from "./types";

export type UploadedFile = components["schemas"]["UploadedFile"];

export async function uploadFile(kind: "identity_document" | "location_photo", file: File): Promise<UploadedFile> {
    const formData = new FormData();
    formData.append("kind", kind);
    formData.append("file", file);

    const response = await fetch(`${import.meta.env.VITE_API_URL ?? "/api"}/files`, {
        method: "POST",
        credentials: "include",
        body: formData,
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

    return response.json();
}
