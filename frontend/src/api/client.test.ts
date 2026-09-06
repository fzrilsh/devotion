// client.ts membaca import.meta.env yang tidak bisa dievaluasi ts-jest, jadi
// modulnya diganti, sama seperti pola di FilePreviewModal.test.tsx. Yang diuji
// di sini pembacaan header Retry-After yang dilakukan client lewat fungsi murni
// parseRetryAfter di lib/problem.ts.
jest.mock("@api/client", () => ({
    ApiError: class ApiError extends Error {},
}));

import { parseRetryAfter } from "@lib/problem";

describe("parseRetryAfter", () => {
    it("mengubah nilai header Retry-After detik menjadi angka", () => {
        expect(parseRetryAfter("42")).toBe(42);
    });

    it("mengembalikan undefined bila header tidak ada", () => {
        expect(parseRetryAfter(null)).toBeUndefined();
    });

    it("mengembalikan undefined bila header bukan angka", () => {
        expect(parseRetryAfter("nanti")).toBeUndefined();
        expect(parseRetryAfter("")).toBeUndefined();
    });
});
