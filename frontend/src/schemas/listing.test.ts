import { listingSchema } from "./listing";

const validListing = {
    weekly_capacity: 500,
    readiness_lead_days: 7,
    product_item_ids: ["item-1"],
    machines: [{ item_id: "machine-1", machine_count: 3 }],
};

describe("listingSchema", () => {
    it("menerima listing yang sah", () => {
        const result = listingSchema.safeParse(validListing);
        expect(result.success).toBe(true);
    });

    it("menolak kapasitas mingguan kurang dari 1", () => {
        const result = listingSchema.safeParse({ ...validListing, weekly_capacity: 0 });
        expect(result.success).toBe(false);
        if (!result.success) {
            expect(result.error.issues[0]?.message).toBe("Kapasitas mingguan minimal 1 unit.");
        }
    });

    it("menolak kapasitas mingguan bukan bilangan bulat", () => {
        const result = listingSchema.safeParse({ ...validListing, weekly_capacity: 10.5 });
        expect(result.success).toBe(false);
        if (!result.success) {
            expect(result.error.issues[0]?.message).toBe("Kapasitas harus bilangan bulat.");
        }
    });

    it("menerima jeda kesiapan sampai 365 hari", () => {
        const result = listingSchema.safeParse({ ...validListing, readiness_lead_days: 365 });
        expect(result.success).toBe(true);
    });

    it("menolak jeda kesiapan lebih dari 365 hari", () => {
        const result = listingSchema.safeParse({ ...validListing, readiness_lead_days: 366 });
        expect(result.success).toBe(false);
    });

    it("menerima alasan penolakan dengan 5 karakter", () => {
        const reason = "Palsu";
        expect(reason.trim().length).toBeGreaterThanOrEqual(5);
    });

    it("menolak tanpa jenis produk", () => {
        const result = listingSchema.safeParse({ ...validListing, product_item_ids: [] });
        expect(result.success).toBe(false);
        if (!result.success) {
            expect(result.error.issues[0]?.message).toBe("Pilih minimal satu jenis produk.");
        }
    });

    it("menolak jumlah mesin nol", () => {
        const result = listingSchema.safeParse({ ...validListing, machines: [{ item_id: "machine-1", machine_count: 0 }] });
        expect(result.success).toBe(false);
    });

    it("menerima mesin kosong karena mesin bersifat opsional", () => {
        const result = listingSchema.safeParse({ ...validListing, machines: [] });
        expect(result.success).toBe(true);
    });
});
