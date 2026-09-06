import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import VerifyEmail from "./VerifyEmail";
import { ApiError } from "@api/client";
import { useAuth, useResendCode, useVerifyEmail } from "@hooks/useAuth";

// @api/client membaca import.meta.env yang tidak bisa dievaluasi ts-jest, jadi
// modulnya diganti, sama seperti pola di FilePreviewModal.test.tsx.
jest.mock("@api/client", () => ({
    ApiError: class ApiError extends Error {
        status: number;
        data: unknown;
        retryAfterSeconds?: number;

        constructor(status: number, data: unknown, retryAfterSeconds?: number) {
            super(`API Error: ${status}`);
            this.status = status;
            this.data = data;
            this.retryAfterSeconds = retryAfterSeconds;
        }
    },
}));

jest.mock("@hooks/useAuth", () => ({
    useAuth: jest.fn(),
    useVerifyEmail: jest.fn(),
    useResendCode: jest.fn(),
}));

const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockedUseVerifyEmail = useVerifyEmail as jest.MockedFunction<typeof useVerifyEmail>;
const mockedUseResendCode = useResendCode as jest.MockedFunction<typeof useResendCode>;

function mockAccount() {
    mockedUseAuth.mockReturnValue({
        user: {
            account_id: "acc-1",
            email: "mitra@contoh.id",
            phone: "+6281234567890",
            email_verified: false,
            phone_verified: false,
            is_admin: false,
            roles: { subcontractor: true, buyer: false },
        },
        isAuthenticated: true,
        isLoading: false,
        isFetching: false,
        refetchMe: jest.fn(),
    } as unknown as ReturnType<typeof useAuth>);
}

function mockVerifyMutation(impl?: (data: unknown) => Promise<unknown>) {
    const fn = jest.fn(impl);
    mockedUseVerifyEmail.mockReturnValue({ mutateAsync: fn, isPending: false } as never);
    return fn;
}

function mockResendMutation(impl?: (data: unknown) => Promise<unknown>) {
    const fn = jest.fn(impl);
    mockedUseResendCode.mockReturnValue({ mutateAsync: fn, isPending: false } as never);
    return fn;
}

function renderPage() {
    return render(
        <MemoryRouter>
            <VerifyEmail />
        </MemoryRouter>,
    );
}

describe("VerifyEmail", () => {
    beforeEach(() => {
        jest.clearAllMocks();
        mockAccount();
        mockVerifyMutation();
        mockResendMutation();
    });

    it("FR-004: digit OTP dapat dihapus dengan backspace", async () => {
        const user = userEvent.setup();
        renderPage();

        const boxes = screen.getAllByRole("textbox") as HTMLInputElement[];
        expect(boxes).toHaveLength(6);

        await user.type(boxes[0]!, "1");
        await user.type(boxes[1]!, "2");
        expect(boxes[0]!).toHaveValue("1");
        expect(boxes[1]!).toHaveValue("2");

        await user.type(boxes[1]!, "{backspace}");
        expect(boxes[1]!).toHaveValue("");
        expect(boxes[0]!).toHaveValue("1");

        await user.type(boxes[0]!, "{backspace}");
        expect(boxes[0]!).toHaveValue("");
    });

    it("FR-004: tempel 6 digit sekaligus mengisi seluruh kotak", async () => {
        const user = userEvent.setup();
        renderPage();

        const boxes = screen.getAllByRole("textbox") as HTMLInputElement[];
        await user.click(boxes[0]!);
        await user.paste("123456");

        const values = screen.getAllByRole("textbox").map((b) => (b as HTMLInputElement).value);
        expect(values).toEqual(["1", "2", "3", "4", "5", "6"]);
    });

    it("FR-004: halaman verifikasi langsung mengirim ulang kode saat dibuka", async () => {
        const resend = mockResendMutation();
        renderPage();

        await waitFor(() => {
            expect(resend).toHaveBeenCalledWith({ target: "mitra@contoh.id", channel: "email" });
        });

        expect(await screen.findByRole("status")).toHaveTextContent("Kode verifikasi baru sudah dikirim ke email Anda.");
    });

    it("FR-004: penolakan rate limit saat buka halaman menampilkan waktu tunggu, bukan galat", async () => {
        mockResendMutation(() => {
            throw new ApiError(429, { code: "RATE_LIMIT_EXCEEDED" }, 42);
        });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText(/Kirim ulang kode dalam/)).toHaveTextContent("00:42");
        });

        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
});
