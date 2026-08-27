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
                    <p className="text-sm sm:text-base font-bold tracking-wide text-white uppercase">Kenapa Devotion</p>
                    <h1 className="text-4xl max-w-xl text-white mt-4 font-bold leading-tight">Lebih dari sekadar direktori konveksi</h1>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-x-8 gap-y-12">
                    {aboutBenefit.map((item, index) => {
                        const Icon = item.icon;
                        return (
                            <div
                                key={index}
                                className="group relative h-full flex flex-col gap-4 rounded-2xl border border-white/10 bg-white p-6 transition-all duration-300 ease-out hover:-translate-y-1 hover:border-industrial-blue-500/40 hover:shadow-[0_20px_40px_-15px_rgba(217,119,87,0.15)]"
                            >
                                <div className="relative w-fit">
                                    <div className="p-4 rounded-xl w-fit bg-industrial-blue-500/10 border border-industrial-blue-500/10 transition-colors duration-300 group-hover:border-industrial-blue-500/30">
                                        <Icon className="text-deep-navy-500/80 text-xl transition-colors duration-300 group-hover:text-industrial-blue-500" />
                                    </div>
                                </div>

                                <p className="font-bold text-deep-navy-500 leading-snug text-xl">{item.title}</p>
                                <span className="h-px w-6 bg-deep-navy-500/50 transition-all duration-300 group-hover:w-10 group-hover:bg-industrial-blue-500/50" />
                                <p className="text-sm text-slate-500 leading-relaxed">{item.description}</p>
                            </div>
                        );
                    })}
                </div>
            </div>
        </section>
    );
}
