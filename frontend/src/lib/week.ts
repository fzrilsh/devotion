/**
 * Week arithmetic for the capacity calendar. Weeks start on Monday and every
 * boundary is computed in WIB (Asia/Jakarta, UTC+7 with no DST), mirroring
 * `platform.WeekStart` in the backend so both sides agree on which Monday a
 * given instant belongs to.
 *
 * Week starts travel as plain `YYYY-MM-DD` strings, the same shape the contract
 * uses for `week_start` (`format: date`), so nothing here ever holds a
 * timestamp for a value that is really a calendar date.
 */

const WIB_OFFSET_MS = 7 * 60 * 60 * 1000;

function toIsoDate(utcDate: Date): string {
    const year = String(utcDate.getUTCFullYear()).padStart(4, "0");
    const month = String(utcDate.getUTCMonth() + 1).padStart(2, "0");
    const day = String(utcDate.getUTCDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

/** Reads an ISO date as a UTC midnight instant, so day arithmetic never crosses a DST seam. */
function fromIsoDate(isoDate: string): Date {
    const [year, month, day] = isoDate.split("-").map(Number);

    return new Date(Date.UTC(year, month - 1, day));
}

/** The calendar date in WIB for an instant, regardless of the browser's own zone. */
export function jakartaIsoDate(instant: Date): string {
    return toIsoDate(new Date(instant.getTime() + WIB_OFFSET_MS));
}

export function addDays(isoDate: string, days: number): string {
    const date = fromIsoDate(isoDate);
    date.setUTCDate(date.getUTCDate() + days);

    return toIsoDate(date);
}

export function addWeeks(weekStart: string, weeks: number): string {
    return addDays(weekStart, weeks * 7);
}

/** Snaps an ISO date back to the Monday of its own week. Mondays are returned unchanged. */
export function weekStartOf(isoDate: string): string {
    const offset = (fromIsoDate(isoDate).getUTCDay() + 6) % 7;

    return addDays(isoDate, -offset);
}

/** The Monday of the week the given instant falls in, read in WIB. */
export function currentWeekStart(now: Date): string {
    return weekStartOf(jakartaIsoDate(now));
}

export function isMonday(isoDate: string): boolean {
    return fromIsoDate(isoDate).getUTCDay() === 1;
}

/** Whole weeks from one Monday to another. Negative when `to` is earlier. */
export function weeksBetween(from: string, to: string): number {
    const days = (fromIsoDate(to).getTime() - fromIsoDate(from).getTime()) / 86_400_000;

    return Math.round(days / 7);
}
