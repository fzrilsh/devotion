import type { CandidateStatus } from "@api/search";
import Loading from "@components/common/Loading";
import { useIncomingCandidates } from "@hooks/useQuota";
import { latestOffer, resolveOfferChain } from "@lib/offers";
import { statusFilters } from "@lib/statusFilters";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuArrowRight, LuInbox, LuSend } from "react-icons/lu";
import { Link } from "react-router-dom";
import { candidateStatusMeta, formatRupiah } from "./meta";

const filterOptions = statusFilters(candidateStatusMeta);

// "Belum ada penawaran" hanya benar sebelum subkontraktor membalas. Status
// offered berarti rantai penawaran sudah ada, jadi kalimat itu akan menyesatkan
// bila rantainya tidak ikut terkirim oleh API.
function offerlessNote(status: CandidateStatus): string {
    if (status === "awaiting_reply") return "Menunggu balasan Anda";
    if (status === "offered") return "Negosiasi berjalan, riwayat penawaran belum termuat";

    return "Tanpa penawaran";
}

export default function Incoming() {
    const [status, setStatus] = useState<CandidateStatus | "all">("all");
    const candidatesQuery = useIncomingCandidates(status === "all" ? undefined : status);
    const candidates = candidatesQuery.data?.pages.flatMap((page) => page.items) ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Request Masuk</h1>
                <p className="mt-1 text-sm text-slate-500">Request kuota dari pemberi order yang ditujukan ke listing Anda. Batas balasan tiap request 72 jam.</p>
            </div>

            <div className="flex flex-wrap gap-2" role="group" aria-label="Saring status request">
                {filterOptions.map((option) => (
                    <button
                        key={option.value}
                        type="button"
                        onClick={() => setStatus(option.value)}
                        className={cn("cursor-pointer rounded-full border px-4 py-2 text-xs font-semibold transition", status === option.value ? "border-industrial-blue-500 bg-industrial-blue-500 text-white" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50")}
                    >
                        {option.label}
                    </button>
                ))}
            </div>

            {candidatesQuery.isLoading ? (
                <Loading />
            ) : candidatesQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Daftar request masuk tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : candidates.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">
                        {status === "all" ? "Belum ada request masuk. Pastikan listing Anda tayang dan kalender kapasitas diperbarui supaya muncul di hasil pencarian." : "Tidak ada request dengan status ini."}
                    </p>

                    {status === "all" ? (
                        <Link to="/listing" className="mt-4 inline-flex items-center gap-2 rounded-xl bg-industrial-blue-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                            Periksa Listing
                        </Link>
                    ) : null}
                </div>
            ) : (
                <ul className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                    {candidates.map((candidate) => {
                        const meta = candidateStatusMeta[candidate.status];

                        // offers dan latest_offer sama-sama opsional di respons incoming.
                        // Ronde terakhir dibaca dari rantai bila yang terkirim hanya rantai.
                        const latest = latestOffer(resolveOfferChain(candidate));

                        return (
                            <li key={candidate.candidate_id}>
                                <Link to={`/requests/incoming/${candidate.candidate_id}`} state={{ candidateId: candidate.candidate_id }} className="group flex items-center gap-4 rounded-2xl border border-slate-200 bg-white p-5 transition-all hover:border-industrial-blue-500/30 hover:shadow-md hover:shadow-slate-200">
                                    <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-deep-navy-500/10 text-deep-navy-600">
                                        <LuSend className="size-5" aria-hidden />
                                    </span>

                                    <div className="min-w-0 flex-1">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <p className="truncate text-sm font-bold text-slate-800">{candidate.business_name || "Pemberi order"}</p>
                                            <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                                        </div>

                                        {latest ? (
                                            <p className="mt-1 text-xs font-semibold text-slate-600">
                                                Ronde {latest.sequence}: {formatRupiah(latest.total_price)} · kesiapan {latest.readiness_lead_days} hari
                                            </p>
                                        ) : (
                                            <p className="mt-1 text-xs text-slate-400">{offerlessNote(candidate.status)}</p>
                                        )}
                                    </div>

                                    <LuArrowRight className="size-4 shrink-0 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-industrial-blue-500" aria-hidden />
                                </Link>
                            </li>
                        );
                    })}
                </ul>
            )}

            {candidatesQuery.hasNextPage ? (
                <div className="text-center">
                    <button type="button" onClick={() => candidatesQuery.fetchNextPage()} disabled={candidatesQuery.isFetchingNextPage} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                        {candidatesQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                    </button>
                </div>
            ) : null}
        </div>
    );
}
