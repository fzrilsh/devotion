import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";
import PaymentMismatchNotice from "./PaymentMismatchNotice";

describe("PaymentMismatchNotice", () => {
    it("tidak merender apa pun ketika server tidak menandai selisih (FR-043)", () => {
        const { container } = render(<PaymentMismatchNotice mismatch={null} audience="party" />);

        expect(container).toBeEmptyDOMElement();
    });

    it("menjelaskan pernyataan yang belum dibalas pihak lawan (FR-043)", () => {
        render(<PaymentMismatchNotice mismatch={{ kind: "missing_counterpart" }} audience="party" />);

        expect(screen.getByRole("status")).toHaveTextContent("Satu pihak sudah menyatakan pembayaran, pihak lawan belum menyatakan apa pun.");
        expect(screen.getByRole("status")).toHaveTextContent("laporkan sengketa");
    });

    it("menyebut selisih hari ketika kedua tanggal berbeda (FR-043)", () => {
        render(<PaymentMismatchNotice mismatch={{ kind: "date_differs", day_difference: 3 }} audience="party" />);

        expect(screen.getByRole("status")).toHaveTextContent("tanggalnya berbeda 3 hari");
    });

    it("menghilangkan angka hari bila server tidak mengirimkannya (FR-043)", () => {
        render(<PaymentMismatchNotice mismatch={{ kind: "date_differs" }} audience="party" />);

        expect(screen.getByRole("status")).toHaveTextContent("tetapi tanggalnya berbeda.");
    });

    it("memakai catatan mediasi untuk admin, bukan ajakan melapor (FR-043)", () => {
        render(<PaymentMismatchNotice mismatch={{ kind: "missing_counterpart" }} audience="admin" />);

        const notice = screen.getByRole("status");

        expect(notice).toHaveTextContent("bahan mediasi");
        expect(notice).toHaveTextContent("bukan siapa yang benar");
        expect(notice).not.toHaveTextContent("laporkan sengketa");
    });
});
