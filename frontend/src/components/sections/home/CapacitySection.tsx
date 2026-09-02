import { motion } from "motion/react";
import { LuArrowRight, LuBadgeCheck, LuCalendarClock, LuCog, LuMapPin, LuPackage, LuShirt, LuStar } from "react-icons/lu";
import { Link } from "react-router-dom";

const demoCapacities = [
    {
        id: "demo-1",
        businessName: "Konveksi Maju Bersama",
        city: "Bandung, Jawa Barat",
        products: "Kaos, Polo Shirt",
        machines: "Mesin Obras, Mesin Jahit Lurus",
        weeklyCapacity: 2000,
        readyInDays: 4,
        rating: 4.9,
        reviewCount: 32,
        completionRate: 98,
        verified: true,
    },
    {
        id: "demo-2",
        businessName: "UD Sejahtera Tekstil",
        city: "Solo, Jawa Tengah",
        products: "Celana, Rok, Seragam",
        machines: "Mesin Jahit Lurus, Mesin Kancing",
        weeklyCapacity: 1500,
        readyInDays: 7,
        rating: 4.7,
        reviewCount: 21,
        completionRate: 95,
        verified: true,
    },
    {
        id: "demo-3",
        businessName: "Karya Busana Nusantara",
        city: "Surabaya, Jawa Timur",
        products: "Jaket, Hoodie, Sweater",
        machines: "Mesin Obras, Mesin Overdeck",
        weeklyCapacity: 800,
        readyInDays: 6,
        rating: 4.5,
        reviewCount: 14,
        completionRate: 92,
        verified: false,
    },
];

const container = {
    hidden: {},
    show: { transition: { staggerChildren: 0.1 } },
};

const item = {
    hidden: { opacity: 0, y: 24 },
    show: { opacity: 1, y: 0, transition: { duration: 0.5, ease: "easeOut" as const } },
};

export default function CapacitySection() {
    return (
        <section id="kapasitas" className="relative flex h-full scroll-mt-24 items-center justify-center bg-slate-50">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="mx-auto mb-12 max-w-2xl text-center">
                    <p className="text-sm font-bold uppercase tracking-widest text-industrial-blue-500">Kapasitas Terbaik</p>
                    <h2 className="mt-3 text-3xl font-extrabold leading-tight tracking-tight text-deep-navy-900 sm:text-4xl">Mitra paling populer minggu ini</h2>
                    <p className="mt-4 text-sm leading-relaxed text-slate-600 sm:text-base">Contoh tampilan mitra dengan rating dan jumlah ulasan tertinggi. Data pada halaman ini bersifat demo.</p>
                </div>

                <motion.div variants={container} initial="hidden" whileInView="show" viewport={{ once: true, margin: "-60px" }} className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
                    {demoCapacities.map((capacity) => (
                        <motion.article
                            key={capacity.id}
                            variants={item}
                            className="group relative flex flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm transition-all duration-300 hover:-translate-y-1 hover:border-industrial-blue-300 hover:shadow-xl hover:shadow-slate-200"
                        >
                            <div className="flex flex-1 flex-col p-6">
                                <div className="mb-1 flex items-start justify-between gap-3">
                                    <h3 className="text-base font-bold leading-snug text-deep-navy-900">{capacity.businessName}</h3>
                                    <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-bold text-amber-700 ring-1 ring-amber-200">
                                        <LuStar className="size-3.5 fill-amber-400 text-amber-400" aria-hidden />
                                        {capacity.rating.toFixed(1)}
                                    </span>
                                </div>

                                <div className="mb-5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                                    <span className="inline-flex items-center gap-1">
                                        <LuMapPin className="size-3.5" aria-hidden />
                                        {capacity.city}
                                    </span>
                                    {capacity.verified && (
                                        <span className="inline-flex items-center gap-1 font-semibold text-industrial-blue-600">
                                            <LuBadgeCheck className="size-3.5" aria-hidden />
                                            Terverifikasi
                                        </span>
                                    )}
                                </div>

                                <dl className="mb-5 space-y-3 border-t border-slate-100 pt-4 text-sm">
                                    <div className="flex items-start gap-2.5">
                                        <LuShirt className="mt-0.5 size-4 shrink-0 text-industrial-blue-500" aria-hidden />
                                        <div className="min-w-0">
                                            <dt className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Jenis produk</dt>
                                            <dd className="truncate text-slate-700" title={capacity.products}>
                                                {capacity.products}
                                            </dd>
                                        </div>
                                    </div>
                                    <div className="flex items-start gap-2.5">
                                        <LuCog className="mt-0.5 size-4 shrink-0 text-industrial-blue-500" aria-hidden />
                                        <div className="min-w-0">
                                            <dt className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Mesin</dt>
                                            <dd className="truncate text-slate-700" title={capacity.machines}>
                                                {capacity.machines}
                                            </dd>
                                        </div>
                                    </div>
                                    <div className="flex items-start gap-2.5">
                                        <LuPackage className="mt-0.5 size-4 shrink-0 text-industrial-blue-500" aria-hidden />
                                        <div>
                                            <dt className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Kapasitas</dt>
                                            <dd className="text-slate-700">{capacity.weeklyCapacity.toLocaleString("id-ID")} pcs/minggu</dd>
                                        </div>
                                    </div>
                                    <div className="flex items-start gap-2.5">
                                        <LuCalendarClock className="mt-0.5 size-4 shrink-0 text-industrial-blue-500" aria-hidden />
                                        <div>
                                            <dt className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Siap dalam</dt>
                                            <dd className="text-slate-700">{capacity.readyInDays} hari</dd>
                                        </div>
                                    </div>
                                </dl>

                                <div className="mt-auto flex items-center justify-between gap-3 border-t border-slate-100 pt-4">
                                    <p className="text-xs text-slate-500">
                                        <span className="font-semibold text-slate-700">{capacity.reviewCount} ulasan</span>
                                        {" · "}
                                        <span className={capacity.completionRate >= 95 ? "font-semibold text-emerald-600" : "font-semibold text-amber-600"}>{capacity.completionRate}% selesai</span>
                                    </p>
                                </div>
                            </div>
                        </motion.article>
                    ))}
                </motion.div>

                <div className="mt-12 text-center">
                    <Link to="/search" className="inline-flex items-center gap-2 text-sm font-semibold text-deep-navy-700 transition-colors hover:text-industrial-blue-500">
                        Lihat semua kapasitas tersedia
                        <LuArrowRight className="size-4" aria-hidden />
                    </Link>
                </div>
            </div>
        </section>
    );
}
