import { useAuth } from "@hooks/useAuth";

// Dipisah dari VerificationGate.tsx supaya berkas komponen itu hanya
// meng-export komponen. Fast refresh Vite mengganti modul yang isinya murni
// komponen tanpa memuat ulang halaman; satu export bukan komponen membuatnya
// jatuh ke full reload, dan state form yang sedang diisi ikut hilang.
export function useAccountVerification() {
    const { user } = useAuth();

    const needsEmail = Boolean(user) && !user?.email_verified;
    const needsPhone = Boolean(user) && !user?.phone_verified;

    return {
        needsEmail,
        needsPhone,
        needsVerification: needsEmail || needsPhone,
    };
}
