import Loading from "@components/common/Loading";
import { useWhatsAppStatus } from "@hooks/useAdmin";
import { LuMessageCircle, LuQrCode, LuRefreshCw, LuTriangleAlert } from "react-icons/lu";

export default function AdminWhatsApp() {
    const statusQuery = useWhatsAppStatus();
    const status = statusQuery.data;

    return (
        <div className="mx-auto space-y-6">
            <div className="flex items-center justify-between gap-3">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">WhatsApp</h1>
                    <p className="mt-1 text-sm text-slate-500">Status sesi WhatsApp yang mengirim kode verifikasi dan notifikasi ke pengguna.</p>
                </div>

                <button type="button" onClick={() => statusQuery.refetch()} disabled={statusQuery.isFetching} className="inline-flex shrink-0 cursor-pointer items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" aria-label="Muat ulang status">
                    <LuRefreshCw className={statusQuery.isFetching ? "size-4 animate-spin" : "size-4"} aria-hidden />
                    Muat Ulang
                </button>
            </div>

            {statusQuery.isLoading ? (
                <Loading />
            ) : statusQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Status WhatsApp tidak dapat dimuat. Coba muat ulang.</p>
                </div>
            ) : status ? (
                <div className="space-y-4">
                    <div className={`flex items-start gap-4 rounded-2xl border p-6 ${status.connected ? "border-emerald-200 bg-emerald-50" : "border-red-200 bg-red-50"}`}>
                        <span className={`grid size-12 shrink-0 place-items-center rounded-xl ${status.connected ? "bg-emerald-500/15 text-emerald-600" : "bg-red-500/15 text-red-600"}`}>
                            <LuMessageCircle className="size-6" aria-hidden />
                        </span>

                        <div>
                            <p className={`text-sm font-bold ${status.connected ? "text-emerald-800" : "text-red-800"}`}>{status.connected ? "Sesi terhubung" : "Sesi terputus"}</p>
                            <p className={`mt-1 text-sm leading-6 ${status.connected ? "text-emerald-700" : "text-red-700"}`}>
                                {status.connected
                                    ? "Kode verifikasi dan notifikasi WhatsApp terkirim normal."
                                    : "Kode verifikasi dan notifikasi WhatsApp tidak terkirim sampai sesi ditautkan ulang. Notifikasi email tetap berjalan."}
                            </p>
                        </div>
                    </div>

                    {status.last_error ? (
                        <div className="flex items-start gap-3 rounded-2xl border border-amber-200 bg-amber-50 p-4">
                            <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden />
                            <div>
                                <p className="text-sm font-semibold text-amber-800">Galat terakhir</p>
                                <p className="mt-1 text-xs leading-5 text-amber-700">{status.last_error}</p>
                            </div>
                        </div>
                    ) : null}

                    {status.qr_code ? (
                        <div className="rounded-2xl border border-slate-200 bg-white p-6 text-center">
                            <div className="flex items-center justify-center gap-2">
                                <LuQrCode className="size-5 text-slate-500" aria-hidden />
                                <h2 className="text-sm font-bold text-slate-800">Tautkan ulang sesi</h2>
                            </div>

                            <p className="mt-2 text-xs leading-5 text-slate-500">Pindai kode QR ini dari aplikasi WhatsApp pada perangkat layanan (menu Perangkat Tertaut).</p>

                            <img src={`data:image/png;base64,${status.qr_code}`} alt="Kode QR penautan sesi WhatsApp" className="mx-auto mt-4 size-56 rounded-xl border border-slate-200" />

                            <p className="mt-3 text-[11px] text-slate-400">Kode QR diperbarui otomatis. Halaman ini memuat ulang status tiap 30 detik.</p>
                        </div>
                    ) : null}
                </div>
            ) : null}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuMessageCircle className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Nomor layanan tidak pernah ditampilkan di sini demi keamanan. Saat sesi terputus, platform tetap melayani; hanya kanal WhatsApp yang terganggu.</p>
            </div>
        </div>
    );
}
