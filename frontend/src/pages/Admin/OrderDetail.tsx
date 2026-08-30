import Loading from "@components/common/Loading";
import { useWorkOrder } from "@hooks/useWorkOrders";
import { LuBanknote, LuCalendarDays, LuPackage } from "react-icons/lu";
import { Link, useParams } from "react-router-dom";
import { formatDateId, formatDateTimeId, formatRupiah, workOrderStatusMeta } from "../WorkOrders/meta";

export default function AdminOrderDetail() {
    const { workOrderId } = useParams<{ workOrderId: string }>();
    const workOrderQuery = useWorkOrder(workOrderId || "");

    if (!workOrderId) {
        return (
            <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                <p className="text-sm font-semibold text-red-700">ID pesanan tidak valid.</p>
            </div>
        );
    }

    if (workOrderQuery.isLoading) {
        return <Loading />;
    }

    if (workOrderQuery.isError) {
        return (
            <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                <p className="text-sm font-semibold text-red-700">Pesanan tidak dapat dimuat. Coba muat ulang halaman.</p>
            </div>
        );
    }

    const workOrder = workOrderQuery.data;
    if (!workOrder) {
        return (
            <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                <p className="text-sm font-semibold text-red-700">Pesanan tidak ditemukan.</p>
            </div>
        );
    }

    const statusMeta = workOrderStatusMeta[workOrder.status];

    return (
        <div className="space-y-6">
            <div className="space-y-2">
                <h1 className="text-2xl font-bold text-slate-900">Detail Pesanan</h1>
                <p className="text-sm text-slate-500">
                    Status: <span className={`inline-block rounded-full px-2.5 py-0.5 text-[11px] font-bold ${statusMeta?.className}`}>{statusMeta?.label}</span>
                </p>
            </div>

            {/* Main Content */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                {/* Left Column - Pesanan */}
                <div className="lg:col-span-2 space-y-6">
                    {/* Informasi Pesanan */}
                    <div className="rounded-2xl border border-slate-200 bg-white p-6">
                        <h2 className="text-lg font-bold text-slate-900 mb-4">Informasi Pesanan</h2>

                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            <div>
                                <p className="text-xs font-semibold text-slate-500 uppercase">Jumlah</p>
                                <p className="mt-1 text-lg font-bold text-slate-900">{workOrder.quantity?.toLocaleString("id-ID") ?? "-"} unit</p>
                            </div>

                            <div>
                                <p className="text-xs font-semibold text-slate-500 uppercase">Nilai Pesanan</p>
                                <p className="mt-1 text-lg font-bold text-slate-900">{formatRupiah(workOrder.total_price ?? 0)}</p>
                            </div>

                            {workOrder.readiness_lead_days !== undefined && (
                                <div>
                                    <p className="text-xs font-semibold text-slate-500 uppercase">Jeda Kesiapan</p>
                                    <p className="mt-1 text-sm text-slate-700">{workOrder.readiness_lead_days} hari</p>
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Riwayat Status */}
                    <div className="rounded-2xl border border-slate-200 bg-white p-6">
                        <h2 className="text-lg font-bold text-slate-900 mb-4">Riwayat Status</h2>

                        <div className="space-y-3">
                            {workOrder.status_history && workOrder.status_history.length > 0 ? (
                                workOrder.status_history.map((history, idx) => (
                                    <div key={idx} className="flex gap-3 pb-3 border-b border-slate-100 last:border-0 last:pb-0">
                                        <div className="flex-1">
                                            <p className="text-sm font-semibold text-slate-800">{history.status ? workOrderStatusMeta[history.status]?.label || history.status : "-"}</p>
                                            <p className="mt-0.5 text-xs text-slate-500">{formatDateTimeId(history.at)}</p>
                                            {history.note && <p className="mt-1 text-xs text-slate-600">{history.note}</p>}
                                        </div>
                                    </div>
                                ))
                            ) : (
                                <p className="text-sm text-slate-500">Belum ada perubahan status.</p>
                            )}
                        </div>
                    </div>

                    {/* Alokasi Kapasitas */}
                    {workOrder.allocations && workOrder.allocations.length > 0 && (
                        <div className="rounded-2xl border border-slate-200 bg-white p-6">
                            <h2 className="text-lg font-bold text-slate-900 mb-4">Alokasi Kapasitas</h2>

                            <div className="overflow-x-auto">
                                <table className="w-full text-sm">
                                    <thead>
                                        <tr className="border-b border-slate-200">
                                            <th className="text-left py-3 px-3 font-semibold text-slate-700">Minggu</th>
                                            <th className="text-right py-3 px-3 font-semibold text-slate-700">Kapasitas</th>
                                            <th className="text-right py-3 px-3 font-semibold text-slate-700">Dialokasikan</th>
                                            <th className="text-right py-3 px-3 font-semibold text-slate-700">Sisa</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {workOrder.allocations.map((alloc, idx) => (
                                            <tr key={idx} className="border-b border-slate-100 hover:bg-slate-50">
                                                <td className="py-3 px-3 text-slate-800">{formatDateId(alloc.week_start)}</td>
                                                <td className="text-right py-3 px-3 text-slate-800">{alloc.capacity?.toLocaleString("id-ID") ?? "-"}</td>
                                                <td className="text-right py-3 px-3 text-slate-800">{alloc.allocated?.toLocaleString("id-ID") ?? "-"}</td>
                                                <td className="text-right py-3 px-3 text-slate-800">{alloc.remaining?.toLocaleString("id-ID") ?? "-"}</td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    )}
                </div>

                {/* Right Column - Pihak & Pembayaran */}
                <div className="space-y-6">
                    {/* Pemberi Order */}
                    <div className="rounded-2xl border border-slate-200 bg-white p-5">
                        <h3 className="text-sm font-bold text-slate-900 mb-3">Pemberi Order</h3>
                        <Link to={`/profile/${workOrder.buyer_profile_id}`} className="text-sm font-semibold text-industrial-blue-600 hover:underline">
                            Lihat Profil
                        </Link>
                    </div>

                    {/* Subkontraktor */}
                    <div className="rounded-2xl border border-slate-200 bg-white p-5">
                        <h3 className="text-sm font-bold text-slate-900 mb-3">Subkontraktor</h3>
                        <Link to={`/profile/${workOrder.subcontractor_profile_id}`} className="text-sm font-semibold text-industrial-blue-600 hover:underline">
                            Lihat Profil
                        </Link>
                    </div>

                    {/* Riwayat Pembayaran */}
                    {workOrder.payments && workOrder.payments.length > 0 && (
                        <div className="rounded-2xl border border-slate-200 bg-white p-5">
                            <h3 className="text-sm font-bold text-slate-900 mb-3 flex items-center gap-2">
                                <LuBanknote className="size-4" aria-hidden />
                                Pernyataan Pembayaran
                            </h3>

                            <div className="space-y-2">
                                {workOrder.payments.map((payment, idx) => (
                                    <div key={idx} className="text-xs pb-2 border-b border-slate-100 last:border-0 last:pb-0">
                                        <p className="font-semibold text-slate-700">
                                            {payment.direction === "sent" ? "Terkirim" : "Diterima"} - {formatDateId(payment.date)}
                                        </p>
                                        {payment.note && <p className="mt-1 text-slate-600">{payment.note}</p>}
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    <div className="rounded-2xl border border-slate-200 bg-white p-6">
                        <h2 className="text-lg font-bold text-slate-900 mb-4">Tenggat Waktu</h2>

                        <div className="space-y-3">
                            {workOrder.readiness_deadline && (
                                <div className="flex items-center gap-3">
                                    <LuCalendarDays className="size-5 text-slate-400" aria-hidden />
                                    <div>
                                        <p className="text-xs font-semibold text-slate-500">Kesiapan Produksi</p>
                                        <p className="mt-0.5 text-sm text-slate-800">{formatDateId(workOrder.readiness_deadline)}</p>
                                    </div>
                                </div>
                            )}

                            {workOrder.deadline && (
                                <div className="flex items-center gap-3">
                                    <LuPackage className="size-5 text-slate-400" aria-hidden />
                                    <div>
                                        <p className="text-xs font-semibold text-slate-500">Deadline Pesanan</p>
                                        <p className="mt-0.5 text-sm text-slate-800">{formatDateId(workOrder.deadline)}</p>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
