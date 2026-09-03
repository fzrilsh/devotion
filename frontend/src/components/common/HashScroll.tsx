import { scrollToSection } from "@lib/anchors";
import { useEffect } from "react";
import { useLocation } from "react-router-dom";

// Anchor navigasi memakai path /#section sehingga tautan tetap benar dari
// halaman mana pun. Router tidak menggulir otomatis, jadi komponen ini
// menggulir ke section saat hash berubah, termasuk saat membuka /#section
// langsung dari address bar.
export default function HashScroll() {
    const { hash, pathname } = useLocation();

    useEffect(() => {
        if (pathname !== "/") return;

        const sectionId = hash.replace(/^#/, "");
        scrollToSection(sectionId, hash ? "smooth" : "auto");
    }, [hash, pathname]);

    return null;
}
