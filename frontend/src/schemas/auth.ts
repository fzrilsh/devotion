import { z } from "zod";

const passwordSchema = z.string().min(8, "Kata sandi minimal 8 karakter");

const phoneSchema = z
    .string()
    .transform((value) => value.replace(/[\s-]/g, ""))
    .pipe(z.string().regex(/^(\+62|62|0)8\d{7,11}$/, "Nomor HP belum sesuai format, contoh: 0812 3456 7890"));

export const loginSchema = z.object({
    email: z.string().email("Email tidak valid"),
    password: passwordSchema,
});

export const registerSchema = z
    .object({
        email: z.string().email("Email tidak valid"),
        phone: phoneSchema,
        business_name: z.string().trim().min(3, "Nama usaha minimal 3 karakter"),
        city_code: z.string().regex(/^\d{4}$/, "Pilih kota atau kabupaten yang valid"),
        password: passwordSchema,
        password_confirmation: passwordSchema,
        roles: z.object({
            subcontractor: z.boolean(),
            buyer: z.boolean(),
        }),
    })
    .refine((data) => data.password === data.password_confirmation, {
        path: ["password_confirmation"],
        message: "Konfirmasi kata sandi belum sama",
    })
    .refine((data) => data.roles.subcontractor || data.roles.buyer, {
        path: ["roles"],
        message: "Pilih minimal satu peran",
    });

export const verificationCodeSchema = z.object({
    code: z.string().regex(/^\d{6}$/, "Kode harus terdiri dari 6 angka"),
});

export const recoverRequestSchema = z.object({
    email: z.string().email("Email tidak valid"),
});

export const recoverConfirmSchema = z
    .object({
        email: z.string().email("Email tidak valid"),
        code: z.string().regex(/^\d{6}$/, "Kode harus terdiri dari 6 angka"),
        new_password: passwordSchema,
        password_confirmation: passwordSchema,
    })
    .refine((data) => data.new_password === data.password_confirmation, {
        path: ["password_confirmation"],
        message: "Konfirmasi kata sandi tidak sama",
    });

export type LoginForm = z.infer<typeof loginSchema>;
export type RegisterForm = z.infer<typeof registerSchema>;
export type VerificationCodeForm = z.infer<typeof verificationCodeSchema>;
export type RecoverRequestForm = z.infer<typeof recoverRequestSchema>;
export type RecoverConfirmForm = z.infer<typeof recoverConfirmSchema>;
