import type { AvailabilityPeriod, PeriodUpdateItem } from "@api/listing";
import Loading from "@components/common/Loading";
import { useListingPeriods, useMyListing, useUpdateListingPeriods } from "@hooks/useListing";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { useMemo, useState } from "react";
import { LuArrowLeft, LuCalendarX, LuChevronLeft, LuChevronRight, LuInfo } from "react-icons/lu";
import { Link } from "react-router-dom";

const WEEKS_PER_PAGE = 12;

function formatWeekStart(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function addIsoDays(isoDate: string, days: number): string {
    const [year, month, day] = isoDate.split("-").map(Number);
    const date = new Date(Date.UTC(year, month - 1, day));
    date.setUTCDate(date.getUTCDate() + days);
    return formatWeekStart(date);
}

const weekFormatter = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", timeZone: "Asia/Jakarta" });
const weekYearFormatter = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "Asia/Jakarta" });

function formatWeekRange(weekStartIso: string): string {
    const [year, month, day] = weekStartIso.split("-").map(Number);
    const start = new Date(Date.UTC(year, month - 1, day));
    const end = new Date(Date.UTC(year, month - 1, day + 6));

    return `${weekFormatter.format(start)} - ${weekYearFormatter.format(end)}`;
}

type DraftPeriod = {
    week_start: string;
    capacity: number;
    marked_full: boolean;
};

export default function Calendar() {
    const listingQuery = useMyListing();
    const [page, setPage] = useState(0);
    const today = useMemo(() => new Date(), []);
    const anchorWeekStart = useMemo(() => formatWeekStart(today), [today]);

    const from = useMemo(() => addIsoDays(anchorWeekStart, page * WEEKS_PER_PAGE), [anchorWeekStart, page]);
    const to = useMemo(() => addIsoDays(anchorWeekStart, page * WEEKS_PER_PAGE + WEEKS_PER_PAGE - 1), [anchorWeekStart, page]);

    const periodsQuery = useListingPeriods(listingQuery.data ? from : undefined, listingQuery.data ? to : undefined);
    const updateMutation = useUpdateListingPeriods();

    const [draft, setDraft] = useState<Record<string, DraftPeriod>>({});
    const [errorMessage, setErrorMessage] = useState("");

    if (listingQuery.isLoading) return <Loading />;

    const listing = listingQuery.data ?? null;

    if (!listing) {
        return (
            <div className="space-y-6">
                <h1 className="text-xl font-bold text-slate-900">Kalender Kapasitas</h1>

                <div className="rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuCalendarX className="mx-auto size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Buat listing kapasitas lebih dulu sebelum mengatur kalender.</p>

                    <Link to="/listing" className="mt-4 inline-flex items-center gap-2 rounded-xl bg-industrial-blue-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                        <LuArrowLeft className="size-4" aria-hidden />
                        Buat Listing
                    </Link>
                </div>
            </div>
        );
    }

    const serverPeriods = periodsQuery.data ?? [];
    const serverByWeek = new Map(serverPeriods.map((period) => [period.week_start, period]));

    const weeks: { week_start: string; server?: AvailabilityPeriod; current: DraftPeriod; dirty: boolean }[] = [];

    for (let index = 0; index < WEEKS_PER_PAGE; index++) {
        const weekStart = addIsoDays(anchorWeekStart, page * WEEKS_PER_PAGE + index);
        const server = serverByWeek.get(weekStart);
        const base: DraftPeriod = server ? { week_start: server.week_start, capacity: server.capacity, marked_full: server.marked_full } : { week_start: weekStart, capacity: listing.weekly_capacity, marked_full: false };
        const current = draft[weekStart] ?? base;
        const dirty = Boolean(draft[weekStart]) && (draft[weekStart].capacity !== base.capacity || draft[weekStart].marked_full !== base.marked_full);

        weeks.push({ week_start: weekStart, server, current, dirty });
    }

    const dirtyWeeks = weeks.filter((week) => week.dirty);

    function updateDraft(weekStart: string, patch: Partial<DraftPeriod>) {
        setErrorMessage("");
        setDraft((previous) => {
            const server = serverByWeek.get(weekStart);
            const base: DraftPeriod = server ? { week_start: server.week_start, capacity: server.capacity, marked_full: server.marked_full } : { week_start: weekStart, capacity: listing!.weekly_capacity, marked_full: false };
            const next = { ...(previous[weekStart] ?? base), ...patch };

            const unchanged = next.capacity === base.capacity && next.marked_full === base.marked_full;
            const updated = { ...previous };

            if (unchanged) {
                delete updated[weekStart];
            } else {
                updated[weekStart] = next;
            }

            return updated;
        });
    }

    async function handleSave() {
        setErrorMessage("");

        const periods: PeriodUpdateItem[] = dirtyWeeks.map((week) => ({
            week_start: week.week_start,
            capacity: week.current.capacity,
            marked_full: week.current.marked_full,
        }));

        const invalid = periods.some((period) => {
            const [year, month, day] = period.week_start.split("-").map(Number);
            return new Date(Date.UTC(year, month - 1, day)).getUTCDay() !== 1;
        });

        if (invalid) {
            setErrorMessage("Ada minggu yang awalnya bukan hari Senin. Muat ulang halaman lalu coba lagi.");
            return;
        }

        try {
            await updateMutation.mutateAsync(periods);
            setDraft({});
        } catch (error) {
            setErrorMessage(getProblemMessage(error, "Perubahan tidak dapat disimpan. Silakan coba lagi.", { 401: "Sesi Anda habis, silakan masuk kembali.", 422: "Data periode tidak sah. Awal minggu harus jatuh pada hari Senin." }));
        }
    }

    const isLocked = (week: (typeof weeks)[number]) => Boolean(week.server && week.server.allocated > 0);

    return (
        <div className="space-y-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">Kalender Kapasitas</h1>
                    <p className="mt-1 text-sm text-slate-500">Atur kapasitas dan tandai minggu penuh. Minggu dimulai Senin, zona waktu WIB.</p>
                </div>

                <Link to="/listing" className="inline-flex shrink-0 items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition-colors hover:bg-slate-50">
                    <LuArrowLeft className="size-4" aria-hidden />
                    Kembali ke Listing
                </Link>
            </div>

            <div className="flex items-center justify-between rounded-2xl border border-slate-200 bg-white px-4 py-3">
                <button type="button" onClick={() => setPage((value) => Math.max(0, value - 1))} disabled={page === 0} className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40">
                    <LuChevronLeft className="size-4" aria-hidden />
                    Sebelumnya
                </button>

                <p className="text-sm font-bold text-slate-800">{formatWeekRange(from)}</p>

                <button type="button" onClick={() => setPage((value) => value + 1)} className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-100">
                    Berikutnya
                    <LuChevronRight className="size-4" aria-hidden />
                </button>
            </div>

            {errorMessage ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                    {errorMessage}
                </div>
            ) : null}

            {periodsQuery.isLoading ? (
                <Loading />
            ) : (
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                    {weeks.map((week) => {
                        const locked = isLocked(week);
                        const allocated = week.server?.allocated ?? 0;

                        return (
                            <div
                                key={week.week_start}
                                className={cn(
                                    "rounded-2xl border p-4 transition-colors",
                                    week.current.marked_full ? "border-red-200 bg-red-50" : week.dirty ? "border-industrial-blue-500/40 bg-industrial-blue-500/5" : "border-slate-200 bg-white",
                                )}
                            >
                                <div className="flex items-start justify-between gap-2">
                                    <div>
                                        <p className="text-sm font-bold text-slate-800">{formatWeekRange(week.week_start)}</p>
                                        {allocated > 0 ? <p className="mt-0.5 text-xs text-slate-400">Terpakai {allocated.toLocaleString("id-ID")} unit untuk pesanan</p> : null}
                                    </div>

                                    {week.current.marked_full ? <span className="rounded-full bg-red-500/10 px-2.5 py-1 text-[11px] font-bold text-red-600">Penuh</span> : null}
                                </div>

                                <div className="mt-3 flex items-center gap-2">
                                    <label className="flex-1">
                                        <span className="sr-only">Kapasitas minggu {formatWeekRange(week.week_start)}</span>
                                        <input
                                            type="number"
                                            min={allocated}
                                            value={Number.isNaN(week.current.capacity) ? "" : week.current.capacity}
                                            disabled={locked || updateMutation.isPending}
                                            onChange={(event) => updateDraft(week.week_start, { capacity: Number(event.target.value) })}
                                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-800 outline-none transition focus:border-industrial-blue-500 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500"
                                        />
                                    </label>
                                    <span className="text-xs text-slate-400">unit</span>
                                </div>

                                <button
                                    type="button"
                                    onClick={() => updateDraft(week.week_start, { marked_full: !week.current.marked_full })}
                                    disabled={locked || updateMutation.isPending}
                                    className={cn(
                                        "mt-3 w-full cursor-pointer rounded-lg border px-3 py-2 text-xs font-bold transition disabled:cursor-not-allowed disabled:opacity-50",
                                        week.current.marked_full ? "border-red-300 bg-white text-red-600 hover:bg-red-50" : "border-slate-300 bg-white text-slate-600 hover:border-red-300 hover:text-red-600",
                                    )}
                                >
                                    {week.current.marked_full ? "Buka Kembali Minggu Ini" : "Tandai Penuh"}
                                </button>

                                {locked ? <p className="mt-2 text-[11px] leading-4 text-slate-400">Minggu ini sudah punya alokasi pesanan, jadi tidak dapat diubah atau ditandai penuh.</p> : null}
                            </div>
                        );
                    })}
                </div>
            )}

            {dirtyWeeks.length > 0 ? (
                <div className="sticky bottom-4 flex items-center justify-between gap-3 rounded-2xl border border-industrial-blue-500/30 bg-white p-4 shadow-lg shadow-slate-200">
                    <p className="text-sm font-semibold text-slate-700">
                        {dirtyWeeks.length} minggu berubah
                    </p>

                    <div className="flex gap-2">
                        <button type="button" onClick={() => setDraft({})} disabled={updateMutation.isPending} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                            Batalkan
                        </button>
                        <button type="button" onClick={handleSave} disabled={updateMutation.isPending} className="cursor-pointer rounded-xl bg-industrial-blue-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-70">
                            {updateMutation.isPending ? "Menyimpan..." : "Simpan Perubahan"}
                        </button>
                    </div>
                </div>
            ) : null}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuInfo className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">
                    Minggu yang belum diatur mengikuti kapasitas mingguan listing ({listing.weekly_capacity.toLocaleString("id-ID")} unit). Kapasitas tidak dapat diturunkan di bawah jumlah yang sudah terpakai pesanan berjalan.
                </p>
            </div>
        </div>
    );
}
