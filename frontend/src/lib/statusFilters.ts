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

    return value != null && Object.prototype.hasOwnProperty.call(meta, value);
}
