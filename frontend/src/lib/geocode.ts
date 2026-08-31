import { toCoordinate, type Coordinate } from "./geo";

// Pencarian nama tempat memakai Nominatim milik OpenStreetMap, sumber yang sama
// dengan tile petanya, jadi tidak ada kunci API dan tidak ada dependency baru.
// Hasilnya hanya membantu menggeser peta; titik yang tersimpan tetap yang
// ditandai pengguna (FR-057).
const NOMINATIM_SEARCH_URL = "https://nominatim.openstreetmap.org/search";

// Pasar yang dilayani hanya Indonesia, jadi hasil dibatasi ke satu negara supaya
// nama kota yang sama di negara lain tidak ikut muncul.
const COUNTRY_CODES = "id";
const RESULT_LIMIT = 6;

// Nominatim meminta paling banyak satu permintaan per detik. Jeda ketik di
// pemanggil menjaga batas itu; ini hanya batas panjang kueri yang layak dikirim.
export const MIN_QUERY_LENGTH = 3;

// Kotak batas hasil: selatan, barat, utara, timur. Dipakai untuk mengepaskan
// pandangan peta ke wilayah hasil, supaya kota tampil sebagai kota dan alamat
// tampil sebagai alamat tanpa menebak tingkat zoom.
export type GeocodeBounds = [[number, number], [number, number]];

export type GeocodePlace = {
    id: string;
    label: string;
    point: Coordinate;
    bounds: GeocodeBounds | null;
};

type NominatimItem = {
    place_id?: number | string;
    display_name?: string;
    lat?: string;
    lon?: string;
    boundingbox?: string[];
};

function parseBounds(raw: unknown): GeocodeBounds | null {
    if (!Array.isArray(raw) || raw.length !== 4) return null;

    const [south, north, west, east] = raw.map((value) => Number(value));

    if (![south, north, west, east].every((value) => Number.isFinite(value))) return null;
    if (!toCoordinate(south, west) || !toCoordinate(north, east)) return null;

    return [
        [south, west],
        [north, east],
    ];
}

// parsePlaces menyaring baris yang koordinatnya tidak sah, bukan hanya memetakan
// bentuknya, karena respons layanan luar tidak dijamin lengkap.
export function parsePlaces(payload: unknown): GeocodePlace[] {
    if (!Array.isArray(payload)) return [];

    const places: GeocodePlace[] = [];

    for (const raw of payload as NominatimItem[]) {
        const point = toCoordinate(Number(raw?.lat), Number(raw?.lon));
        const label = typeof raw?.display_name === "string" ? raw.display_name.trim() : "";

        if (!point || !label) continue;

        places.push({
            id: String(raw?.place_id ?? `${point.latitude},${point.longitude}`),
            label,
            point,
            bounds: parseBounds(raw?.boundingbox),
        });
    }

    return places;
}

export async function searchPlaces(query: string, signal?: AbortSignal): Promise<GeocodePlace[]> {
    const trimmed = query.trim();
    if (trimmed.length < MIN_QUERY_LENGTH) return [];

    const params = new URLSearchParams({
        q: trimmed,
        format: "jsonv2",
        addressdetails: "0",
        limit: String(RESULT_LIMIT),
        countrycodes: COUNTRY_CODES,
        "accept-language": "id",
    });

    const response = await fetch(`${NOMINATIM_SEARCH_URL}?${params.toString()}`, { signal, headers: { Accept: "application/json" } });

    if (!response.ok) throw new Error(`Nominatim ${response.status}`);

    return parsePlaces(await response.json());
}
