import { MIN_QUERY_LENGTH, searchPlaces, type GeocodePlace } from "@lib/geocode";
import { useEffect, useState } from "react";

const DEBOUNCE_MS = 600;

type PlaceSearchState = {
    places: GeocodePlace[];
    isSearching: boolean;
    error: string;
};

type ResolvedSearch = { query: string; places: GeocodePlace[]; error: string };

export function usePlaceSearch(query: string): PlaceSearchState {
    const trimmed = query.trim();
    const active = trimmed.length >= MIN_QUERY_LENGTH;

    const [resolved, setResolved] = useState<ResolvedSearch>({ query: "", places: [], error: "" });

    useEffect(() => {
        if (!active) return;

        const controller = new AbortController();

        const timer = window.setTimeout(async () => {
            try {
                const places = await searchPlaces(trimmed, controller.signal);
                setResolved({ query: trimmed, places, error: "" });
            } catch {
                if (controller.signal.aborted) return;

                setResolved({ query: trimmed, places: [], error: "Pencarian tempat sedang tidak dapat diakses. Tandai titik langsung pada peta." });
            }
        }, DEBOUNCE_MS);

        return () => {
            window.clearTimeout(timer);
            controller.abort();
        };
    }, [trimmed, active]);

    if (!active) return { places: [], isSearching: false, error: "" };

    if (resolved.query !== trimmed) return { places: [], isSearching: true, error: "" };

    return { places: resolved.places, isSearching: false, error: resolved.error };
}
