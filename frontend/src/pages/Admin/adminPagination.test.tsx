import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { getDisputes, getItemProposals, getLateOrders } from "@api/admin";
import { apiClient } from "@api/client";
import AdminVerificationQueue from "./VerificationQueue";
import AdminProposals from "./Proposals";
import AdminDisputes from "./Disputes";
import AdminLateOrders from "./LateOrders";

jest.mock("@api/client", () => ({
    apiClient: jest.fn(),
    apiUrl: (path: string) => path,
}));

const mockFetchNextPage = jest.fn();

jest.mock("@hooks/useAdmin", () => ({
    useVerificationQueue: jest.fn(() => ({
        data: {
            pages: [{
                items: [{ request_id: "v-1", business_name: "Usaha A", status: "pending", submitted_at: "2024-01-01T00:00:00Z" }],
                pagination: { has_next: true, next_cursor: "next-1" },
            }],
        },
        isLoading: false,
        isError: false,
        hasNextPage: true,
        isFetchingNextPage: false,
        fetchNextPage: mockFetchNextPage,
    })),
    useDecideVerification: jest.fn(() => ({ mutateAsync: jest.fn(), isPending: false })),
    useItemProposals: jest.fn(() => ({
        data: {
            pages: [{
                items: [{ proposal_id: "p-1", proposed_name: "Kain Katun", kind: "product", status: "pending", created_at: "2024-01-01T00:00:00Z" }],
                pagination: { has_next: true, next_cursor: "next-2" },
            }],
        },
        isLoading: false,
        isError: false,
        hasNextPage: true,
        isFetchingNextPage: false,
        fetchNextPage: mockFetchNextPage,
    })),
    useDecideProposal: jest.fn(() => ({ mutateAsync: jest.fn(), isPending: false })),
    useDisputes: jest.fn(() => ({
        data: {
            pages: [{
                items: [{ dispute_id: "d-1", work_order_id: "wo-1", report_body: "Ada masalah", status: "reported", created_at: "2024-01-01T00:00:00Z" }],
                pagination: { has_next: true, next_cursor: "next-3" },
            }],
        },
        isLoading: false,
        isError: false,
        hasNextPage: true,
        isFetchingNextPage: false,
        fetchNextPage: mockFetchNextPage,
    })),
    useMediateDispute: jest.fn(() => ({ mutateAsync: jest.fn(), isPending: false })),
    useResolveDispute: jest.fn(() => ({ mutateAsync: jest.fn(), isPending: false })),
    useLateOrders: jest.fn(() => ({
        data: {
            pages: [{
                items: [{ work_order_id: "wo-2", status: "production", buyer_profile_id: "b-1", subcontractor_profile_id: "s-1", quantity: 10, deadline: "2024-01-10", total_price: 1500000, readiness_deadline: "2024-01-08" }],
                pagination: { has_next: true, next_cursor: "next-4" },
            }],
        },
        isLoading: false,
        isError: false,
        hasNextPage: true,
        isFetchingNextPage: false,
        fetchNextPage: mockFetchNextPage,
    })),
}));

jest.mock("@hooks/useWorkOrders", () => ({
    useWorkOrder: jest.fn(() => ({
        data: { buyer_profile_id: "b-1", subcontractor_profile_id: "s-1", payment_mismatch: null },
        isLoading: false,
        isError: false,
    })),
}));

describe("admin pagination controls", () => {
    beforeEach(() => {
        mockFetchNextPage.mockClear();
        (apiClient as jest.Mock).mockReset();
    });

    function renderWithRouter(ui: React.ReactElement) {
        return render(<MemoryRouter>{ui}</MemoryRouter>);
    }

    it("meneruskan opaque cursor apa adanya pada halaman sengketa berikutnya", async () => {
        (apiClient as jest.Mock).mockResolvedValue({
            items: [{ dispute_id: "d-2", work_order_id: "wo-99", report_body: "Lanjutan", status: "reported", created_at: "2024-01-02T00:00:00Z" }],
            pagination: { has_next: true, next_cursor: "opaque-page-2" },
        });

        const result = await getDisputes({ status: "reported", cursor: "opaque-page-1" });

        expect(apiClient).toHaveBeenCalledWith("/admin/disputes?status=reported&cursor=opaque-page-1");
        expect(result.pagination.next_cursor).toBe("opaque-page-2");
    });

    it("meneruskan opaque cursor apa adanya pada halaman proposal berikutnya", async () => {
        (apiClient as jest.Mock).mockResolvedValue({
            items: [{ proposal_id: "p-2", kind: "machine", proposed_name: "Mesin B", status: "pending", created_at: "2024-01-02T00:00:00Z" }],
            pagination: { has_next: true, next_cursor: "opaque-prop-2" },
        });

        const result = await getItemProposals({ cursor: "opaque-prop-1" });

        expect(apiClient).toHaveBeenCalledWith("/admin/proposals?cursor=opaque-prop-1");
        expect(result.pagination.next_cursor).toBe("opaque-prop-2");
    });

    it("meneruskan opaque cursor apa adanya pada halaman pesanan terlambat berikutnya", async () => {
        (apiClient as jest.Mock).mockResolvedValue({
            items: [{ work_order_id: "wo-99", status: "production", buyer_profile_id: "b-1", subcontractor_profile_id: "s-1", quantity: 2, deadline: "2024-01-12", total_price: 200000, readiness_deadline: "2024-01-10" }],
            pagination: { has_next: true, next_cursor: "opaque-late-2" },
        });

        const result = await getLateOrders({ cursor: "opaque-late-1" });

        expect(apiClient).toHaveBeenCalledWith("/admin/late-orders?cursor=opaque-late-1");
        expect(result.pagination.next_cursor).toBe("opaque-late-2");
    });

    it("menampilkan tombol muat lebih banyak pada antrean verifikasi", () => {
        renderWithRouter(<AdminVerificationQueue />);

        fireEvent.click(screen.getByRole("button", { name: /muat lebih banyak/i }));
        expect(mockFetchNextPage).toHaveBeenCalledTimes(1);
    });

    it("menampilkan tombol muat lebih banyak pada usulan item", () => {
        renderWithRouter(<AdminProposals />);

        fireEvent.click(screen.getByRole("button", { name: /muat lebih banyak/i }));
        expect(mockFetchNextPage).toHaveBeenCalledTimes(1);
    });

    it("menampilkan tombol muat lebih banyak pada sengketa", () => {
        renderWithRouter(<AdminDisputes />);

        fireEvent.click(screen.getByRole("button", { name: /muat lebih banyak/i }));
        expect(mockFetchNextPage).toHaveBeenCalledTimes(1);
    });

    it("menampilkan tombol muat lebih banyak pada pesanan terlambat", () => {
        renderWithRouter(<AdminLateOrders />);

        fireEvent.click(screen.getByRole("button", { name: /muat lebih banyak/i }));
        expect(mockFetchNextPage).toHaveBeenCalledTimes(1);
    });
});
