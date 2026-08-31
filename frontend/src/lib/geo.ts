// Jarak dihitung sendiri dengan haversine, tanpa PostGIS maupun dependency peta,
// dan bersifat informatif saja: tidak menyaring dan tidak mengubah urutan hasil
// pencarian (FR-064).
const EARTH_RADIUS_KM = 6371;

function toRadians(degrees: number): number {
    return (degrees * Math.PI) / 180;
}

export type Coordinate = { latitude: number; longitude: number };

export function toCoordinate(latitude?: number | null, longitude?: number | null): Coordinate | null {
    if (latitude == null || longitude == null) return null;
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return null;
    if (Math.abs(latitude) > 90 || Math.abs(longitude) > 180) return null;

    return { latitude, longitude };
}

export function haversineKm(from: Coordinate, to: Coordinate): number {
    const deltaLat = toRadians(to.latitude - from.latitude);
    const deltaLon = toRadians(to.longitude - from.longitude);

    const a = Math.sin(deltaLat / 2) ** 2 + Math.cos(toRadians(from.latitude)) * Math.cos(toRadians(to.latitude)) * Math.sin(deltaLon / 2) ** 2;

    return 2 * EARTH_RADIUS_KM * Math.asin(Math.min(1, Math.sqrt(a)));
}

export function formatDistanceKm(km: number): string {
    if (km < 1) return "kurang dari 1 km";
    if (km < 10) return `${km.toFixed(1).replace(".", ",")} km`;

    return `${Math.round(km).toLocaleString("id-ID")} km`;
}
