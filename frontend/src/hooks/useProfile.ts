import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getMyProfile, getProfileReviews, getPublicProfile, updateMyProfile, type ProfileUpdateRequest } from "@api/profile";
import { useAuth } from "@hooks/useAuth";

const profileKeys = {
    me: (accountId: string | undefined) => ["profile", "me", accountId] as const,
    public: (profileId: string) => ["profile", "public", profileId] as const,
    reviews: (profileId: string) => ["profile", "reviews", profileId] as const,
};

export function useMyProfile() {
    const { user } = useAuth();

    return useQuery({
        queryKey: profileKeys.me(user?.account_id),
        queryFn: getMyProfile,
        enabled: Boolean(user && !user.is_admin),
        staleTime: 5 * 60 * 1000,
        gcTime: 30 * 60 * 1000,
        retry: false,
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
    });
}

export function usePublicProfile(profileId: string) {
    return useQuery({
        queryKey: profileKeys.public(profileId),
        queryFn: () => getPublicProfile(profileId),
        enabled: !!profileId,
        staleTime: 10 * 60 * 1000,
        gcTime: 30 * 60 * 1000,
        retry: false,
    });
}

// Ulasan per transaksi beserta identitas pengulas dan tanggalnya (FR-049).
// Kursor diteruskan apa adanya supaya urutan antar halaman tetap stabil.
export function useProfileReviews(profileId: string) {
    return useInfiniteQuery({
        queryKey: profileKeys.reviews(profileId),
        queryFn: ({ pageParam }) => getProfileReviews(profileId, pageParam),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: (lastPage) => (lastPage.pagination.has_next ? (lastPage.pagination.next_cursor ?? undefined) : undefined),
        enabled: Boolean(profileId),
        staleTime: 30 * 1000,
        retry: false,
    });
}

export function useUpdateProfile() {
    const queryClient = useQueryClient();
    const { user } = useAuth();

    return useMutation({
        mutationFn: (data: ProfileUpdateRequest) => updateMyProfile(data),
        onSuccess: (updatedProfile) => {
            queryClient.setQueryData(profileKeys.me(user?.account_id), updatedProfile);
        },
    });
}

export function useProfile() {
    const profileQuery = useMyProfile();
    const updateMutation = useUpdateProfile();

    return {
        profile: profileQuery.data ?? null,
        isLoading: profileQuery.isLoading,
        isFetching: profileQuery.isFetching,
        error: profileQuery.error,

        updateProfile: updateMutation.mutateAsync,
        updatePending: updateMutation.isPending,
        updateError: updateMutation.error,

        refetch: profileQuery.refetch,
    };
}
