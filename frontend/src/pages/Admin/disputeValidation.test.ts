import { validateDisputeResolution } from "./disputeValidation";

describe("validateDisputeResolution", () => {
    it("memerlukan pihak dan catatan saat hasil mediasi dibatalkan (FR-048)", () => {
        expect(validateDisputeResolution({ result: "cancelled", liableProfileId: "", note: "" })).toBe("Pilih pihak yang menanggung pembatalan.");
        expect(validateDisputeResolution({ result: "cancelled", liableProfileId: "buyer-1", note: "" })).toBe("Catatan wajib diisi saat hasil mediasi dibatalkan.");
        expect(validateDisputeResolution({ result: "cancelled", liableProfileId: "buyer-1", note: "Alasan pembatalan jelas." })).toBeNull();
    });
});
