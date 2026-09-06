import type { Offer, RequestCandidate } from "@api/search";
import Loading from "@components/common/Loading";
import { useMasterProducts } from "@hooks/useListing";
import { useAcceptOffer, useCounterOffer, useQuotaRequest } from "@hooks/useQuota";
import { canAcceptOffer, latestOffer, resolveOfferChain } from "@lib/offers";
import { getProblemMessage } from "@lib/problem";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuArrowLeft, LuCircleCheck, LuClock, LuHandshake, LuHourglass, LuSend } from "react-icons/lu";
import { Link, useNavigate, useParams } from "react-router-dom";
import { formatDayTimeId as formatDateTimeId, formatRupiah } from "@lib/datetime";
import { candidateStatusMeta, formatDateShort } from "./meta";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

function OfferHistory({ offers }: { offers: Offer[] }) {
    if (offers.length === 0) return null;

    return (
        <ol className="mt-3 space-y-2 border-t border-slate-100 pt-3">
            {offers.map((offer) => (
                <li key={offer.offer_id} className="flex items-center justify-between gap-3 text-xs">
                    <span className="flex items-center gap-2 text-slate-500">
                        <span className={cn("rounded-full px-2 py-0.5 font-bold", offer.party === "subcontractor" ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "bg-deep-navy-500/10 text-deep-navy-600")}>
                            Ronde {offer.sequence} · {offer.party === "subcontractor" ? "Subkontraktor" : "Anda"}
                        </span>
                        {offer.note ? <span className="truncate text-slate-400">{offer.note}</span> : null}
                    </span>

                    <span className="shrink-0 font-bold tabular-nums text-slate-700">{formatRupiah(offer.total_price)}</span>
                </li>
            ))}
        </ol>
    );
}

function CandidateCard({ candidate, agreed, onAccept, accepting }: { candidate: RequestCandidate; agreed: boolean; onAccept: (offerId: string) => Promise<void>; accepting: boolean }) {
    const counterMutation = useCounterOffer();
    const [counterOpen, setCounterOpen] = useState(false);
    const [counterPrice, setCounterPrice] = useState("");
    const [counterNote, setCounterNote] = useState("");
    const [error, setError] = useState("");

    const meta = candidateStatusMeta[candidate.status];
    const offers = resolveOfferChain(candidate);
    const latest = latestOffer(offers);

    async function handleAccept() {
        if (!latest) return;
        setError("");

        try {
            await onAccept(latest.offer_id);
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    async function handleCounter() {
        if (!latest) return;
        setError("");

        const price = Number(counterPrice);
        if (!Number.isInteger(price) || price < 1) {
            setError("Masukkan harga counter dalam rupiah bulat.");
            return;
        }

        try {
            await counterMutation.mutateAsync({ offerId: latest.offer_id, data: { total_price: price, note: counterNote.trim() || undefined } });
            setCounterOpen(false);
            setCounterPrice("");
            setCounterNote("");
        } catch (err) {
            setError(getProblemMessage(err));
        }
    }

    return (
        <div className={cn("rounded-2xl border bg-white p-5 transition-colors", candidate.status === "agreed" ? "border-emerald-300" : "border-slate-200")}>
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                    <p className="truncate text-sm font-bold text-slate-900">{candidate.business_name || "Subkontraktor"}</p>
                    <span className={cn("mt-1 inline-block rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                </div>

                {latest ? (
                    <div className="text-right">
                        <p className="text-lg font-extrabold tabular-nums text-slate-900">{formatRupiah(latest.total_price)}</p>
                        <p className="text-xs text-slate-400">Jeda kesiapan {latest.readiness_lead_days} hari</p>
                    </div>
                ) : (
                    <p className="text-right text-xs text-slate-400">Jeda kesiapan belum ditawarkan</p>
                )}
            </div>

            <OfferHistory offers={offers} />

            {error ? (
                <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                    {error}
                </div>
            ) : null}

            {canAcceptOffer(candidate.status, latest, "buyer") && !agreed ? (
                <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                    {counterOpen ? (
                        <div className="space-y-3">
                            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                                <input type="number" min={1} value={counterPrice} onChange={(event) => setCounterPrice(event.target.value)} placeholder="Harga counter (rupiah)" aria-label="Harga counter" className={inputClassName} />
                                <input type="text" value={counterNote} onChange={(event) => setCounterNote(event.target.value)} placeholder="Catatan (opsional)" aria-label="Catatan counter" className={inputClassName} maxLength={500} />
                            </div>

                            <div className="flex gap-2">
                                <button
                                    type="button"
                                    onClick={handleCounter}
                                    disabled={counterMutation.isPending}
                                    className="flex-1 cursor-pointer rounded-xl bg-deep-navy-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-deep-navy-600 disabled:cursor-not-allowed disabled:opacity-60"
                                >
                                    {counterMutation.isPending ? "Mengirim..." : "Kirim Counter"}
                                </button>
                                <button type="button" onClick={() => setCounterOpen(false)} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                    Batal
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className="flex gap-2">
                            <button
                                type="button"
                                onClick={handleAccept}
                                disabled={accepting}
                                className="inline-flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60"
                            >
                                <LuHandshake className="size-4" aria-hidden />
                                {accepting ? "Memproses..." : "Terima Penawaran"}
                            </button>
                            <button type="button" onClick={() => setCounterOpen(true)} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                Counter
                            </button>
                        </div>
                    )}

                    <p className="text-[11px] leading-4 text-slate-400">Menerima penawaran membentuk pesanan dan mengalokasikan kapasitas; kandidat lain otomatis tidak dilanjutkan.</p>
                </div>
            ) : null}

            {candidate.status === "agreed" ? (
                <p className="mt-3 flex items-center gap-1.5 border-t border-emerald-100 pt-3 text-xs font-semibold text-emerald-700">
                    <LuCircleCheck className="size-4" aria-hidden />
                    Kesepakatan terbentuk. Pesanan sudah dibuat dan kapasitas teralokasi.
                </p>
            ) : null}
        </div>
    );
}

export default function SentDetail() {
    const { requestId = "" } = useParams();
    const navigate = useNavigate();
    const requestQuery = useQuotaRequest(requestId);
    const productsQuery = useMasterProducts();

    // Gunakan satu mutasi penerimaan untuk seluruh request. Saat satu kandidat
    // diterima, pesanan bersama dibuat, semua kartu kandidat dinonaktifkan,
    // dan banner pesanan yang dibuat ditampilkan.
    const acceptMutation = useAcceptOffer();

    if (requestQuery.isLoading) return <Loading />;

    if (requestQuery.isError || !requestQuery.data) {
        return (
            <div className="space-y-6">
                <h1 className="text-xl font-bold text-slate-900">Detail Request</h1>
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Request tidak ditemukan atau Anda tidak berwenang melihatnya.</p>
                    <Link to="/quota-requests" className="mt-3 inline-flex items-center gap-2 text-sm font-bold text-red-800 underline underline-offset-2">
                        <LuArrowLeft className="size-4" aria-hidden />
                        Kembali ke request terkirim
                    </Link>
                </div>
            </div>
        );
    }

    const request = requestQuery.data;
    const productName = productsQuery.isLoading
        ? "Memuat jenis produk..."
        : productsQuery.data?.find((product) => product.item_id === request.product_item_id)?.name ?? "Jenis produk tidak tersedia";
    const agreedCandidate = request.candidates.find((candidate) => candidate.status === "agreed");
    const agreedOrder = acceptMutation.data;

    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <Link to="/quota-requests" className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                    <LuArrowLeft className="size-3.5" aria-hidden />
                    Request
                </Link>

                <h1 className="text-xl font-bold text-slate-900">Detail Request</h1>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <div className="flex items-start gap-4">
                    <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                        <LuSend className="size-6" aria-hidden />
                    </span>

                    <div className="min-w-0 flex-1">
                        <p className="text-lg font-extrabold text-slate-900">Detail permintaan kuota</p>
                        <p className="mt-0.5 text-xs text-slate-400">Informasi yang Anda kirim ke kandidat terpilih</p>
                    </div>
                </div>

                <dl className="mt-5 grid grid-cols-1 gap-4 border-t border-slate-100 pt-5 sm:grid-cols-2 lg:grid-cols-3">
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">Jenis Produk</dt>
                        <dd className="mt-1 text-sm font-bold text-slate-900">{productName}</dd>
                    </div>
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">Jumlah</dt>
                        <dd className="mt-1 text-sm font-bold text-slate-900">{request.quantity.toLocaleString("id-ID")} unit</dd>
                    </div>
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">Bahan</dt>
                        <dd className="mt-1 text-sm font-bold text-slate-900">{request.material}</dd>
                    </div>
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">Deadline Produksi</dt>
                        <dd className="mt-1 text-sm font-bold text-slate-900">{formatDateShort(request.deadline)}</dd>
                    </div>
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">Dikirim</dt>
                        <dd className="mt-1 text-sm font-bold text-slate-900">{formatDateShort(request.created_at)}</dd>
                    </div>
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">Batas Balasan</dt>
                        <dd className="mt-1 flex items-center gap-1.5 text-sm font-bold text-slate-900">
                            <LuClock className="size-3.5 text-slate-400" aria-hidden />
                            {formatDateTimeId(request.expires_at)}
                        </dd>
                    </div>
                </dl>

                <div className="mt-5 flex items-center gap-1.5 border-t border-slate-100 pt-4 text-sm text-slate-600">
                    <LuHourglass className="size-4 text-slate-400" aria-hidden />
                    <span><strong className="font-bold text-slate-900">{request.candidates.length}</strong> kandidat tujuan</span>
                </div>

                {request.note ? (
                    <div className="mt-4 rounded-xl bg-slate-50 p-4">
                        <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Catatan Kebutuhan</p>
                        <p className="mt-1 text-sm leading-6 text-slate-700">{request.note}</p>
                    </div>
                ) : null}
            </div>

            {agreedOrder ? (
                <div className="flex flex-col items-start justify-between gap-3 rounded-2xl border border-emerald-200 bg-emerald-50 p-5 sm:flex-row sm:items-center">
                    <p className="text-sm font-semibold text-emerald-800">Kesepakatan terbentuk. Pesanan baru sudah dibuat dan kapasitas teralokasi.</p>

                    <button type="button" onClick={() => navigate(`/orders/${agreedOrder.work_order_id}`)} className="shrink-0 cursor-pointer rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700">
                        Lihat Pesanan
                    </button>
                </div>
            ) : null}

            <div>
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Perbandingan Kandidat</h2>

                <div className="mt-3 grid grid-cols-1 gap-4 lg:grid-cols-2">
                    {request.candidates.map((candidate) => (
                        <CandidateCard
                            key={candidate.candidate_id}
                            candidate={candidate}
                            agreed={Boolean(agreedCandidate)}
                            accepting={acceptMutation.isPending}
                            onAccept={async (offerId) => {
                                await acceptMutation.mutateAsync(offerId);
                            }}
                        />
                    ))}
                </div>
            </div>
        </div>
    );
}
