import { confirmPasswordRecovery, getMe, login, logout, register, requestPasswordRecovery, resendCode, verifyEmail, verifyPhone, type LoginRequest, type RecoverConfirmRequest, type RecoverRequest, type RegisterRequest, type ResendCodeRequest, type VerificationCodeRequest } from "@api/auth";
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
            queryClient.setQueryData(authKeys.me, null);
            queryClient.removeQueries({ queryKey: authKeys.me });
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
