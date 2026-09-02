import { z } from "zod";

export const profileSchema = z.object({
    business_name: z.string().min(3, "Nama usaha minimal 3 karakter").max(150, "Maksimal 150 karakter"),
    description: z.string().max(2000, "Deskripsi maksimal 2000 karakter").optional(),
    province_code: z.string().optional(),
    city_code: z.string().optional(),
    latitude: z.number().min(-11.5, "Latitude di luar wilayah Indonesia").max(6.5, "Latitude di luar wilayah Indonesia").nullable().optional(),
    longitude: z.number().min(94.5, "Longitude di luar wilayah Indonesia").max(141.5, "Longitude di luar wilayah Indonesia").nullable().optional(),
});

export type ProfileForm = z.infer<typeof profileSchema>;
