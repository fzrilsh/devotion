import { LuBadgeCheck, LuMap, LuShieldCheck, LuZap } from "react-icons/lu";

const aboutBenefit = [
    {
        icon: LuShieldCheck,
        title: "Mitra Terverifikasi",
        description: "Mitra telah melalui verifikasi dokumen dan kapasitas produksi.",
    },
    {
        icon: LuZap,
        title: "Respons Cepat",
        description: "Rata-rata konfirmasi mitra dalam waktu 4 jam kerja.",
    },
    {
        icon: LuMap,
        title: "Jaringan Luas",
        description: "Lebih dari 1.200 mitra aktif di 34 kota Indonesia.",
    },
    {
        icon: LuBadgeCheck,
        title: "Transaksi Aman",
        description: "Pembayaran terproteksi dengan jaminan kepuasan.",
    },
];

export default function AboutSection() {
    return (
        <section className="bg-slate-100 h-full flex items-center justify-center scroll-mt-24">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-16 items-center *:content-center">
                    <div className="flex flex-col gap-4 items-start">
                        <p className="text-sm sm:text-base font-bold tracking-wide uppercase text-industrial-blue-500">Mengapa Devotion</p>
                        <h2 className="max-w-2xl text-2xl font-bold tracking-tight sm:text-3xl lg:text-4xl">Bukan sekadar solusi, kami hadir untuk berkembang bersama.</h2>
                        <div className=""> Devotion hadir dengan pendekatan yang mengutamakan kualitas, kebutuhan, dan hubungan jangka panjang. Kami percaya bahwa solusi terbaik lahir dari pemahaman yang baik.</div>
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                        {aboutBenefit.map((item, index) => {
                            const Icon = item.icon;

                            return (
                                <div key={index} className="group relative h-full flex gap-4">
                                    <div className="relative w-fit">
                                        <Icon className="text-deep-navy-500/80 text-xl transition-colors duration-300 group-hover:text-industrial-blue-500" />
                                    </div>
                                    <div>
                                        <p className="font-bold text-deep-navy-500 text-lg">{item.title}</p>
                                        <p className="text-sm text-slate-500 leading-relaxed">{item.description}</p>
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>
        </section>
    );
}
