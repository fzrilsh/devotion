import { z } from "zod";

export const listingSchema = z.object({
    weekly_capacity: z
        .number("Kapasitas mingguan wajib diisi.")
        .int("Kapasitas harus bilangan bulat.")
        .min(1, "Kapasitas mingguan minimal 1 unit."),
    readiness_lead_days: z
        .number("Jeda kesiapan wajib diisi.")
        .int("Jeda kesiapan harus bilangan bulat.")
        .min(0, "Jeda kesiapan minimal 0 hari.")
        .max(90, "Jeda kesiapan maksimal 90 hari."),
    product_item_ids: z.array(z.string()).min(1, "Pilih minimal satu jenis produk."),
    machines: z.array(
        z.object({
            item_id: z.string(),
            machine_count: z.number().int("Jumlah mesin harus bilangan bulat.").min(1, "Jumlah mesin minimal 1.").max(999, "Jumlah mesin maksimal 999."),
        }),
    ),
});

export type ListingForm = z.infer<typeof listingSchema>;
