import { LuHandshake, LuLayers, LuScissors, LuSearch } from "react-icons/lu";

const steps = [
    {
        num: "01",
        icon: LuScissors,
        title: "Isi kebutuhan produksi",
        description: "Masukkan jenis produk, jumlah, bahan, dan tenggat waktu yang Anda butuhkan.",
    },
    {
        num: "02",
        icon: LuSearch,
        title: "Temukan kandidat yang sesuai",
        description: "Sistem menampilkan daftar subkontraktor yang memiliki kapasitas dan kemampuan relevan.",
    },
    {
        num: "03",
        icon: LuLayers,
        title: "Bandingkan penawaran",
        description: "Lihat dan bandingkan profil, kapasitas, mesin, dan kesiapan mulai dari setiap kandidat.",
    },
    {
        num: "04",
        icon: LuHandshake,
        title: "Sepakati dan pantau pesanan",
        description: "Komunikasikan detail langsung dan pantau progres pesanan dari satu tempat.",
    },
];

export default function HowItWorksSection() {
    return (
        <section className="h-full flex items-center justify-center scroll-mt-24">
            <div className="relative mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
                <div className="mb-8">
                    <p className="text-sm sm:text-base font-bold tracking-wide text-deep-navy-500 uppercase">Cara Kerja</p>
                    <h1 className="text-4xl max-w-xl mt-4 font-bold leading-tight">
                        <span className="text-industrial-orange-500">Empat langkah</span> menuju kerja sama produksi
                    </h1>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8">
                    {steps.map((step, index) => (
                        <div className="relative" key={step.num}>
                            {index < steps.length - 1 && <div className="hidden lg:block absolute top-6.5 left-24 w-full h-px bg-deep-navy-500 z-0" style={{ width: "calc(100% - 5.75rem)" }} />}

                            <div className="relative z-10">
                                <div className="flex items-center gap-4 mb-4">
                                    <div className="p-4 rounded-xl w-fit bg-deep-navy-500">
                                        <step.icon className="text-white text-xl" />
                                    </div>
                                    <span className="text-industrial-orange-500 text-sm font-semibold tracking-widest">{step.num}</span>
                                </div>
                                <p className="font-bold">{step.title}</p>
                                <p className="text-sm text-justify text-slate-500 leading-relaxed">{step.description}</p>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </section>
    );
}
