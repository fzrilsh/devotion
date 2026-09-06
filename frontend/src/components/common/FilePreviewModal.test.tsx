import "@testing-library/jest-dom";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import FilePreviewModal from "./FilePreviewModal";

// @api/client membaca import.meta.env yang tidak bisa dievaluasi ts-jest, jadi
// modulnya diganti, sama seperti pola di useQuota.test.tsx.
jest.mock("@api/client", () => ({
    apiUrl: (path: string) => `/api${path}`,
}));

// JPEG dimulai FF D8 FF, PNG dimulai 89 50 4E 47, PDF dimulai %PDF.
const JPEG_BYTES = [0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10];
const PNG_BYTES = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a];
const PDF_BYTES = [0x25, 0x50, 0x44, 0x46, 0x2d, 0x31];

function mockFileResponse(bytes: number[], status = 200) {
    return {
        ok: status === 200,
        status,
        blob: () => Promise.resolve(new Blob([new Uint8Array(bytes)], { type: "application/octet-stream" })),
    };
}

describe("FilePreviewModal", () => {
    afterEach(() => {
        jest.restoreAllMocks();
        delete (global as { fetch?: unknown }).fetch;
    });

    function stubFetch(response: ReturnType<typeof mockFileResponse>) {
        const fetchMock = jest.fn().mockResolvedValue(response);
        (global as { fetch: unknown }).fetch = fetchMock;
        return fetchMock;
    }

    it("menampilkan gambar JPEG dari berkas yang diambil dengan kredensial (FR-009)", async () => {
        const fetchMock = stubFetch(mockFileResponse(JPEG_BYTES));

        render(<FilePreviewModal fileId="abc-123" title="Dokumen" onClose={() => {}} />);

        await waitFor(() => expect(screen.getByRole("img", { name: "Dokumen" })).toBeInTheDocument());
        expect(fetchMock).toHaveBeenCalledWith("/api/files/abc-123", expect.objectContaining({ credentials: "include" }));
    });

    it("menampilkan gambar PNG dari magic bytes (FR-009)", async () => {
        stubFetch(mockFileResponse(PNG_BYTES));

        render(<FilePreviewModal fileId="png-1" title="Foto Lokasi" onClose={() => {}} />);

        await waitFor(() => expect(screen.getByRole("img", { name: "Foto Lokasi" })).toBeInTheDocument());
    });

    it("merender PDF di dalam iframe viewer bawaan (FR-009)", async () => {
        stubFetch(mockFileResponse(PDF_BYTES));

        render(<FilePreviewModal fileId="pdf-1" title="Dokumen" onClose={() => {}} />);

        await waitFor(() => expect(screen.getByTitle("Dokumen")).toBeInTheDocument());
    });

    it("menutup modal lewat tombol Escape (FR-009)", async () => {
        const user = userEvent.setup();
        const onClose = jest.fn();
        stubFetch(mockFileResponse(JPEG_BYTES));

        render(<FilePreviewModal fileId="esc-1" title="Dokumen" onClose={onClose} />);
        await screen.findByRole("img", { name: "Dokumen" });

        await user.keyboard("{Escape}");

        expect(onClose).toHaveBeenCalled();
    });

    it("menutup modal lewat tombol Tutup (FR-009)", async () => {
        const user = userEvent.setup();
        const onClose = jest.fn();
        stubFetch(mockFileResponse(JPEG_BYTES));

        render(<FilePreviewModal fileId="close-1" title="Dokumen" onClose={onClose} />);
        await screen.findByRole("img", { name: "Dokumen" });

        await user.click(screen.getByRole("button", { name: /tutup/i }));

        expect(onClose).toHaveBeenCalled();
    });

    it("menutup modal lewat klik backdrop (FR-009)", async () => {
        const user = userEvent.setup();
        const onClose = jest.fn();
        stubFetch(mockFileResponse(JPEG_BYTES));

        render(<FilePreviewModal fileId="backdrop-1" title="Dokumen" onClose={onClose} />);
        await screen.findByRole("img", { name: "Dokumen" });

        await user.click(screen.getByTestId("backdrop"));

        expect(onClose).toHaveBeenCalled();
    });

    it("menaikkan zoom gambar saat diklik lalu kembali ke 1x setelah 4x (FR-009)", async () => {
        const user = userEvent.setup();
        stubFetch(mockFileResponse(JPEG_BYTES));

        render(<FilePreviewModal fileId="zoom-1" title="Dokumen" onClose={() => {}} />);
        const img = await screen.findByRole("img", { name: "Dokumen" });

        expect(img).toHaveStyle({ transform: "scale(1)" });
        await user.click(img);
        expect(img).toHaveStyle({ transform: "scale(2)" });
        await user.click(img);
        expect(img).toHaveStyle({ transform: "scale(4)" });
        await user.click(img);
        expect(img).toHaveStyle({ transform: "scale(1)" });
    });

    it("menampilkan pesan galat ketika berkas ditolak server (FR-009)", async () => {
        stubFetch(mockFileResponse([], 403));

        render(<FilePreviewModal fileId="err-1" title="Dokumen" onClose={() => {}} />);

        await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Berkas tidak dapat dimuat. Sesi Anda mungkin sudah berakhir. Silakan muat ulang halaman."));
    });

    it("tidak merender apa pun ketika fileId kosong", () => {
        const { container } = render(<FilePreviewModal fileId={null} title="Dokumen" onClose={() => {}} />);

        expect(container).toBeEmptyDOMElement();
    });
});
