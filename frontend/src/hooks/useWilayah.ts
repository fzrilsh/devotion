import { getCities, getProvinces } from "@api/master";
import { useQuery } from "@tanstack/react-query";

const wilayahKeys = {
    provinces: ["wilayah", "provinces"] as const,
    cities: (provinceCode?: string) => ["wilayah", "cities", provinceCode] as const,
};

export function useProvinces() {
    return useQuery({
        queryKey: wilayahKeys.provinces,
        queryFn: getProvinces,
        staleTime: 60 * 60 * 1000,
        gcTime: 24 * 60 * 60 * 1000,
        retry: false,
    });
}

export function useCities(provinceCode?: string, options?: { enabled?: boolean }) {
    return useQuery({
        queryKey: wilayahKeys.cities(provinceCode),
        queryFn: () => getCities(provinceCode),
        enabled: !!provinceCode && options?.enabled !== false,
        staleTime: 60 * 60 * 1000,
        gcTime: 24 * 60 * 60 * 1000,
        retry: false,
    });
}

export function useWilayah(provinceCode?: string) {
    const provincesQuery = useProvinces();
    const citiesQuery = useCities(provinceCode, { enabled: !!provinceCode });

    return {
        provinces: provincesQuery.data ?? [],
        cities: citiesQuery.data ?? [],
        isLoading: provincesQuery.isLoading || citiesQuery.isLoading,
        isFetching: provincesQuery.isFetching || citiesQuery.isFetching,
        error: provincesQuery.error || citiesQuery.error,

        refetchProvinces: provincesQuery.refetch,
        refetchCities: citiesQuery.refetch,

        getProvinceName: (code: string) => {
            const province = provincesQuery.data?.find((p) => p.code === code);
            return province?.name ?? code;
        },

        getCityName: (code: string) => {
            const city = citiesQuery.data?.find((c) => c.code === code);
            return city?.name ?? code;
        },
    };
}
