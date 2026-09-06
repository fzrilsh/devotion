import type { DisputeResult } from "@api/admin";

export function validateDisputeResolution({
    result,
    liableProfileId,
    note,
}: {
    result: DisputeResult;
    liableProfileId: string;
    note: string;
}): string | null {
    if (result === "cancelled" && !liableProfileId) {
        return "Pilih pihak yang menanggung pembatalan.";
    }

    if (result === "cancelled" && !note.trim()) {
        return "Catatan wajib diisi saat hasil mediasi dibatalkan.";
    }

    return null;
}
