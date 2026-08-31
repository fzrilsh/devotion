import { isStatusFilter, statusFilters } from "./statusFilters";

const meta = {
    reported: { label: "Dilaporkan" },
    in_mediation: { label: "Dalam Mediasi" },
    resolved: { label: "Selesai" },
};

describe("statusFilters", () => {
    it("menurunkan chip dari peta label dan menjaga urutan kuncinya", () => {
        expect(statusFilters(meta)).toEqual([
            { value: "all", label: "Semua" },
            { value: "reported", label: "Dilaporkan" },
            { value: "in_mediation", label: "Dalam Mediasi" },
            { value: "resolved", label: "Selesai" },
        ]);
    });

    it("dapat menempatkan chip Semua di akhir", () => {
        const filters = statusFilters(meta, { allLast: true });
        expect(filters[filters.length - 1]).toEqual({ value: "all", label: "Semua" });
        expect(statusFilters(meta, { allLast: true })[0]).toEqual({ value: "reported", label: "Dilaporkan" });
    });

    it("ikut bertambah saat peta label menambah status", () => {
        const extended = { ...meta, escalated: { label: "Dieskalasi" } };
        expect(statusFilters(extended)).toHaveLength(statusFilters(meta).length + 1);
    });
});

describe("isStatusFilter", () => {
    it("menerima all dan status yang dikenal", () => {
        expect(isStatusFilter(meta, "all")).toBe(true);
        expect(isStatusFilter(meta, "resolved")).toBe(true);
    });

    it("menolak null, kosong, dan status yang tidak dikenal", () => {
        expect(isStatusFilter(meta, null)).toBe(false);
        expect(isStatusFilter(meta, "")).toBe(false);
        expect(isStatusFilter(meta, "escalated")).toBe(false);
    });

    it("menolak properti bawaan Object.prototype", () => {
        expect(isStatusFilter(meta, "toString")).toBe(false);
        expect(isStatusFilter(meta, "constructor")).toBe(false);
    });
});
