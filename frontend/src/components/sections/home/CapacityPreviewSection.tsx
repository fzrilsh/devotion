import { cn } from "@lib/utils";
import { FaCheck } from "react-icons/fa6";
import { LuArrowRight } from "react-icons/lu";

const CapacityLists = [
    {
        id: 1,
        name: "Konveksi Maju Bersama",
        city: "Bandung, Jawa Barat",
        productType: "Kaos, Polo Shirt",
        machineType: "Mesin Obras, Mesin Jahit Lurus",
        weeklyCapacity: "2.000 pcs/minggu",
        readyIn: "3-5 hari kerja",
        matchScore: 94,
        verified: true,
    },
    {
        id: 2,
        name: "UD Sejahtera Tekstil",
        city: "Solo, Jawa Tengah",
        productType: "Celana, Rok, Seragam",
        machineType: "Mesin Jahit Lurus, Mesin Kancing",
        weeklyCapacity: "1.500 pcs/minggu",
        readyIn: "7-10 hari kerja",
        matchScore: 87,
        verified: true,
    },
    {
        id: 3,
        name: "Karya Busana Nusantara",
        city: "Surabaya, Jawa Timur",
        productType: "Jaket, Hoodie, Sweater",
        machineType: "Mesin Obras, Mesin Overdeck",
        weeklyCapacity: "800 pcs/minggu",
        readyIn: "5-7 hari kerja",
        matchScore: 79,
        verified: false,
    },
];

export default function CapacityPreviewSection() {
    return (
        <section className="bg-slate-50 h-full flex items-center justify-center scroll-mt-24 relative">
            <svg className="absolute inset-0 w-full h-full opacity-[0.4] pointer-events-none" xmlns="http://www.w3.org/2000/svg">
                <defs>
                    <pattern id="grid" width="64" height="64" patternUnits="userSpaceOnUse">
                        <path className="stroke-slate-200" d="M 64 0 L 0 0 0 64" fill="none" strokeWidth="1" />
                    </pattern>
                </defs>
                <rect width="100%" height="100%" fill="url(#grid)" />
            </svg>

            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-10">
                    <div>
                        <div className="mb-3 flex items-center gap-3">
                            <span className="h-px w-8 bg-industrial-orange-500" />
                            <p className="text-sm sm:text-base font-bold tracking-wide text-industrial-orange-600 uppercase">Kenapa Devotion</p>
                        </div>
                        <h1 className="text-4xl max-w-xl text-deep-navy-900 font-bold leading-tight">Contoh kapasitas yang tersedia</h1>
                    </div>
                    <p className="text-slate-600 text-sm max-w-xs sm:text-right leading-relaxed">Data di bawah adalah contoh demo. Skor kecocokan dihitung berdasarkan kriteria produksi, bukan rating atau jarak.</p>
                </div>

                <div className="flex flex-col gap-4">
                    {CapacityLists.map((list) => {
                        const scoreColor = list.matchScore >= 90 ? "text-emerald-700 bg-emerald-50 ring-1 ring-emerald-200" : list.matchScore >= 80 ? "text-sky-700 bg-sky-50 ring-1 ring-sky-200" : "text-industrial-blue-700 bg-industrial-blue-50 ring-1 ring-industrial-blue-200";

                        return (
                            <div key={list.id} className="bg-white border border-slate-200 rounded-xl p-6 flex flex-col sm:flex-row sm:items-center gap-5 shadow-sm transition-all duration-200 hover:shadow-md hover:border-industrial-orange-300 group">
                                <div className="flex-1 min-w-0">
                                    <div className="flex flex-wrap items-center gap-2 mb-1">
                                        <span className="font-bold text-lg text-deep-navy-900" style={{ fontFamily: "var(--font-display)" }}>
                                            {list.name}
                                        </span>
                                        {list.verified && (
                                            <span className="inline-flex items-center gap-1 text-xs font-bold text-industrial-blue-700 bg-industrial-blue-50 ring-1 ring-industrial-blue-200 px-2 py-0.5 rounded">
                                                <FaCheck className="w-3 h-3" />
                                                Terverifikasi
                                            </span>
                                        )}
                                    </div>
                                    <p className="text-slate-500 text-sm mb-4">{list.city}</p>

                                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
                                        <div>
                                            <p className="text-industrial-blue-600 font-semibold mb-1">Jenis Produk</p>
                                            <p className="text-slate-700">{list.productType}</p>
                                        </div>
                                        <div>
                                            <p className="text-industrial-blue-600 font-semibold mb-1">Mesin</p>
                                            <p className="text-slate-700">{list.machineType}</p>
                                        </div>
                                        <div>
                                            <p className="text-industrial-blue-600 font-semibold mb-1">Kapasitas</p>
                                            <p className="text-slate-700">{list.weeklyCapacity}</p>
                                        </div>
                                        <div>
                                            <p className="text-industrial-blue-600 font-semibold mb-1">Siap dalam</p>
                                            <p className="text-slate-700">{list.readyIn}</p>
                                        </div>
                                    </div>
                                </div>

                                <div className="flex sm:flex-col items-center sm:items-end gap-4 sm:gap-3 shrink-0">
                                    <div className={cn("px-4 py-2 rounded-lg text-center min-w-20", scoreColor)}>
                                        <p className="text-2xl font-bold leading-none">{list.matchScore}</p>
                                        <p className="text-[10px] font-bold mt-1 tracking-wide">SKOR COCOK</p>
                                    </div>

                                    <a className="inline-flex items-center gap-1.5 bg-deep-navy-800 hover:bg-deep-navy-900 text-white text-sm font-semibold px-5 py-2.5 rounded-lg transition-colors whitespace-nowrap focus:outline-none focus-visible:ring-2 focus-visible:ring-industrial-blue-500 focus-visible:ring-offset-2">
                                        Lihat detail
                                        <LuArrowRight className="w-3.5 h-3.5" />
                                    </a>
                                </div>
                            </div>
                        );
                    })}
                </div>

                <div className="mt-10 text-center">
                    <a href="/" className="inline-flex items-center gap-2 text-deep-navy-700 hover:text-industrial-orange-600 text-sm font-semibold transition-colors">
                        Lihat semua kapasitas tersedia
                        <LuArrowRight className="w-4 h-4" />
                    </a>
                </div>
            </div>
        </section>
    );
}
