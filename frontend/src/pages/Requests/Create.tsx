import { ApiError } from "@api/client";
import Loading from "@components/common/Loading";
import VerificationGate from "@components/common/VerificationGate";
import { useAccountVerification } from "@hooks/useAccountVerification";
import { useMasterProducts } from "@hooks/useListing";
import { useCreateQuotaRequest } from "@hooks/useQuota";
import { useEffect, useState } from "react";
import { LuArrowLeft, LuSend, LuTriangleAlert, LuUsers } from "react-icons/lu";
import { Link, useLocation, useNavigate } from "react-router-dom";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

type CreateLocationState = {
    listingIds?: string[];
    listingNames?: string[];
    productItemId?: string;
    productName?: string;
    quantity?: number;
    deadline?: string;
};

function getProblemMessage(error: unknown): string {
    if (error instanceof ApiError) {
        if (typeof error.data === "object" && error.data !== null && "detail" in error.data && typeof error.data.detail === "string") {
            return error.data.detail;
        }

        if (error.status === 401) return "Sesi Anda habis, silakan masuk kembali.";
        if (error.status === 429) return "Terlalu banyak request dalam waktu singkat. Coba lagi beberapa saat.";
    }

    return "Request tidak dapat dikirim. Silakan coba lagi.";
}

export default function Create() {
    const navigate = useNavigate();
    const location = useLocation();
    const state = (location.state as CreateLocationState | null) ?? {};

    const productsQuery = useMasterProducts();
    const createMutation = useCreateQuotaRequest();
    const { needsVerification } = useAccountVerification();

    const [productItemId, setProductItemId] = useState(state.productItemId ?? "");
    const [quantity, setQuantity] = useState(state.quantity != null ? String(state.quantity) : "");
    const [material, setMaterial] = useState("");
    const [deadline, setDeadline] = useState(state.deadline ?? "");
    const [note, setNote] = useState("");
    const [errorMessage, setErrorMessage] = useState("");

    const listingIds = state.listingIds ?? [];
    const listingNames = state.listingNames ?? [];
    const products = (productsQuery.data ?? []).filter((item) => item.active);

    useEffect(() => {
        if (listingIds.length === 0) {
            navigate("/search", { replace: true });
        }
    }, [listingIds.length, navigate]);

    if (listingIds.length === 0) return <Loading />;

    async function handleSubmit(event: React.FormEvent) {
        event.preventDefault();
        setErrorMessage("");

        const parsedQuantity = Number(quantity);

        if (!productItemId) {
            setErrorMessage("Pilih jenis produk yang dibutuhkan.");
            return;
        }

        if (!Number.isInteger(parsedQuantity) || parsedQuantity < 1) {
            setErrorMessage("Jumlah kebutuhan harus bilangan bulat minimal 1.");
            return;
        }

        if (material.trim().length < 3) {
            setErrorMessage("Sebutkan bahan yang diminta, minimal 3 karakter.");
            return;
        }

        if (!deadline) {
            setErrorMessage("Tentukan tanggal deadline produksi.");
            return;
        }

        try {
            const request = await createMutation.mutateAsync({
                listing_ids: listingIds,
                product_item_id: productItemId,
                quantity: parsedQuantity,
                material: material.trim(),
                deadline,
                note: note.trim() || undefined,
            });

            navigate(`/quota-requests/${request.request_id}`, { replace: true });
        } catch (error) {
            setErrorMessage(getProblemMessage(error));
        }
    }

    return (
        <div className="mx-auto space-y-6">
            <div className="flex items-center gap-3">
                <Link to="/search" className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-600 transition hover:bg-slate-50">
                    <LuArrowLeft className="size-3.5" aria-hidden />
                    Pencarian
                </Link>

                <div>
                    <h1 className="text-xl font-bold text-slate-900">Buat Request Kuota</h1>
                    <p className="mt-1 text-sm text-slate-500">Satu request terkirim ke seluruh kandidat terpilih sekaligus.</p>
                </div>
            </div>

            <div className="rounded-2xl border border-industrial-blue-500/20 bg-industrial-blue-500/5 p-5">
                <div className="flex items-center gap-2.5">
                    <LuUsers className="size-4.5 text-industrial-blue-600" aria-hidden />
                    <p className="text-sm font-bold text-slate-800">{listingIds.length} kandidat tujuan</p>
                </div>

                <div className="mt-3 flex flex-wrap gap-1.5">
                    {listingNames.map((name, index) => (
                        <span key={index} className="rounded-full border border-industrial-blue-500/20 bg-white px-3 py-1 text-xs font-semibold text-industrial-blue-700">
                            {name}
                        </span>
                    ))}
                </div>

                <p className="mt-3 text-xs leading-5 text-slate-500">Batas waktu balasan 72 jam ditetapkan sistem sejak request dikirim.</p>
            </div>

            {needsVerification ? (
                <VerificationGate action="mengirim request kuota" />
            ) : (
                <form onSubmit={handleSubmit} className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
                {errorMessage ? (
                    <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                        {errorMessage}
                    </div>
                ) : null}

                <div>
                    <label htmlFor="product_item_id" className={labelClassName}>
                        Jenis Produk <span className="text-red-500">*</span>
                    </label>
                    <select id="product_item_id" value={productItemId} onChange={(event) => setProductItemId(event.target.value)} className={inputClassName}>
                        <option value="">Pilih produk</option>
                        {products.map((item) => (
                            <option key={item.item_id} value={item.item_id}>
                                {item.name}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <label htmlFor="quantity" className={labelClassName}>
                            Jumlah (unit) <span className="text-red-500">*</span>
                        </label>
                        <input id="quantity" type="number" min={1} value={quantity} onChange={(event) => setQuantity(event.target.value)} className={inputClassName} placeholder="Misalnya 3000" />
                    </div>

                    <div>
                        <label htmlFor="deadline" className={labelClassName}>
                            Deadline Produksi <span className="text-red-500">*</span>
                        </label>
                        <input id="deadline" type="date" value={deadline} onChange={(event) => setDeadline(event.target.value)} className={inputClassName} />
                    </div>
                </div>

                <div>
                    <label htmlFor="material" className={labelClassName}>
                        Bahan <span className="text-red-500">*</span>
                    </label>
                    <input id="material" type="text" value={material} onChange={(event) => setMaterial(event.target.value)} className={inputClassName} placeholder="Misalnya: katun combed 30s" />
                </div>

                <div>
                    <label htmlFor="note" className={labelClassName}>
                        Catatan (opsional)
                    </label>
                    <textarea id="note" rows={3} value={note} onChange={(event) => setNote(event.target.value)} className={inputClassName} placeholder="Detail tambahan seperti ukuran, warna, atau standar kemasan" />
                </div>

                <button type="submit" disabled={createMutation.isPending} className="inline-flex w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-industrial-blue-500 px-6 py-3 font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-70">
                    <LuSend className="size-4" aria-hidden />
                    {createMutation.isPending ? "Mengirim..." : `Kirim ke ${listingIds.length} Kandidat`}
                </button>
                </form>
            )}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuTriangleAlert className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Request yang sudah dikirim tidak dapat ditarik kembali. Pastikan jumlah, bahan, dan deadline sudah benar sebelum mengirim.</p>
            </div>
        </div>
    );
}
