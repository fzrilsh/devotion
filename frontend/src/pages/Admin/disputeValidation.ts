export function validateDisputeResolution({
    result,
    liableProfileId,
    note,
}: {
    result: "continued" | "confirmed" | "cancelled";
    liableProfileId: string;
    note: string;
}): string | null {
    if (result === "cancelled" && !liableProfileId) {
        return "Pilih pihak yang menanggung pembatalan.";
    }

    if (result === "cancelled" && !note) {
        return "Catatan wajib diisi saat hasil mediasi dibatalkan.";
    }

    return null;
}
