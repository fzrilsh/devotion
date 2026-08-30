import type { Notification } from "@api/notifications";
import Loading from "@components/common/Loading";
import { useMarkNotificationRead, useNotifications } from "@hooks/useNotifications";
import { useAuth } from "@hooks/useAuth";
import { getNotificationLink } from "@lib/notificationLinks";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuBellOff, LuCalendarClock, LuCalendarX, LuCheck, LuCircleAlert, LuClipboardList, LuCreditCard, LuFileInput, LuFileOutput, LuFileText, LuInbox, LuLoaderCircle, LuSettings2, LuShieldCheck, LuStar } from "react-icons/lu";
import { Link } from "react-router-dom";

const eventMeta: Record<Notification["event"], { label: string; icon: React.ElementType; className: string }> = {
    request_received: { label: "Request masuk", icon: LuFileInput, className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    offer_received: { label: "Penawaran", icon: LuFileOutput, className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    counter_offer: { label: "Counter-offer", icon: LuFileOutput, className: "bg-violet-500/10 text-violet-600" },
    agreement_formed: { label: "Kesepakatan", icon: LuClipboardList, className: "bg-emerald-500/10 text-emerald-600" },
    order_status_changed: { label: "Status pesanan", icon: LuClipboardList, className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    payment_record: { label: "Pembayaran", icon: LuCreditCard, className: "bg-emerald-500/10 text-emerald-600" },
    deadline_approaching: { label: "Tenggat mendekat", icon: LuCalendarClock, className: "bg-amber-500/10 text-amber-600" },
    deadline_passed: { label: "Tenggat lewat", icon: LuCalendarX, className: "bg-red-500/10 text-red-600" },
    verification_decision: { label: "Verifikasi", icon: LuShieldCheck, className: "bg-emerald-500/10 text-emerald-600" },
    rating_request: { label: "Permintaan ulasan", icon: LuStar, className: "bg-amber-500/10 text-amber-600" },
    order_cancelled: { label: "Pesanan dibatalkan", icon: LuCircleAlert, className: "bg-red-500/10 text-red-600" },
    confirmation_due_approaching: { label: "Konfirmasi otomatis", icon: LuCalendarClock, className: "bg-amber-500/10 text-amber-600" },
    order_auto_closed: { label: "Pesanan ditutup otomatis", icon: LuCheck, className: "bg-slate-500/10 text-slate-600" },
    item_proposal_decision: { label: "Usulan item", icon: LuFileText, className: "bg-industrial-blue-500/10 text-industrial-blue-600" },
    calendar_stale: { label: "Kalender kedaluwarsa", icon: LuCalendarX, className: "bg-amber-500/10 text-amber-600" },
    request_expired: { label: "Request kedaluwarsa", icon: LuCalendarX, className: "bg-amber-500/10 text-amber-600" },
};

function formatRelativeTime(isoDate: string): string {
    const date = new Date(isoDate);
    const diffMs = Date.now() - date.getTime();
    const diffMinutes = Math.floor(diffMs / 60000);

    if (diffMinutes < 1) return "Baru saja";
    if (diffMinutes < 60) return `${diffMinutes} menit lalu`;

    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) return `${diffHours} jam lalu`;

    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays} hari lalu`;

    return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "long", year: "numeric", timeZone: "Asia/Jakarta" }).format(date);
}

function NotificationItem({ notification, isBuyer }: { notification: Notification; isBuyer: boolean }) {
    const markRead = useMarkNotificationRead();
    const meta = eventMeta[notification.event] ?? { label: "Notifikasi", icon: LuInbox, className: "bg-slate-500/10 text-slate-600" };
    const Icon = meta.icon;
    const link = getNotificationLink(notification, isBuyer);

    return (
        <li className={cn("flex gap-4 rounded-2xl border p-4 transition-colors", notification.read ? "border-slate-200 bg-white" : "border-industrial-blue-500/20 bg-industrial-blue-500/5")}>
            <span className={cn("grid size-11 shrink-0 place-items-center rounded-xl", meta.className)}>
                <Icon className="size-5" aria-hidden />
            </span>

            <div className="min-w-0 flex-1">
                <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                        <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">{meta.label}</p>
                        <h3 className={cn("mt-0.5 truncate text-sm", notification.read ? "font-semibold text-slate-700" : "font-bold text-slate-900")}>{notification.title || meta.label}</h3>
                    </div>

                    <span className="shrink-0 text-xs text-slate-400">{formatRelativeTime(notification.created_at)}</span>
                </div>

                {notification.body ? <p className="mt-1 text-sm leading-6 text-slate-500">{notification.body}</p> : null}

                <div className="mt-3 flex items-center gap-3">
                    {link ? (
                        <Link to={link.to} className="text-xs font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            {link.label}
                        </Link>
                    ) : null}

                    {!notification.read ? (
                        <button
                            type="button"
                            onClick={() => markRead.mutate(notification.notification_id)}
                            disabled={markRead.isPending}
                            className="inline-flex cursor-pointer items-center gap-1.5 text-xs font-bold text-slate-500 transition-colors hover:text-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                            <LuCheck className="size-3.5" aria-hidden />
                            Tandai dibaca
                        </button>
                    ) : null}
                </div>
            </div>

            {!notification.read ? <span className="mt-1.5 size-2 shrink-0 rounded-full bg-industrial-blue-500" aria-label="Belum dibaca" /> : null}
        </li>
    );
}

export default function Notifications() {
    const [unreadOnly, setUnreadOnly] = useState(false);
    const notificationsQuery = useNotifications({ unread: unreadOnly });
    const { user } = useAuth();
    const isBuyer = Boolean(user?.roles?.buyer);

    const items = notificationsQuery.data?.pages.flatMap((page) => page.items) ?? [];
    const unreadCount = notificationsQuery.data?.pages[0]?.unread_count ?? 0;

    return (
        <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">Notifikasi</h1>
                    {unreadCount > 0 ? <p className="mt-1 text-sm text-slate-500">{unreadCount} notifikasi belum dibaca</p> : null}
                </div>

                <div className="flex items-center gap-2">
                    <div className="flex rounded-xl border border-slate-200 bg-white p-1">
                        <button type="button" onClick={() => setUnreadOnly(false)} className={cn("cursor-pointer rounded-lg px-3.5 py-1.5 text-sm font-semibold transition-colors", !unreadOnly ? "bg-industrial-blue-500 text-white" : "text-slate-500 hover:text-slate-700")}>
                            Semua
                        </button>

                        <button type="button" onClick={() => setUnreadOnly(true)} className={cn("cursor-pointer rounded-lg px-3.5 py-1.5 text-sm font-semibold transition-colors", unreadOnly ? "bg-industrial-blue-500 text-white" : "text-slate-500 hover:text-slate-700")}>
                            Belum dibaca
                        </button>
                    </div>

                    <Link to="/notifications/preferences" className="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition-colors hover:bg-slate-50">
                        <LuSettings2 className="size-4" aria-hidden />
                        Preferensi
                    </Link>
                </div>
            </div>

            {notificationsQuery.isLoading ? (
                <Loading />
            ) : notificationsQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Notifikasi tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : items.length === 0 ? (
                <div className="flex flex-col items-center rounded-2xl border border-dashed border-slate-300 bg-white p-12 text-center">
                    <span className="grid size-14 place-items-center rounded-2xl bg-slate-100 text-slate-400">
                        <LuBellOff className="size-7" aria-hidden />
                    </span>

                    <p className="mt-4 text-sm font-semibold text-slate-700">{unreadOnly ? "Tidak ada notifikasi yang belum dibaca." : "Belum ada notifikasi."}</p>
                    <p className="mt-1 max-w-sm text-sm text-slate-500">Pemberitahuan pesanan, penawaran, dan verifikasi akan muncul di sini.</p>
                </div>
            ) : (
                <>
                    <ul className="space-y-3">
                        {items.map((notification) => (
                            <NotificationItem key={notification.notification_id} notification={notification} isBuyer={isBuyer} />
                        ))}
                    </ul>

                    {notificationsQuery.hasNextPage ? (
                        <div className="flex justify-center">
                            <button
                                type="button"
                                onClick={() => notificationsQuery.fetchNextPage()}
                                disabled={notificationsQuery.isFetchingNextPage}
                                className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-5 py-2.5 text-sm font-semibold text-slate-600 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
                            >
                                {notificationsQuery.isFetchingNextPage ? <LuLoaderCircle className="size-4 animate-spin" aria-hidden /> : null}
                                {notificationsQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                            </button>
                        </div>
                    ) : null}
                </>
            )}
        </div>
    );
}
