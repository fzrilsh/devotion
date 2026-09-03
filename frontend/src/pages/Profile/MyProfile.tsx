import Loading from "@components/common/Loading";
import LocationMap from "@components/common/LocationMap";
import VerificationGate from "@components/common/VerificationGate";
import LocationPicker from "@components/common/LocationPicker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useAccountVerification } from "@hooks/useAccountVerification";
import { useAuth, useUpdateMyRoles } from "@hooks/useAuth";
import { useProfile } from "@hooks/useProfile";
import { useWilayah } from "@hooks/useWilayah";
import { cn } from "@lib/utils";
import { getProblemMessage } from "@lib/problem";
import { profileSchema, type ProfileForm } from "@schemas/profile";
import { useEffect, useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { LuCircleCheck, LuCircleX, LuClock3, LuMail, LuMapPin, LuPencil, LuPhone, LuSearch, LuShieldCheck, LuStar, LuStore, LuX } from "react-icons/lu";
import { Link } from "react-router-dom";

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-800 outline-none transition-all duration-200 placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10";
const labelClassName = "mb-2 block text-sm font-semibold text-slate-500";

const verificationMeta = {
    approved: { label: "Terverifikasi", icon: LuCircleCheck, className: "border-emerald-200 bg-emerald-50 text-emerald-700" },
    pending: { label: "Menunggu Verifikasi", icon: LuClock3, className: "border-amber-200 bg-amber-50 text-amber-700" },
    rejected: { label: "Verifikasi Ditolak", icon: LuCircleX, className: "border-red-200 bg-red-50 text-red-700" },
} as const;

function maskPhone(phone: string): string {
    const digits = phone.replace(/\D/g, "");
    if (digits.length <= 4) return phone;

    return `${"*".repeat(digits.length - 4)}${digits.slice(-4)}`;
}

function AccountCard({ admin }: { admin?: boolean }) {
    const { user } = useAuth();

    const channels = [
        {
            key: "email" as const,
            icon: LuMail,
            title: "Email",
            value: user?.email,
            verified: user?.email_verified ?? false,
            to: "/auth/verify-email",
            state: { email: user?.email },
        },
        {
            key: "phone" as const,
            icon: LuPhone,
            title: "Nomor HP",
            value: user?.phone ? maskPhone(user.phone) : undefined,
            verified: user?.phone_verified ?? false,
            to: "/auth/verify-phone",
            state: { email: user?.email },
        },
    ];

    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-6">
            <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Akun</h3>

            <div className="mt-4 space-y-3">
                {channels.map((channel) => (
                    <div key={channel.key} className="rounded-xl border border-slate-200 bg-white p-4">
                        <div className="flex items-center gap-3">
                            <span className="grid size-10 shrink-0 place-items-center rounded-lg bg-slate-100 text-slate-500">
                                <channel.icon className="size-4.5" aria-hidden />
                            </span>

                            <div className="min-w-0">
                                <p className="text-sm font-bold text-slate-800">{channel.title}</p>
                                <p className="truncate text-xs text-slate-400">{channel.value || "Belum diisi"}</p>
                            </div>
                        </div>

                        <div className="mt-3 border-t border-slate-100 pt-3">
                            {channel.verified ? (
                                <span className="inline-flex items-center gap-1.5 text-xs font-bold text-emerald-700">
                                    <LuCircleCheck className="size-3.5 shrink-0" aria-hidden />
                                    Terverifikasi
                                </span>
                            ) : (
                                <Link to={channel.to} state={channel.state} className="inline-flex w-full items-center justify-center rounded-lg bg-industrial-blue-500 px-4 py-2 text-xs font-bold text-white transition hover:bg-industrial-blue-600">
                                    Verifikasi
                                </Link>
                            )}
                        </div>
                    </div>
                ))}
            </div>

            {admin ? <p className="mt-4 text-xs leading-5 text-slate-400">Akun admin tidak memiliki profil usaha. Verifikasi email dan nomor HP tetap diperlukan untuk keamanan akun.</p> : null}
        </div>
    );
}

const roleOptions = [
    {
        key: "buyer" as const,
        icon: LuSearch,
        title: "Pemberi order",
        description: "Mencari subkontraktor dan mengirim request kuota ketika order melebihi kapasitas sendiri.",
    },
    {
        key: "subcontractor" as const,
        icon: LuStore,
        title: "Subkontraktor",
        description: "Menayangkan kapasitas produksi yang menganggur dan menerima request kuota dari pemberi order.",
    },
];

function RolesCard() {
    const { user } = useAuth();
    const updateRoles = useUpdateMyRoles();

    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    const roles = { buyer: Boolean(user?.roles?.buyer), subcontractor: Boolean(user?.roles?.subcontractor) };
    const activeCount = Number(roles.buyer) + Number(roles.subcontractor);

    async function toggleRole(key: "buyer" | "subcontractor") {
        setError("");
        setSuccess("");

        const next = { ...roles, [key]: !roles[key] };

        if (!next.buyer && !next.subcontractor) {
            setError("Akun usaha harus memiliki setidaknya satu peran.");
            return;
        }

        try {
            await updateRoles.mutateAsync(next);
            setSuccess(next[key] ? `Peran ${key === "buyer" ? "pemberi order" : "subkontraktor"} ditambahkan.` : `Peran ${key === "buyer" ? "pemberi order" : "subkontraktor"} dicabut.`);
        } catch (err) {
            setError(getProblemMessage(err, "Terjadi kesalahan. Silakan coba lagi.", { 401: "Sesi Anda habis, silakan masuk kembali.", 403: "Anda tidak berwenang mengubah profil ini." }));
        }
    }

    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-6">
            <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Peran Usaha</h3>
            <p className="mt-2 text-xs leading-5 text-slate-500">Satu akun dapat menjalankan kedua peran sekaligus. Menu navigasi mengikuti peran yang aktif.</p>

            {error ? (
                <div className="mt-3 rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700" role="alert" aria-live="polite">
                    {error}
                </div>
            ) : null}

            {success ? (
                <div className="mt-3 rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-700" role="status" aria-live="polite">
                    {success}
                </div>
            ) : null}

            <div className="mt-4 space-y-3">
                {roleOptions.map((option) => {
                    const active = roles[option.key];
                    const lastRole = active && activeCount === 1;

                    return (
                        <div key={option.key} className={cn("rounded-xl border p-4", active ? "border-industrial-blue-500/30 bg-industrial-blue-500/5" : "border-slate-200 bg-white")}>
                            <div className="flex items-start gap-3">
                                <span className={cn("grid size-10 shrink-0 place-items-center rounded-lg", active ? "bg-industrial-blue-500/10 text-industrial-blue-600" : "bg-slate-100 text-slate-400")}>
                                    <option.icon className="size-4.5" aria-hidden />
                                </span>

                                <div className="min-w-0">
                                    <p className="text-sm font-bold text-slate-800">{option.title}</p>
                                    <p className="mt-0.5 text-xs leading-5 text-slate-500">{option.description}</p>
                                </div>
                            </div>

                            <div className={cn("mt-3 border-t pt-3", active ? "border-industrial-blue-500/15" : "border-slate-100")}>
                                <button
                                    type="button"
                                    onClick={() => toggleRole(option.key)}
                                    disabled={updateRoles.isPending || lastRole}
                                    aria-pressed={active}
                                    title={lastRole ? "Peran terakhir tidak dapat dicabut." : undefined}
                                    className={cn(
                                        "inline-flex w-full cursor-pointer items-center justify-center rounded-lg px-4 py-2 text-xs font-bold transition disabled:cursor-not-allowed disabled:opacity-60",
                                        active ? "border border-slate-300 bg-white text-slate-600 hover:bg-slate-50" : "bg-industrial-blue-500 text-white hover:bg-industrial-blue-600",
                                    )}
                                >
                                    {updateRoles.isPending ? "Memproses..." : active ? "Cabut" : "Aktifkan"}
                                </button>

                                {lastRole ? <p className="mt-2 text-xs leading-5 text-slate-400">Peran terakhir tidak dapat dicabut.</p> : null}
                            </div>
                        </div>
                    );
                })}
            </div>

            <p className="mt-4 text-xs leading-5 text-slate-400">Peran tidak dapat dicabut selama masih ada pesanan aktif pada peran tersebut.</p>
        </div>
    );
}

function AdminProfile() {
    const { user } = useAuth();

    return (
        <div className="mx-auto space-y-6">
            <div>
                <h1 className="text-xl font-bold text-slate-900">Profil Akun</h1>
                <p className="mt-1 text-sm text-slate-500">Informasi akun administrator Anda.</p>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex items-center gap-4">
                        <span className="grid size-14 shrink-0 place-items-center rounded-2xl bg-linear-to-br from-industrial-blue-500 to-deep-navy-500 text-xl font-extrabold text-white shadow-lg">{(user?.email?.charAt(0) ?? "A").toUpperCase()}</span>

                        <div className="min-w-0">
                            <h2 className="truncate text-lg font-extrabold tracking-tight text-slate-900">{user?.email}</h2>
                            <p className="mt-0.5 flex items-center gap-1.5 text-sm text-slate-500">
                                <LuPhone className="size-4 shrink-0 text-slate-400" aria-hidden />
                                {user?.phone || "Nomor HP belum diisi"}
                            </p>
                        </div>
                    </div>

                    <span className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-industrial-blue-500/20 bg-industrial-blue-500/10 px-3 py-1.5 text-xs font-bold text-industrial-blue-600">
                        <LuShieldCheck className="size-3.5" aria-hidden />
                        Administrator
                    </span>
                </div>
            </div>

            <AccountCard admin />
        </div>
    );
}

export default function MyProfile() {
    const { user, isLoading: authLoading } = useAuth();
    const { needsVerification } = useAccountVerification();

    const { profile, isLoading, updateProfile, updatePending } = useProfile();
    const [editMode, setEditMode] = useState(false);

    const profileValues = useMemo<ProfileForm>(
        () => ({
            business_name: profile?.business_name || "",
            description: profile?.description || "",
            province_code: profile?.province_code || "",
            city_code: profile?.city_code || "",
            latitude: profile?.latitude ?? null,
            longitude: profile?.longitude ?? null,
        }),
        [profile],
    );

    const {
        register,
        handleSubmit,
        setError,
        setValue,
        control,
        reset,
        formState: { errors },
    } = useForm<ProfileForm>({
        resolver: zodResolver(profileSchema),
        values: profileValues,
    });

    const provinceCode = useWatch({ control, name: "province_code" }) ?? "";
    const { provinces, cities } = useWilayah(provinceCode);

    async function onSubmit(values: ProfileForm) {
        try {
            await updateProfile({
                business_name: values.business_name,
                description: values.description,
                city_code: values.city_code,
                latitude: values.latitude ?? null,
                longitude: values.longitude ?? null,
            });
            setEditMode(false);
        } catch (error) {
            setError("root", { message: getProblemMessage(error, "Terjadi kesalahan. Silakan coba lagi.", { 401: "Sesi Anda habis, silakan masuk kembali.", 403: "Anda tidak berwenang mengubah profil ini." }) });
        }
    }

    const watchedLatitude = useWatch({ control, name: "latitude" });
    const watchedLongitude = useWatch({ control, name: "longitude" });

    const resolveLocationRegion = useMemo(
        () => async (latitude: number, longitude: number) => {
            try {
                const response = await fetch(`https://nominatim.openstreetmap.org/reverse?format=jsonv2&lat=${latitude}&lon=${longitude}&zoom=10&accept-language=id`, {
                    headers: { Accept: "application/json" },
                });

                if (!response.ok) return;

                const payload = (await response.json()) as { address?: { city?: string; town?: string; village?: string; county?: string; state?: string; state_district?: string; country?: string; country_code?: string } };
                const address = payload.address ?? {};
                const cityName = address.city ?? address.town ?? address.village ?? address.county ?? "";
                const provinceName = address.state ?? address.state_district ?? "";

                if (!cityName && !provinceName) return;

                const cityMatch = cities.find((city) => city.name.toLowerCase() === cityName.toLowerCase());
                const provinceMatch = provinces.find((province) => province.name.toLowerCase() === provinceName.toLowerCase());

                if (provinceMatch) {
                    setValue("province_code", provinceMatch.code, { shouldDirty: true });
                }

                if (cityMatch) {
                    setValue("city_code", cityMatch.code, { shouldDirty: true });
                }
            } catch {
                // Ignore reverse-geocoding failures; the user can still keep the manually selected region.
            }
        },
        [cities, provinces, setValue],
    );

    function handleLocationChange(latitude: number, longitude: number) {
        setValue("latitude", latitude, { shouldDirty: true });
        setValue("longitude", longitude, { shouldDirty: true });
    }

    function handleClearLocation() {
        setValue("latitude", null, { shouldDirty: true });
        setValue("longitude", null, { shouldDirty: true });
    }

    useEffect(() => {
        if (watchedLatitude == null || watchedLongitude == null) return;

        void resolveLocationRegion(watchedLatitude, watchedLongitude);
    }, [resolveLocationRegion, watchedLatitude, watchedLongitude]);

    if (authLoading) return <Loading />;

    if (user?.is_admin) return <AdminProfile />;

    if (isLoading) return <Loading />;

    const status = profile?.verification_status && profile.verification_status in verificationMeta ? verificationMeta[profile.verification_status as keyof typeof verificationMeta] : null;
    const StatusIcon = status?.icon ?? LuClock3;
    const location = [profile?.city_name, profile?.province_name].filter(Boolean).join(", ");

    return (
        <div className="mx-auto space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-xl font-bold text-slate-900">Profil Usaha</h1>

                {!editMode && !needsVerification ? (
                    <button type="button" onClick={() => setEditMode(true)} className="inline-flex cursor-pointer items-center gap-2 rounded-xl bg-industrial-blue-500 px-5 py-2.5 text-sm font-semibold text-white transition-all duration-200 hover:bg-industrial-blue-600">
                        <LuPencil className="size-4" aria-hidden />
                        Edit Profil
                    </button>
                ) : null}
            </div>

            <div className="relative overflow-hidden rounded-2xl border border-slate-200 bg-white">
                <div className="relative flex flex-col gap-4 p-6 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex items-center gap-4">
                        <div className="pb-1">
                            <h2 className="text-lg font-extrabold tracking-tight text-slate-900">{profile?.business_name || "Nama usaha belum diisi"}</h2>

                            <p className="mt-0.5 flex items-center gap-1.5 text-sm text-slate-500">
                                <LuMapPin className="size-4 shrink-0 text-slate-400" aria-hidden />
                                {location || "Lokasi belum diisi"}
                            </p>
                        </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-2 pb-1">
                        <span className={cn("inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-bold", status ? status.className : "border-slate-200 bg-slate-50 text-slate-500")}>
                            <StatusIcon className="size-3.5" aria-hidden />
                            {status ? status.label : "Verifikasi belum diajukan"}
                        </span>

                        {profile?.identity_verified ? (
                            <span className="inline-flex items-center gap-1.5 rounded-full border border-industrial-blue-500/20 bg-industrial-blue-500/10 px-3 py-1.5 text-xs font-bold text-industrial-blue-600">
                                <LuShieldCheck className="size-3.5" aria-hidden />
                                Identitas terverifikasi
                            </span>
                        ) : null}
                    </div>

                    {!editMode && status?.label !== "Terverifikasi" ? (
                        <div className="border-t border-slate-100 bg-slate-50/60 px-6 py-4 sm:flex sm:items-center sm:justify-between sm:gap-4">
                            <p className="text-sm text-slate-500">Dapatkan lencana terverifikasi pada profil publik dan hasil pencarian Anda.</p>

                            <Link
                                to="/verification"
                                className="mt-3 inline-flex shrink-0 items-center gap-2 rounded-xl bg-industrial-blue-500 px-4 py-2.5 text-sm font-semibold text-white transition-all duration-200 hover:bg-industrial-blue-600 sm:mt-0"
                            >
                                <LuShieldCheck className="size-4" aria-hidden />
                                {status?.label === "Menunggu Verifikasi" ? "Lihat Status Pengajuan" : status?.label === "Verifikasi Ditolak" ? "Ajukan Ulang Verifikasi" : "Ajukan Verifikasi"}
                            </Link>
                        </div>
                    ) : null}
                </div>
            </div>

            {profile?.verification_status === "rejected" ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">
                    Verifikasi identitas Anda ditolak. Periksa alasan penolakan dan{" "}
                    <Link to="/verification" className="font-bold text-red-800 underline underline-offset-2 hover:text-red-900">
                        ajukan ulang di halaman Verifikasi
                    </Link>
                    .
                </div>
            ) : null}

            {editMode && needsVerification ? (
                <VerificationGate action="mengubah profil usaha" />
            ) : editMode ? (
                <div className="rounded-2xl border border-slate-200 bg-white p-6">
                    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
                        {errors.root?.message && (
                            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">
                                {errors.root.message}
                            </div>
                        )}

                        <div>
                            <label htmlFor="business_name" className={labelClassName}>
                                Nama Usaha <span className="text-red-500">*</span>
                            </label>
                            <input id="business_name" type="text" className={cn(inputClassName, errors.business_name && "border-red-400 focus:border-red-500")} {...register("business_name")} />
                            {errors.business_name && <p className="mt-1 text-sm text-red-600">{errors.business_name.message}</p>}
                        </div>

                        <div>
                            <label htmlFor="description" className={labelClassName}>
                                Deskripsi
                            </label>
                            <textarea id="description" rows={4} className={cn(inputClassName, errors.description && "border-red-400 focus:border-red-500")} {...register("description")} />
                            {errors.description && <p className="mt-1 text-sm text-red-600">{errors.description.message}</p>}
                        </div>

                        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <div>
                                <label htmlFor="province_code" className={labelClassName}>
                                    Provinsi
                                </label>
                                <select id="province_code" className={cn(inputClassName, errors.province_code && "border-red-400 focus:border-red-500")} {...register("province_code")}>
                                    <option value="">Pilih provinsi</option>
                                    {provinces.map((prov) => (
                                        <option key={prov.code} value={prov.code}>
                                            {prov.name}
                                        </option>
                                    ))}
                                </select>
                                {errors.province_code && <p className="mt-1 text-sm text-red-600">{errors.province_code.message}</p>}
                            </div>
                            <div>
                                <label htmlFor="city_code" className={labelClassName}>
                                    Kota/Kabupaten
                                </label>
                                <select id="city_code" disabled={!provinceCode || cities.length === 0} className={cn(inputClassName, errors.city_code && "border-red-400 focus:border-red-500")} {...register("city_code")}>
                                    <option value="">Pilih kota</option>
                                    {cities.map((city) => (
                                        <option key={city.code} value={city.code}>
                                            {city.name}
                                        </option>
                                    ))}
                                </select>
                                {errors.city_code && <p className="mt-1 text-sm text-red-600">{errors.city_code.message}</p>}
                            </div>
                        </div>

                        <div>
                            <div className="mb-2 flex items-center justify-between">
                                <label className="text-sm font-semibold text-slate-500">Titik Lokasi Usaha (opsional)</label>

                                {watchedLatitude != null && watchedLongitude != null ? (
                                    <button type="button" onClick={handleClearLocation} className="cursor-pointer text-xs font-semibold text-red-500 transition-colors hover:text-red-600">
                                        Hapus titik
                                    </button>
                                ) : null}
                            </div>

                            <LocationPicker latitude={watchedLatitude ?? null} longitude={watchedLongitude ?? null} onChange={handleLocationChange} onLocationSelected={async (latitude, longitude) => {
                                await resolveLocationRegion(latitude, longitude);
                            }} disabled={updatePending} />

                            <p className="mt-1.5 text-xs leading-5 text-slate-400">
                                {watchedLatitude != null && watchedLongitude != null ? `Koordinat terpilih: ${watchedLatitude}, ${watchedLongitude}. Klik peta untuk memindahkan titik.` : "Klik pada peta untuk menandai lokasi usaha Anda. Titik ini membantu pemberi order memperkirakan jarak."}
                            </p>

                            {errors.latitude && <p className="mt-1 text-sm text-red-600">{errors.latitude.message}</p>}
                            {errors.longitude && <p className="mt-1 text-sm text-red-600">{errors.longitude.message}</p>}
                        </div>

                        <div className="flex gap-4 pt-2">
                            <button type="submit" disabled={updatePending} className="flex-1 cursor-pointer rounded-xl bg-industrial-blue-500 px-6 py-3 font-semibold text-white transition hover:bg-industrial-blue-600 disabled:cursor-not-allowed disabled:opacity-70">
                                {updatePending ? "Menyimpan..." : "Simpan Perubahan"}
                            </button>
                            <button
                                type="button"
                                onClick={() => {
                                    setEditMode(false);
                                    reset();
                                }}
                                className="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 px-6 py-3 font-semibold text-slate-600 transition hover:bg-slate-50"
                            >
                                <LuX className="size-4" aria-hidden />
                                Batal
                            </button>
                        </div>
                    </form>
                </div>
            ) : (
                <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                    <div className="rounded-2xl border border-slate-200 bg-white p-6 lg:col-span-2">
                        <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Tentang Usaha</h3>

                        <p className="mt-3 whitespace-pre-line text-sm leading-6 text-slate-700">{profile?.description || "Belum ada deskripsi. Ceritakan keahlian, jenis produk, dan kapasitas produksi usaha Anda agar calon mitra lebih percaya."}</p>

                        <div className="mt-6 grid grid-cols-1 gap-4 border-t border-slate-100 pt-6 sm:grid-cols-2">
                            <div>
                                <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">Kota / Kabupaten</p>
                                <p className="mt-1 text-sm font-semibold text-slate-800">{profile?.city_name || "-"}</p>
                            </div>

                            <div>
                                <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">Provinsi</p>
                                <p className="mt-1 text-sm font-semibold text-slate-800">{profile?.province_name || "-"}</p>
                            </div>

                            <div className="col-span-2">
                                {profile?.latitude != null && profile?.longitude != null ? (
                                    <LocationMap latitude={profile.latitude} longitude={profile.longitude} label={profile.business_name} />
                                ) : (
                                    <div className="flex h-80 items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-slate-50 px-4 text-center text-sm text-slate-500">
                                        Lokasi usaha belum ditandai. Tambahkan titik lokasi pada form edit profil agar calon mitra dapat melihat area usaha Anda.
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    <div className="space-y-6">
                        <AccountCard />

                        <RolesCard />

                        <div className="rounded-2xl border border-slate-200 bg-white p-6">
                            <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Reputasi</h3>

                            {profile?.reputation ? (
                                <div className="mt-4 space-y-4">
                                    <div className="flex items-center gap-3">
                                        <span className="grid size-11 place-items-center rounded-xl bg-amber-50 text-amber-500">
                                            <LuStar className="size-5" aria-hidden />
                                        </span>

                                        <div>
                                            <p className="text-lg font-extrabold text-slate-900">{profile.reputation.average_rating != null ? Number(profile.reputation.average_rating).toFixed(1) : "-"}</p>
                                            <p className="text-xs text-slate-400">{profile.reputation.review_count ?? 0} ulasan</p>
                                        </div>
                                    </div>

                                    <div className="rounded-xl bg-slate-50 p-3">
                                        <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">Tingkat penyelesaian</p>

                                        {profile.reputation.enough_data && profile.reputation.completion_rate != null ? (
                                            <p className="mt-1 text-sm font-bold text-slate-800">{Math.round(profile.reputation.completion_rate)}%</p>
                                        ) : (
                                            <p className="mt-1 text-sm text-slate-500">Belum cukup data untuk menampilkan tingkat penyelesaian.</p>
                                        )}
                                    </div>
                                </div>
                            ) : (
                                <p className="mt-3 text-sm text-slate-500">Belum ada reputasi. Selesaikan pesanan pertama Anda untuk mulai membangun reputasi.</p>
                            )}
                        </div>

                        <div className="rounded-2xl border border-industrial-blue-500/20 bg-industrial-blue-500/5 p-6">
                            <div className="flex items-center gap-3">
                                <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-white text-industrial-blue-500 shadow-sm">
                                    <LuShieldCheck className="size-5" aria-hidden />
                                </span>

                                <h3 className="text-sm font-bold text-slate-800">Profil yang lengkap lebih dipercaya</h3>
                            </div>

                            <p className="mt-3 text-xs leading-5 text-slate-500">Isi deskripsi dan lokasi usaha Anda dengan lengkap. Mitra potensial melihat profil ini sebelum mengirim request kuota.</p>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
