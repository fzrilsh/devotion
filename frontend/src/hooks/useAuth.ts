import { getMe, login, logout, type LoginRequest } from "@api/auth";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const authKeys = {
    me: ["auth", "me"] as const,
};

export function useMe() {
    return useQuery({
        queryKey: authKeys.me,
        queryFn: getMe,
        staleTime: 60_000,
        retry: false,
    });
}

export function useLogin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: LoginRequest) => login(payload),
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
            queryClient.invalidateQueries({ queryKey: authKeys.me });
        },
    });
}

export function useAuth() {
    const meQuery = useMe();
    const loginMutation = useLogin();
    const logoutMutation = useLogout();

    return {
        user: meQuery.data ?? null,
        isAuthenticated: Boolean(meQuery.data),
        isLoading: meQuery.isLoading,
        isFetching: meQuery.isFetching,
        login: loginMutation.mutateAsync,
        logout: logoutMutation.mutateAsync,
        loginPending: loginMutation.isPending,
        logoutPending: logoutMutation.isPending,
        loginError: loginMutation.error,
        logoutError: logoutMutation.error,
        refetchMe: meQuery.refetch,
    };
}
