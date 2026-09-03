import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import Footer from "./Footer";

describe("Footer", () => {
    it("merender tautan jelajah sebagai path /#section agar benar dari halaman mana pun", () => {
        render(
            <MemoryRouter initialEntries={["/tentang"]}>
                <Footer />
            </MemoryRouter>,
        );

        const beranda = screen.getByRole("link", { name: "Beranda" });
        expect(beranda).toHaveAttribute("href", "/#beranda");

        const kapasitas = screen.getByRole("link", { name: "Kapasitas" });
        expect(kapasitas).toHaveAttribute("href", "/#kapasitas");
    });
});
