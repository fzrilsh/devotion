import { ApiError } from "@api/client";
import type { Offer } from "@api/search";
import Loading from "@components/common/Loading";
import { useCounterOffer, useIncomingCandidate, useRejectCandidate, useSendOffer } from "@hooks/useQuota";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuArrowLeft, LuCircleCheck, LuHourglass, LuSend, LuTriangleAlert, LuUser } from "react-icons/lu";
import { Link, useLocation, useParams } from "react-router-dom";
import { candidateStatusMeta, formatDateShort, formatRupiah } from "./meta";
import { formatDayTimeId as formatDateTimeId } from "@lib/datetime";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

type DetailLocationState = { candidateId?: string };

type CapacityInfo = { quantityRequested: number; remainingCapacity: number; untilWeek?: string };

function getProblem(error: unknown): { code?: string; detail: string; meta?: Record<string, unknown> } | null {
    if (!(error instanceof ApiError)) return null;

    if (typeof error.data === "object" && error.data !== null && "detail" in error.data && typeof error.data.detail === "string") {
        const data = error.data as { code?: string; detail: string; meta?: Record<string, unknown> };
        return { code: data.code, detail: data.detail, meta: data.meta };
    }

    if (error.status === 401) return { detail: "Sesi Anda habis, silakan masuk kembali." };
    if (error.status === 403) return { detail: "Anda tidak berwenang melakukan aksi ini." };
    if (error.status === 410) return { detail: "Batas waktu balasan request ini sudah terlampaui." };

    return null;
}

function getProblemMessage(error: unknown): string {
    return getProblem(error)?.detail ?? "Aksi tidak dapat diproses. Silakan coba lagi.";
}

// Penolakan kapasitas membawa meta terstruktur (FR-035) supaya pengguna bisa
// melihat angka pastinya, bukan hanya kalimat penjelasan.
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
    // Param rute ini adalah candidate_id, bukan request_id. Detail request utuh
    // hanya bisa dibaca pemberi order; subkontraktor membaca kandidatnya sendiri.
    const { requestId: paramCandidateId = "" } = useParams();
    const location = useLocation();
    const stateCandidateId = (location.state as DetailLocationState | null)?.candidateId;
    const candidateId = stateCandidateId ?? paramCandidateId;

    const candidateQuery = useIncomingCandidate(candidateId);

    const sendOfferMutation = useSendOffer(candidateId);
    const rejectMutation = useRejectCandidate(candidateId);
    const counterMutation = useCounterOffer();

    const [panel, setPanel] = useState<"none" | "offer" | "counter" | "reject">("none");
    const [totalPrice, setTotalPrice] = useState("");
    const [leadDays, setLeadDays] = useState("");
    const [offerNote, setOfferNote] = useState("");
    const [counterPrice, setCounterPrice] = useState("");
    const [counterNote, setCounterNote] = useState("");
    const [rejectReason, setRejectReason] = useState("");
    const [error, setError] = useState("");
    const [capacityInfo, setCapacityInfo] = useState<CapacityInfo | null>(null);

    if (candidateQuery.isLoading) return <Loading />;

    if (candidateQuery.isError || !candidateQuery.data) {
        return (
            <div className="space-y-6">
                <h1 className="text-xl font-bold text-slate-900">Detail Request Masuk</h1>
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Request tidak ditemukan atau sudah tidak tersedia untuk listing Anda.</p>
                    <Link to="/requests/incoming" className="mt-3 inline-flex items-center gap-2 text-sm font-bold text-red-800 underline underline-offset-2">
                        <LuArrowLeft className="size-4" aria-hidden />
                        Kembali ke request masuk
                    </Link>
                </div>
            </div>
        );
    }

    const candidate = candidateQuery.data;
    const latest = candidate.latest_offer;
    const offers = candidate.offers ?? [];

    const canOffer = candidate.status === "awaiting_reply";
    const canCounter = candidate.status === "offered" && latest?.party === "buyer";
    const waitingBuyer = candidate.status === "offered" && latest?.party === "subcontractor";
    const terminal = ["rejected", "expired", "not_continued", "agreed"].includes(candidate.status);
    const busy = sendOfferMutation.isPending || rejectMutation.isPending || counterMutation.isPending;

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

        if (!latest) return;

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
                    <p className="mt-4 flex items-center gap-2 border-t border-slate-100 pt-4 text-sm text-slate-500">
                        <LuHourglass className="size-4 text-amber-500" aria-hidden />
                        Penawaran Anda sudah terkirim. Menunggu keputusan pemberi order.
                    </p>
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

                {canOffer || canCounter ? (
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
                                    <input id="offer_note" type="text" value={offerNote} onChange={(event) => setOfferNote(event.target.value)} className={inputClassName} placeholder="Syarat atau detail tambahan" />
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
                                        <input id="counter_note" type="text" value={counterNote} onChange={(event) => setCounterNote(event.target.value)} className={inputClassName} placeholder="Alasan penyesuaian harga" />
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
                                {canCounter ? (
                                    <button type="button" onClick={() => setPanel("counter")} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                                        Counter Penawaran
                                    </button>
                                ) : (
                                    <button type="button" onClick={() => setPanel("offer")} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                                        Buat Penawaran
                                    </button>
                                )}

                                <button type="button" onClick={() => setPanel("reject")} className="cursor-pointer rounded-xl border border-red-300 bg-white px-4 py-2.5 text-sm font-semibold text-red-600 transition hover:bg-red-50">
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
