import { formatRupiah } from "./meta";

describe("formatRupiah", () => {
    it("memformat bilangan bulat sebagai rupiah tanpa desimal", () => {
        const formatted = formatRupiah(1500000);
        expect(formatted).toContain("1.500.000");
        expect(formatted).not.toContain(",00");
    });

    it("memformat nol", () => {
        expect(formatRupiah(0)).toContain("0");
    });

    it("tidak membulatkan ke pecahan", () => {
        const formatted = formatRupiah(999999);
        expect(formatted).toContain("999.999");
    });
});
