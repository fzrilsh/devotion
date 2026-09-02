import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";

// @api/search pulls in @api/client, which reads import.meta.env. ts-jest cannot
// evaluate that, so the module is replaced wholesale.
jest.mock("@api/search", () => ({
    acceptOffer: jest.fn(),
    counterOffer: jest.fn(),
    createQuotaRequest: jest.fn(),
    getIncomingCandidates: jest.fn(),
    getQuotaRequest: jest.fn(),
    getSentQuotaRequests: jest.fn(),
    rejectCandidate: jest.fn(),
    searchSubcontractors: jest.fn(),
    sendOffer: jest.fn(),
}));

import type { CandidateStatus, IncomingCandidate } from "@api/search";
import { CANDIDATE_LISTS_NOT_LOADED, CANDIDATE_NOT_IN_LOADED_LISTS, quotaKeys, useIncomingCandidate } from "./useQuota";

function candidate(id: string, status: CandidateStatus): IncomingCandidate {
    return {
        candidate_id: id,
        listing_id: "11111111-1111-4111-8111-111111111111",
        profile_id: "22222222-2222-4222-8222-222222222222",
        business_name: "Konveksi Contoh",
        status,
        quantity: 500,
        deadline: "2026-10-05",
        capacity_in_range: 800,
        can_fulfill: true,
    };
}

/** Mirrors what useIncomingCandidates writes: one infinite-query entry per status filter. */
function seedList(client: QueryClient, status: CandidateStatus | undefined, items: IncomingCandidate[]) {
    client.setQueryData(quotaKeys.incoming(status), {
        pages: [{ items, pagination: { has_next: false, next_cursor: null } }],
        pageParams: [undefined],
    });
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

    it("FR-030: menemukan kandidat dari daftar tanpa filter", async () => {
        seedList(client, undefined, [candidate("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "awaiting_reply")]);

        const { result } = renderCandidateHook(client, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa");

        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data?.candidate_id).toBe("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa");
    });

    // The regression this test pins: the writer keys each list by its status
    // filter, so a reader that only looks at the "all" entry finds nothing for
    // any candidate opened from a filtered list.
    it("FR-031: menemukan kandidat yang dibuka dari daftar yang difilter status", async () => {
        seedList(client, "offered", [candidate("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "offered")]);

        const { result } = renderCandidateHook(client, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb");

        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data?.status).toBe("offered");
    });

    it("FR-031: mencari di seluruh entri daftar, bukan hanya yang pertama", async () => {
        seedList(client, undefined, [candidate("cccccccc-cccc-4ccc-8ccc-cccccccccccc", "awaiting_reply")]);
        seedList(client, "rejected", [candidate("dddddddd-dddd-4ddd-8ddd-dddddddddddd", "rejected")]);

        const { result } = renderCandidateHook(client, "dddddddd-dddd-4ddd-8ddd-dddddddddddd");

        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data?.status).toBe("rejected");
    });

    it("FR-030: membedakan daftar yang belum dimuat dari kandidat yang tidak ada", async () => {
        const { result } = renderCandidateHook(client, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee");

        await waitFor(() => expect(result.current.isError).toBe(true));
        expect((result.current.error as Error).message).toBe(CANDIDATE_LISTS_NOT_LOADED);
    });

    it("FR-030: melaporkan kandidat tidak ada bila daftar sudah dimuat tanpa kandidat itu", async () => {
        seedList(client, undefined, [candidate("ffffffff-ffff-4fff-8fff-ffffffffffff", "awaiting_reply")]);

        const { result } = renderCandidateHook(client, "99999999-9999-4999-8999-999999999999");

        await waitFor(() => expect(result.current.isError).toBe(true));
        expect((result.current.error as Error).message).toBe(CANDIDATE_NOT_IN_LOADED_LISTS);
    });

    // A prefix read of the lists must not pick up detail entries, or a stale
    // detail entry would be walked as if it were a page of list items.
    it("FR-030: entri detail tidak ikut terbaca saat menyapu prefix daftar", async () => {
        client.setQueryData(quotaKeys.incomingDetail("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), candidate("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "awaiting_reply"));

        const entries = client.getQueriesData({ queryKey: quotaKeys.incomingLists });

        expect(entries).toHaveLength(0);
    });
});
