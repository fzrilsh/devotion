import { getHealth } from "@api/system";
import { useQuery } from "@tanstack/react-query";

export const systemKeys = {
    health: ["system", "health"] as const,
};

// Basis data gagal atau penyimpanan penuh dibalas 503, dan justru itu keadaan yang
// paling perlu tampil di halaman admin. Karena apiClient melemparkan galat pada
// respons non-2xx, isi badannya tidak terbaca; kartu status membaca kegagalan
// permintaan sebagai instance tidak sehat, bukan sebagai galat jaringan biasa.
export function useHealth() {
    return useQuery({
        queryKey: systemKeys.health,
        queryFn: getHealth,
        staleTime: 15 * 1000,
        refetchInterval: 60 * 1000,
        retry: false,
    });
}
