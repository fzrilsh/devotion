import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { scrollToSection } from "@lib/anchors";

export function sectionIdFromHref(href: string): string {
    const hashIndex = href.indexOf("#");
    return hashIndex >= 0 ? href.slice(hashIndex + 1) : "";
}

export interface NavItem {
    href: string;
    label?: string;
}

export function useHeader(navItems: NavItem[] = [], offset: number = 0) {
    const [isScrolled, setIsScrolled] = useState<boolean>(false);
    const [isOpen, setIsOpen] = useState<boolean>(false);
    const [activeLink, setActiveLink] = useState<string>(navItems[0]?.href || "");
    const navigate = useNavigate();
    const { pathname } = useLocation();

    const toggleMenu = useCallback(() => setIsOpen((prev) => !prev), []);
    const closeMenu = useCallback(() => setIsOpen(false), []);

    const handleClick = useCallback((e: React.MouseEvent<HTMLAnchorElement, MouseEvent>, href: string) => {
        const hashIndex = href.indexOf("#");
        if (hashIndex < 0) return;

        e.preventDefault();
        const sectionId = href.slice(hashIndex + 1);

        if (pathname !== "/") {
            navigate(`/#${sectionId}`);
            return;
        }

        if (scrollToSection(sectionId)) {
            setActiveLink(href);
        }
    }, [navigate, pathname]);

    useEffect(() => {
        let ticking = false;

        function onScroll() {
            if (ticking) return;
            ticking = true;

            requestAnimationFrame(() => {
                const scrollTop = window.scrollY;
                setIsScrolled(scrollTop > 5);
                setIsOpen(false);

                if (navItems.length > 0) {
                    const scrollPosition = scrollTop + offset;

                    for (const nav of navItems) {
                        const section = document.getElementById(sectionIdFromHref(nav.href));
                        if (!section) continue;

                        const sectionTop = section.offsetTop;
                        const sectionBottom = sectionTop + section.offsetHeight;

                        if (scrollPosition >= sectionTop && scrollPosition < sectionBottom) {
                            setActiveLink(nav.href);
                            break;
                        }
                    }
                }

                ticking = false;
            });
        }

        window.addEventListener("scroll", onScroll, { passive: true });
        return () => window.removeEventListener("scroll", onScroll);
    }, [navItems, offset]);

    useEffect(() => {
        window.addEventListener("pageshow", closeMenu);
        window.addEventListener("popstate", closeMenu);

        return () => {
            window.removeEventListener("pageshow", closeMenu);
            window.removeEventListener("popstate", closeMenu);
        };
    }, [closeMenu]);

    return { isScrolled, isOpen, setIsOpen, toggleMenu, closeMenu, activeLink, handleClick };
}
