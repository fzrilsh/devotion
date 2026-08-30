import type { SearchCandidate, SearchParams } from "@api/search";
import Loading from "@components/common/Loading";
import { useMasterMachines, useMasterProducts } from "@hooks/useListing";
import { useProfile } from "@hooks/useProfile";
import { useSearch } from "@hooks/useQuota";
import { useWilayah } from "@hooks/useWilayah";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuCheck, LuCircleAlert, LuMapPin, LuSearch, LuShieldCheck, LuStar, LuX } from "react-icons/lu";
import { useNavigate } from "react-router-dom";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

const regionOptions = [
    { value: "city" as const, label: "Kota saya" },
    { value: "province" as const, label: "Provinsi saya" },
    { value: "national" as const, label: "Nasional" },
];

function formatNumber(value: number): string {
    return value.toLocaleString("id-ID");
}

function CandidateCard({ candidate, selected, onToggle }: { candidate: SearchCandidate; selected: boolean; onToggle: () => void }) {
    const unmet = candidate.criteria.filter((criterion) => !criterion.met);
    const reputation = candidate.reputation;

    return (
        <div className={cn("rounded-2xl border bg-white p-5 transition-all", selected ? "border-industrial-blue-500 shadow-md shadow-industrial-blue-500/10" : "border-slate-200 hover:border-industrial-blue-500/30")}>
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                        <h3 className="truncate text-sm font-bold text-slate-900">{candidate.business_name}</h3>

                        {candidate.identity_verified ? (
                            <span className="inline-flex items-center gap-1 rounded-full border border-industrial-blue-500/20 bg-industrial-blue-500/10 px-2 py-0.5 text-[10px] font-bold text-industrial-blue-600">
                                <LuShieldCheck className="size-3" aria-hidden />
                                Terverifikasi
                            </span>
                        ) : null}

                        {candidate.stale_calendar ? (
                            <span className="inline-flex items-center gap-1 rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[10px] font-bold text-amber-600">
                                <LuCircleAlert className="size-3" aria-hidden />
                                Kalender belum diperbarui
                            </span>
                        ) : null}
                    </div>

                    <p className="mt-1 flex items-center gap-1.5 text-xs text-slate-500">
                        <LuMapPin className="size-3.5 shrink-0 text-slate-400" aria-hidden />
                        {candidate.city_name || "Lokasi tidak tersedia"}
                        {candidate.distance_km != null ? ` · ±${Math.round(candidate.distance_km)} km` : ""}
                    </p>
                </div>

                <span className={cn("grid size-9 shrink-0 place-items-center rounded-full text-sm font-extrabold", candidate.score === 4 ? "bg-emerald-500/10 text-emerald-600" : "bg-slate-100 text-slate-500")}>{candidate.score}/4</span>
            </div>

            <div className="mt-3 flex flex-wrap gap-1.5">
                {candidate.criteria.map((criterion) => (
                    <span
                        key={criterion.name}
                        title={criterion.detail ?? undefined}
                        className={cn("inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-semibold", criterion.met ? "bg-emerald-500/10 text-emerald-700" : "bg-red-500/10 text-red-600")}
                    >
                        {criterion.met ? <LuCheck className="size-3" aria-hidden /> : <LuX className="size-3" aria-hidden />}
                        {criterion.name}
                    </span>
                ))}
            </div>

            <div className="mt-4 grid grid-cols-3 gap-2 border-t border-slate-100 pt-3 text-center">
                <div>
                    <p className="text-sm font-extrabold tabular-nums text-slate-800">{candidate.weekly_capacity != null ? formatNumber(candidate.weekly_capacity) : "-"}</p>
                    <p className="text-[10px] text-slate-400">unit/minggu</p>
                </div>
                <div>
                    <p className="text-sm font-extrabold tabular-nums text-slate-800">{candidate.total_capacity_until_deadline != null ? formatNumber(candidate.total_capacity_until_deadline) : "-"}</p>
                    <p className="text-[10px] text-slate-400">sisa s.d. deadline</p>
                </div>
                <div>
                    <p className="text-sm font-extrabold tabular-nums text-slate-800">{candidate.readiness_lead_days != null ? `${candidate.readiness_lead_days} hari` : "-"}</p>
                    <p className="text-[10px] text-slate-400">jeda kesiapan</p>
                </div>
            </div>

            <div className="mt-3 flex items-center justify-between gap-3">
                <p className="flex items-center gap-1.5 text-xs text-slate-500">
                    <LuStar className="size-3.5 text-amber-400" aria-hidden />
                    {reputation?.average_rating != null ? Number(reputation.average_rating).toFixed(1) : "-"}
                    <span className="text-slate-300">·</span>
                    {reputation?.enough_data && reputation.completion_rate != null ? `${Math.round(reputation.completion_rate)}% selesai` : "Tingkat penyelesaian belum cukup data"}
                </p>

                <button
                    type="button"
                    onClick={onToggle}
                    disabled={unmet.length === 4}
                    className={cn(
                        "shrink-0 cursor-pointer rounded-xl px-4 py-2 text-xs font-bold transition disabled:cursor-not-allowed disabled:opacity-40",
                        selected ? "bg-deep-navy-500 text-white hover:bg-deep-navy-600" : "bg-industrial-blue-500 text-white hover:bg-industrial-blue-600",
                    )}
                >
                    {selected ? "Dipilih" : "Pilih"}
                </button>
            </div>
        </div>
    );
}

export default function Search() {
    const navigate = useNavigate();
    const { profile } = useProfile();
    const productsQuery = useMasterProducts();
    const machinesQuery = useMasterMachines();

    const [productItemId, setProductItemId] = useState("");
    const [machineItemId, setMachineItemId] = useState("");
    const [quantity, setQuantity] = useState("");
    const [deadline, setDeadline] = useState("");
    const [maxLeadDays, setMaxLeadDays] = useState("");
    const [regionLevel, setRegionLevel] = useState<"city" | "province" | "national">("city");
    const [formError, setFormError] = useState("");

    const [submitted, setSubmitted] = useState<SearchParams | null>(null);
    const [selected, setSelected] = useState<Record<string, SearchCandidate>>({});

    const { getCityName, getProvinceName } = useWilayah(profile?.province_code || undefined);

    const searchQuery = useSearch(submitted);
    const candidates = searchQuery.data?.pages.flatMap((page) => page.items) ?? [];
    const selectedList = Object.values(selected);

    const products = (productsQuery.data ?? []).filter((item) => item.active);
    const machines = (machinesQuery.data ?? []).filter((item) => item.active);

    function handleSearch(event: React.FormEvent) {
        event.preventDefault();
        setFormError("");

        const parsedQuantity = Number(quantity);

        if (!productItemId) {
            setFormError("Pilih jenis produk yang dibutuhkan.");
            return;
        }

        if (!Number.isInteger(parsedQuantity) || parsedQuantity < 1) {
            setFormError("Jumlah kebutuhan harus bilangan bulat minimal 1.");
            return;
        }

        if (!deadline) {
            setFormError("Tentukan tanggal deadline produksi.");
            return;
        }

        const params: SearchParams = {
            product_item_id: productItemId,
            quantity: parsedQuantity,
            deadline,
            region_level: regionLevel,
        };

        if (machineItemId) params.machine_item_id = machineItemId;

        const parsedLead = Number(maxLeadDays);
        if (maxLeadDays && Number.isInteger(parsedLead) && parsedLead >= 0) params.max_lead_days = parsedLead;

        if (regionLevel === "city" && profile?.city_code) params.city_code = profile.city_code;
        if (regionLevel === "province" && profile?.province_code) params.province_code = profile.province_code;

        setSelected({});
        setSubmitted(params);
    }

    function toggleCandidate(candidate: SearchCandidate) {
        setSelected((previous) => {
            const next = { ...previous };

            if (next[candidate.listing_id]) {
                delete next[candidate.listing_id];
            } else {
                next[candidate.listing_id] = candidate;
            }

            return next;
        });
    }

    function handleContinue() {
        if (!submitted || selectedList.length === 0) return;

        navigate("/quota-requests/new", {
            state: {
                listingIds: selectedList.map((candidate) => candidate.listing_id),
                listingNames: selectedList.map((candidate) => candidate.business_name),
                productItemId: submitted.product_item_id,
                productName: products.find((item) => item.item_id === submitted.product_item_id)?.name,
                quantity: submitted.quantity,
                deadline: submitted.deadline,
            },
        });
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Cari Subkontraktor</h1>
                <p className="mt-1 text-sm text-slate-500">Tentukan kebutuhan produksi Anda. Hasil diurutkan dari kecocokan kriteria paling tinggi.</p>
            </div>

            <form onSubmit={handleSearch} className="rounded-2xl border border-slate-200 bg-white p-6">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
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

                    <div>
                        <label htmlFor="machine_item_id" className={labelClassName}>
                            Jenis Mesin (opsional)
                        </label>
                        <select id="machine_item_id" value={machineItemId} onChange={(event) => setMachineItemId(event.target.value)} className={inputClassName}>
                            <option value="">Semua mesin</option>
                            {machines.map((item) => (
                                <option key={item.item_id} value={item.item_id}>
                                    {item.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label htmlFor="quantity" className={labelClassName}>
                            Jumlah Kebutuhan (unit) <span className="text-red-500">*</span>
                        </label>
                        <input id="quantity" type="number" min={1} value={quantity} onChange={(event) => setQuantity(event.target.value)} className={inputClassName} placeholder="Misalnya 3000" />
                    </div>

                    <div>
                        <label htmlFor="deadline" className={labelClassName}>
                            Deadline Produksi <span className="text-red-500">*</span>
                        </label>
                        <input id="deadline" type="date" value={deadline} onChange={(event) => setDeadline(event.target.value)} className={inputClassName} />
                    </div>

                    <div>
                        <label htmlFor="max_lead_days" className={labelClassName}>
                            Jeda Kesiapan Maksimal (hari)
                        </label>
                        <input id="max_lead_days" type="number" min={0} value={maxLeadDays} onChange={(event) => setMaxLeadDays(event.target.value)} className={inputClassName} placeholder="Kosongkan bila bebas" />
                    </div>

                    <div>
                        <span className={labelClassName}>Cakupan Wilayah</span>
                        <div className="flex gap-2">
                            {regionOptions.map((option) => (
                                <button
                                    key={option.value}
                                    type="button"
                                    onClick={() => setRegionLevel(option.value)}
                                    className={cn("flex-1 cursor-pointer rounded-xl border px-3 py-2.5 text-xs font-semibold transition", regionLevel === option.value ? "border-industrial-blue-500 bg-industrial-blue-500 text-white" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50")}
                                >
                                    {option.label}
                                </button>
                            ))}
                        </div>
                        <p className="mt-1.5 text-xs text-slate-400">
                            {regionLevel === "city" ? `Sekitar ${profile?.city_code ? getCityName(profile.city_code) : "kota Anda"}.` : regionLevel === "province" ? `Seluruh ${profile?.province_code ? getProvinceName(profile.province_code) : "provinsi Anda"}.` : "Seluruh Indonesia."}
                        </p>
                    </div>
                </div>

                {formError ? (
                    <div className="mt-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">
                        {formError}
                    </div>
                ) : null}

                <button type="submit" disabled={searchQuery.isFetching} className="mt-5 inline-flex w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-industrial-blue-500 px-6 py-3 font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-70">
                    <LuSearch className="size-4" aria-hidden />
                    {searchQuery.isFetching ? "Mencari..." : "Cari Subkontraktor"}
                </button>
            </form>

            {submitted ? (
                searchQuery.isLoading ? (
                    <Loading />
                ) : searchQuery.isError ? (
                    <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                        <p className="text-sm font-semibold text-red-700">Hasil pencarian tidak dapat dimuat. Periksa kembali kriteria Anda.</p>
                    </div>
                ) : candidates.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                        <p className="text-sm text-slate-500">Tidak ada subkontraktor yang cocok pada cakupan ini. Coba perluas wilayah atau longgarkan kriteria.</p>
                    </div>
                ) : (
                    <>
                        <p className="text-sm text-slate-500">
                            {candidates.length} kandidat ditemukan, diurutkan dari skor kecocokan tertinggi.
                        </p>

                        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                            {candidates.map((candidate) => (
                                <CandidateCard key={candidate.listing_id} candidate={candidate} selected={Boolean(selected[candidate.listing_id])} onToggle={() => toggleCandidate(candidate)} />
                            ))}
                        </div>

                        {searchQuery.hasNextPage ? (
                            <div className="text-center">
                                <button type="button" onClick={() => searchQuery.fetchNextPage()} disabled={searchQuery.isFetchingNextPage} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                                    {searchQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                                </button>
                            </div>
                        ) : null}
                    </>
                )
            ) : null}

            {selectedList.length > 0 ? (
                <div className="sticky bottom-4 flex items-center justify-between gap-3 rounded-2xl border border-industrial-blue-500/30 bg-white p-4 shadow-lg shadow-slate-200">
                    <p className="text-sm font-semibold text-slate-700">{selectedList.length} kandidat dipilih</p>

                    <button type="button" onClick={handleContinue} className="cursor-pointer rounded-xl bg-deep-navy-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-deep-navy-600">
                        Lanjut Buat Request Kuota
                    </button>
                </div>
            ) : null}
        </div>
    );
}
