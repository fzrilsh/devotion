import { getHealth } from "@api/system";
import { useQuery } from "@tanstack/react-query";

export const systemKeys = {
    health: ["system", "health"] as const,
};

export function useHealth() {
    return useQuery({
        queryKey: systemKeys.health,
        queryFn: getHealth,
        staleTime: 15 * 1000,
        refetchInterval: 60 * 1000,
        retry: false,
    });
}
