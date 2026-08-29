import { ApiError } from "@api/client";
import type { CatalogItem } from "@api/listing";
import Loading from "@components/common/Loading";
import VerificationGate, { useAccountVerification } from "@components/common/VerificationGate";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCreateListing, useListingVisibility, useMasterMachines, useMasterProducts, useMyListing, useProposeMasterItem, useUpdateListing } from "@hooks/useListing";
import { cn } from "@lib/utils";
import { listingSchema, type ListingForm } from "@schemas/listing";
import { motion } from "motion/react";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { LuArrowRight, LuCalendarDays, LuCog, LuEye, LuEyeOff, LuInfo, LuLightbulb, LuPackage, LuPencil, LuSend, LuShirt, LuTriangleAlert, LuX } from "react-icons/lu";
import { Link } from "react-router-dom";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

function getProblemMessage(error: unknown): string {
    if (error instanceof ApiError) {
        if (typeof error.data === "object" && error.data !== null && "detail" in error.data && typeof error.data.detail === "string") {
            return error.data.detail;
        }

        if (error.status === 401) return "Sesi Anda habis, silakan masuk kembali.";
        if (error.status === 403) return "Anda tidak berwenang mengubah listing ini.";
    }

    return "Terjadi kesalahan. Silakan coba lagi.";
}

function ProposalBox({ kind, title }: { kind: "product" | "machine"; title: string }) {
    const proposeMutation = useProposeMasterItem();
    const [name, setName] = useState("");
    const [message, setMessage] = useState<{ tone: "success" | "error"; text: string } | null>(null);

    async function handleSubmit() {
        setMessage(null);

        const trimmed = name.trim();
        if (trimmed.length < 3) {
            setMessage({ tone: "error", text: "Nama usulan minimal 3 karakter." });
            return;
        }

        try {
            await proposeMutation.mutateAsync({ kind, proposedName: trimmed });
            setName("");
            setMessage({ tone: "success", text: "Usulan terkirim. Admin akan meninjau sebelum item masuk daftar baku." });
        } catch (error) {
            setMessage({ tone: "error", text: getProblemMessage(error) });
        }
    }

    return (
        <div className="mt-3 rounded-xl border border-dashed border-slate-300 bg-slate-50 p-3">
            <p className="text-xs font-semibold text-slate-500">{title}</p>

            <div className="mt-2 flex gap-2">
                <input
                    type="text"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    onKeyDown={(event) => {
                        if (event.key === "Enter") {
                            event.preventDefault();
                            void handleSubmit();
                        }
                    }}
                    placeholder="Nama item baru"
                    className="flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs text-slate-800 outline-none transition focus:border-industrial-blue-500"
                />
                <button
                    type="button"
                    onClick={() => void handleSubmit()}
                    disabled={proposeMutation.isPending}
                    className="inline-flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg bg-slate-700 px-3 py-2 text-xs font-bold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    <LuSend className="size-3.5" aria-hidden />
                    Usulkan
                </button>
            </div>

            {message ? <p className={cn("mt-2 text-xs", message.tone === "success" ? "text-emerald-600" : "text-red-600")}>{message.text}</p> : null}
        </div>
    );
}

function ListingFormCard({ products, machines, defaultValues, isEdit, onCancel, onSaved }: { products: CatalogItem[]; machines: CatalogItem[]; defaultValues: ListingForm; isEdit: boolean; onCancel: () => void; onSaved: () => void }) {
    const createMutation = useCreateListing();
    const updateMutation = useUpdateListing();
    const mutation = isEdit ? updateMutation : createMutation;

    const {
        register,
        handleSubmit,
        setValue,
        setError,
        control,
        formState: { errors },
    } = useForm<ListingForm>({
        resolver: zodResolver(listingSchema),
        defaultValues,
    });

    const selectedProducts = useWatch({ control, name: "product_item_ids" }) ?? [];
    const selectedMachines = useWatch({ control, name: "machines" }) ?? [];

    function toggleProduct(itemId: string) {
        const next = selectedProducts.includes(itemId) ? selectedProducts.filter((id) => id !== itemId) : [...selectedProducts, itemId];
        setValue("product_item_ids", next, { shouldValidate: true, shouldDirty: true });
    }

    function toggleMachine(itemId: string) {
        const exists = selectedMachines.some((machine) => machine.item_id === itemId);
        const next = exists ? selectedMachines.filter((machine) => machine.item_id !== itemId) : [...selectedMachines, { item_id: itemId, machine_count: 1 }];
        setValue("machines", next, { shouldDirty: true });
    }

    function setMachineCount(itemId: string, count: number) {
        setValue(
            "machines",
            selectedMachines.map((machine) => (machine.item_id === itemId ? { ...machine, machine_count: count } : machine)),
            { shouldDirty: true },
        );
    }

    async function onSubmit(values: ListingForm) {
        try {
            await mutation.mutateAsync({
                weekly_capacity: values.weekly_capacity,
                readiness_lead_days: values.readiness_lead_days,
                product_item_ids: values.product_item_ids,
                machines: values.machines,
            });
            onSaved();
        } catch (error) {
            setError("root", { message: getProblemMessage(error) });
        }
    }

    return (
        <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3, ease: "easeOut" }} className="overflow-hidden rounded-2xl border border-slate-200 bg-white">
            <div className="border-b border-slate-100 bg-linear-to-r from-industrial-blue-500/5 to-transparent px-6 py-5">
                <h2 className="text-base font-extrabold tracking-tight text-slate-900">{isEdit ? "Ubah Listing" : "Buat Listing Kapasitas"}</h2>
                <p className="mt-1 text-xs leading-5 text-slate-500">{isEdit ? "Perbarui detail kapasitas Anda. Perubahan langsung memengaruhi hasil pencarian." : "Isi detail kapasitas produksi Anda. Listing yang lengkap lebih mudah ditemukan pemberi order."}</p>
            </div>

            <form onSubmit={handleSubmit(onSubmit)} className="space-y-6 p-6" noValidate>
                {errors.root?.message ? (
                    <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                        {errors.root.message}
                    </div>
                ) : null}

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <label htmlFor="weekly_capacity" className={labelClassName}>
                            Kapasitas per Minggu (unit) <span className="text-red-500">*</span>
                        </label>
                        <input id="weekly_capacity" type="number" min={1} inputMode="numeric" className={cn(inputClassName, errors.weekly_capacity && "border-red-400 focus:border-red-500")} {...register("weekly_capacity", { valueAsNumber: true })} />
                        {errors.weekly_capacity && <p className="mt-1 text-sm text-red-600">{errors.weekly_capacity.message}</p>}
                    </div>

                    <div>
                        <label htmlFor="readiness_lead_days" className={labelClassName}>
                            Jeda Kesiapan (hari) <span className="text-red-500">*</span>
                        </label>
                        <input id="readiness_lead_days" type="number" min={0} max={90} inputMode="numeric" className={cn(inputClassName, errors.readiness_lead_days && "border-red-400 focus:border-red-500")} {...register("readiness_lead_days", { valueAsNumber: true })} />
                        <p className="mt-1 text-xs text-slate-400">Berapa hari Anda butuh sebelum produksi bisa mulai setelah kesepakatan.</p>
                        {errors.readiness_lead_days && <p className="mt-1 text-sm text-red-600">{errors.readiness_lead_days.message}</p>}
                    </div>
                </div>

                <div>
                    <p className={labelClassName}>
                        Jenis Produk yang Dikerjakan <span className="text-red-500">*</span>
                    </p>

                    <div className="flex flex-wrap gap-2">
                        {products.map((item) => {
                            const selected = selectedProducts.includes(item.item_id);

                            return (
                                <button
                                    key={item.item_id}
                                    type="button"
                                    onClick={() => toggleProduct(item.item_id)}
                                    aria-pressed={selected}
                                    className={cn(
                                        "cursor-pointer rounded-full border px-4 py-2 text-xs font-semibold transition-all",
                                        selected ? "border-industrial-blue-500 bg-industrial-blue-500 text-white shadow-sm" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50 hover:text-industrial-blue-600",
                                    )}
                                >
                                    {item.name}
                                </button>
                            );
                        })}
                    </div>

                    {errors.product_item_ids && <p className="mt-1 text-sm text-red-600">{errors.product_item_ids.message}</p>}

                    <ProposalBox kind="product" title="Jenis produk tidak ada di daftar? Usulkan ke admin." />
                </div>

                <div>
                    <p className={labelClassName}>Mesin yang Dimiliki</p>

                    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
                        {machines.map((item) => {
                            const entry = selectedMachines.find((machine) => machine.item_id === item.item_id);

                            return (
                                <div key={item.item_id} className={cn("flex items-center justify-between gap-3 rounded-xl border p-3 transition-colors", entry ? "border-industrial-blue-500/40 bg-industrial-blue-500/5" : "border-slate-200 bg-white")}>
                                    <label className="flex flex-1 cursor-pointer items-center gap-3">
                                        <input type="checkbox" checked={Boolean(entry)} onChange={() => toggleMachine(item.item_id)} className="size-4 rounded border-slate-300 accent-industrial-blue-500" />
                                        <span className="text-sm font-semibold text-slate-700">{item.name}</span>
                                    </label>

                                    {entry ? (
                                        <div className="flex items-center gap-2">
                                            <input
                                                type="number"
                                                min={1}
                                                max={999}
                                                value={entry.machine_count}
                                                onChange={(event) => setMachineCount(item.item_id, Math.max(1, Number(event.target.value) || 1))}
                                                aria-label={`Jumlah ${item.name}`}
                                                className="w-20 rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-center text-sm text-slate-800 outline-none transition focus:border-industrial-blue-500"
                                            />
                                            <span className="text-xs text-slate-400">unit</span>
                                        </div>
                                    ) : null}
                                </div>
                            );
                        })}
                    </div>

                    {errors.machines && <p className="mt-1 text-sm text-red-600">{typeof errors.machines.message === "string" ? errors.machines.message : "Periksa jumlah mesin."}</p>}

                    <ProposalBox kind="machine" title="Mesin tidak ada di daftar? Usulkan ke admin." />
                </div>

                <div className="flex gap-3 border-t border-slate-100 pt-5">
                    <button type="submit" disabled={mutation.isPending} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-6 py-3 font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-70">
                        {mutation.isPending ? "Menyimpan..." : isEdit ? "Simpan Perubahan" : "Buat Listing"}
                    </button>

                    {isEdit ? (
                        <button type="button" onClick={onCancel} className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 px-6 py-3 font-semibold text-slate-600 transition hover:bg-slate-50">
                            <LuX className="size-4" aria-hidden />
                            Batal
                        </button>
                    ) : null}
                </div>
            </form>
        </motion.div>
    );
}

export default function Listing() {
    const listingQuery = useMyListing();
    const productsQuery = useMasterProducts();
    const machinesQuery = useMasterMachines();
    const visibilityMutation = useListingVisibility();
    const { needsVerification } = useAccountVerification();
    const [editMode, setEditMode] = useState(false);

    if (listingQuery.isLoading || productsQuery.isLoading || machinesQuery.isLoading) {
        return <Loading />;
    }

    // getMyListing mengembalikan null saat listing belum ada (404), jadi
    // isError di sini hanya untuk kegagalan sungguhan.
    const listing = listingQuery.data ?? null;
    const loadError = listingQuery.isError;
    const blocked = needsVerification && !listing;

    const products = (productsQuery.data ?? []).filter((item) => item.active);
    const machines = (machinesQuery.data ?? []).filter((item) => item.active);
    const totalMachines = listing ? listing.machines.reduce((total, machine) => total + machine.machine_count, 0) : 0;

    const formDefaults: ListingForm = listing
        ? {
              weekly_capacity: listing.weekly_capacity,
              readiness_lead_days: listing.readiness_lead_days,
              product_item_ids: listing.product_items.map((item) => item.item_id),
              machines: listing.machines.map((machine) => ({ item_id: machine.item.item_id, machine_count: machine.machine_count })),
          }
        : {
              weekly_capacity: 100,
              readiness_lead_days: 7,
              product_item_ids: [],
              machines: [],
          };

    return (
        <div className="space-y-6">
            <div className="flex items-center flex-col sm:flex-row justify-between gap-4">
                <div>
                    <h1 className="text-xl font-bold text-slate-900">Listing Kapasitas</h1>
                    <p className="mt-1 text-sm text-slate-500">Tawarkan kapasitas produksi Anda agar dapat ditemukan pemberi order.</p>
                </div>

                <div className="flex shrink-0 items-center gap-2 w-full sm:w-auto">
                    {listing && !editMode && !needsVerification ? (
                        <button type="button" onClick={() => setEditMode(true)} className="inline-flex w-full sm:w-auto justify-center cursor-pointer items-center gap-2 rounded-xl bg-industrial-blue-500 px-4 py-2 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                            <LuPencil className="size-4" aria-hidden />
                            Edit Listing
                        </button>
                    ) : null}

                    <Link to="/listing/calendar" className="inline-flex items-center gap-2 w-full sm:w-auto rounded-xl border justify-center border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition-colors hover:bg-slate-50">
                        <LuCalendarDays className="size-4" aria-hidden />
                        Kalender
                    </Link>
                </div>
            </div>

            {loadError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Listing tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : blocked ? (
                <VerificationGate action="membuat listing kapasitas" />
            ) : !listing || editMode ? (
                needsVerification ? (
                    <VerificationGate action="mengubah listing kapasitas" />
                ) : (
                    <ListingFormCard products={products} machines={machines} defaultValues={formDefaults} isEdit={Boolean(listing)} onCancel={() => setEditMode(false)} onSaved={() => setEditMode(false)} />
                )
            ) : (
                <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3, ease: "easeOut" }} className="space-y-6">
                    {/* Status tayang */}
                    <div className={cn("flex flex-col gap-4 rounded-2xl border p-5 sm:flex-row sm:items-center sm:justify-between", listing.published ? "border-emerald-200 bg-emerald-50" : "border-amber-200 bg-amber-50")}>
                        <div className="flex items-center gap-3">
                            <span className={cn("grid size-11 shrink-0 place-items-center rounded-xl", listing.published ? "bg-emerald-500/10 text-emerald-600" : "bg-amber-500/10 text-amber-600")}>
                                {listing.published ? <LuEye className="size-5" aria-hidden /> : <LuEyeOff className="size-5" aria-hidden />}
                            </span>

                            <div>
                                <p className="text-sm font-bold text-slate-800">{listing.published ? "Listing sedang tayang" : "Listing tidak tayang"}</p>
                                <p className="text-xs text-slate-500">{listing.published ? "Pemberi order dapat menemukan listing Anda di pencarian." : "Listing Anda tidak muncul di hasil pencarian. Tayangkan agar mulai menerima request."}</p>
                            </div>
                        </div>

                        <button
                            type="button"
                            onClick={() => visibilityMutation.mutate(!listing.published)}
                            disabled={visibilityMutation.isPending}
                            className={cn(
                                "shrink-0 cursor-pointer rounded-xl px-5 py-2.5 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-60",
                                listing.published ? "border border-slate-300 bg-white text-slate-600 hover:bg-slate-100" : "bg-emerald-600 text-white hover:bg-emerald-700",
                            )}
                        >
                            {visibilityMutation.isPending ? "Memproses..." : listing.published ? "Nonaktifkan" : "Tayangkan"}
                        </button>
                    </div>

                    {visibilityMutation.isError ? (
                        <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3" role="alert">
                            <LuTriangleAlert className="mt-0.5 size-4.5 shrink-0 text-red-500" aria-hidden />
                            <p className="text-sm text-red-700">{getProblemMessage(visibilityMutation.error)}</p>
                        </div>
                    ) : null}

                    {/* Ringkasan angka */}
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                        <div className="rounded-2xl border border-slate-200 bg-white p-5">
                            <div className="flex items-center gap-3">
                                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                                    <LuPackage className="size-5" aria-hidden />
                                </span>

                                <div className="min-w-0">
                                    <p className="truncate text-2xl font-extrabold tabular-nums text-slate-900">{listing.weekly_capacity.toLocaleString("id-ID")}</p>
                                    <p className="text-xs text-slate-400">unit per minggu</p>
                                </div>
                            </div>
                        </div>

                        <div className="rounded-2xl border border-slate-200 bg-white p-5">
                            <div className="flex items-center gap-3">
                                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-amber-500/10 text-amber-600">
                                    <LuCalendarDays className="size-5" aria-hidden />
                                </span>

                                <div className="min-w-0">
                                    <p className="truncate text-2xl font-extrabold tabular-nums text-slate-900">{listing.readiness_lead_days} hari</p>
                                    <p className="text-xs text-slate-400">jeda kesiapan produksi</p>
                                </div>
                            </div>
                        </div>

                        <div className="rounded-2xl border border-slate-200 bg-white p-5">
                            <div className="flex items-center gap-3">
                                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-deep-navy-500/10 text-deep-navy-600">
                                    <LuCog className="size-5" aria-hidden />
                                </span>

                                <div className="min-w-0">
                                    <p className="truncate text-2xl font-extrabold tabular-nums text-slate-900">{totalMachines}</p>
                                    <p className="text-xs text-slate-400">total mesin tercatat</p>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Detail produk dan mesin */}
                    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                        <div className="rounded-2xl border border-slate-200 bg-white p-6">
                            <div className="flex items-center gap-2.5">
                                <LuShirt className="size-4.5 text-industrial-blue-500" aria-hidden />
                                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Jenis Produk</h2>
                            </div>

                            <div className="mt-4 flex flex-wrap gap-2">
                                {listing.product_items.map((item) => (
                                    <span key={item.item_id} className="rounded-full border border-industrial-blue-500/20 bg-industrial-blue-500/5 px-3.5 py-1.5 text-xs font-semibold text-industrial-blue-700">
                                        {item.name}
                                    </span>
                                ))}
                            </div>
                        </div>

                        <div className="rounded-2xl border border-slate-200 bg-white p-6">
                            <div className="flex items-center gap-2.5">
                                <LuCog className="size-4.5 text-industrial-blue-500" aria-hidden />
                                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Mesin</h2>
                            </div>

                            {listing.machines.length > 0 ? (
                                <ul className="mt-3 divide-y divide-slate-100">
                                    {listing.machines.map((machine) => (
                                        <li key={machine.item.item_id} className="flex items-center justify-between gap-3 py-3 first:pt-1 last:pb-0">
                                            <span className="text-sm font-semibold text-slate-700">{machine.item.name}</span>
                                            <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">{machine.machine_count} unit</span>
                                        </li>
                                    ))}
                                </ul>
                            ) : (
                                <p className="mt-3 text-sm text-slate-500">Belum ada mesin yang dicantumkan.</p>
                            )}
                        </div>
                    </div>

                    {/* Aksi lanjutan */}
                    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                        <Link to="/listing/calendar" className="group flex items-center gap-4 rounded-2xl border border-slate-200 bg-white p-5 transition-all hover:border-industrial-blue-500/30 hover:shadow-md hover:shadow-slate-200">
                            <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-industrial-blue-500/10 text-industrial-blue-600">
                                <LuCalendarDays className="size-5" aria-hidden />
                            </span>

                            <span className="min-w-0 flex-1">
                                <span className="block text-sm font-bold text-slate-800 transition-colors group-hover:text-industrial-blue-600">Atur Kalender Ketersediaan</span>
                                <span className="mt-0.5 block text-xs leading-5 text-slate-500">Tandai minggu yang penuh dan perbarui sisa kapasitas per minggu agar listing tetap muncul di pencarian.</span>
                            </span>

                            <LuArrowRight className="size-4 shrink-0 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-industrial-blue-500" aria-hidden />
                        </Link>

                        <div className="flex items-start gap-4 rounded-2xl bg-deep-navy-400 border border-slate-200 p-5">
                            <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-white/20 text-amber-500 shadow-sm">
                                <LuLightbulb className="size-5" aria-hidden />
                            </span>

                            <div>
                                <p className="text-sm font-bold text-white">Satu angka untuk semua produk</p>
                                <p className="mt-0.5 text-xs leading-5 text-slate-200">Kapasitas mingguan berlaku untuk seluruh jenis produk pada listing. Jika produk Anda sangat beragam, isi angka rata-rata produksi mingguan.</p>
                            </div>
                        </div>
                    </div>

                    <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                        <LuInfo className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                        <p className="text-xs leading-5 text-slate-500">Jenis produk atau mesin Anda belum ada di daftar baku? Masuk mode edit dan usulkan item baru ke admin dari form listing.</p>
                    </div>
                </motion.div>
            )}
        </div>
    );
}
