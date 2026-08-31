import { useCallback, useEffect, useState } from "react";

export interface NavItem {
    href: string;
    label?: string;
}

export function useHeader(navItems: NavItem[] = [], offset: number = 0) {
    const [isScrolled, setIsScrolled] = useState<boolean>(false);
    const [isOpen, setIsOpen] = useState<boolean>(false);
    const [activeLink, setActiveLink] = useState<string>(navItems[0]?.href || "");

    const toggleMenu = useCallback(() => setIsOpen((prev) => !prev), []);
    const closeMenu = useCallback(() => setIsOpen(false), []);

    const handleClick = useCallback((e: React.MouseEvent<HTMLAnchorElement, MouseEvent>, href: string) => {
        if (href.startsWith("#")) {
            e.preventDefault();
            const targetId = href.replace("#", "");
            const targetElement = document.getElementById(targetId);

            if (targetElement) {
                targetElement.scrollIntoView({ behavior: "smooth" });
                setActiveLink(href);
            }
        }
    }, []);

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
                        const section = document.querySelector<HTMLElement>(nav.href);
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
