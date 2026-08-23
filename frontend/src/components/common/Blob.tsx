import { cn } from "@lib/utils";
import { motion, useAnimation } from "motion/react";
import { useEffect } from "react";

interface BlobProps {
    className?: string;
    size?: "sm" | "md" | "lg";
    animate?: boolean;
}

const sizeMap: Record<NonNullable<BlobProps["size"]>, string> = {
    sm: "w-[300px] h-[300px]",
    md: "w-[450px] h-[450px]",
    lg: "w-[600px] h-[600px]",
};

export default function Blob({ className, size = "md", animate = true }: BlobProps) {
    const controls = useAnimation();

    useEffect(() => {
        async function sequence() {
            await controls.start({
                opacity: 1,
                scale: 1,
                transition: { duration: 0.8, ease: "easeOut" },
            });

            if (animate) {
                controls.start({
                    scale: [1, 1.08, 0.96, 1],
                    x: [0, 20, -15, 0],
                    y: [0, -15, 10, 0],
                    borderRadius: ["60% 40% 30% 70% / 60% 30% 70% 40%", "30% 60% 70% 40% / 50% 60% 30% 60%", "50% 50% 40% 60% / 40% 60% 50% 50%", "60% 40% 30% 70% / 60% 30% 70% 40%"],
                    transition: {
                        duration: 14,
                        repeat: Infinity,
                        ease: "easeInOut",
                    },
                });
            }
        }

        sequence();
    }, [controls, animate]);

    return <motion.div aria-hidden="true" initial={{ opacity: 0, scale: 0.85 }} animate={controls} className={cn("absolute -z-10 rounded-[60%_40%_30%_70%/60%_30%_70%_40%] blur-3xl pointer-events-none", sizeMap[size], className)} />;
}
