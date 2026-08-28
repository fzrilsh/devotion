import { confirmPasswordRecovery, getMe, login, logout, register, requestPasswordRecovery, resendCode, updateMyRoles, verifyEmail, verifyPhone, type LoginRequest, type RecoverConfirmRequest, type RecoverRequest, type RegisterRequest, type ResendCodeRequest, type RolesUpdateRequest, type VerificationCodeRequest } from "@api/auth";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const authKeys = {
    me: ["auth", "me"] as const,
};

export function useMe() {
    return useQuery({
        queryKey: authKeys.me,
        queryFn: getMe,

        staleTime: 5 * 60 * 1000,
        gcTime: 30 * 60 * 1000,

        retry: false,

        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
        refetchOnMount: false,

        // Jangan pernah menampilkan data akun dari cache saat fetch ulang.
        // Setelah ganti akun, data lama harus hilang sebelum data baru tiba.
        placeholderData: undefined,
    });
}

export function useLogin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: LoginRequest) => login(data),
        onSuccess: (account) => {
            queryClient.setQueryData(authKeys.me, account);
        },
    });
}

export function useLogout() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: logout,
        onSuccess: () => {
            // Bersihkan seluruh cache, bukan hanya data akun. Kalau hanya
            // authKeys.me yang dihapus, query lain (profil, listing, pesanan)
            // masih menyimpan data akun sebelumnya dan ikut ke sesi akun baru.
            queryClient.clear();
        },
    });
}

export function useRegister() {
    return useMutation({
        mutationFn: (data: RegisterRequest) => register(data),
    });
}

export function useVerifyEmail() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: VerificationCodeRequest) => verifyEmail(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: authKeys.me });
        },
    });
}

export function useVerifyPhone() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: VerificationCodeRequest) => verifyPhone(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: authKeys.me });
        },
    });
}

export function useResendCode() {
    return useMutation({
        mutationFn: (data: ResendCodeRequest) => resendCode(data),
    });
}

export function useUpdateMyRoles() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: RolesUpdateRequest) => updateMyRoles(data),
        onSuccess: (account) => {
            queryClient.setQueryData(authKeys.me, account);
        },
    });
}

export function useRequestPasswordRecovery() {
    return useMutation({
        mutationFn: (data: RecoverRequest) => requestPasswordRecovery(data),
    });
}

export function useConfirmPasswordRecovery() {
    return useMutation({
        mutationFn: (data: RecoverConfirmRequest) => confirmPasswordRecovery(data),
    });
}

export function useAuth() {
    const meQuery = useMe();

    return {
        user: meQuery.data ?? null,
        isAuthenticated: Boolean(meQuery.data),
        isLoading: meQuery.isLoading,
        isFetching: meQuery.isFetching,

        refetchMe: meQuery.refetch,
    };
}
