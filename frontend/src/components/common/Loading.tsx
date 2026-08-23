import { fadeIn } from "@data/animation";
import { motion } from "motion/react";

interface LoadingProps {
    onComplete?: () => void;
}

export default function Loading({ onComplete }: LoadingProps) {
    return (
        <motion.div
            className="fixed inset-0 z-50 flex flex-col items-center justify-center gap-6"
            {...fadeIn}
            onAnimationComplete={(definition) => {
                if (definition === "exit") {
                    onComplete?.();
                }
            }}
        >
            <div className="flex flex-col items-center gap-3">
                <motion.h1 className="text-deep-navy-500 font-extrabold text-4xl" {...fadeIn}>
                    Devo<span className="text-industrial-orange-500">tion</span>
                </motion.h1>

                <svg width="240" height="8" viewBox="0 0 240 8" className="overflow-visible">
                    <line x1="0" y1="4" x2="240" y2="4" stroke="#fff" strokeWidth="1" />
                    <motion.line x1="0" y1="4" x2="240" y2="4" stroke="#1a365d" strokeWidth="4" strokeLinecap="round" strokeDasharray="6 6" initial={{ strokeDashoffset: 0 }} animate={{ strokeDashoffset: -24 }} transition={{ duration: 0.8, repeat: Infinity, ease: "linear" }} />
                </svg>
            </div>
        </motion.div>
    );
}
