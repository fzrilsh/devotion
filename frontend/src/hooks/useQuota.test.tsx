import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";

// @api/search pulls in @api/client, which reads import.meta.env. ts-jest cannot
// evaluate that, so the module is replaced wholesale.
jest.mock("@api/search", () => ({
    acceptOffer: jest.fn(),
    counterOffer: jest.fn(),
    createQuotaRequest: jest.fn(),
    getIncomingCandidate: jest.fn(),
    getIncomingCandidates: jest.fn(),
    getQuotaRequest: jest.fn(),
    getSentQuotaRequests: jest.fn(),
    rejectCandidate: jest.fn(),
    searchSubcontractors: jest.fn(),
    sendOffer: jest.fn(),
}));

import { getIncomingCandidate, rejectCandidate, type CandidateStatus, type IncomingCandidate } from "@api/search";
import { quotaKeys, useIncomingCandidate, useRejectCandidate } from "./useQuota";

function candidate(id: string, status: CandidateStatus): IncomingCandidate {
    return {
        candidate_id: id,
        listing_id: "11111111-1111-4111-8111-111111111111",
        profile_id: "22222222-2222-4222-8222-222222222222",
        business_name: "Konveksi Contoh",
        status,
        material: "Katun combed 30s",
        quantity: 500,
        deadline: "2026-10-05",
        note: "Catatan dari pembeli",
        capacity_in_range: 800,
        can_fulfill: true,
    };
}

function renderCandidateHook(client: QueryClient, candidateId: string) {
    function wrapper({ children }: { children: ReactNode }) {
        return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    }

    return renderHook(() => useIncomingCandidate(candidateId), { wrapper });
}

describe("useIncomingCandidate", () => {
    let client: QueryClient;

    beforeEach(() => {
        client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    });

    it("FR-030: memuat kandidat langsung dari server tanpa daftar yang tersimpan", async () => {
        const direct = candidate("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", "awaiting_reply");
        (getIncomingCandidate as jest.Mock).mockResolvedValueOnce(direct);

        const { result } = renderCandidateHook(client, direct.candidate_id);

        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data).toEqual(direct);
        expect(getIncomingCandidate).toHaveBeenCalledWith(direct.candidate_id);
    });

    it("FR-030: tidak lagi gagal saat daftar incoming belum pernah dibuka", async () => {
        const direct = candidate("ffffffff-ffff-4fff-8fff-ffffffffffff", "awaiting_reply");
        (getIncomingCandidate as jest.Mock).mockResolvedValueOnce(direct);

        const { result } = renderCandidateHook(client, direct.candidate_id);

        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.error).toBeNull();
    });

    // A prefix read of the lists must not pick up detail entries, or a stale
    // detail entry would be walked as if it were a page of list items.
    it("FR-030: entri detail tidak ikut terbaca saat menyapu prefix daftar", async () => {
        client.setQueryData(quotaKeys.incomingDetail("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), candidate("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "awaiting_reply"));

        const entries = client.getQueriesData({ queryKey: quotaKeys.incomingLists });

        expect(entries).toHaveLength(0);
    });
});

describe("useRejectCandidate", () => {
    it("FR-031: memperbarui daftar dan detail setelah kandidat ditolak", async () => {
        const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
        const candidateId = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
        (rejectCandidate as jest.Mock).mockResolvedValueOnce(undefined);
        const refetchQueries = jest.spyOn(client, "refetchQueries").mockResolvedValue(undefined);

        function wrapper({ children }: { children: ReactNode }) {
            return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
        }

        const { result } = renderHook(() => useRejectCandidate(candidateId), { wrapper });
        await result.current.mutateAsync("Kapasitas sudah penuh.");

        expect(rejectCandidate).toHaveBeenCalledWith(candidateId, "Kapasitas sudah penuh.");
        expect(refetchQueries).toHaveBeenNthCalledWith(1, { queryKey: quotaKeys.incomingLists });
        expect(refetchQueries).toHaveBeenNthCalledWith(2, { queryKey: quotaKeys.incomingDetail(candidateId) });
    });
});
