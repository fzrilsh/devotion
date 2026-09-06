import { normalizePhone } from "./phone";

// The backend rejects anything that does not match `^\+62[0-9]{8,13}$` on the raw
// request body (internal/account/handlers.go), so every accepted form must come
// out of normalizePhone with the leading plus intact (FR-001, FR-002).
const backendPhoneRe = /^\+62[0-9]{8,13}$/;

describe("normalizePhone", () => {
    it("mengubah awalan 0 menjadi +62 (FR-001)", () => {
        expect(normalizePhone("081234567890")).toBe("+6281234567890");
    });

    it("menambahkan plus pada awalan 62 (FR-001)", () => {
        expect(normalizePhone("6281234567890")).toBe("+6281234567890");
    });

    it("mempertahankan nomor yang sudah E.164 (FR-001)", () => {
        expect(normalizePhone("+6281234567890")).toBe("+6281234567890");
    });

    it("melengkapi nomor yang diketik tanpa awalan (FR-001)", () => {
        expect(normalizePhone("81234567890")).toBe("+6281234567890");
    });

    it("membuang spasi, tanda hubung, dan tanda kurung (FR-001)", () => {
        expect(normalizePhone("0812 3456-7890")).toBe("+6281234567890");
        expect(normalizePhone("(0812) 3456 7890")).toBe("+6281234567890");
    });

    it("mengembalikan string kosong bila tidak ada angka", () => {
        expect(normalizePhone("")).toBe("");
        expect(normalizePhone("   ")).toBe("");
    });

    it("hasilnya selalu diterima pola backend (FR-002)", () => {
        const typed = ["081234567890", "6281234567890", "+6281234567890", "81234567890", "0812 3456-7890", "0851234567"];

        for (const value of typed) {
            expect(normalizePhone(value)).toMatch(backendPhoneRe);
        }
    });
});
