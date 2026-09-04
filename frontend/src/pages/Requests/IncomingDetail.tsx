import type { Offer } from "@api/search";
import Loading from "@components/common/Loading";
import { useAcceptOffer, useCounterOffer, useIncomingCandidate, useRejectCandidate, useReloadIncomingCandidate, useSendOffer, useSessionOffers } from "@hooks/useQuota";
import { canAcceptOffer, canCounterOffer, canSendFirstOffer, isChainFromSessionOnly, isOfferChainMissing, isTerminalCandidate, isWaitingBuyer, latestOffer, resolveOfferChain } from "@lib/offers";
import { getProblem, getProblemMessage } from "@lib/problem";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuArrowLeft, LuCircleCheck, LuCircleX, LuHandshake, LuHourglass, LuRefreshCw, LuSend, LuTriangleAlert, LuUser } from "react-icons/lu";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { candidateStatusMeta, formatDateShort, formatRupiah } from "./meta";
import { formatDayTimeId as formatDateTimeId } from "@lib/datetime";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

type DetailLocationState = { candidateId?: string };

type CapacityInfo = { quantityRequested: number; remainingCapacity: number; untilWeek?: string };

function getCapacityInfo(error: unknown): CapacityInfo | null {
    const problem = getProblem(error);

    if (problem?.code !== "INSUFFICIENT_CAPACITY" || !problem.meta) return null;

    const { quantity_requested, remaining_capacity, until_week } = problem.meta;

    if (typeof quantity_requested !== "number" || typeof remaining_capacity !== "number") return null;

    return { quantityRequested: quantity_requested, remainingCapacity: remaining_capacity, untilWeek: typeof until_week === "string" ? until_week : undefined };
}

function OfferHistory({ offers }: { offers: Offer[] }) {
    if (offers.length === 0) return null;

    return (
        <ol className="mt-3 space-y-2 border-t border-slate-100 pt-3">
            {offers.map((offer) => (
                <li key={offer.offer_id} className="flex items-center justify-between gap-3 text-xs">
                    <span className="flex items-center gap-2 text-slate-500">
                        <span className={cn("rounded-full px-2 py-0.5 font-bold", offer.party === "subcontractor" ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "bg-deep-navy-500/10 text-deep-navy-600")}>
                            Ronde {offer.sequence} · {offer.party === "subcontractor" ? "Anda" : "Pemberi Order"}
                        </span>
                        {offer.note ? <span className="truncate text-slate-400">{offer.note}</span> : null}
                    </span>

                    <span className="shrink-0 font-bold tabular-nums text-slate-700">{formatRupiah(offer.total_price)}</span>
                </li>
            ))}
        </ol>
    );
}

export default function IncomingDetail() {
    const { requestId: paramCandidateId = "" } = useParams();
    const location = useLocation();
    const navigate = useNavigate();
    const stateCandidateId = (location.state as DetailLocationState | null)?.candidateId;
    const candidateId = stateCandidateId ?? paramCandidateId;

    const candidateQuery = useIncomingCandidate(candidateId);
    const reloadCandidate = useReloadIncomingCandidate(candidateId);
    const sessionOffers = useSessionOffers(candidateId);

    const sendOfferMutation = useSendOffer(candidateId);
    const rejectMutation = useRejectCandidate(candidateId);
    const counterMutation = useCounterOffer(candidateId);
    const acceptMutation = useAcceptOffer();

    const [panel, setPanel] = useState<"none" | "offer" | "counter" | "reject">("none");
    const [totalPrice, setTotalPrice] = useState("");
    const [leadDays, setLeadDays] = useState("");
    const [offerNote, setOfferNote] = useState("");
    const [counterPrice, setCounterPrice] = useState("");
    const [counterNote, setCounterNote] = useState("");
    const [rejectReason, setRejectReason] = useState("");
    const [error, setError] = useState("");
    const [capacityInfo, setCapacityInfo] = useState<CapacityInfo | null>(null);
    const [reloading, setReloading] = useState(false);

    // The lists are what actually go to the server, so the spinner has to cover
    // their refetch too, not only the cache-only detail entry.
    const refreshing = reloading || candidateQuery.isFetching;

    async function handleReload() {
        setReloading(true);

        try {
            await reloadCandidate();
        } finally {
            setReloading(false);
        }
    }

    if (candidateQuery.isLoading) return <Loading />;

    if (candidateQuery.isError || !candidateQuery.data) {
        return (
            <div className="space-y-6">
                <h1 className="text-xl font-bold text-slate-900">Detail Request Masuk</h1>
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Kandidat tidak ditemukan atau sudah tidak tersedia untuk listing Anda.</p>
                    <Link to="/requests/incoming" className="mt-3 inline-flex items-center gap-2 text-sm font-bold text-red-800 underline underline-offset-2">
                        <LuArrowLeft className="size-4" aria-hidden />
                        Kembali ke request masuk
                    </Link>
                </div>
            </div>
        );
    }

    const candidate = candidateQuery.data;

    const offers = resolveOfferChain(candidate, sessionOffers);
    const latest = latestOffer(offers);

    const canOffer = canSendFirstOffer(candidate.status, offers);
    const canCounter = canCounterOffer(candidate.status, latest);
    const canAccept = canAcceptOffer(candidate.status, latest, "subcontractor");
    const waitingBuyer = isWaitingBuyer(candidate.status, latest);
    const terminal = isTerminalCandidate(candidate.status);
    const offerChainMissing = isOfferChainMissing(candidate.status, offers);
    const chainFromSessionOnly = isChainFromSessionOnly(candidate, offers);
    const busy = sendOfferMutation.isPending || rejectMutation.isPending || counterMutation.isPending || acceptMutation.isPending;

    async function handleSendOffer(event: React.FormEvent) {
        event.preventDefault();
        setError("");
        setCapacityInfo(null);

        const price = Number(totalPrice);
        const lead = Number(leadDays);

        if (!Number.isInteger(price) || price < 1) {
            setError("Masukkan total harga dalam rupiah bulat.");
            return;
        }

        if (!Number.isInteger(lead) || lead < 0 || lead > 365) {
            setError("Jeda kesiapan harus bilangan bulat 0 sampai 365 hari.");
            return;
        }

        try {
            await sendOfferMutation.mutateAsync({ total_price: price, readiness_lead_days: lead, note: offerNote.trim() || undefined });
            setPanel("none");
            setTotalPrice("");
            setLeadDays("");
            setOfferNote("");
        } catch (err) {
            setCapacityInfo(getCapacityInfo(err));
            setError(getProblemMessage(err));
        }
    }

    async function handleCounter(event: React.FormEvent) {
        event.preventDefault();
        setError("");

        if (!latest?.offer_id) return;

        const price = Number(counterPrice);
        if (!Number.isInteger(price) || price < 1) {
            setError("Masukkan harga counter dalam rupiah bulat.");
            return;
        }

        try {
            await counterMutation.mutateAsync({ offerId: latest.offer_id, data: { total_price: price, note: counterNote.trim() || undefined } });
            setPanel("none");
            setCounterPrice("");
            setCounterNote("");
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    async function handleReject(event: React.FormEvent) {
        event.preventDefault();
        setError("");

        const reason = rejectReason.trim();
        if (reason.length < 5) {
            setError("Tuliskan alasan penolakan, minimal 5 karakter.");
            return;
        }

        try {
            await rejectMutation.mutateAsync(reason);
            setPanel("none");
            setRejectReason("");
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    async function handleAccept() {
        if (!latest?.offer_id) return;
        setError("");

        try {
            const workOrder = await acceptMutation.mutateAsync(latest.offer_id);
            navigate(`/orders/${workOrder.work_order_id}`);
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <Link to="/requests/incoming" className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                    <LuArrowLeft className="size-3.5" aria-hidden />
                    Request Masuk
                </Link>

                <h1 className="text-xl font-bold text-slate-900">Detail Request Masuk</h1>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <div className="flex items-start gap-4">
                    <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-deep-navy-500/10 text-deep-navy-600">
                        <LuSend className="size-6" aria-hidden />
                    </span>

                    <div className="min-w-0 flex-1">
                        <p className="flex items-center gap-2 text-lg font-extrabold text-slate-900">
                            <LuUser className="size-5 shrink-0 text-slate-400" aria-hidden />
                            {candidate.business_name || "Pemberi order"}
                        </p>
                        <p className="mt-0.5 text-xs text-slate-400">Request kuota untuk listing Anda. Batas balasan tiap request 72 jam sejak dikirim.</p>
                    </div>

                    <span className={cn("shrink-0 rounded-full px-3 py-1 text-[11px] font-bold", candidateStatusMeta[candidate.status].className)}>{candidateStatusMeta[candidate.status].label}</span>
                </div>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Detail Permintaan</h2>

                <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Jumlah &amp; Bahan</p>
                        <p className="mt-1 text-lg font-extrabold text-slate-900">{candidate.quantity.toLocaleString("id-ID")} unit {candidate.material}</p>
                    </div>

                    <div>
                        <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Tenggat Pesanan</p>
                        <p className="mt-1 text-lg font-extrabold text-slate-900">{formatDateShort(candidate.deadline)}</p>
                    </div>
                </div>

                {candidate.note ? <p className="mt-4 border-t border-slate-100 pt-4 text-sm leading-6 text-slate-600">{candidate.note}</p> : null}

                <div className={cn("mt-4 flex items-start gap-3 rounded-xl border p-4", candidate.can_fulfill ? "border-emerald-200 bg-emerald-50" : "border-amber-200 bg-amber-50")}>
                    {candidate.can_fulfill ? (
                        <LuCircleCheck className="mt-0.5 size-5 shrink-0 text-emerald-600" aria-hidden />
                    ) : (
                        <LuCircleX className="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden />
                    )}
                    <p className={cn("text-sm leading-6", candidate.can_fulfill ? "text-emerald-800" : "text-amber-800")}>
                        {candidate.can_fulfill
                            ? <>Kapasitas Anda mencukupi: {candidate.capacity_in_range.toLocaleString("id-ID")} potong tersisa dari minggu kesiapan sampai tenggat, untuk {candidate.quantity.toLocaleString("id-ID")} potong yang diminta.</>
                            : <>Kapasitas Anda kurang: {candidate.capacity_in_range.toLocaleString("id-ID")} potong tersisa dari minggu kesiapan sampai tenggat, sementara diminta {candidate.quantity.toLocaleString("id-ID")} potong. Naikkan kapasitas mingguan di halaman Listing atau tolak request ini.</>}
                    </p>
                </div>

                {candidate.rejection_reason ? (
                    <div className="mt-4 rounded-xl border border-red-200 bg-red-50 p-4">
                        <p className="text-xs font-semibold uppercase tracking-wide text-red-500">Alasan Penolakan</p>
                        <p className="mt-1 text-sm leading-6 text-red-700">{candidate.rejection_reason}</p>
                    </div>
                ) : null}
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <div className="flex flex-wrap items-start justify-between gap-3">
                    <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Negosiasi</h2>

                    {latest ? (
                        <div className="text-right">
                            <p className="text-lg font-extrabold tabular-nums text-slate-900">{formatRupiah(latest.total_price)}</p>
                            <p className="text-xs text-slate-400">
                                ronde {latest.sequence} oleh {latest.party === "subcontractor" ? "Anda" : "pemberi order"} · kesiapan {latest.readiness_lead_days} hari · {formatDateTimeId(latest.created_at)}
                            </p>
                        </div>
                    ) : null}
                </div>

                <OfferHistory offers={offers} />

                {error ? (
                    <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                        <p>{error}</p>

                        {capacityInfo ? (
                            <dl className="mt-2 space-y-1 border-t border-red-200 pt-2">
                                <div className="flex justify-between gap-3">
                                    <dt className="text-red-600">Jumlah diminta</dt>
                                    <dd className="font-bold tabular-nums">{capacityInfo.quantityRequested.toLocaleString("id-ID")} potong</dd>
                                </div>
                                <div className="flex justify-between gap-3">
                                    <dt className="text-red-600">Kapasitas tersisa Anda</dt>
                                    <dd className="font-bold tabular-nums">{capacityInfo.remainingCapacity.toLocaleString("id-ID")} potong</dd>
                                </div>
                                {capacityInfo.untilWeek ? (
                                    <div className="flex justify-between gap-3">
                                        <dt className="text-red-600">Dihitung sampai minggu</dt>
                                        <dd className="font-bold">{formatDateShort(capacityInfo.untilWeek)}</dd>
                                    </div>
                                ) : null}
                                <p className="pt-1 text-[11px] leading-4 text-red-600">Naikkan kapasitas mingguan di halaman Listing, atau tolak request ini bila deadline-nya memang tidak terjangkau.</p>
                            </dl>
                        ) : null}
                    </div>
                ) : null}

                {waitingBuyer ? (
                    <div className="mt-4 border-t border-slate-100 pt-4">
                        <p className="flex items-center gap-2 text-sm text-slate-500">
                            <LuHourglass className="size-4 shrink-0 text-amber-500" aria-hidden />
                            Penawaran Anda sudah terkirim. Menunggu keputusan pemberi order.
                        </p>

                        {chainFromSessionOnly ? (
                            <>
                                <p className="mt-2 text-xs leading-5 text-slate-400">
                                    Ronde di atas berasal dari balasan server atas penawaran yang baru Anda kirim, bukan dari daftar request. Bila pemberi order sudah meng-counter, balasannya belum tentu terlihat di sini: muat ulang data untuk memeriksa.
                                </p>

                                <button
                                    type="button"
                                    onClick={handleReload}
                                    disabled={refreshing}
                                    className="mt-3 inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
                                >
                                    <LuRefreshCw className={cn("size-4", refreshing && "animate-spin")} aria-hidden />
                                    {refreshing ? "Memuat..." : "Muat ulang data"}
                                </button>
                            </>
                        ) : null}
                    </div>
                ) : null}

                {candidate.status === "agreed" ? (
                    <div className="mt-4 border-t border-emerald-100 pt-4">
                        <p className="flex items-center gap-2 text-sm font-semibold text-emerald-700">
                            <LuCircleCheck className="size-4" aria-hidden />
                            Kesepakatan terbentuk. Pesanan sudah dibuat dan kapasitas Anda teralokasi.
                        </p>
                        <Link to="/orders" className="mt-2 inline-flex items-center gap-1.5 text-xs font-bold text-emerald-800 underline underline-offset-2">
                            Buka daftar pesanan
                        </Link>
                    </div>
                ) : null}

                {terminal && candidate.status !== "agreed" ? (
                    <p className="mt-4 flex items-center gap-2 border-t border-slate-100 pt-4 text-sm text-slate-500">
                        <LuTriangleAlert className="size-4 text-slate-400" aria-hidden />
                        Request ini sudah berakhir dengan status {candidateStatusMeta[candidate.status].label.toLowerCase()}.
                    </p>
                ) : null}

                {offerChainMissing ? (
                    <div className="mt-4 border-t border-amber-100 pt-4">
                        <p className="flex items-start gap-2 text-sm text-slate-600">
                            <LuTriangleAlert className="mt-0.5 size-4 shrink-0 text-amber-500" aria-hidden />
                            <span>
                                Riwayat penawaran request ini tidak ikut terkirim oleh server, jadi tombol counter belum bisa ditampilkan: meng-counter membutuhkan nomor penawaran yang asli, dan itu hanya bisa datang dari server. Negosiasi masih berjalan dan belum ada kesepakatan, jadi request ini belum berakhir. Coba muat ulang data; bila tetap kosong, laporkan ke admin dengan menyebut status <strong>{candidateStatusMeta[candidate.status].label.toLowerCase()}</strong>. Anda tetap bisa menolak request ini.
                            </span>
                        </p>

                        <button
                            type="button"
                            onClick={handleReload}
                            disabled={refreshing}
                            className="mt-3 inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                            <LuRefreshCw className={cn("size-4", refreshing && "animate-spin")} aria-hidden />
                            {refreshing ? "Memuat..." : "Muat ulang data"}
                        </button>
                    </div>
                ) : null}

                {canOffer || canCounter || canAccept || offerChainMissing ? (
                    <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                        {panel === "offer" ? (
                            <form onSubmit={handleSendOffer} className="space-y-3">
                                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                                    <div>
                                        <label htmlFor="total_price" className={labelClassName}>
                                            Total Harga (rupiah) <span className="text-red-500">*</span>
                                        </label>
                                        <input id="total_price" type="number" min={1} value={totalPrice} onChange={(event) => setTotalPrice(event.target.value)} className={inputClassName} placeholder="Misalnya 15000000" />
                                    </div>

                                    <div>
                                        <label htmlFor="readiness_lead_days" className={labelClassName}>
                                            Jeda Kesiapan (hari) <span className="text-red-500">*</span>
                                        </label>
                                        <input id="readiness_lead_days" type="number" min={0} max={365} value={leadDays} onChange={(event) => setLeadDays(event.target.value)} className={inputClassName} placeholder="Misalnya 7" />
                                    </div>
                                </div>

                                <div>
                                    <label htmlFor="offer_note" className={labelClassName}>
                                        Catatan (opsional)
                                    </label>
                                    <input id="offer_note" type="text" value={offerNote} onChange={(event) => setOfferNote(event.target.value)} className={inputClassName} placeholder="Syarat atau detail tambahan" maxLength={500} />
                                </div>

                                <div className="flex gap-2">
                                    <button type="submit" disabled={busy} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60">
                                        {sendOfferMutation.isPending ? "Mengirim..." : "Kirim Penawaran"}
                                    </button>
                                    <button type="button" onClick={() => setPanel("none")} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                        Batal
                                    </button>
                                </div>

                                <p className="text-[11px] leading-4 text-slate-400">Penawaran ditolak sistem bila jumlah request melampaui total kapasitas Anda sampai deadline.</p>
                            </form>
                        ) : panel === "counter" ? (
                            <form onSubmit={handleCounter} className="space-y-3">
                                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                                    <div>
                                        <label htmlFor="counter_price" className={labelClassName}>
                                            Harga Counter (rupiah) <span className="text-red-500">*</span>
                                        </label>
                                        <input id="counter_price" type="number" min={1} value={counterPrice} onChange={(event) => setCounterPrice(event.target.value)} className={inputClassName} placeholder="Misalnya 14500000" />
                                    </div>

                                    <div>
                                        <label htmlFor="counter_note" className={labelClassName}>
                                            Catatan (opsional)
                                        </label>
                                        <input id="counter_note" type="text" value={counterNote} onChange={(event) => setCounterNote(event.target.value)} className={inputClassName} placeholder="Alasan penyesuaian harga" maxLength={500} />
                                    </div>
                                </div>

                                <div className="flex gap-2">
                                    <button type="submit" disabled={busy} className="flex-1 cursor-pointer rounded-xl bg-deep-navy-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-deep-navy-600 disabled:cursor-not-allowed disabled:opacity-60">
                                        {counterMutation.isPending ? "Mengirim..." : "Kirim Counter"}
                                    </button>
                                    <button type="button" onClick={() => setPanel("none")} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                        Batal
                                    </button>
                                </div>
                            </form>
                        ) : panel === "reject" ? (
                            <form onSubmit={handleReject} className="space-y-3">
                                <div>
                                    <label htmlFor="reject_reason" className={labelClassName}>
                                        Alasan Penolakan <span className="text-red-500">*</span>
                                    </label>
                                    <textarea id="reject_reason" rows={3} value={rejectReason} onChange={(event) => setRejectReason(event.target.value)} className={inputClassName} placeholder="Misalnya: kapasitas sudah penuh sampai tanggal deadline" minLength={5} maxLength={500} />
                                </div>

                                <div className="flex gap-2">
                                    <button type="submit" disabled={busy} className="flex-1 cursor-pointer rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60">
                                        {rejectMutation.isPending ? "Mengirim..." : "Tolak Request"}
                                    </button>
                                    <button type="button" onClick={() => setPanel("none")} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                        Batal
                                    </button>
                                </div>
                            </form>
                        ) : (
                            <div className="flex flex-wrap gap-2">
                                {canAccept ? (
                                    <button type="button" onClick={handleAccept} disabled={busy} className="flex-1 cursor-pointer rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60">
                                        <span className="inline-flex items-center justify-center gap-2">
                                            <LuHandshake className="size-4" aria-hidden />
                                            {acceptMutation.isPending ? "Memproses..." : "Terima Penawaran"}
                                        </span>
                                    </button>
                                ) : null}

                                {canCounter ? (
                                    <button type="button" onClick={() => setPanel("counter")} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                                        Counter Penawaran
                                    </button>
                                ) : null}

                                {!canAccept && !canCounter && canOffer ? (
                                    <button type="button" onClick={() => setPanel("offer")} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                                        Buat Penawaran
                                    </button>
                                ) : null}

                                <button type="button" onClick={() => setPanel("reject")} className={cn("cursor-pointer rounded-xl border border-red-300 bg-white px-4 py-2.5 text-sm font-semibold text-red-600 transition hover:bg-red-50", (offerChainMissing || canAccept || canCounter) && "flex-1")}>
                                    Tolak
                                </button>
                            </div>
                        )}
                    </div>
                ) : null}
            </div>
        </div>
    );
}
