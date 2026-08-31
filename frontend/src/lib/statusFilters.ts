// Daftar chip filter status diturunkan dari peta label, bukan ditulis ulang sebagai
// literal di tiap halaman. Peta labelnya bertipe Record<Enum, ...> sehingga nilai
// enum baru dari openapi.yaml menggagalkan typecheck di satu tempat, dan chip
// filternya ikut bertambah tanpa disunting. Urutan mengikuti urutan kunci pada
// peta label, jadi urutan tampilan diatur di sana.

export type StatusMeta = { label: string };

export type StatusFilter<T extends string> = { value: T | "all"; label: string };

type Options = { allLabel?: string; allLast?: boolean };

export function statusFilters<T extends string>(meta: Record<T, StatusMeta>, options?: Options): StatusFilter<T>[] {
    const all: StatusFilter<T> = { value: "all", label: options?.allLabel ?? "Semua" };
    const entries = (Object.entries(meta) as [T, StatusMeta][]).map(([value, { label }]) => ({ value, label }));

    return options?.allLast ? [...entries, all] : [all, ...entries];
}

export function isStatusFilter<T extends string>(meta: Record<T, StatusMeta>, value: string | null): value is T | "all" {
    if (value === "all") return true;

    // hasOwnProperty.call, bukan `value in meta`, supaya "toString" dan properti
    // bawaan Object.prototype lain tidak lolos sebagai status yang sah.
    return value != null && Object.prototype.hasOwnProperty.call(meta, value);
}
