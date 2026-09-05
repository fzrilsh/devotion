import Loading from "@components/common/Loading";
import PaymentMismatchNotice from "@components/common/PaymentMismatchNotice";
import { useMasterProducts } from "@hooks/useListing";
import { useWorkOrder } from "@hooks/useWorkOrders";
import { cn } from "@lib/utils";
import { LuArrowLeft, LuArrowRight, LuBanknote, LuCalendarDays, LuEye, LuPackage, LuShieldAlert } from "react-icons/lu";
import { Link, useNavigate, useParams } from "react-router-dom";
import { formatDateId, formatDateTimeId, formatRupiah, workOrderStatusMeta } from "../WorkOrders/meta";

function ErrorCard({ message, onBack }: { message: string; onBack: () => void }) {
    return (
        <div className="space-y-6">
            <h1 className="text-xl font-bold text-slate-900">Detail Pesanan</h1>

            <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                <p className="text-sm font-semibold text-red-700">{message}</p>

                <button type="button" onClick={onBack} className="mt-3 inline-flex cursor-pointer items-center gap-2 text-sm font-bold text-red-800 underline underline-offset-2">
                    <LuArrowLeft className="size-4" aria-hidden />
                    Kembali ke antrean
                </button>
            </div>
        </div>
    );
}

export default function AdminOrderDetail() {
    const { workOrderId = "" } = useParams();
    const navigate = useNavigate();
    const workOrderQuery = useWorkOrder(workOrderId);
    const productsQuery = useMasterProducts();

    function handleBack() {
        navigate(-1);
    }

    if (!workOrderId) {
        return <ErrorCard message="Id pesanan tidak ada pada alamat halaman." onBack={handleBack} />;
    }

    if (workOrderQuery.isLoading) {
        return <Loading />;
    }

    if (workOrderQuery.isError || !workOrderQuery.data) {
        return <ErrorCard message="Pesanan tidak dapat dimuat. Periksa kembali id pada alamat, lalu coba muat ulang halaman." onBack={handleBack} />;
    }

    const order = workOrderQuery.data;
    const productName = productsQuery.isLoading
        ? "Memuat jenis produk..."
        : productsQuery.data?.find((product) => product.item_id === order.product_item_id)?.name ?? "Jenis produk tidak tersedia";
    const statusMeta = workOrderStatusMeta[order.status];

    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <button type="button" onClick={handleBack} className="inline-flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                    <LuArrowLeft className="size-3.5" aria-hidden />
                    Kembali
                </button>

                <h1 className="text-xl font-bold text-slate-900">Detail Pesanan</h1>

                <span className={cn("ml-auto shrink-0 rounded-full px-3 py-1 text-xs font-bold", statusMeta.className)}>{statusMeta.label}</span>
            </div>

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuEye className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">
                    Anda membuka pesanan ini sebagai admin pengawas, bukan sebagai pihak pesanan. Halaman ini baca saja: perubahan status tetap dilakukan subkontraktor dan pemberi order. Admin mengubah nasib pesanan lewat keputusan sengketa.
                </p>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                <div className="space-y-6 lg:col-span-2">
                    <div className="rounded-2xl border border-slate-200 bg-white p-6">
                        <div className="flex items-start gap-4">
                            <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                                <LuPackage className="size-6" aria-hidden />
                            </span>

                            <div className="min-w-0">
                                <p className="text-lg font-extrabold text-slate-900">{productName}</p>
                                <p className="mt-0.5 text-xs text-slate-400">{order.quantity.toLocaleString("id-ID")} unit · Deadline {formatDateId(order.deadline)}</p>
                            </div>
                        </div>

                        <dl className="mt-6 grid grid-cols-1 gap-4 border-t border-slate-100 pt-6 sm:grid-cols-2 lg:grid-cols-3">
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Jenis Produk</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{productName}</dd>
                            </div>
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Jumlah</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{order.quantity.toLocaleString("id-ID")} unit</dd>
                            </div>
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Bahan</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{order.material || "-"}</dd>
                            </div>
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Deadline Produksi</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{formatDateId(order.deadline)}</dd>
                            </div>
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Nilai Pesanan</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{formatRupiah(order.total_price)}</dd>
                            </div>
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Jeda Kesiapan</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{order.readiness_lead_days != null ? `${order.readiness_lead_days} hari` : "-"}</dd>
                            </div>
                            <div>
                                <dt className="text-xs font-semibold uppercase tracking-wider text-slate-400">Tenggat Kesiapan</dt>
                                <dd className="mt-1 text-sm font-bold text-slate-800">{formatDateId(order.readiness_deadline)}</dd>
                            </div>
                        </dl>
                    </div>

                    {order.status === "shipped" && order.auto_confirm_at ? (
                        <div className="flex items-start gap-3 rounded-2xl border border-sky-200 bg-sky-50 p-4">
                            <LuCalendarDays className="mt-0.5 size-5 shrink-0 text-sky-600" aria-hidden />
                            <p className="text-xs leading-5 text-sky-800">Pesanan dikonfirmasi diterima secara otomatis pada {formatDateTimeId(order.auto_confirm_at)} bila tidak ada konfirmasi atau sengketa sebelumnya.</p>
                        </div>
                    ) : null}

                    <div className="rounded-2xl border border-slate-200 bg-white p-6">
                        <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Riwayat Status</h2>

                        {order.status_history && order.status_history.length > 0 ? (
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
                        ) : (
                            <p className="mt-3 text-sm text-slate-500">Belum ada perubahan status yang tercatat.</p>
                        )}
                    </div>

                    {order.allocations && order.allocations.length > 0 ? (
                        <div className="rounded-2xl border border-slate-200 bg-white p-6">
                            <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Alokasi Kapasitas</h2>

                            <div className="mt-3 overflow-x-auto">
                                <table className="w-full text-sm">
                                    <caption className="sr-only">Alokasi kapasitas pesanan ini per minggu</caption>
                                    <thead>
                                        <tr className="border-b border-slate-200 text-xs uppercase tracking-wider text-slate-400">
                                            <th scope="col" className="px-3 py-3 text-left font-semibold">
                                                Minggu
                                            </th>
                                            <th scope="col" className="px-3 py-3 text-right font-semibold">
                                                Kapasitas
                                            </th>
                                            <th scope="col" className="px-3 py-3 text-right font-semibold">
                                                Dialokasikan
                                            </th>
                                            <th scope="col" className="px-3 py-3 text-right font-semibold">
                                                Sisa
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {order.allocations.map((period) => (
                                            <tr key={period.week_start} className="border-b border-slate-100 last:border-0">
                                                <th scope="row" className="px-3 py-3 text-left font-medium text-slate-800">
                                                    {formatDateId(period.week_start)}
                                                </th>
                                                <td className="px-3 py-3 text-right tabular-nums text-slate-800">{period.capacity.toLocaleString("id-ID")}</td>
                                                <td className="px-3 py-3 text-right tabular-nums text-slate-800">{period.allocated.toLocaleString("id-ID")}</td>
                                                <td className="px-3 py-3 text-right tabular-nums text-slate-800">{period.remaining.toLocaleString("id-ID")}</td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    ) : null}
                </div>

                <div className="space-y-6">
                    <div className="rounded-2xl border border-slate-200 bg-white p-5">
                        <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Pihak Pesanan</h2>

                        <div className="mt-3 space-y-3">
                            <div>
                                <p className="text-xs font-semibold text-slate-500">Pemberi Order</p>
                                <Link to={`/profile/${order.buyer_profile_id}`} className="mt-0.5 inline-flex items-center gap-1.5 text-sm font-semibold text-industrial-blue-600 hover:underline">
                                    Lihat profil
                                    <LuArrowRight className="size-3.5" aria-hidden />
                                </Link>
                            </div>

                            <div className="border-t border-slate-100 pt-3">
                                <p className="text-xs font-semibold text-slate-500">Subkontraktor</p>
                                <Link to={`/profile/${order.subcontractor_profile_id}`} className="mt-0.5 inline-flex items-center gap-1.5 text-sm font-semibold text-industrial-blue-600 hover:underline">
                                    Lihat profil
                                    <LuArrowRight className="size-3.5" aria-hidden />
                                </Link>
                            </div>
                        </div>
                    </div>

                    <div className="rounded-2xl border border-slate-200 bg-white p-5">
                        <h2 className="flex items-center gap-2 text-sm font-bold uppercase tracking-wider text-slate-400">
                            <LuShieldAlert className="size-4" aria-hidden />
                            Sengketa
                        </h2>

                        <div className="mt-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 text-xs leading-5 text-slate-600">
                            Laporan sengketa dipelihara pada antrean sengketa admin, bukan diambil dari halaman pertama daftar. Buka antrean untuk melihat riwayat yang relevan terhadap pesanan ini.
                        </div>

                        <Link to="/admin/disputes" className="mt-4 inline-flex items-center gap-1.5 text-xs font-bold text-industrial-blue-500 transition-colors hover:text-industrial-blue-600">
                            Buka antrean sengketa
                            <LuArrowRight className="size-3.5" aria-hidden />
                        </Link>
                    </div>

                    <div className="rounded-2xl border border-slate-200 bg-white p-5">
                        <h2 className="flex items-center gap-2 text-sm font-bold uppercase tracking-wider text-slate-400">
                            <LuBanknote className="size-4" aria-hidden />
                            Pernyataan Pembayaran
                        </h2>

                        <PaymentMismatchNotice mismatch={order.payment_mismatch} audience="admin" className="mt-3" />

                        {order.payments && order.payments.length > 0 ? (
                            <ul className="mt-3 space-y-2">
                                {order.payments.map((payment) => (
                                    <li key={payment.payment_id} className="rounded-xl bg-slate-50 px-3 py-2.5">
                                        <div className="flex items-center justify-between gap-3">
                                            <p className="text-xs font-semibold text-slate-700">{payment.direction === "sent" ? "Pembayaran dikirim" : "Pembayaran diterima"}</p>
                                            <span className="shrink-0 text-xs text-slate-500">{formatDateId(payment.date)}</span>
                                        </div>

                                        {payment.note ? <p className="mt-1 text-xs text-slate-500">{payment.note}</p> : null}
                                    </li>
                                ))}
                            </ul>
                        ) : (
                            <p className="mt-3 text-xs text-slate-500">Belum ada pernyataan pembayaran dari kedua pihak.</p>
                        )}

                        <p className="mt-3 text-[11px] leading-4 text-slate-400">Platform tidak menahan maupun memproses dana. Yang tercatat hanya pernyataan kedua pihak, tanpa jumlah uang, jadi pernyataan yang bertentangan diselesaikan lewat mediasi.</p>
                    </div>
                </div>
            </div>
        </div>
    );
}
