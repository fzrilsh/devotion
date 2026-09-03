import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import HashScroll from "./HashScroll";
import { scrollToSection } from "@lib/anchors";

jest.mock("@lib/anchors", () => ({
    scrollToSection: jest.fn(),
}));

const mockedScrollToSection = scrollToSection as jest.MockedFunction<typeof scrollToSection>;

beforeEach(() => {
    mockedScrollToSection.mockClear();
});

function renderAt(path: string) {
    return render(
        <MemoryRouter initialEntries={[path]}>
            <HashScroll />
        </MemoryRouter>,
    );
}

describe("HashScroll", () => {
    it("menggulir ke section saat dibuka di path / dengan hash", () => {
        renderAt("/#kapasitas");

        expect(mockedScrollToSection).toHaveBeenCalledWith("kapasitas", "smooth");
    });

    it("tidak menggulir saat pengguna berada di path lain", () => {
        renderAt("/tentang#kapasitas");

        expect(mockedScrollToSection).not.toHaveBeenCalled();
    });

    it("menggulir ke puncak saat berada di / tanpa hash", () => {
        renderAt("/");

        expect(mockedScrollToSection).toHaveBeenCalledWith("", "auto");
    });
});
