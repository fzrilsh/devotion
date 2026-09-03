import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import UnverifiedRoute from "./UnverifiedRoute";
import { useAuth } from "@hooks/useAuth";

jest.mock("@hooks/useAuth", () => ({
    useAuth: jest.fn(),
}));

const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

// fadeIn uses whileInView, which reads IntersectionObserver, an API that
// jsdom does not implement. This stub is enough to render the guard in tests.
class IntersectionObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
}

Object.defineProperty(globalThis, "IntersectionObserver", {
    writable: true,
    configurable: true,
    value: IntersectionObserverStub,
});

function mockAccount({ emailVerified, phoneVerified, isLoading = false }: { emailVerified: boolean; phoneVerified: boolean; isLoading?: boolean }) {
    mockedUseAuth.mockReturnValue({
        user: isLoading
            ? null
            : {
                  account_id: "acc-1",
                  email: "mitra@contoh.id",
                  phone: "+6281234567890",
                  email_verified: emailVerified,
                  phone_verified: phoneVerified,
                  is_admin: false,
                  roles: { subcontractor: true, buyer: false },
              },
        isAuthenticated: !isLoading,
        isLoading,
        isFetching: false,
        refetchMe: jest.fn(),
    } as unknown as ReturnType<typeof useAuth>);
}

function renderRoute() {
    return render(
        <MemoryRouter initialEntries={["/auth/verify-email"]}>
            <Routes>
                <Route path="/auth/verify-email" element={<UnverifiedRoute />}>
                    <Route index element={<div>halaman verifikasi</div>} />
                </Route>
                <Route path="/profile/me" element={<div>profil saya</div>} />
            </Routes>
        </MemoryRouter>,
    );
}

describe("UnverifiedRoute", () => {
    it("FR-002: akun dengan email dan nomor HP terverifikasi dibawa keluar dari halaman verifikasi", () => {
        mockAccount({ emailVerified: true, phoneVerified: true });
        renderRoute();

        expect(screen.getByText("profil saya")).toBeInTheDocument();
        expect(screen.queryByText("halaman verifikasi")).not.toBeInTheDocument();
    });

    it("FR-002: akun dengan satu kanal belum terverifikasi tetap membuka halaman verifikasi", () => {
        mockAccount({ emailVerified: true, phoneVerified: false });
        renderRoute();

        expect(screen.getByText("halaman verifikasi")).toBeInTheDocument();
        expect(screen.queryByText("profil saya")).not.toBeInTheDocument();
    });

    it("FR-002: sesi yang masih dimuat menampilkan layar pemuatan, bukan pengalihan", () => {
        mockAccount({ emailVerified: false, phoneVerified: false, isLoading: true });
        renderRoute();

        expect(screen.queryByText("halaman verifikasi")).not.toBeInTheDocument();
        expect(screen.queryByText("profil saya")).not.toBeInTheDocument();
    });
});
