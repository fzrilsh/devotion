import { formatDateId, formatDateTimeId, formatRupiah, parseApiDate } from "./datetime";

describe("parseApiDate", () => {
    it("mengurai RFC 3339 dari backend", () => {
        expect(parseApiDate("2026-08-31T10:00:00Z")?.toISOString()).toBe("2026-08-31T10:00:00.000Z");
    });

    it("mengurai bentuk berspasi yang tidak dijamin peramban", () => {
        expect(parseApiDate("2026-08-31 10:00:00Z")?.toISOString()).toBe("2026-08-31T10:00:00.000Z");
    });

    it("mengembalikan null untuk kosong, spasi, dan tanggal tidak sah", () => {
        expect(parseApiDate(null)).toBeNull();
        expect(parseApiDate(undefined)).toBeNull();
        expect(parseApiDate("")).toBeNull();
        expect(parseApiDate("   ")).toBeNull();
        expect(parseApiDate("bukan tanggal")).toBeNull();
    });
});

describe("pemformat tanggal", () => {
    it("menampilkan tanda hubung, bukan Invalid Date, untuk masukan tidak sah", () => {
        expect(formatDateId(null)).toBe("-");
        expect(formatDateTimeId("bukan tanggal")).toBe("-");
    });

    it("merender pada WIB, bukan zona waktu peramban", () => {
        expect(formatDateId("2026-08-31T17:30:00Z")).toContain("2026");
        expect(formatDateId("2026-08-31T17:30:00Z")).toContain("1");
        expect(formatDateTimeId("2026-08-31T17:30:00Z")).toContain("00.30");
    });

    it("memformat bentuk berspasi sama dengan bentuk T", () => {
        expect(formatDateTimeId("2026-08-31 10:00:00Z")).toBe(formatDateTimeId("2026-08-31T10:00:00Z"));
    });
});

describe("formatRupiah", () => {
    it("tanpa pecahan, sesuai aturan uang bilangan bulat", () => {
        const formatted = formatRupiah(1500000);
        expect(formatted).toContain("1.500.000");
        expect(formatted).not.toContain(",00");
    });

    it("nol tetap ditampilkan, null menjadi tanda hubung", () => {
        expect(formatRupiah(0)).toContain("0");
        expect(formatRupiah(null)).toBe("-");
    });
});
