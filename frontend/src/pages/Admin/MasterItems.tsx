import { ApiError } from "@api/client";
import type { CatalogItem } from "@api/admin";
import Loading from "@components/common/Loading";
import { useCreateMasterItem, useMasterItems, useUpdateMasterItem } from "@hooks/useAdmin";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuCircleCheck, LuCog, LuInbox, LuPackage, LuPencil, LuPlus, LuX } from "react-icons/lu";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

function getProblemMessage(error: unknown, fallback: string): string {
    if (error instanceof ApiError) {
        if (typeof error.data === "object" && error.data !== null && "detail" in error.data && typeof error.data.detail === "string") {
            return error.data.detail;
        }
    }

    return fallback;
}

function ItemRow({ item }: { item: CatalogItem }) {
    const updateMutation = useUpdateMasterItem();
    const [editing, setEditing] = useState(false);
    const [name, setName] = useState(item.name);
    const [error, setError] = useState("");

    async function saveName() {
        setError("");

        const trimmed = name.trim();
        if (trimmed.length < 2 || trimmed.length > 100) {
            setError("Nama item harus 2 sampai 100 karakter.");
            return;
        }

        try {
            await updateMutation.mutateAsync({ itemId: item.item_id, data: { name: trimmed } });
            setEditing(false);
        } catch (err) {
            setError(getProblemMessage(err, "Nama tidak dapat disimpan."));
        }
    }

    async function toggleActive() {
        setError("");

        try {
            await updateMutation.mutateAsync({ itemId: item.item_id, data: { active: !item.active } });
        } catch (err) {
            setError(getProblemMessage(err, "Status item tidak dapat diubah."));
        }
    }

    return (
        <li className="flex flex-wrap items-center gap-3 px-5 py-4">
            <span className={cn("grid size-9 shrink-0 place-items-center rounded-lg", item.kind === "product" ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "bg-violet-500/10 text-violet-600")}>
                {item.kind === "product" ? <LuPackage className="size-4" aria-hidden /> : <LuCog className="size-4" aria-hidden />}
            </span>

            <div className="min-w-0 flex-1">
                {editing ? (
                    <div className="flex items-center gap-2">
                        <input type="text" value={name} onChange={(event) => setName(event.target.value)} className={cn(inputClassName, "py-2")} minLength={2} maxLength={100} aria-label="Nama item" autoFocus />
                        <button type="button" onClick={saveName} disabled={updateMutation.isPending} className="shrink-0 cursor-pointer rounded-lg bg-industrial-blue-500 p-2 text-white transition hover:bg-industrial-blue-600 disabled:opacity-60" aria-label="Simpan nama">
                            <LuCircleCheck className="size-4" aria-hidden />
                        </button>
                        <button
                            type="button"
                            onClick={() => {
                                setEditing(false);
                                setName(item.name);
                                setError("");
                            }}
                            className="shrink-0 cursor-pointer rounded-lg border border-slate-300 p-2 text-slate-500 transition hover:bg-slate-50"
                            aria-label="Batal mengubah nama"
                        >
                            <LuX className="size-4" aria-hidden />
                        </button>
                    </div>
                ) : (
                    <p className={cn("truncate text-sm font-semibold", item.active ? "text-slate-800" : "text-slate-400 line-through")}>{item.name}</p>
                )}

                {error ? (
                    <p className="mt-1 text-xs text-red-600" role="alert">
                        {error}
                    </p>
                ) : null}
            </div>

            <span className={cn("shrink-0 rounded-full px-2.5 py-0.5 text-[11px] font-bold", item.active ? "bg-emerald-500/10 text-emerald-600" : "bg-slate-200 text-slate-500")}>{item.active ? "Aktif" : "Nonaktif"}</span>

            <div className="flex shrink-0 gap-2">
                {!editing ? (
                    <button
                        type="button"
                        onClick={() => {
                            setEditing(true);
                            setName(item.name);
                        }}
                        className="cursor-pointer rounded-lg border border-slate-300 p-2 text-slate-500 transition hover:bg-slate-50"
                        aria-label={`Ubah nama ${item.name}`}
                    >
                        <LuPencil className="size-4" aria-hidden />
                    </button>
                ) : null}

                <button
                    type="button"
                    onClick={toggleActive}
                    disabled={updateMutation.isPending}
                    className={cn("cursor-pointer rounded-lg border px-3 py-2 text-xs font-semibold transition disabled:cursor-not-allowed disabled:opacity-60", item.active ? "border-red-300 text-red-600 hover:bg-red-50" : "border-emerald-300 text-emerald-600 hover:bg-emerald-50")}
                >
                    {item.active ? "Nonaktifkan" : "Aktifkan"}
                </button>
            </div>
        </li>
    );
}

export default function AdminMasterItems() {
    const [kind, setKind] = useState<"product" | "machine">("product");
    const itemsQuery = useMasterItems(kind);
    const createMutation = useCreateMasterItem();

    const [newName, setNewName] = useState("");
    const [formError, setFormError] = useState("");

    const items = itemsQuery.data ?? [];

    async function handleCreate(event: React.FormEvent) {
        event.preventDefault();
        setFormError("");

        const trimmed = newName.trim();
        if (trimmed.length < 2 || trimmed.length > 100) {
            setFormError("Nama item harus 2 sampai 100 karakter.");
            return;
        }

        try {
            await createMutation.mutateAsync({ kind, name: trimmed });
            setNewName("");
        } catch (err) {
            setFormError(getProblemMessage(err, "Item tidak dapat ditambahkan."));
        }
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Item Baku</h1>
                <p className="mt-1 text-sm text-slate-500">Kelola daftar baku produk dan mesin yang dipakai pada listing dan request kuota.</p>
            </div>

            <div className="flex flex-wrap gap-2" role="group" aria-label="Jenis item">
                {(
                    [
                        { value: "product", label: "Produk" },
                        { value: "machine", label: "Mesin" },
                    ] as const
                ).map((option) => (
                    <button
                        key={option.value}
                        type="button"
                        onClick={() => setKind(option.value)}
                        className={cn("cursor-pointer rounded-full border px-4 py-2 text-xs font-semibold transition", kind === option.value ? "border-industrial-blue-500 bg-industrial-blue-500 text-white" : "border-slate-300 bg-white text-slate-600 hover:border-industrial-blue-500/50")}
                    >
                        {option.label}
                    </button>
                ))}
            </div>

            <form onSubmit={handleCreate} className="rounded-2xl border border-slate-200 bg-white p-5">
                <label htmlFor="new_item_name" className="mb-2 block text-sm font-semibold text-slate-500">
                    Tambah {kind === "product" ? "produk" : "mesin"} baru
                </label>

                <div className="flex gap-2">
                    <input id="new_item_name" type="text" value={newName} onChange={(event) => setNewName(event.target.value)} className={inputClassName} placeholder={kind === "product" ? "Misalnya: kemeja flanel" : "Misalnya: mesin obras"} minLength={2} maxLength={100} />
                    <button type="submit" disabled={createMutation.isPending} className="inline-flex shrink-0 cursor-pointer items-center gap-2 rounded-xl bg-industrial-blue-500 px-4 py-3 text-sm font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-60">
                        <LuPlus className="size-4" aria-hidden />
                        {createMutation.isPending ? "Menyimpan..." : "Tambah"}
                    </button>
                </div>

                {formError ? (
                    <p className="mt-2 text-xs text-red-600" role="alert">
                        {formError}
                    </p>
                ) : null}
            </form>

            {itemsQuery.isLoading ? (
                <Loading />
            ) : itemsQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">Daftar item tidak dapat dimuat. Coba muat ulang halaman.</p>
                </div>
            ) : items.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Belum ada item {kind === "product" ? "produk" : "mesin"}. Tambahkan lewat formulir di atas.</p>
                </div>
            ) : (
                <div className="rounded-2xl border border-slate-200 bg-white">
                    <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
                        <h2 className="text-sm font-bold text-slate-800">{items.length} item {kind === "product" ? "produk" : "mesin"}</h2>
                    </div>

                    <ul className="divide-y divide-slate-100">
                        {items.map((item) => (
                            <ItemRow key={item.item_id} item={item} />
                        ))}
                    </ul>
                </div>
            )}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuPackage className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Menonaktifkan item tidak menghapus listing lama yang memakainya; item hanya berhenti ditawarkan pada formulir baru.</p>
            </div>
        </div>
    );
}