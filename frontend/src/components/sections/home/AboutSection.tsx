import { LuHandshake, LuLayers, LuRefreshCw, LuSearch } from "react-icons/lu";

const aboutBenefit = [
    {
        icon: LuSearch,
        title: "Kapasitas produksi lebih mudah ditemukan",
        description: "Temukan subkontraktor yang sesuai kebutuhan Anda tanpa harus mengandalkan jaringan pribadi atau rekomendasi mulut ke mulut.",
    },
    {
        icon: LuLayers,
        title: "Kriteria kandidat dapat dibandingkan dengan jelas",
        description: "Bandingkan jenis mesin, kapasitas mingguan, dan kesiapan mulai dari beberapa subkontraktor sekaligus dalam satu tampilan.",
    },
    {
        icon: LuRefreshCw,
        title: "Informasi kapasitas lebih aktual",
        description: "Subkontraktor memperbarui ketersediaan kapasitas secara mandiri, sehingga data yang Anda lihat mencerminkan kondisi nyata.",
    },
    {
        icon: LuHandshake,
        title: "Cocok untuk UMKM konveksi",
        description: "Dirancang khusus untuk ekosistem konveksi Indonesia, bukan platform impor yang dipaksakan untuk kebutuhan lokal.",
    },
];

export default function AboutSection() {
    return (
        <section className="bg-deep-navy-500 h-full flex items-center justify-center mt-24 scroll-mt-24">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="mb-8">
                    <p className="text-sm sm:text-base font-bold tracking-wide text-industrial-orange-500 uppercase">Kenapa Devotion</p>
                    <h1 className="text-4xl max-w-xl text-white mt-4 font-bold leading-tight">Lebih dari sekadar direktori konveksi</h1>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
                    {aboutBenefit.map((item, index) => {
                        const Icon = item.icon;
                        return (
                            <div className="group transition-all duration-200 hover:-translate-y-0.5 h-full flex flex-col gap-4" key={index}>
                                <div className="p-4 rounded-xl w-fit bg-industrial-orange-100/20">
                                    <Icon className="text-white group-hover:text-industrial-orange-500 text-xl" />
                                </div>
                                <p className="font-bold text-white">{item.title}</p>
                                <p className="text-sm text-justify text-slate-300 leading-relaxed">{item.description}</p>
                            </div>
                        );
                    })}
                </div>
            </div>
        </section>
    );
}
