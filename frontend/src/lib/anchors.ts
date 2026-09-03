export function scrollToSection(id: string, behavior: ScrollBehavior = "smooth"): boolean {
    if (!id) {
        window.scrollTo({ top: 0, behavior });
        return true;
    }

    const element = document.getElementById(id);
    if (!element) return false;

    element.scrollIntoView({ behavior });
    return true;
}
