import Loading from "@components/common/Loading";
import { useLateOrders } from "@hooks/useAdmin";
import { LuArrowRight, LuCircleCheck, LuClock, LuInbox } from "react-icons/lu";
import { Link } from "react-router-dom";
import { formatDateId, formatRupiah, workOrderStatusMeta } from "../WorkOrders/meta";

export default function AdminLateOrders() {
    const lateOrdersQuery = useLateOrders();
    const orders = lateOrdersQuery.data?.items ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Pesanan Terlambat</h1>
                <p className="mt-1 text-sm text-slate-500">Pesanan yang sudah melewati tenggat kesiapan produksi dan membutuhkan perhatian admin.</p>
            </div>

            {lateOrdersQuery.isLoading ? (
                <Loading />
            ) : lateOrdersQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Daftar pesanan terlambat tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : orders.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuCircleCheck className="size-10 text-emerald-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Tidak ada pesanan yang melewati tenggat. Semua pesanan berjalan sesuai jadwal.</p>
                </div>
            ) : (
                <ul className="space-y-3">
                    {orders.map((order) => {
                        const meta = workOrderStatusMeta[order.status];

                        return (
                            <li key={order.work_order_id}>
                                <Link to={`/orders/${order.work_order_id}`} className="group flex items-center gap-4 rounded-2xl border border-amber-200 bg-white p-5 transition-all hover:border-amber-400 hover:shadow-md hover:shadow-slate-200">
                                    <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-amber-500/10 text-amber-600">
                                        <LuClock className="size-5" aria-hidden />
                                    </span>

                                    <div className="min-w-0 flex-1">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <p className="text-sm font-bold text-slate-800">{order.quantity.toLocaleString("id-ID")} unit</p>
                                            <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${meta.className}`}>{meta.label}</span>
                                        </div>

                                        <p className="mt-1 text-xs text-slate-400">
                                            Tenggat kesiapan {formatDateId(order.readiness_deadline ?? order.deadline)}
                                            {order.total_price != null ? ` · Nilai ${formatRupiah(order.total_price)}` : ""}
                                        </p>
                                    </div>

                                    <LuArrowRight className="size-4 shrink-0 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-amber-500" aria-hidden />
                                </Link>
                            </li>
                        );
                    })}
                </ul>
            )}

            <div className="flex items-start gap-3 rounded-2xl border border-amber-500/20 bg-amber-50 p-4">
                <LuClock className="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden />
                <p className="text-xs leading-5 text-amber-800">Keterlambatan kesiapan memengaruhi reputasi subkontraktor. Hubungi kedua pihak lewat kontak pada detail pesanan sebelum sengketa dibuka.</p>
            </div>

            {orders.length > 0 ? (
                <div className="flex items-center gap-2 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <LuInbox className="size-4 shrink-0 text-slate-400" aria-hidden />
                    <p className="text-xs text-slate-500">{orders.length} pesanan membutuhkan tindak lanjut.</p>
                </div>
            ) : null}
        </div>
    );
}