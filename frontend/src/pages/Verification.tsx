import { apiUrl } from "@api/client";
import type { FileKind, VerificationStatus } from "@api/verification";
import Loading from "@components/common/Loading";
import { useProfile } from "@hooks/useProfile";
import { useMyVerificationRequests, useSubmitVerification, useUploadFile } from "@hooks/useVerification";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { useRef, useState } from "react";
import { Link } from "react-router-dom";
import { LuCircleCheck, LuCloudUpload, LuFileCheck, LuFileImage, LuHourglass, LuRefreshCw, LuShieldCheck, LuTriangleAlert, LuUserPen, LuX } from "react-icons/lu";
import { formatDateTimeLongId as formatDateTimeId } from "@lib/datetime";

const MAX_FILE_SIZE = 5 * 1024 * 1024;
const ACCEPTED_TYPES = "image/jpeg,image/png,application/pdf";

const statusMeta: Record<VerificationStatus, { label: string; className: string }> = {
    pending: { label: "Menunggu Keputusan", className: "bg-amber-500/10 text-amber-600" },
    approved: { label: "Disetujui", className: "bg-emerald-500/10 text-emerald-600" },
    rejected: { label: "Ditolak", className: "bg-red-500/10 text-red-600" },
};

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

const documentLinkClassName = "inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-600 transition hover:border-industrial-blue-500/50 hover:bg-industrial-blue-500/5 hover:text-industrial-blue-600";

function DocumentButton({ fileId, label }: { fileId: string; label: string }) {
    return (
        <a href={apiUrl(`/files/${fileId}`)} target="_blank" rel="noreferrer" className={documentLinkClassName}>
            <LuFileImage className="size-3.5" aria-hidden />
            {label}
        </a>
    );
}

type FileSlotProps = {
    id: string;
    label: string;
    hint: string;
    file: File | null;
    error: string;
    disabled?: boolean;
    onSelect: (file: File | null) => void;
};

function FileSlot({ id, label, hint, file, error, disabled, onSelect }: FileSlotProps) {
    const inputRef = useRef<HTMLInputElement>(null);

    return (
        <div>
            <span className={labelClassName}>
                {label} <span className="text-red-500">*</span>
            </span>

            <input
                ref={inputRef}
                id={id}
                type="file"
                accept={ACCEPTED_TYPES}
                disabled={disabled}
                className="sr-only"
                onChange={(event) => onSelect(event.target.files?.[0] ?? null)}
            />

            {file ? (
                <div className="flex items-center justify-between gap-3 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3">
                    <span className="flex min-w-0 items-center gap-2.5">
                        <LuFileCheck className="size-5 shrink-0 text-emerald-600" aria-hidden />
                        <span className="min-w-0">
                            <span className="block truncate text-sm font-semibold text-emerald-800">{file.name}</span>
                            <span className="block text-xs text-emerald-600">{(file.size / 1024 / 1024).toFixed(2)} MB</span>
                        </span>
                    </span>

                    <button
                        type="button"
                        disabled={disabled}
                        onClick={() => {
                            onSelect(null);
                            if (inputRef.current) inputRef.current.value = "";
                        }}
                        className="shrink-0 cursor-pointer rounded-lg p-1.5 text-emerald-700 transition hover:bg-emerald-100 disabled:cursor-not-allowed"
                        aria-label={`Hapus ${label}`}
                    >
                        <LuX className="size-4" aria-hidden />
                    </button>
                </div>
            ) : (
                <button
                    type="button"
                    disabled={disabled}
                    onClick={() => inputRef.current?.click()}
                    className="flex w-full cursor-pointer flex-col items-center gap-2 rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-center transition hover:border-industrial-blue-500/50 hover:bg-industrial-blue-500/5 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    <LuCloudUpload className="size-7 text-slate-400" aria-hidden />
                    <span className="text-sm font-semibold text-slate-600">Pilih berkas</span>
                    <span className="text-xs text-slate-400">{hint}</span>
                </button>
            )}

            {error ? (
                <p className="mt-2 text-xs text-red-600" role="alert">
                    {error}
                </p>
            ) : null}
        </div>
    );
}

export default function Verification() {
    const { profile, isLoading: profileLoading } = useProfile();
    const requestsQuery = useMyVerificationRequests();
    const uploadMutation = useUploadFile();
    const submitMutation = useSubmitVerification();

    const [identityNumber, setIdentityNumber] = useState("");
    const [identityFile, setIdentityFile] = useState<File | null>(null);
    const [locationFile, setLocationFile] = useState<File | null>(null);
    const [fileErrors, setFileErrors] = useState<Partial<Record<FileKind, string>>>({});
    const [errorMessage, setErrorMessage] = useState("");

    const requests = requestsQuery.data ?? [];
    const latest = requests[0] ?? null;
    const pendingRequest = requests.find((request) => request.status === "pending");
    const approved = Boolean(profile?.identity_verified);
    const formLocked = approved || Boolean(pendingRequest);
    const busy = uploadMutation.isPending || submitMutation.isPending;

    function selectFile(kind: FileKind, file: File | null) {
        setFileErrors((previous) => ({ ...previous, [kind]: "" }));
        setErrorMessage("");

        if (file && file.size > MAX_FILE_SIZE) {
            setFileErrors((previous) => ({ ...previous, [kind]: "Ukuran berkas maksimal 5 MB." }));
            return;
        }

        if (kind === "identity_document") setIdentityFile(file);
        else setLocationFile(file);
    }

    async function handleSubmit(event: React.FormEvent) {
        event.preventDefault();
        setErrorMessage("");

        const number = identityNumber.trim();
        if (number.length < 8 || number.length > 32) {
            setErrorMessage("Nomor identitas usaha harus 8 sampai 32 karakter.");
            return;
        }

        if (!identityFile) {
            setFileErrors((previous) => ({ ...previous, identity_document: "Unggah dokumen identitas usaha." }));
            return;
        }

        if (!locationFile) {
            setFileErrors((previous) => ({ ...previous, location_photo: "Unggah foto lokasi usaha." }));
            return;
        }

        try {
            const [identity, location] = await Promise.all([
                uploadMutation.mutateAsync({ kind: "identity_document", file: identityFile }),
                uploadMutation.mutateAsync({ kind: "location_photo", file: locationFile }),
            ]);

            await submitMutation.mutateAsync({
                identity_number: number,
                identity_file_id: identity.file_id,
                location_file_id: location.file_id,
            });

            setIdentityNumber("");
            setIdentityFile(null);
            setLocationFile(null);
        } catch (error) {
            setErrorMessage(getProblemMessage(error, "Pengajuan verifikasi tidak dapat dikirim. Silakan coba lagi.", { 413: "Ukuran berkas maksimal 5 MB.", 415: "Tipe berkas tidak diizinkan. Gunakan JPG, PNG, atau PDF." }));
        }
    }

    if (profileLoading || requestsQuery.isLoading) return <Loading />;

    return (
        <div className="mx-auto space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Verifikasi Identitas</h1>
                <p className="mt-1 text-sm text-slate-500">Ajukan verifikasi identitas usaha untuk mendapatkan lencana terverifikasi pada profil Anda.</p>
            </div>

            {approved ? (
                <div className="flex items-start gap-4 rounded-2xl border border-emerald-200 bg-emerald-50 p-6">
                    <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-emerald-500/15 text-emerald-600">
                        <LuShieldCheck className="size-6" aria-hidden />
                    </span>

                    <div>
                        <p className="text-sm font-bold text-emerald-800">Usaha Anda sudah terverifikasi</p>
                        <p className="mt-1 text-sm leading-6 text-emerald-700">
                            Lencana terverifikasi tampil pada profil publik dan hasil pencarian.
                            {latest?.decided_at ? ` Disetujui pada ${formatDateTimeId(latest.decided_at)}.` : ""}
                        </p>

                        <Link to="/profile/me" className="mt-3 inline-flex items-center gap-1.5 rounded-xl border border-emerald-300 bg-white px-4 py-2 text-xs font-bold text-emerald-700 transition hover:bg-emerald-50">
                            <LuUserPen className="size-3.5" aria-hidden />
                            Lihat profil saya
                        </Link>
                    </div>
                </div>
            ) : pendingRequest ? (
                <div className="flex items-start gap-4 rounded-2xl border border-amber-200 bg-amber-50 p-6">
                    <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-amber-500/15 text-amber-600">
                        <LuHourglass className="size-6" aria-hidden />
                    </span>

                    <div>
                        <p className="text-sm font-bold text-amber-800">Pengajuan sedang ditinjau admin</p>
                        <p className="mt-1 text-sm leading-6 text-amber-700">Diajukan pada {formatDateTimeId(pendingRequest.submitted_at)}. Anda akan menerima notifikasi setelah ada keputusan.</p>

                        <button
                            type="button"
                            onClick={() => requestsQuery.refetch()}
                            disabled={requestsQuery.isFetching}
                            className="mt-3 inline-flex cursor-pointer items-center gap-1.5 rounded-xl border border-amber-300 bg-white px-4 py-2 text-xs font-bold text-amber-700 transition hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                            <LuRefreshCw className={cn("size-3.5", requestsQuery.isFetching && "animate-spin")} aria-hidden />
                            {requestsQuery.isFetching ? "Memeriksa..." : "Periksa keputusan"}
                        </button>
                    </div>
                </div>
            ) : null}

            {!approved && latest?.status === "rejected" && !pendingRequest ? (
                <div className="flex items-start gap-4 rounded-2xl border border-red-200 bg-red-50 p-6">
                    <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-red-500/15 text-red-600">
                        <LuTriangleAlert className="size-6" aria-hidden />
                    </span>

                    <div>
                        <p className="text-sm font-bold text-red-800">Pengajuan sebelumnya ditolak</p>
                        <p className="mt-1 text-sm leading-6 text-red-700">{latest.reason || "Dokumen tidak memenuhi syarat."} Perbaiki dokumen lalu ajukan kembali lewat formulir di bawah.</p>
                    </div>
                </div>
            ) : null}

            {!formLocked ? (
                <form onSubmit={handleSubmit} className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
                    {errorMessage ? (
                        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert" aria-live="polite">
                            {errorMessage}
                        </div>
                    ) : null}

                    <div>
                        <label htmlFor="identity_number" className={labelClassName}>
                            Nomor Identitas Usaha <span className="text-red-500">*</span>
                        </label>
                        <input id="identity_number" type="text" value={identityNumber} onChange={(event) => setIdentityNumber(event.target.value)} className={inputClassName} placeholder="Misalnya nomor NIB atau KTP penanggung jawab" minLength={8} maxLength={32} />
                        <p className="mt-1.5 text-xs text-slate-400">8 sampai 32 karakter. Nomor ini hanya terlihat oleh admin.</p>
                    </div>

                    <FileSlot id="identity_file" label="Dokumen Identitas" hint="JPG, PNG, atau PDF, maksimal 5 MB" file={identityFile} error={fileErrors.identity_document ?? ""} disabled={busy} onSelect={(file) => selectFile("identity_document", file)} />

                    <FileSlot id="location_file" label="Foto Lokasi Usaha" hint="JPG, PNG, atau PDF, maksimal 5 MB" file={locationFile} error={fileErrors.location_photo ?? ""} disabled={busy} onSelect={(file) => selectFile("location_photo", file)} />

                    <button type="submit" disabled={busy} className="inline-flex w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-industrial-blue-500 px-6 py-3 font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-70">
                        <LuShieldCheck className="size-4" aria-hidden />
                        {busy ? "Mengirim pengajuan..." : "Ajukan Verifikasi"}
                    </button>
                </form>
            ) : null}

            {requests.length > 0 ? (
                <div className="rounded-2xl border border-slate-200 bg-white p-6">
                    <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">Riwayat Pengajuan</h2>

                    <ol className="mt-4 space-y-3">
                        {requests.map((request) => {
                            const meta = statusMeta[request.status];

                            return (
                                <li key={request.request_id} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-100 bg-slate-50 px-4 py-3">
                                    <div className="min-w-0">
                                        <span className={cn("inline-block rounded-full px-2.5 py-0.5 text-[11px] font-bold", meta.className)}>{meta.label}</span>
                                        <p className="mt-1 text-xs text-slate-400">
                                            Diajukan {formatDateTimeId(request.submitted_at)}
                                            {request.decided_at ? ` · Diputuskan ${formatDateTimeId(request.decided_at)}` : ""}
                                        </p>
                                    </div>

                                    <div className="flex shrink-0 items-center gap-2">
                                        {request.identity_file_id ? <DocumentButton fileId={request.identity_file_id} label="Dokumen Identitas" /> : null}
                                        {request.location_file_id ? <DocumentButton fileId={request.location_file_id} label="Foto Lokasi" /> : null}
                                        {request.status === "approved" ? <LuCircleCheck className="size-5 shrink-0 text-emerald-500" aria-hidden /> : null}
                                    </div>
                                </li>
                            );
                        })}
                    </ol>
                </div>
            ) : null}

            <div className="flex items-start gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <LuShieldCheck className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                <p className="text-xs leading-5 text-slate-500">Verifikasi tidak menahan listing Anda untuk tayang dan tidak memengaruhi urutan hasil pencarian. Verifikasi hanya menambah lencana kepercayaan pada profil Anda.</p>
            </div>
        </div>
    );
}
