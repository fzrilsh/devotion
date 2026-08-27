const stats = [
    { value: "50+", label: "Usaha siap ditemukan", sub: "data demo" },
    { value: "12+", label: "Kota dan kabupaten", sub: "data demo" },
    { value: "5+", label: "Jenis produk konveksi", sub: "data demo" },
];

export default function TrustStatsSection() {
    return (
        <section className="bg-industrial-blue-500 h-full flex items-center justify-center scroll-mt-24 relative">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-8 text-center sm:text-left justify-items-center">
                    {stats.map((stat) => (
                        <div key={stat.label} className="flex flex-col sm:flex-row sm:items-baseline gap-1 sm:gap-3">
                            <span className="text-5xl font-extrabold text-white">
                                {stat.value}
                            </span>
                            <div>
                                <p className="text-white/90 font-semibold text-sm">{stat.label}</p>
                                <p className="text-white/50 text-xs">{stat.sub}</p>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </section>
    );
}
