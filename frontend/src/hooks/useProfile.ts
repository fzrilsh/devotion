import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getMyProfile, getPublicProfile, updateMyProfile, type ProfileUpdateRequest } from "@api/profile";
import { useAuth } from "@hooks/useAuth";

const profileKeys = {
    me: ["profile", "me"] as const,
    public: (profileId: string) => ["profile", "public", profileId] as const,
};

export function useMyProfile() {
    const { user } = useAuth();

    return useQuery({
        queryKey: profileKeys.me,
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

export function useUpdateProfile() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: ProfileUpdateRequest) => updateMyProfile(data),
        onSuccess: (updatedProfile) => {
            queryClient.setQueryData(profileKeys.me, updatedProfile);
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
