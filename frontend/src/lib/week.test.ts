import { addDays, addWeeks, currentWeekStart, isMonday, jakartaIsoDate, weekStartOf, weeksBetween } from "./week";

// The backend rejects any week_start that is not a Monday
// (internal/listing/http.go validatePeriodInput: "Awal minggu harus hari Senin."),
// and platform.WeekStart snaps every boundary in Asia/Jakarta. These tests hold
// the client to the same rule (FR-017, FR-021).
describe("weekStartOf", () => {
    it("membiarkan hari Senin apa adanya (FR-017)", () => {
        expect(weekStartOf("2026-08-31")).toBe("2026-08-31");
    });

    it("membulatkan hari tengah minggu ke Senin sebelumnya (FR-017)", () => {
        expect(weekStartOf("2026-09-02")).toBe("2026-08-31");
        expect(weekStartOf("2026-09-03")).toBe("2026-08-31");
    });

    it("membulatkan hari Minggu ke Senin minggu yang sama, bukan Senin berikutnya (FR-017)", () => {
        expect(weekStartOf("2026-09-06")).toBe("2026-08-31");
    });

    it("menyeberangi batas bulan dan tahun", () => {
        expect(weekStartOf("2026-01-01")).toBe("2025-12-29");
        expect(weekStartOf("2026-03-01")).toBe("2026-02-23");
    });
});

describe("addWeeks", () => {
    it("melangkah tujuh hari per minggu, bukan satu hari (FR-017)", () => {
        expect(addWeeks("2026-08-31", 1)).toBe("2026-09-07");
        expect(addWeeks("2026-08-31", 11)).toBe("2026-11-16");
    });

    it("hasilnya tetap hari Senin sepanjang 26 minggu horizon (FR-088)", () => {
        for (let week = 0; week < 26; week++) {
            expect(isMonday(addWeeks("2026-08-31", week))).toBe(true);
        }
    });

    it("dapat melangkah mundur", () => {
        expect(addWeeks("2026-08-31", -1)).toBe("2026-08-24");
    });
});

describe("addDays", () => {
    it("menyeberangi akhir bulan", () => {
        expect(addDays("2026-08-31", 6)).toBe("2026-09-06");
    });

    it("menghitung tahun kabisat", () => {
        expect(addDays("2028-02-28", 1)).toBe("2028-02-29");
    });
});

describe("jakartaIsoDate", () => {
    it("memakai tanggal WIB, bukan tanggal UTC, untuk instan larut malam (aturan 4)", () => {
        // 2026-08-30T17:30:00Z masih 30 Agustus di UTC tapi sudah 31 Agustus di WIB.
        expect(jakartaIsoDate(new Date("2026-08-30T17:30:00Z"))).toBe("2026-08-31");
    });

    it("tidak menggeser instan yang jauh dari batas hari", () => {
        expect(jakartaIsoDate(new Date("2026-09-02T03:00:00Z"))).toBe("2026-09-02");
    });
});

describe("currentWeekStart", () => {
    it("mengembalikan Senin dari minggu instan itu di WIB (FR-017)", () => {
        expect(currentWeekStart(new Date("2026-09-02T03:00:00Z"))).toBe("2026-08-31");
    });

    it("berpindah minggu tepat pada Senin 00.00 WIB, bukan pada Senin 00.00 UTC", () => {
        // Minggu 2026-09-06 pukul 17.30 UTC sudah Senin 2026-09-07 pukul 00.30 WIB.
        expect(currentWeekStart(new Date("2026-09-06T17:30:00Z"))).toBe("2026-09-07");
        expect(currentWeekStart(new Date("2026-09-06T16:30:00Z"))).toBe("2026-08-31");
    });

    it("hasilnya selalu hari Senin", () => {
        for (let day = 0; day < 14; day++) {
            const instant = new Date(Date.UTC(2026, 8, 1 + day, 9, 0, 0));
            expect(isMonday(currentWeekStart(instant))).toBe(true);
        }
    });
});

describe("weeksBetween", () => {
    it("menghitung jumlah minggu penuh", () => {
        expect(weeksBetween("2026-08-31", "2026-11-16")).toBe(11);
        expect(weeksBetween("2026-08-31", "2026-08-31")).toBe(0);
    });

    it("bernilai negatif bila minggu tujuan lebih awal", () => {
        expect(weeksBetween("2026-08-31", "2026-08-24")).toBe(-1);
    });
});
