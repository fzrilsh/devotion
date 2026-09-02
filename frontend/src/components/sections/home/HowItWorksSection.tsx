import { motion } from "motion/react";
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

const container = {
    hidden: {},
    show: {
        transition: {
            staggerChildren: 0.12,
        },
    },
};

const card = {
    hidden: { opacity: 0, y: 32 },
    show: { opacity: 1, y: 0, transition: { duration: 0.55, ease: "easeOut" as const } },
};

export default function HowItWorksSection() {
    return (
        <section id="cara-kerja" className="relative h-full scroll-mt-24 overflow-hidden bg-deep-navy-500">
            <div aria-hidden className="pointer-events-none absolute inset-0">
                <div className="absolute -top-40 left-1/2 size-128 -translate-x-1/2 rounded-full bg-industrial-blue-500/10 blur-3xl" />
                <div className="absolute -bottom-48 -right-24 size-96 rounded-full bg-industrial-orange-500/10 blur-3xl" />
            </div>

            <div className="relative mx-auto w-full max-w-7xl px-4 py-20 sm:px-6 lg:px-8 lg:py-28">
                <div className="mx-auto mb-14 max-w-2xl text-center">
                    <motion.div initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true, margin: "-80px" }} transition={{ duration: 0.5, ease: "easeOut" }}>
                        <h2 className="mt-5 text-3xl font-extrabold leading-tight tracking-tight text-white sm:text-4xl">Dari kebutuhan produksi ke kesepakatan, dalam empat langkah.</h2>
                        <p className="mt-4 text-sm leading-relaxed text-white/60 sm:text-base">Alur yang sama dipakai pemberi order dan pemilik kapasitas. Tidak ada perantara tersembunyi, semua tercatat di platform.</p>
                    </motion.div>
                </div>

                <motion.ol variants={container} initial="hidden" whileInView="show" viewport={{ once: true, margin: "-80px" }} className="relative grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
                    {steps.map((step) => {
                        const Icon = step.icon;

                        return (
                            <motion.li key={step.num} variants={card} className="group relative">
                                <div className="relative flex h-full flex-col rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm transition-colors duration-300 hover:border-industrial-blue-400/40 hover:bg-white/10">
                                    <div className="relative mb-6 flex items-center justify-between">
                                        <span className="relative grid size-14 place-items-center rounded-2xl bg-industrial-blue-500 text-white shadow-lg shadow-industrial-blue-500/30 transition-transform duration-300 group-hover:-translate-y-1">
                                            <Icon className="size-6" aria-hidden />
                                        </span>
                                        <span aria-hidden className="text-4xl font-extrabold tracking-tight text-white/10 transition-colors duration-300 group-hover:text-industrial-blue-400/30">
                                            {step.num}
                                        </span>
                                    </div>

                                    <h3 className="text-base font-bold text-white">{step.title}</h3>
                                    <p className="mt-2 text-sm leading-relaxed text-white/60">{step.description}</p>
                                </div>
                            </motion.li>
                        );
                    })}
                </motion.ol>
            </div>
        </section>
    );
}
