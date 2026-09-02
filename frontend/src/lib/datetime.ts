const jakarta = "Asia/Jakarta";

export function parseApiDate(value?: string | null): Date | null {
    if (!value) return null;

    const raw = String(value).trim();
    if (!raw) return null;

    const normalized = raw.includes("T") ? raw : raw.replace(" ", "T");
    const date = new Date(normalized);

    return Number.isNaN(date.getTime()) ? null : date;
}

function format(value: string | null | undefined, options: Intl.DateTimeFormatOptions): string {
    const date = parseApiDate(value);
    if (!date) return "-";

    return new Intl.DateTimeFormat("id-ID", { ...options, timeZone: jakarta }).format(date);
}

/** 31 Agu 2026 */
export function formatDateId(value?: string | null): string {
    return format(value, { day: "numeric", month: "short", year: "numeric" });
}

/** 31 Agustus 2026 */
export function formatDateLongId(value?: string | null): string {
    return format(value, { day: "numeric", month: "long", year: "numeric" });
}

/** 31 Agu 2026, 17.05 */
export function formatDateTimeId(value?: string | null): string {
    return format(value, { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" });
}

/** 31 Agustus 2026, 17.05 */
export function formatDateTimeLongId(value?: string | null): string {
    return format(value, { day: "numeric", month: "long", year: "numeric", hour: "2-digit", minute: "2-digit" });
}

/** 31 Agu, 17.05, untuk tenggat yang selalu dekat sehingga tahun tidak menambah informasi */
export function formatDayTimeId(value?: string | null): string {
    return format(value, { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}

export function formatRupiah(amount?: number | null): string {
    if (amount == null) return "-";

    return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(amount);
}
