import type { PaymentMismatch } from "@api/workOrders";
import { cn } from "@lib/utils";
import { LuTriangleAlert } from "react-icons/lu";

// Penjelasan ketidakcocokan pernyataan pembayaran. Bendera datang dari server
// (FR-043); halaman tidak membandingkan sendiri daftar payments, karena aturan
// pembandingnya milik domain dan hanya boleh hidup di satu tempat.
function describe(mismatch: PaymentMismatch): string {
    if (mismatch.kind === "missing_counterpart") {
        return "Satu pihak sudah menyatakan pembayaran, pihak lawan belum menyatakan apa pun.";
    }

    const days = mismatch.day_difference;

    return days ? `Kedua pihak menyatakan pembayaran, tetapi tanggalnya berbeda ${days} hari.` : "Kedua pihak menyatakan pembayaran, tetapi tanggalnya berbeda.";
}

export default function PaymentMismatchNotice({ mismatch, audience, className }: { mismatch?: PaymentMismatch | null; audience: "party" | "admin"; className?: string }) {
    if (!mismatch) return null;

    const note = audience === "admin" ? "Selisih ini dikirim server sebagai bahan mediasi. Platform tidak memverifikasi terjadinya pembayaran, jadi tandanya menunjukkan pertentangan pernyataan, bukan siapa yang benar." : "Selisih ini ditunjukkan apa adanya. Bila pihak lawan tidak memperbaiki pernyataannya, laporkan sengketa agar admin menengahi.";

    return (
        <div className={cn("flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4", className)} role="status">
            <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden />

            <div className="min-w-0">
                <p className="text-sm font-bold text-amber-800">Pernyataan pembayaran tidak cocok</p>
                <p className="mt-1 text-xs leading-5 text-amber-700">{describe(mismatch)}</p>
                <p className="mt-1.5 text-[11px] leading-4 text-amber-600">{note}</p>
            </div>
        </div>
    );
}
