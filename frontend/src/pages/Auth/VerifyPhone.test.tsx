import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import VerifyPhone from "./VerifyPhone";
import { ApiError } from "@api/client";
import { useAuth, useResendCode, useVerifyPhone } from "@hooks/useAuth";

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
    useVerifyPhone: jest.fn(),
    useResendCode: jest.fn(),
}));

const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockedUseVerifyPhone = useVerifyPhone as jest.MockedFunction<typeof useVerifyPhone>;
const mockedUseResendCode = useResendCode as jest.MockedFunction<typeof useResendCode>;

function mockAccount() {
    mockedUseAuth.mockReturnValue({
        user: {
            account_id: "acc-1",
            email: "mitra@contoh.id",
            phone: "+6281234567890",
            email_verified: true,
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
    mockedUseVerifyPhone.mockReturnValue({ mutateAsync: fn, isPending: false } as never);
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
            <VerifyPhone />
        </MemoryRouter>,
    );
}

describe("VerifyPhone", () => {
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

        await user.type(boxes[0]!, "7");
        await user.type(boxes[1]!, "8");
        expect(boxes[0]!).toHaveValue("7");
        expect(boxes[1]!).toHaveValue("8");

        await user.type(boxes[1]!, "{backspace}");
        expect(boxes[1]!).toHaveValue("");

        await user.type(boxes[0]!, "{backspace}");
        expect(boxes[0]!).toHaveValue("");
    });

    it("FR-004: halaman verifikasi langsung mengirim ulang kode WhatsApp saat dibuka", async () => {
        const resend = mockResendMutation();
        renderPage();

        await waitFor(() => {
            expect(resend).toHaveBeenCalledWith({ target: "+6281234567890", channel: "whatsapp" });
        });

        expect(await screen.findByRole("status")).toHaveTextContent("Kode verifikasi baru sudah dikirim melalui WhatsApp.");
    });

    it("FR-004: penolakan rate limit saat buka halaman menampilkan waktu tunggu, bukan galat", async () => {
        mockResendMutation(() => {
            throw new ApiError(429, { code: "RATE_LIMIT_EXCEEDED" }, 90);
        });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText(/Kirim ulang kode dalam/)).toHaveTextContent("01:30");
        });

        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
});
