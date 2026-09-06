import type { QuotaRequestDetail } from "@api/search";
import Loading from "@components/common/Loading";
import { useSentQuotaRequests } from "@hooks/useQuota";
import { cn } from "@lib/utils";
import { LuArrowRight, LuInbox, LuPlus, LuSend } from "react-icons/lu";
import { Link } from "react-router-dom";
import { candidateStatusMeta, formatDateShort } from "./meta";

function requestSummary(request: QuotaRequestDetail): { offered: number; awaiting: number; agreed: number } {
    let offered = 0;
    let awaiting = 0;
    let agreed = 0;

    for (const candidate of request.candidates) {
        if (candidate.status === "offered") offered++;
        if (candidate.status === "awaiting_reply") awaiting++;
        if (candidate.status === "agreed") agreed++;
    }

    return { offered, awaiting, agreed };
}

export default function Sent() {
    const requestsQuery = useSentQuotaRequests();
    const requests = requestsQuery.data?.pages.flatMap((page) => page.items) ?? [];

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">Request Terkirim</h1>
                    <p className="mt-1 text-sm text-slate-500">Request kuota yang Anda kirim ke calon subkontraktor.</p>
                </div>

                <Link to="/search" className="inline-flex shrink-0 items-center gap-2 rounded-xl bg-industrial-blue-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-industrial-blue-600">
                    <LuPlus className="size-4" aria-hidden />
                    Buat Request
                </Link>
            </div>

            {requestsQuery.isLoading ? (
                <Loading />
            ) : requestsQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Daftar request tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : requests.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Belum ada request terkirim. Cari subkontraktor lalu kirim request kuota dari hasil pencarian.</p>

                    <Link to="/search" className="mt-4 inline-flex items-center gap-2 rounded-xl bg-industrial-blue-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                        <LuPlus className="size-4" aria-hidden />
                        Cari Subkontraktor
                    </Link>
                </div>
            ) : (
                <ul className="space-y-3">
                    {requests.map((request) => {
                        const summary = requestSummary(request);

                        return (
                            <li key={request.request_id}>
                                <Link to={`/quota-requests/${request.request_id}`} className="group flex items-center gap-4 rounded-2xl border border-slate-200 bg-white p-5 transition-all hover:border-industrial-blue-500/30 hover:shadow-md hover:shadow-slate-200">
                                    <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                                        <LuSend className="size-5" aria-hidden />
                                    </span>

                                    <div className="min-w-0 flex-1">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <p className="text-sm font-bold text-slate-800">
                                                {request.quantity.toLocaleString("id-ID")} unit {request.material}
                                            </p>
                                        </div>

                                        <p className="mt-1 text-xs text-slate-400">
                                            Deadline {formatDateShort(request.deadline)} · {request.candidates.length} kandidat · Dikirim {formatDateShort(request.created_at)}
                                        </p>

                                        <div className="mt-2 flex flex-wrap gap-1.5">
                                            {summary.agreed > 0 ? <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", candidateStatusMeta.agreed.className)}>{summary.agreed} sepakat</span> : null}
                                            {summary.offered > 0 ? <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", candidateStatusMeta.offered.className)}>{summary.offered} penawaran</span> : null}
                                            {summary.awaiting > 0 ? <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", candidateStatusMeta.awaiting_reply.className)}>{summary.awaiting} menunggu</span> : null}
                                        </div>
                                    </div>

                                    <LuArrowRight className="size-4 shrink-0 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-industrial-blue-500" aria-hidden />
                                </Link>
                            </li>
                        );
                    })}
                </ul>
            )}

            {requestsQuery.hasNextPage ? (
                <div className="text-center">
                    <button type="button" onClick={() => requestsQuery.fetchNextPage()} disabled={requestsQuery.isFetchingNextPage} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                        {requestsQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                    </button>
                </div>
            ) : null}
        </div>
    );
}
