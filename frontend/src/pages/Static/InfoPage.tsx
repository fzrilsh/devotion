import HomeLayout from "@components/layout/HomeLayout";
import { motion } from "motion/react";
import type { ReactNode } from "react";

export type InfoSection = {
    heading: string;
    body: string[];
};

type InfoPageProps = {
    title: string;
    intro: string;
    sections: InfoSection[];
    children?: ReactNode;
};

export default function InfoPage({ title, intro, sections, children }: InfoPageProps) {
    return (
        <HomeLayout>
            <div className="bg-slate-50 pt-36 pb-20">
                <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4, ease: "easeOut" }} className="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
                    <div className="text-center">
                        <h1 className="text-3xl font-extrabold tracking-tight text-deep-navy-900 sm:text-4xl">{title}</h1>
                        <p className="mt-4 text-sm leading-relaxed text-slate-600 sm:text-base">{intro}</p>
                    </div>

                    <div className="mt-12 space-y-8">
                        {sections.map((section, index) => (
                            <section key={index} className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-8">
                                <h2 className="text-lg font-bold text-deep-navy-900">{section.heading}</h2>

                                <div className="mt-3 space-y-3">
                                    {section.body.map((paragraph, pIndex) => (
                                        <p key={pIndex} className="text-sm leading-7 text-slate-600">
                                            {paragraph}
                                        </p>
                                    ))}
                                </div>
                            </section>
                        ))}

                        {children}
                    </div>
                </motion.div>
            </div>
        </HomeLayout>
    );
}
