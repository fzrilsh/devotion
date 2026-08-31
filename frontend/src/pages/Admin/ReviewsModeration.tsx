import { ApiError } from "@api/client";
import Loading from "@components/common/Loading";
import { useHideReview } from "@hooks/useAdmin";
import { useProfileReviews } from "@hooks/useProfile";
import { cn } from "@lib/utils";
import { useState } from "react";
import { LuEyeOff, LuInbox, LuSearch, LuStar } from "react-icons/lu";
import { formatDateId } from "@lib/datetime";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";

const UUID_PATTERN = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

function getProblemMessage(error: unknown): string {
    if (error instanceof ApiError) {
        if (typeof error.data === "object" && error.data !== null && "detail" in error.data && typeof error.data.detail === "string") {
            return error.data.detail;
        }

        if (error.status === 404) return "Profil dengan ID tersebut tidak ditemukan.";
    }

    return "Ulasan tidak dapat dimuat. Silakan coba lagi.";
}

function StarRating({ rating }: { rating: number }) {
    return (
        <span className="flex items-center gap-0.5" aria-label={`Rating ${rating} dari 5`}>
            {[1, 2, 3, 4, 5].map((star) => (
                <LuStar key={star} className={cn("size-3.5", star <= rating ? "fill-amber-400 text-amber-400" : "text-slate-300")} aria-hidden />
            ))}
        </span>
    );
}

export default function AdminReviewsModeration() {
    const [profileIdInput, setProfileIdInput] = useState("");
    const [profileId, setProfileId] = useState("");
    const [formError, setFormError] = useState("");

    const reviewsQuery = useProfileReviews(profileId);
    const hideMutation = useHideReview();

    const [hideTarget, setHideTarget] = useState("");
    const [hideReason, setHideReason] = useState("");
    const [actionError, setActionError] = useState("");

    const reviews = reviewsQuery.data?.pages.flatMap((page) => page.items) ?? [];

    function handleSearch(event: React.FormEvent) {
        event.preventDefault();
        setFormError("");

        const trimmed = profileIdInput.trim();
        if (!UUID_PATTERN.test(trimmed)) {
            setFormError("Masukkan ID profil yang sah (format UUID). ID profil bisa disalin dari alamat halaman profil publik.");
            return;
        }

        setHideTarget("");
        setHideReason("");
        setActionError("");
        setProfileId(trimmed);
    }

    async function handleHide(reviewId: string) {
        setActionError("");

        const reason = hideReason.trim();
        if (reason.length < 5) {
            setActionError("Tuliskan alasan penyembunyian, minimal 5 karakter.");
            return;
        }

        try {
            await hideMutation.mutateAsync({ reviewId, reason });
            setHideTarget("");
            setHideReason("");
        } catch (err) {
            setActionError(getProblemMessage(err));
        }
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Moderasi Ulasan</h1>
                <p className="mt-1 text-sm text-slate-500">Tinjau ulasan pada sebuah profil usaha dan sembunyikan yang melanggar. Ulasan tersembunyi tidak tampil dan tidak dihitung reputasi.</p>
            </div>

            <form onSubmit={handleSearch} className="rounded-2xl border border-slate-200 bg-white p-5">
                <label htmlFor="profile_id" className="mb-2 block text-sm font-semibold text-slate-500">
                    ID profil usaha
                </label>

                <div className="flex gap-2">
                    <input id="profile_id" type="text" value={profileIdInput} onChange={(event) => setProfileIdInput(event.target.value)} className={cn(inputClassName, "font-mono text-xs")} placeholder="Contoh: 3f8a2c1e-..." aria-describedby="profile_id_hint" />
                    <button type="submit" className="inline-flex shrink-0 cursor-pointer items-center gap-2 rounded-xl bg-industrial-blue-500 px-4 py-3 text-sm font-semibold text-white transition hover:bg-industrial-blue-600">
                        <LuSearch className="size-4" aria-hidden />
                        Tampilkan
                    </button>
                </div>

                <p id="profile_id_hint" className="mt-2 text-xs text-slate-400">
                    Salin ID dari alamat profil publik, misalnya /profile/<span className="font-mono">ID-profil</span>.
                </p>

                {formError ? (
                    <p className="mt-2 text-xs text-red-600" role="alert">
                        {formError}
                    </p>
                ) : null}
            </form>

            {!profileId ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuSearch className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Masukkan ID profil usaha untuk menampilkan ulasannya.</p>
                </div>
            ) : reviewsQuery.isLoading ? (
                <Loading />
            ) : reviewsQuery.isError ? (
                <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center">
                    <p className="text-sm font-semibold text-red-700">{getProblemMessage(reviewsQuery.error)}</p>
                </div>
            ) : reviews.length === 0 ? (
                <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center">
                    <LuInbox className="size-10 text-slate-300" aria-hidden />
                    <p className="mt-3 text-sm text-slate-500">Profil ini belum punya ulasan yang tampil.</p>
                </div>
            ) : (
                <div className="space-y-4">
                    {reviews.map((review) => (
                        <div key={review.review_id} className="rounded-2xl border border-slate-200 bg-white p-5">
                            <div className="flex flex-wrap items-start justify-between gap-3">
                                <div className="min-w-0">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <p className="text-sm font-bold text-slate-900">{review.author_business_name || "Pengguna"}</p>
                                        <StarRating rating={review.rating} />
                                    </div>

                                    <p className="mt-1 text-xs text-slate-400">
                                        Ulasan {formatDateId(review.created_at)}
                                        {review.transaction_date ? ` · Transaksi ${formatDateId(review.transaction_date)}` : ""}
                                    </p>
                                </div>

                                {hideTarget !== review.review_id ? (
                                    <button
                                        type="button"
                                        onClick={() => {
                                            setHideTarget(review.review_id);
                                            setHideReason("");
                                            setActionError("");
                                        }}
                                        className="inline-flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg border border-red-300 bg-white px-3 py-2 text-xs font-semibold text-red-600 transition hover:bg-red-50"
                                    >
                                        <LuEyeOff className="size-3.5" aria-hidden />
                                        Sembunyikan
                                    </button>
                                ) : null}
                            </div>

                            {review.text ? <p className="mt-3 text-sm leading-6 text-slate-700">{review.text}</p> : <p className="mt-3 text-sm italic text-slate-400">Tanpa teks ulasan.</p>}

                            {hideTarget === review.review_id ? (
                                <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                                    <label htmlFor={`hide-reason-${review.review_id}`} className="block text-sm font-semibold text-slate-500">
                                        Alasan Penyembunyian <span className="text-red-500">*</span>
                                    </label>
                                    <textarea id={`hide-reason-${review.review_id}`} rows={2} value={hideReason} onChange={(event) => setHideReason(event.target.value)} className={inputClassName} placeholder="Misalnya: mengandung ujaran yang melanggar ketentuan" minLength={5} maxLength={1000} />

                                    {actionError ? (
                                        <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                                            {actionError}
                                        </div>
                                    ) : null}

                                    <div className="flex gap-2">
                                        <button type="button" onClick={() => handleHide(review.review_id)} disabled={hideMutation.isPending} className="flex-1 cursor-pointer rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60">
                                            {hideMutation.isPending ? "Menyimpan..." : "Sembunyikan Ulasan"}
                                        </button>
                                        <button type="button" onClick={() => setHideTarget("")} className="cursor-pointer rounded-xl border border-slate-300 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50">
                                            Batal
                                        </button>
                                    </div>
                                </div>
                            ) : null}
                        </div>
                    ))}

                    {reviewsQuery.hasNextPage ? (
                        <div className="text-center">
                            <button type="button" onClick={() => reviewsQuery.fetchNextPage()} disabled={reviewsQuery.isFetchingNextPage} className="cursor-pointer rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60">
                                {reviewsQuery.isFetchingNextPage ? "Memuat..." : "Muat lebih banyak"}
                            </button>
                        </div>
                    ) : null}
                </div>
            )}
        </div>
    );
}
