import { apiClient } from "./client";
import type { components } from "./types";

export type UploadedFile = components["schemas"]["UploadedFile"];
export type VerificationRequest = components["schemas"]["VerificationRequest"];
export type VerificationStatus = components["schemas"]["VerificationStatus"];

export type FileKind = "identity_document" | "location_photo";

export async function uploadFile(kind: FileKind, file: File): Promise<UploadedFile> {
    const formData = new FormData();
    formData.set("kind", kind);
    formData.set("file", file);

    return apiClient<UploadedFile>("/files", { method: "POST", body: formData });
}

export async function getMyVerificationRequests(): Promise<VerificationRequest[]> {
    return apiClient<VerificationRequest[]>("/verification");
}

export async function submitVerification(data: { identity_number: string; identity_file_id: string; location_file_id: string }): Promise<VerificationRequest> {
    return apiClient<VerificationRequest>("/verification", { method: "POST", body: JSON.stringify(data) });
}
