import type { Dispute, ItemProposal, VerificationRequest } from "@api/admin";
import Loading from "@components/common/Loading";
import { useDisputes, useItemProposals, useLateOrders, useVerificationQueue } from "@hooks/useAdmin";
import { cn } from "@lib/utils";
import { LuArrowRight, LuCalendarX, LuClipboardList, LuClock, LuInbox, LuMessageSquare, LuMessagesSquare, LuShieldAlert, LuShieldCheck, LuTriangleAlert } from "react-icons/lu";
import { Link } from "react-router-dom";

function formatDate(isoDate?: string | null): string {
    if (!isoDate) return "-";

    const normalized = isoDate.trim().replace(" ", "T");
    const date = new Date(normalized);

    if (Number.isNaN(date.getTime())) return "-";

    return new Intl.DateTimeFormat("id-ID", {
        day: "numeric",
        month: "short",
        year: "numeric",
        timeZone: "Asia/Jakarta",
    }).format(date);
}

function QueueCard({ title, description, count, tone, icon: Icon, to }: { title: string; description: string; count: number; tone: "blue" | "amber" | "red" | "violet"; icon: React.ElementType; to: string }) {
    const tones = {
        blue: { card: "border-industrial-blue-500/20 hover:border-industrial-blue-500/40", icon: "bg-industrial-blue-500/10 text-industrial-blue-600", badge: "bg-industrial-blue-500/10 text-industrial-blue-600" },
        amber: { card: "border-amber-500/20 hover:border-amber-500/40", icon: "bg-amber-500/10 text-amber-600", badge: "bg-amber-500/10 text-amber-600" },
        red: { card: "border-red-500/20 hover:border-red-500/40", icon: "bg-red-500/10 text-red-600", badge: "bg-red-500/10 text-red-600" },
        violet: { card: "border-violet-500/20 hover:border-violet-500/40", icon: "bg-violet-500/10 text-violet-600", badge: "bg-violet-500/10 text-violet-600" },
    }[tone];

    return (
        <Link to={to} className={cn("group relative overflow-hidden rounded-2xl border bg-white p-5 transition-all duration-200 hover:shadow-lg hover:shadow-slate-200", tones.card)}>
            <div className="flex items-start justify-between gap-3">
                <span className={cn("grid size-11 place-items-center rounded-xl", tones.icon)}>
                    <Icon className="size-5" aria-hidden />
                </span>

                <span className={cn("rounded-full px-3 py-1 text-lg font-extrabold tabular-nums", tones.badge)}>{count}</span>
            </div>

            <h3 className="mt-4 text-sm font-bold text-slate-800">{title}</h3>
            <p className="mt-1 text-xs leading-5 text-slate-500">{description}</p>

            <span className="mt-3 inline-flex items-center gap-1.5 text-xs font-bold text-slate-400 transition-colors group-hover:text-industrial-blue-600">
                Buka antrean
                <LuArrowRight className="size-3.5 transition-transform duration-200 group-hover:translate-x-0.5" aria-hidden />
            </span>
        </Link>
    );
}

function QueueSection({ title, to, icon: Icon, isLoading, isError, isEmpty, emptyMessage, children }: { title: string; to: string; icon: React.ElementType; isLoading: boolean; isError: boolean; isEmpty: boolean; emptyMessage: string; children: React.ReactNode }) {
    return (
        <section className="flex flex-col rounded-2xl border border-slate-200 bg-white">
            <header className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
                <div className="flex items-center gap-3">
                    <span className="grid size-9 place-items-center rounded-lg bg-slate-100 text-slate-500">
                        <Icon className="size-4.5" aria-hidden />
                    </span>
                    <h2 className="text-sm font-bold text-slate-800">{title}</h2>
                </div>

                <Link to={to} className="inline-flex items-center gap-1.5 text-xs font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                    Lihat semua
                    <LuArrowRight className="size-3.5" aria-hidden />
                </Link>
            </header>

            <div className="flex-1 p-5">
                {isLoading ? (
                    <Loading />
                ) : isError ? (
                    <p className="py-6 text-center text-sm text-red-600">Antrean tidak dapat dimuat.</p>
                ) : isEmpty ? (
                    <div className="flex flex-col items-center py-6 text-center">
                        <LuInbox className="size-8 text-slate-300" aria-hidden />
                        <p className="mt-2 text-sm text-slate-500">{emptyMessage}</p>
                    </div>
                ) : (
                    <ul className="divide-y divide-slate-100">{children}</ul>
                )}
            </div>
        </section>
    );
}

const disputeStatusMeta: Record<Dispute["status"], { label: string; className: string }> = {
    reported: { label: "Dilaporkan", className: "bg-red-500/10 text-red-600" },
    in_mediation: { label: "Mediasi", className: "bg-amber-500/10 text-amber-600" },
    resolved: { label: "Selesai", className: "bg-emerald-500/10 text-emerald-600" },
};

function todayLabel(): string {
    const today = new Date();
    const hari = today.toLocaleDateString("id-ID", { weekday: "long" });
    const day = today.getDate();
    const month = today.toLocaleString("id-ID", { month: "long" });
    const year = today.getFullYear();

    return `${hari}, ${day} ${month} ${year}`;
}

export default function AdminDashboard() {
    const verificationQuery = useVerificationQueue("pending");
    const proposalsQuery = useItemProposals();
    const disputesQuery = useDisputes();
    const lateOrdersQuery = useLateOrders();

    const verificationItems = verificationQuery.data?.items ?? [];
    const pendingProposals = (proposalsQuery.data ?? []).filter((proposal) => proposal.status === "pending");
    const openDisputes = (disputesQuery.data ?? []).filter((dispute) => dispute.status !== "resolved");
    const lateOrders = lateOrdersQuery.data?.items ?? [];

    return (
        <div className="space-y-6">
            <div>
                <p className="text-xs font-semibold text-deep-navy-300 uppercase tracking-widest mb-1">{todayLabel()}</p>
                <h1 className="text-xl font-bold text-slate-900">Dasbor Admin</h1>
                <p className="mt-1 text-sm text-slate-500">Ringkasan antrean moderasi dan pengawasan platform.</p>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
                <QueueCard title="Antrean Verifikasi" description="Pengajuan verifikasi identitas yang menunggu keputusan." count={verificationItems.length} tone="blue" icon={LuShieldCheck} to="/admin/verification" />
                <QueueCard title="Usulan Item" description="Usulan produk dan mesin baru dari pengguna." count={pendingProposals.length} tone="violet" icon={LuMessageSquare} to="/admin/proposals" />
                <QueueCard title="Sengketa Terbuka" description="Sengketa yang masih dilaporkan atau dalam mediasi." count={openDisputes.length} tone="red" icon={LuMessagesSquare} to="/admin/disputes" />
                <QueueCard title="Pesanan Terlambat" description="Pesanan yang sudah melewati tenggat kesiapan produksi." count={lateOrders.length} tone="amber" icon={LuClock} to="/admin/late-orders" />
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                <QueueSection title="Verifikasi menunggu" to="/admin/verification" icon={LuShieldCheck} isLoading={verificationQuery.isLoading} isError={verificationQuery.isError} isEmpty={verificationItems.length === 0} emptyMessage="Tidak ada pengajuan verifikasi yang menunggu.">
                    {verificationItems.slice(0, 5).map((request: VerificationRequest) => (
                        <li key={request.request_id} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                            <div className="min-w-0">
                                <p className="truncate text-sm font-semibold text-slate-800">{request.business_name || "Usaha tanpa nama"}</p>
                                <p className="text-xs text-slate-400">Diajukan {formatDate(request.submitted_at)}</p>
                            </div>

                            <span className="shrink-0 rounded-full bg-amber-500/10 px-2.5 py-1 text-[11px] font-bold text-amber-600">Menunggu</span>
                        </li>
                    ))}
                </QueueSection>

                <QueueSection title="Sengketa terbuka" to="/admin/disputes" icon={LuShieldAlert} isLoading={disputesQuery.isLoading} isError={disputesQuery.isError} isEmpty={openDisputes.length === 0} emptyMessage="Tidak ada sengketa yang perlu ditangani.">
                    {openDisputes.slice(0, 5).map((dispute: Dispute) => {
                        const meta = disputeStatusMeta[dispute.status];

                        return (
                            <li key={dispute.dispute_id} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                                <div className="min-w-0">
                                    <p className="truncate text-sm font-semibold text-slate-800">{dispute.report_body}</p>
                                    <p className="text-xs text-slate-400">Dilaporkan {formatDate(dispute.created_at)}</p>
                                </div>

                                <span className={cn("shrink-0 rounded-full px-2.5 py-1 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                            </li>
                        );
                    })}
                </QueueSection>

                <QueueSection title="Pesanan terlambat" to="/admin/late-orders" icon={LuCalendarX} isLoading={lateOrdersQuery.isLoading} isError={lateOrdersQuery.isError} isEmpty={lateOrders.length === 0} emptyMessage="Tidak ada pesanan yang melewati tenggat.">
                    {lateOrders.slice(0, 5).map((order) => (
                        <li key={order.work_order_id} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                            <div className="min-w-0">
                                <p className="truncate text-sm font-semibold text-slate-800">{order.quantity.toLocaleString("id-ID")} unit</p>
                                <p className="text-xs text-slate-400">Tenggat kesiapan {formatDate(order.readiness_deadline ?? order.deadline)}</p>
                            </div>

                            <Link to={`/admin/orders/${order.work_order_id}`} className="inline-flex shrink-0 items-center gap-1.5 text-xs font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                                Detail
                                <LuArrowRight className="size-3.5" aria-hidden />
                            </Link>
                        </li>
                    ))}
                </QueueSection>

                <QueueSection title="Usulan item menunggu" to="/admin/proposals" icon={LuClipboardList} isLoading={proposalsQuery.isLoading} isError={proposalsQuery.isError} isEmpty={pendingProposals.length === 0} emptyMessage="Tidak ada usulan item yang menunggu keputusan.">
                    {pendingProposals.slice(0, 5).map((proposal: ItemProposal) => (
                        <li key={proposal.proposal_id} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                            <div className="min-w-0">
                                <p className="truncate text-sm font-semibold text-slate-800">{proposal.proposed_name}</p>
                                <p className="text-xs text-slate-400">Diusulkan {formatDate(proposal.created_at)}</p>
                            </div>

                            <span className="shrink-0 rounded-full bg-violet-500/10 px-2.5 py-1 text-[11px] font-bold text-violet-600">{proposal.kind === "product" ? "Produk" : "Mesin"}</span>
                        </li>
                    ))}
                </QueueSection>
            </div>

            <div className="flex items-start gap-3 rounded-2xl border border-amber-500/20 bg-amber-50 p-4">
                <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden />
                <p className="text-xs leading-5 text-amber-800">Pesanan yang melewati tenggat kesiapan dan sengketa yang terbuka memengaruhi reputasi kedua pihak. Tinjau antrean di atas secara berkala agar penyelesaian tidak tertunda.</p>
            </div>
        </div>
    );
}
