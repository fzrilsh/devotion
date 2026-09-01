import { ApiError } from "@api/client";
import type { WorkOrderDetail } from "@api/workOrders";
import Loading from "@components/common/Loading";
import PaymentMismatchNotice from "@components/common/PaymentMismatchNotice";
import { useAuth } from "@hooks/useAuth";
import { useCancelWorkOrder, useChangeWorkOrderStatus, useConfirmWorkOrder, useRecordPayment, useReportDispute, useSubmitReview, useWorkOrder, useWorkOrderContacts } from "@hooks/useWorkOrders";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { useState } from "react";
import { LuArrowLeft, LuArrowRight, LuBanknote, LuCalendarDays, LuCircleAlert, LuMail, LuPackage, LuPhone, LuShieldAlert, LuStar, LuTriangleAlert, LuUser } from "react-icons/lu";
import { Link, useParams } from "react-router-dom";
import { formatDateId, formatDateTimeId, formatRupiah, getWorkOrderSide, paymentDirectionForSide, paymentDirectionLabel, transitionsForSide, workOrderSideMeta, workOrderStatusMeta } from "./meta";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

function ActionPanel({ title, icon: Icon, tone = "slate", error, onClose, children }: { title: string; icon: React.ElementType; tone?: "slate" | "red" | "amber"; error: string; onClose: () => void; children: React.ReactNode }) {
    const tones = {
        slate: "border-slate-200 bg-slate-50",
        red: "border-red-200 bg-red-50",
        amber: "border-amber-200 bg-amber-50",
    }[tone];

    return (
        <div className={cn("rounded-2xl border p-5", tones)}>
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                    <Icon className="size-4.5 text-slate-600" aria-hidden />
                    <h3 className="text-sm font-bold text-slate-800">{title}</h3>
                </div>

                <button type="button" onClick={onClose} className="cursor-pointer text-xs font-semibold text-slate-500 hover:text-slate-700">
                    Tutup
                </button>
            </div>

            {error ? (
                <div className="mt-3 rounded-lg border border-red-200 bg-white px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                    {error}
                </div>
            ) : null}

            <div className="mt-4">{children}</div>
        </div>
    );
}

function waLink(whatsapp: string): string {
    return `https://wa.me/${whatsapp.replace(/\D/g, "")}`;
}

const counterpartyRoleLabel = { buyer: "Pemberi order", subcontractor: "Subkontraktor" } as const;

function CounterpartyContacts({ workOrderId, isParty }: { workOrderId: string; isParty: boolean }) {
    const contactsQuery = useWorkOrderContacts(workOrderId, isParty);

    if (!isParty) return null;

    if (contactsQuery.isLoading) {
        return (
            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Kontak Pihak Lawan</h2>
                <p className="mt-3 text-sm text-slate-500">Memuat kontak...</p>
            </div>
        );
    }

    if (contactsQuery.isError || !contactsQuery.data) {
        return (
            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Kontak Pihak Lawan</h2>
                <p className="mt-3 text-sm text-slate-500">Kontak pihak lawan tidak dapat dimuat. Coba muat ulang halaman.</p>
            </div>
        );
    }

    const { counterparty } = contactsQuery.data;

    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-6">
            <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Kontak Pihak Lawan</h2>
            <p className="mt-2 text-xs leading-5 text-slate-500">Pembayaran dan koordinasi produksi berjalan langsung antar kedua pihak di luar platform. Hubungi lewat kontak di bawah.</p>

            <div className="mt-4 flex items-center gap-3 border-b border-slate-100 pb-4">
                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-deep-navy-500/10 text-deep-navy-600">
                    <LuUser className="size-5" aria-hidden />
                </span>

                <div className="min-w-0">
                    <p className="truncate text-sm font-bold text-slate-800">{counterparty.business_name}</p>
                    <p className="text-xs text-slate-400">{counterpartyRoleLabel[counterparty.role]}</p>
                </div>
            </div>

            <dl className="mt-4 space-y-3">
                <div className="flex items-center justify-between gap-3">
                    <dt className="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-slate-400">
                        <LuMail className="size-4" aria-hidden />
                        Email
                    </dt>

                    <dd className="min-w-0">
                        <a href={`mailto:${counterparty.email}`} className="truncate text-sm font-semibold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            {counterparty.email}
                        </a>
                    </dd>
                </div>

                <div className="flex items-center justify-between gap-3">
                    <dt className="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-slate-400">
                        <LuPhone className="size-4" aria-hidden />
                        WhatsApp
                    </dt>

                    <dd className="min-w-0">
                        <a href={waLink(counterparty.whatsapp)} target="_blank" rel="noreferrer" className="truncate text-sm font-semibold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            {counterparty.whatsapp}
                        </a>
                    </dd>
                </div>
            </dl>
        </div>
    );
}

const statusActions: { status: "production" | "completed" | "shipped"; label: string; icon: React.ElementType }[] = [
    { status: "production", label: "Mulai Produksi", icon: LuPackage },
    { status: "completed", label: "Tandai Selesai", icon: LuCalendarDays },
    { status: "shipped", label: "Tandai Dikirim", icon: LuArrowRight },
];

export default function Detail() {
    const { workOrderId = "" } = useParams();
    const { user } = useAuth();
    const orderQuery = useWorkOrder(workOrderId);

    const changeStatus = useChangeWorkOrderStatus(workOrderId);
    const confirmOrder = useConfirmWorkOrder(workOrderId);
    const cancelOrder = useCancelWorkOrder(workOrderId);
    const recordPayment = useRecordPayment(workOrderId);
    const reportDispute = useReportDispute(workOrderId);
    const submitReview = useSubmitReview(workOrderId);

    const [panel, setPanel] = useState<"cancel" | "payment" | "dispute" | "review" | null>(null);
    const [actionError, setActionError] = useState("");
    const [actionSuccess, setActionSuccess] = useState("");

    const [cancelReason, setCancelReason] = useState("");
    const [paymentDate, setPaymentDate] = useState("");
    const [paymentNote, setPaymentNote] = useState("");
    const [disputeBody, setDisputeBody] = useState("");
    const [rating, setRating] = useState(0);
    const [reviewText, setReviewText] = useState("");

    function resetMessages() {
        setActionError("");
        setActionSuccess("");
    }

    function openPanel(next: typeof panel) {
        resetMessages();
        setPanel((current) => (current === next ? null : next));
    }

    async function runAction(action: () => Promise<unknown>, successMessage: string) {
        resetMessages();

        try {
            await action();
            setPanel(null);
            setActionSuccess(successMessage);
        } catch (error) {
            setActionError(getProblemMessage(error, "Aksi tidak dapat diproses. Silakan coba lagi.", { 401: "Sesi Anda habis, silakan masuk kembali.", 403: "Anda tidak berwenang melakukan aksi ini pada pesanan ini." }));
        }
    }

    if (orderQuery.isLoading) return <Loading />;

    if (orderQuery.isError || !orderQuery.data) {
        const status = orderQuery.error instanceof ApiError ? orderQuery.error.status : 0;
        const message =
            status === 403
                ? "Pesanan ini tidak terdaftar untuk profil usaha Anda. Periksa kembali bahwa id di tautan adalah id pesanan, bukan id request atau id kandidat."
                : status === 404
                  ? "Pesanan dengan id ini tidak ditemukan. Id pesanan diberikan lewat notifikasi kesepakatan dan daftar pesanan; id request kuota dan id kandidat bukan id pesanan."
                  : "Pesanan tidak dapat dimuat. Coba muat ulang halaman.";

        return (
            <div className="space-y-6">
                <h1 className="text-xl font-bold text-slate-900">Detail Pesanan</h1>
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">{message}</p>
                    <Link to="/orders" className="mt-3 inline-flex items-center gap-2 text-sm font-bold text-red-800 underline underline-offset-2">
                        <LuArrowLeft className="size-4" aria-hidden />
                        Kembali ke daftar pesanan
                    </Link>
                </div>
            </div>
        );
    }

    const order: WorkOrderDetail = orderQuery.data;
    const meta = workOrderStatusMeta[order.status];
    const side = getWorkOrderSide(order, user?.profile_id ?? null);
    const transitions = transitionsForSide(order.allowed_transitions ?? [], side);
    const availableStatusActions = statusActions.filter((action) => transitions.includes(action.status));
    const canConfirm = transitions.includes("confirmed");
    const canDispute = transitions.includes("in_mediation");
    const canCancel = order.self_cancellable && side !== null;
    const canRecordPayment = order.can_record_payment && side !== null;
    const paymentDirection = side ? paymentDirectionForSide[side] : "sent";
    const canReview = order.can_review;
    const anyPending = changeStatus.isPending || confirmOrder.isPending || cancelOrder.isPending || recordPayment.isPending || reportDispute.isPending || submitReview.isPending;

    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <Link to="/orders" className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                    <LuArrowLeft className="size-3.5" aria-hidden />
                    Pesanan
                </Link>

                <h1 className="text-xl font-bold text-slate-900">Detail Pesanan</h1>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex items-center gap-4">
                        <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                            <LuPackage className="size-6" aria-hidden />
                        </span>

                        <div>
                            <p className="text-lg font-extrabold text-slate-900">{order.quantity.toLocaleString("id-ID")} unit</p>
                            <p className="mt-0.5 text-xs text-slate-400">Tenggat {formatDateId(order.deadline)}</p>
                        </div>
                    </div>

                    <div className="flex shrink-0 flex-wrap items-center gap-2">
                        {side ? <span className={cn("inline-flex items-center rounded-full px-3.5 py-1.5 text-xs font-bold", workOrderSideMeta[side].className)}>{workOrderSideMeta[side].label}</span> : null}
                        <span className={cn("inline-flex items-center rounded-full px-3.5 py-1.5 text-xs font-bold", meta.className)}>{meta.label}</span>
                    </div>
                </div>

                <div className="mt-6 grid grid-cols-1 gap-4 border-t border-slate-100 pt-6 sm:grid-cols-3">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">Nilai Pesanan</p>
                        <p className="mt-1 text-sm font-bold text-slate-800">{formatRupiah(order.total_price)}</p>
                    </div>

                    <div>
                        <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">Jeda Kesiapan</p>
                        <p className="mt-1 text-sm font-bold text-slate-800">{order.readiness_lead_days != null ? `${order.readiness_lead_days} hari` : "-"}</p>
                    </div>

                    <div>
                        <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">Tenggat Kesiapan</p>
                        <p className="mt-1 text-sm font-bold text-slate-800">{formatDateId(order.readiness_deadline)}</p>
                    </div>
                </div>
            </div>

            {order.status === "shipped" && order.auto_confirm_at ? (
                <div className="flex items-start gap-3 rounded-2xl border border-sky-200 bg-sky-50 p-4">
                    <LuCalendarDays className="mt-0.5 size-5 shrink-0 text-sky-600" aria-hidden />
                    <p className="text-xs leading-5 text-sky-800">Pesanan akan dikonfirmasi diterima secara otomatis pada {formatDateTimeId(order.auto_confirm_at)} bila tidak ada konfirmasi atau sengketa sebelumnya.</p>
                </div>
            ) : null}

            {actionSuccess ? (
                <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700" role="status" aria-live="polite">
                    {actionSuccess}
                </div>
            ) : null}

            <div className="flex flex-wrap gap-2">
                {availableStatusActions.map((action) => (
                    <button
                        key={action.status}
                        type="button"
                        disabled={anyPending}
                        onClick={() => runAction(() => changeStatus.mutateAsync({ newStatus: action.status }), `Status pesanan berubah menjadi ${workOrderStatusMeta[action.status].label}.`)}
                        className="inline-flex cursor-pointer items-center gap-2 rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        <action.icon className="size-4" aria-hidden />
                        {changeStatus.isPending ? "Memproses..." : action.label}
                    </button>
                ))}

                {canConfirm ? (
                    <button
                        type="button"
                        disabled={anyPending}
                        onClick={() => runAction(() => confirmOrder.mutateAsync(), "Penerimaan barang dikonfirmasi.")}
                        className="inline-flex cursor-pointer items-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        <LuCalendarDays className="size-4" aria-hidden />
                        {confirmOrder.isPending ? "Memproses..." : "Konfirmasi Diterima"}
                    </button>
                ) : null}

                {canCancel ? (
                    <button type="button" onClick={() => openPanel("cancel")} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-red-300 bg-white px-4 py-2.5 text-sm font-semibold text-red-600 transition hover:bg-red-50">
                        <LuCircleAlert className="size-4" aria-hidden />
                        Batalkan Pesanan
                    </button>
                ) : null}

                {canRecordPayment ? (
                    <button type="button" onClick={() => openPanel("payment")} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                        <LuBanknote className="size-4" aria-hidden />
                        Catat Pembayaran
                    </button>
                ) : null}

                {canDispute ? (
                    <button type="button" onClick={() => openPanel("dispute")} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-amber-300 bg-white px-4 py-2.5 text-sm font-semibold text-amber-700 transition hover:bg-amber-50">
                        <LuShieldAlert className="size-4" aria-hidden />
                        Laporkan Sengketa
                    </button>
                ) : null}

                {canReview ? (
                    <button type="button" onClick={() => openPanel("review")} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                        <LuStar className="size-4" aria-hidden />
                        Beri Ulasan
                    </button>
                ) : null}
            </div>

            {panel === "cancel" ? (
                <ActionPanel title="Batalkan Pesanan" icon={LuCircleAlert} tone="red" error={actionError} onClose={() => setPanel(null)}>
                    <p className="text-xs leading-5 text-slate-500">Pembatalan membalik seluruh alokasi kapasitas dan membebani tingkat penyelesaian Anda.</p>

                    <label className="mt-3 block">
                        <span className="mb-1.5 block text-xs font-semibold text-slate-500">Alasan pembatalan</span>
                        <textarea rows={3} value={cancelReason} onChange={(event) => setCancelReason(event.target.value)} className={inputClassName} placeholder="Jelaskan mengapa pesanan dibatalkan" />
                    </label>

                    <button
                        type="button"
                        disabled={cancelOrder.isPending || cancelReason.trim().length < 3}
                        onClick={() => runAction(() => cancelOrder.mutateAsync(cancelReason.trim()), "Pesanan dibatalkan dan kapasitas dikembalikan.")}
                        className="mt-3 w-full cursor-pointer rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {cancelOrder.isPending ? "Membatalkan..." : "Ya, Batalkan Pesanan"}
                    </button>
                </ActionPanel>
            ) : null}

            {panel === "payment" ? (
                <ActionPanel title="Catat Pernyataan Pembayaran" icon={LuBanknote} error={actionError} onClose={() => setPanel(null)}>
                    <p className="text-xs leading-5 text-slate-500">Platform tidak menahan atau memproses dana. Yang dicatat hanya pernyataan Anda, tanpa jumlah uang.</p>

                    <div className="mt-3 flex items-center gap-2.5 rounded-xl border border-slate-200 bg-white px-3 py-2.5">
                        <LuBanknote className="size-4 shrink-0 text-slate-500" aria-hidden />
                        <p className="text-xs font-semibold text-slate-700">{paymentDirectionLabel[paymentDirection]}</p>
                    </div>

                    <label className="mt-3 block">
                        <span className="mb-1.5 block text-xs font-semibold text-slate-500">Tanggal pembayaran</span>
                        <input type="date" value={paymentDate} onChange={(event) => setPaymentDate(event.target.value)} className={inputClassName} />
                    </label>

                    <label className="mt-3 block">
                        <span className="mb-1.5 block text-xs font-semibold text-slate-500">Catatan (opsional)</span>
                        <input type="text" value={paymentNote} onChange={(event) => setPaymentNote(event.target.value)} className={inputClassName} placeholder="Misalnya: pelunasan tahap pertama" />
                    </label>

                    <button
                        type="button"
                        disabled={recordPayment.isPending || !paymentDate}
                        onClick={() => runAction(() => recordPayment.mutateAsync({ direction: paymentDirection, date: paymentDate, note: paymentNote || undefined }), "Pernyataan pembayaran tercatat.")}
                        className="mt-3 w-full cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {recordPayment.isPending ? "Mencatat..." : "Simpan Pernyataan"}
                    </button>
                </ActionPanel>
            ) : null}

            {panel === "dispute" ? (
                <ActionPanel title="Laporkan Sengketa" icon={LuShieldAlert} tone="amber" error={actionError} onClose={() => setPanel(null)}>
                    <p className="text-xs leading-5 text-slate-500">Sengketa menghentikan hitungan konfirmasi otomatis dan ditengahi oleh admin.</p>

                    <label className="mt-3 block">
                        <span className="mb-1.5 block text-xs font-semibold text-slate-500">Uraian masalah</span>
                        <textarea rows={4} value={disputeBody} onChange={(event) => setDisputeBody(event.target.value)} className={inputClassName} placeholder="Jelaskan kronologi dan masalah yang terjadi" />
                    </label>

                    <button
                        type="button"
                        disabled={reportDispute.isPending || disputeBody.trim().length < 10}
                        onClick={() => runAction(() => reportDispute.mutateAsync(disputeBody.trim()), "Sengketa dilaporkan. Admin akan meninjau dan menengahi.")}
                        className="mt-3 w-full cursor-pointer rounded-xl bg-amber-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-amber-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {reportDispute.isPending ? "Mengirim..." : "Kirim Laporan Sengketa"}
                    </button>
                </ActionPanel>
            ) : null}

            {panel === "review" ? (
                <ActionPanel title="Beri Ulasan" icon={LuStar} error={actionError} onClose={() => setPanel(null)}>
                    <p className="text-xs leading-5 text-slate-500">Ulasan tidak anonim: nama usaha dan tanggal transaksi Anda akan tampil.</p>

                    <div className="mt-3 flex gap-1.5" role="radiogroup" aria-label="Rating">
                        {[1, 2, 3, 4, 5].map((value) => (
                            <button key={value} type="button" onClick={() => setRating(value)} aria-label={`${value} bintang`} aria-pressed={rating === value} className="cursor-pointer">
                                <LuStar className={cn("size-7 transition", value <= rating ? "fill-amber-400 text-amber-400" : "text-slate-300 hover:text-amber-300")} />
                            </button>
                        ))}
                    </div>

                    <label className="mt-3 block">
                        <span className="mb-1.5 block text-xs font-semibold text-slate-500">Ulasan (opsional)</span>
                        <textarea rows={3} value={reviewText} onChange={(event) => setReviewText(event.target.value)} className={inputClassName} placeholder="Ceritakan pengalaman bekerja sama" />
                    </label>

                    <button
                        type="button"
                        disabled={submitReview.isPending || rating === 0}
                        onClick={() => runAction(() => submitReview.mutateAsync({ rating, text: reviewText || undefined }), "Ulasan terkirim. Terima kasih.")}
                        className="mt-3 w-full cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {submitReview.isPending ? "Mengirim..." : "Kirim Ulasan"}
                    </button>
                </ActionPanel>
            ) : null}

            {actionError && !panel ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                    {actionError}
                </div>
            ) : null}

            <CounterpartyContacts workOrderId={workOrderId} isParty={side !== null} />

            {order.allocations && order.allocations.length > 0 ? (
                <div className="rounded-2xl border border-slate-200 bg-white p-6">
                    <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Alokasi Kapasitas</h2>

                    <ul className="mt-3 divide-y divide-slate-100">
                        {order.allocations.map((period) => (
                            <li key={period.week_start} className="flex items-center justify-between gap-3 py-2.5 first:pt-0 last:pb-0">
                                <span className="text-sm text-slate-600">Minggu {formatDateId(period.week_start)}</span>
                                <span className="text-sm font-bold text-slate-800">{period.allocated.toLocaleString("id-ID")} unit</span>
                            </li>
                        ))}
                    </ul>
                </div>
            ) : null}

            {(order.payments && order.payments.length > 0) || order.payment_mismatch ? (
                <div className="rounded-2xl border border-slate-200 bg-white p-6">
                    <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Pernyataan Pembayaran</h2>

                    <PaymentMismatchNotice mismatch={order.payment_mismatch} audience="party" className="mt-3" />

                    <ul className="mt-3 space-y-2">
                        {order.payments?.map((payment) => (
                            <li key={payment.payment_id} className="flex items-center justify-between gap-3 rounded-xl bg-slate-50 px-4 py-3">
                                <div>
                                    <p className="text-sm font-semibold text-slate-700">{payment.direction === "sent" ? "Pembayaran dikirim" : "Pembayaran diterima"}</p>
                                    {payment.note ? <p className="text-xs text-slate-400">{payment.note}</p> : null}
                                </div>

                                <span className="text-xs font-semibold text-slate-500">{formatDateId(payment.date)}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            ) : null}

            {order.status_history && order.status_history.length > 0 ? (
                <div className="rounded-2xl border border-slate-200 bg-white p-6">
                    <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Riwayat Status</h2>

                    <ol className="mt-4 space-y-0">
                        {order.status_history.map((entry, index) => {
                            const entryMeta = entry.status ? workOrderStatusMeta[entry.status] : null;

                            return (
                                <li key={index} className="relative flex gap-4 pb-5 last:pb-0">
                                    {index < order.status_history!.length - 1 ? <span className="absolute left-1.75 top-5 h-full w-px bg-slate-200" aria-hidden /> : null}

                                    <span className={cn("relative mt-1 size-3.5 shrink-0 rounded-full border-2 border-white ring-2", index === 0 ? "bg-industrial-blue-500 ring-industrial-blue-500/30" : "bg-slate-300 ring-slate-200")} aria-hidden />

                                    <div className="min-w-0 flex-1">
                                        <div className="flex flex-wrap items-center gap-2">
                                            {entryMeta ? <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", entryMeta.className)}>{entryMeta.label}</span> : null}
                                            <span className="text-xs text-slate-400">{formatDateTimeId(entry.at)}</span>
                                        </div>

                                        {entry.note ? <p className="mt-1 text-xs leading-5 text-slate-500">{entry.note}</p> : null}
                                    </div>
                                </li>
                            );
                        })}
                    </ol>
                </div>
            ) : null}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Pembayaran terjadi langsung antar pihak, platform hanya mencatat pernyataan. Bila terjadi masalah setelah produksi dimulai, gunakan laporan sengketa agar ditengahi admin.</p>
            </div>
        </div>
    );
}
