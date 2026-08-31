import { ApiError } from "@api/client";
import HomeLayout from "@components/layout/HomeLayout";
import Loading from "@components/common/Loading";
import { usePublicProfile } from "@hooks/useProfile";
import { LuBadgeCheck, LuMapPin, LuShieldCheck, LuStar, LuWrench } from "react-icons/lu";
import { Link, useParams } from "react-router-dom";

export default function PublicProfile() {
    const { profileId } = useParams<{ profileId: string }>();
    const { data: profile, isLoading, error } = usePublicProfile(profileId ?? "");

    if (isLoading) {
        return (
            <HomeLayout>
                <Loading />
            </HomeLayout>
        );
    }

    if (error || !profile) {
        const notFound = error instanceof ApiError && error.status === 404;

        return (
            <HomeLayout>
                <div className="mx-auto max-w-3xl px-5 py-20 text-center">
                    <h1 className="text-2xl font-extrabold tracking-tight text-slate-900">{notFound ? "Profil tidak ditemukan" : "Profil tidak dapat dimuat"}</h1>
                    <p className="mt-2 text-sm text-slate-500">{notFound ? "Profil usaha yang Anda cari tidak tersedia atau sudah tidak aktif." : "Terjadi kesalahan saat memuat profil. Silakan coba lagi."}</p>
                    <Link to="/" className="mt-6 inline-block rounded-xl bg-industrial-blue-500 px-6 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-industrial-blue-600">
                        Kembali ke Beranda
                    </Link>
                </div>
            </HomeLayout>
        );
    }

    const location = [profile.city_name, profile.province_name].filter(Boolean).join(", ");
    const reputation = profile.reputation;
    const listing = profile.listing;

    return (
        <HomeLayout>
            <div className="bg-slate-50 mt-24 min-h-screen h-full">
                <div className="mx-auto max-w-4xl px-5 py-10">
                    {/* Kepala profil */}
                    <div className="rounded-xl border border-slate-200 bg-white p-6 sm:p-8">
                        <div className="flex flex-col gap-6 sm:flex-row sm:items-start">
                            <span className="grid size-16 shrink-0 place-items-center rounded-2xl bg-linear-to-tl from-deep-navy-500 to-deep-navy-800 text-2xl font-extrabold text-white">{profile.business_name.charAt(0).toUpperCase()}</span>

                            <div className="min-w-0 flex-1">
                                <div className="flex flex-wrap items-center gap-2">
                                    <h1 className="text-2xl font-extrabold tracking-tight text-slate-900">{profile.business_name}</h1>

                                    {profile.identity_verified ? (
                                        <span className="inline-flex items-center gap-1 rounded-full bg-industrial-blue-500/10 px-2.5 py-1 text-xs font-semibold text-industrial-blue-500">
                                            <LuBadgeCheck className="size-3.5" aria-hidden />
                                            Terverifikasi
                                        </span>
                                    ) : null}
                                </div>

                                {location ? (
                                    <p className="mt-1.5 inline-flex items-center gap-1.5 text-sm text-slate-500">
                                        <LuMapPin className="size-4" aria-hidden />
                                        {location}
                                    </p>
                                ) : null}

                                {profile.description ? <p className="mt-4 whitespace-pre-line text-sm leading-6 text-slate-600">{profile.description}</p> : <p className="mt-4 text-sm text-slate-400">Belum ada deskripsi usaha.</p>}
                            </div>
                        </div>
                    </div>

                    {/* Reputasi */}
                    <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
                        <div className="rounded-xl border border-slate-200 bg-white p-5">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Tingkat Penyelesaian</p>
                            {reputation?.enough_data && reputation.completion_rate != null ? <p className="mt-2 text-2xl font-extrabold text-slate-900">{Math.round(reputation.completion_rate)}%</p> : <p className="mt-2 text-sm font-semibold text-slate-500">Belum cukup data</p>}
                        </div>

                        <div className="rounded-xl border border-slate-200 bg-white p-5">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Rating Rata-rata</p>
                            {reputation && reputation.review_count > 0 && reputation.average_rating != null ? (
                                <p className="mt-2 inline-flex items-center gap-1.5 text-2xl font-extrabold text-slate-900">
                                    <LuStar className="size-5 text-industrial-blue-500" aria-hidden />
                                    {reputation.average_rating.toFixed(1)}
                                </p>
                            ) : (
                                <p className="mt-2 text-sm font-semibold text-slate-500">Belum ada rating</p>
                            )}
                        </div>

                        <div className="rounded-xl border border-slate-200 bg-white p-5">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Ulasan</p>
                            <p className="mt-2 text-2xl font-extrabold text-slate-900">{reputation?.review_count ?? 0}</p>
                        </div>
                    </div>

                    {/* Listing kapasitas */}
                    {listing ? (
                        <div className="mt-6 rounded-xl border border-slate-200 bg-white p-6 sm:p-8">
                            <h2 className="text-lg font-bold text-slate-900">Kapasitas Produksi</h2>

                            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                                <div className="rounded-xl bg-slate-50 p-4">
                                    <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Kapasitas Mingguan</p>
                                    <p className="mt-1.5 text-lg font-bold text-slate-900">{listing.weekly_capacity.toLocaleString("id-ID")} unit</p>
                                </div>

                                <div className="rounded-xl bg-slate-50 p-4">
                                    <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Waktu Persiapan</p>
                                    <p className="mt-1.5 text-lg font-bold text-slate-900">{listing.readiness_lead_days} hari</p>
                                </div>
                            </div>

                            {listing.product_items.length > 0 ? (
                                <div className="mt-5">
                                    <p className="text-sm font-semibold text-slate-700">Produk yang dikerjakan</p>
                                    <div className="mt-2 flex flex-wrap gap-2">
                                        {listing.product_items.map((item) => (
                                            <span key={item.item_id} className="rounded-full bg-industrial-blue-500/10 px-3 py-1 text-xs font-semibold text-industrial-blue-500">
                                                {item.name}
                                            </span>
                                        ))}
                                    </div>
                                </div>
                            ) : null}

                            {listing.machines.length > 0 ? (
                                <div className="mt-5">
                                    <p className="text-sm font-semibold text-slate-700">Mesin</p>
                                    <ul className="mt-2 space-y-1.5">
                                        {listing.machines.map(({ item, machine_count }) => (
                                            <li key={item.item_id} className="inline-flex items-center gap-2 text-sm text-slate-600">
                                                <LuWrench className="size-4 text-slate-400" aria-hidden />
                                                {item.name} ({machine_count} unit)
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            ) : null}
                        </div>
                    ) : null}

                    {/* Catatan kepercayaan */}
                    <div className="mt-6 flex items-start gap-3 rounded-xl border border-slate-200 bg-white p-4">
                        <LuShieldCheck className="mt-0.5 size-5 shrink-0 text-industrial-blue-500" aria-hidden />
                        <p className="text-xs leading-5 text-slate-500">Transaksi dan pembayaran dilakukan langsung antara Anda dan pemilik usaha. Devotion hanya mempertemukan kedua pihak dan tidak menahan dana pihak mana pun.</p>
                    </div>
                </div>
            </div>
        </HomeLayout>
    );
}
