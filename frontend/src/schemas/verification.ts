import { z } from "zod";

export const verificationSchema = z.object({
    identity_number: z.string().min(8, "Minimal 8 karakter").max(32, "Maksimal 32 karakter"),
    identity_file_id: z.string().uuid("Dokumen identitas wajib diunggah"),
    location_file_id: z.string().uuid("Foto lokasi wajib diunggah"),
});

export type VerificationForm = z.infer<typeof verificationSchema>;
