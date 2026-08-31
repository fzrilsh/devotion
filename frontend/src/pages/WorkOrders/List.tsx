import type { WorkOrderStatus } from "@api/workOrders";
import Loading from "@components/common/Loading";
import { useAuth } from "@hooks/useAuth";
import { useWorkOrders } from "@hooks/useWorkOrders";
import { isStatusFilter, statusFilters } from "@lib/statusFilters";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuArrowRight, LuClipboardList, LuInbox } from "react-icons/lu";
import { Link, useSearchParams } from "react-router-dom";
import { formatDateId, formatRupiah, getWorkOrderSide, workOrderSideMeta, workOrderStatusMeta } from "./meta";

const filterOptions = statusFilters(workOrderStatusMeta);

const roleFilters = [
    { value: undefined, label: "Semua Posisi" },
    { value: "as_buyer" as const, label: "Sebagai Pemberi Order" },
    { value: "as_subcontractor" as const, label: "Sebagai Subkontraktor" },
];

function parseStatusParam(value: string | null): WorkOrderStatus | "all" {
    return isStatusFilter(workOrderStatusMeta, value) ? value : "all";
}

export default function List() {
    const { user } = useAuth();

    // Notifikasi permintaan ulasan menautkan ke ?status=confirmed supaya pengguna
    // mendarat pada pesanan yang memang bisa diulas (FR-047).
    const [searchParams, setSearchParams] = useSearchParams();
    const status = parseStatusParam(searchParams.get("status"));
    const [role, setRole] = useState<"as_buyer" | "as_subcontractor" | undefined>(undefined);

    function setStatus(next: WorkOrderStatus | "all") {
        setSearchParams(next === "all" ? {} : { status: next }, { replace: true });
    }

    const ordersQuery = useWorkOrders(status === "all" ? [] : [status], role);
    const orders = ordersQuery.data?.pages.flatMap((page) => page.items) ?? [];
    const myProfileId = user?.profile_id ?? null;
    const hasBothRoles = Boolean(user?.roles?.buyer && user?.roles?.subcontractor);

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Pesanan</h1>
                <p className="mt-1 text-sm text-slate-500">Pesanan yang terbentuk dari penawaran yang diterima, baik sebagai pemberi order maupun subkontraktor.</p>
            </div>

            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div className="flex flex-wrap gap-2">
                    {filterOptions.map((filter) => (
                        <button
                            key={filter.value}
                            type="button"
                            onClick={() => setStatus(filter.value)}
                            className={cn(
                                "cursor-pointer rounded-full border px-3.5 py-1.5 text-xs font-semibold transition-all",
                                status === filter.value ? "border-industrial-blue-500 bg-industrial-blue-500 text-white shadow-sm" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50 hover:text-industrial-blue-600",
                            )}
                        >
                            {filter.label}
                        </button>
                    ))}
                </div>

                {hasBothRoles ? (
                    <select
                        value={role ?? ""}
                        onChange={(event) => setRole((event.target.value || undefined) as "as_buyer" | "as_subcontractor" | undefined)}
                        className="rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-600 outline-none transition focus:border-industrial-blue-500"
                        aria-label="Saring berdasarkan posisi saya"
                    >
                        {roleFilters.map((filter) => (
                            <option key={filter.label} value={filter.value ?? ""}>
                                {filter.label}
                            </option>
                        ))}
                    </select>
                ) : null}
            </div>

            {ordersQuery.isLoading ? (
                <Loading />
            ) : ordersQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Daftar pesanan tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : orders.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Belum ada pesanan pada filter ini. Pesanan yang terbentuk dari penawaran yang diterima akan muncul di sini.</p>
                </div>
            ) : (
                <ul className="space-y-3">
                    {orders.map((order) => {
                        const meta = workOrderStatusMeta[order.status];
                        const side = getWorkOrderSide(order, myProfileId);

                        return (
                            <li key={order.work_order_id}>
                                <Link to={`/orders/${order.work_order_id}`} className="group flex items-center gap-4 rounded-2xl border border-slate-200 bg-white p-5 transition-all hover:border-industrial-blue-500/30 hover:shadow-md hover:shadow-slate-200">
                                    <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                                        <LuClipboardList className="size-5" aria-hidden />
                                    </span>

                                    <div className="min-w-0 flex-1">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <p className="text-sm font-bold text-slate-800">{order.quantity.toLocaleString("id-ID")} unit</p>
                                            <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                                            {side ? <span className={cn("rounded-full px-2.5 py-0.5 text-[11px] font-bold", workOrderSideMeta[side].className)}>{workOrderSideMeta[side].label}</span> : null}
                                        </div>

                                        <p className="mt-1 text-xs text-slate-400">
                                            Tenggat {formatDateId(order.deadline)}
                                            {order.total_price != null ? ` · Nilai ${formatRupiah(order.total_price)}` : ""}
                                        </p>
                                    </div>

                                    <LuArrowRight className="size-4 shrink-0 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-industrial-blue-500" aria-hidden />
                                </Link>
                            </li>
                        );
                    })}
                </ul>
            )}

            {ordersQuery.hasNextPage ? (
                <div className="text-center">
                    <button
                        type="button"
                        onClick={() => ordersQuery.fetchNextPage()}
                        disabled={ordersQuery.isFetchingNextPage}
                        className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {ordersQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                    </button>
                </div>
            ) : null}
        </div>
    );
}
