import { scrollToSection } from "./anchors";

describe("scrollToSection", () => {
    it("menggulir ke elemen dengan id yang diminta", () => {
        const scrollIntoView = jest.fn();
        const element = document.createElement("div");
        element.id = "kapasitas";
        element.scrollIntoView = scrollIntoView;
        document.body.appendChild(element);

        const found = scrollToSection("kapasitas");

        expect(found).toBe(true);
        expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth" });

        element.remove();
    });

    it("mengembalikan false saat elemen tidak ditemukan tanpa melempar galat", () => {
        expect(scrollToSection("tidak-ada")).toBe(false);
    });

    it("menggulir ke puncak halaman saat id kosong", () => {
        const scrollTo = jest.fn();
        window.scrollTo = scrollTo;

        const found = scrollToSection("");

        expect(found).toBe(true);
        expect(scrollTo).toHaveBeenCalledWith({ top: 0, behavior: "smooth" });
    });
});
