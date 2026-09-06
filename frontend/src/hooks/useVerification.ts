import { getMyVerificationRequests, submitVerification, uploadFile, type FileKind } from "@api/verification";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const verificationKeys = {
    mine: ["verification", "mine"] as const,
};

export function useMyVerificationRequests() {
    return useQuery({
        queryKey: verificationKeys.mine,
        queryFn: getMyVerificationRequests,
        staleTime: 30 * 1000,
    });
}

export function useUploadFile() {
    return useMutation({
        mutationFn: ({ kind, file }: { kind: FileKind; file: File }) => uploadFile(kind, file),
    });
}

export function useSubmitVerification() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: submitVerification,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: verificationKeys.mine });
            queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
        },
    });
}
