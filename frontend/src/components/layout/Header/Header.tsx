import { fadeIn } from "@data/animation";
import { mainNavigation } from "@data/navigation";
import { motion } from "motion/react";
import HeaderList from "./HeaderList";
import { cn } from "@lib/utils";
import { useHeader } from "./useHeader";

export default function Header() {
    const { isScrolled, isOpen, setIsOpen } = useHeader();
    const toggleMenu = () => setIsOpen(!isOpen);

    return (
        <motion.header {...fadeIn} className={cn("fixed top-0 left-0 right-0 mx-auto w-full z-50 transition-all duration-200", isScrolled ? "bg-white/80 backdrop-blur-xl" : "", isOpen ? "bg-white" : "")}>
            <nav className="flex items-center max-w-7xl justify-between gap-4 mx-auto w-full px-4 py-8 sm:px-6 lg:px-8">
                <a href="/" className="flex items-center justify-center">
                    <h1 className="text-deep-navy-500 font-bold text-2xl">
                        Devo<span className="text-industrial-orange-500">tion</span>
                    </h1>
                </a>

                <ul className="flex items-center justify-center gap-2 max-lg:hidden">
                    <HeaderList items={mainNavigation} />
                </ul>

                <div className="flex items-center gap-2">
                    <div className="hidden sm:flex items-center gap-2">
                        <a href="/auth/login" className={cn("inline-flex items-center justify-center rounded-md px-6 py-2 font-medium text-industrial-orange-500 hover:bg-primary/10  transition-all duration-200 hover:-translate-y-0.5")}>
                            Masuk
                        </a>

                        <a href="/auth/register" className={cn("inline-flex items-center justify-center rounded-md px-6 py-2 font-semibold bg-industrial-orange-500 text-white hover:bg-industrial-orange-600 transition-all duration-200 hover:-translate-y-0.5")}>
                            Daftar
                        </a>
                    </div>

                    <div className="lg:hidden">
                        <div onClick={toggleMenu} className={cn("flex h-11 w-11 cursor-pointer items-center justify-center rounded-md p-2 shadow-2xl hover:bg-white")}>
                            <div className="flex flex-col items-end gap-2">
                                <span className={cn("block h-1 w-7 origin-center rounded-full transition-all duration-300 ease-in-out", isOpen ? "w-7 translate-y-1.5 -rotate-45 bg-deep-navy-500" : "bg-industrial-orange-500")} />
                                <span className={cn("block h-1 w-7 origin-center rounded-full transition-all duration-300 ease-in-out", isOpen ? "w-7 -translate-y-1.5 rotate-45 bg-deep-navy-500" : "bg-industrial-orange-500")} />
                            </div>
                        </div>
                    </div>
                </div>
            </nav>
        </motion.header>
    );
}
