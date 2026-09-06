import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import VerificationGate from "./VerificationGate";

jest.mock("@hooks/useAuth", () => ({
    useAuth: jest.fn(),
}));

import { useAuth } from "@hooks/useAuth";

const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

function renderGate(action = "membuat listing kapasitas") {
    return render(
        <MemoryRouter>
            <VerificationGate action={action} />
        </MemoryRouter>,
    );
}

function mockAccount({ emailVerified, phoneVerified }: { emailVerified: boolean; phoneVerified: boolean }) {
    mockedUseAuth.mockReturnValue({
        user: {
            account_id: "acc-1",
            email: "mitra@contoh.id",
            phone: "+6281234567890",
            email_verified: emailVerified,
            phone_verified: phoneVerified,
            is_admin: false,
            roles: { subcontractor: true, buyer: false },
        },
        isAuthenticated: true,
        isLoading: false,
        isFetching: false,
        refetchMe: jest.fn(),
    } as unknown as ReturnType<typeof useAuth>);
}

describe("VerificationGate", () => {
    it("FR-002: menampilkan aksi yang diblokir dalam pesan", () => {
        mockAccount({ emailVerified: false, phoneVerified: false });
        renderGate("membuat listing kapasitas");

        expect(screen.getByRole("alert")).toHaveTextContent("Sebelum membuat listing kapasitas, verifikasi email dan nomor WhatsApp Anda terlebih dulu.");
    });

    it("FR-002: menawarkan tautan verifikasi untuk kedua kanal bila keduanya belum terverifikasi", () => {
        mockAccount({ emailVerified: false, phoneVerified: false });
        renderGate();

        const links = screen.getAllByRole("link", { name: "Verifikasi" });
        expect(links).toHaveLength(2);
        expect(links[0]).toHaveAttribute("href", "/auth/verify-email");
        expect(links[1]).toHaveAttribute("href", "/auth/verify-phone");
    });

    it("FR-002: hanya meminta verifikasi kanal yang belum terverifikasi", () => {
        mockAccount({ emailVerified: true, phoneVerified: false });
        renderGate("mengirim request kuota");

        expect(screen.getByRole("alert")).toHaveTextContent("verifikasi nomor WhatsApp Anda terlebih dulu");
        expect(screen.getByText("Terverifikasi")).toBeInTheDocument();

        const links = screen.getAllByRole("link", { name: "Verifikasi" });
        expect(links).toHaveLength(1);
        expect(links[0]).toHaveAttribute("href", "/auth/verify-phone");
    });
});
