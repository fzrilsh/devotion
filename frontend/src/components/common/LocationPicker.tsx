import "leaflet/dist/leaflet.css";

import { usePlaceSearch } from "@hooks/usePlaceSearch";
import type { Coordinate } from "@lib/geo";
import type { GeocodeBounds, GeocodePlace } from "@lib/geocode";
import { MIN_QUERY_LENGTH } from "@lib/geocode";
import { cn } from "@lib/utils";
import { useEffect, useRef, useState } from "react";
import { LuLoaderCircle, LuMapPin, LuSearch, LuX } from "react-icons/lu";
import { MapContainer, Marker, TileLayer, useMap, useMapEvents } from "react-leaflet";

type LocationPickerProps = {
    latitude: number | null;
    longitude: number | null;
    onChange: (latitude: number, longitude: number) => void;
    disabled?: boolean;
};

const DEFAULT_CENTER: [number, number] = [-2.5489, 118.0149];
const DEFAULT_ZOOM = 4;
const MARKER_ZOOM = 13;

// Batas zoom saat mengepaskan pandangan ke hasil pencarian. Kotak batas sebuah
// kota bisa sangat kecil (satu kelurahan) atau sangat luas (satu kabupaten);
// tanpa batas ini, hasil sempit membuat peta melompat ke zoom maksimum.
const SEARCH_MAX_ZOOM = 16;

const inputClassName = "w-full rounded-xl border border-slate-300 bg-white py-2.5 pl-10 pr-10 text-sm text-slate-800 outline-none transition-all placeholder:text-slate-400 focus:border-industrial-blue-500 focus:ring-2 focus:ring-industrial-blue-500/10 disabled:cursor-not-allowed disabled:bg-slate-50";

// focusTarget adalah satu perintah gerak untuk peta. Objeknya dibuat baru tiap
// kali supaya memilih tempat yang sama dua kali tetap menggerakkan peta.
type FocusTarget = { point: Coordinate; bounds: GeocodeBounds | null };

function ClickHandler({ onChange, disabled }: { onChange: (latitude: number, longitude: number) => void; disabled?: boolean }) {
    useMapEvents({
        click(event) {
            if (disabled) return;
            onChange(Number(event.latlng.lat.toFixed(6)), Number(event.latlng.lng.toFixed(6)));
        },
    });

    return null;
}

// MapController hanya bergerak saat diperintah lewat focus, bukan setiap kali
// titiknya berubah. Menggerakkan peta pada tiap perubahan titik berarti setiap
// klik memaksa zoom kembali ke MARKER_ZOOM, jadi pengguna yang sudah memperbesar
// peta untuk menandai lokasi dengan tepat justru terlempar keluar.
function MapController({ focus }: { focus: FocusTarget | null }) {
    const map = useMap();

    useEffect(() => {
        if (!focus) return;

        if (focus.bounds) {
            map.fitBounds(focus.bounds, { maxZoom: SEARCH_MAX_ZOOM });
            return;
        }

        // Zoom yang sedang dipakai dipertahankan bila sudah lebih dekat daripada
        // MARKER_ZOOM, jadi berpindah tempat tidak pernah memperkecil peta.
        map.setView([focus.point.latitude, focus.point.longitude], Math.max(map.getZoom(), MARKER_ZOOM));
    }, [map, focus]);

    return null;
}

function PlaceResults({ places, isSearching, error, query, onSelect }: { places: GeocodePlace[]; isSearching: boolean; error: string; query: string; onSelect: (place: GeocodePlace) => void }) {
    const trimmed = query.trim();

    if (error) {
        return (
            <p className="mt-2 text-xs text-amber-600" role="status">
                {error}
            </p>
        );
    }

    if (trimmed.length > 0 && trimmed.length < MIN_QUERY_LENGTH) {
        return <p className="mt-2 text-xs text-slate-400">Tuliskan minimal {MIN_QUERY_LENGTH} karakter.</p>;
    }

    if (isSearching) {
        return (
            <p className="mt-2 flex items-center gap-1.5 text-xs text-slate-400" role="status">
                <LuLoaderCircle className="size-3.5 animate-spin" aria-hidden />
                Mencari tempat...
            </p>
        );
    }

    if (places.length === 0) {
        if (trimmed.length < MIN_QUERY_LENGTH) return null;

        return (
            <p className="mt-2 text-xs text-slate-400" role="status">
                Tempat tidak ditemukan. Coba nama lain, atau tandai titiknya langsung pada peta.
            </p>
        );
    }

    return (
        <ul className="mt-2 max-h-44 overflow-y-auto rounded-xl border border-slate-200 bg-white py-1" aria-label="Hasil pencarian tempat">
            {places.map((place) => (
                <li key={place.id}>
                    <button type="button" onClick={() => onSelect(place)} className="flex w-full cursor-pointer items-start gap-2 px-3 py-2 text-left transition hover:bg-industrial-blue-500/5">
                        <LuMapPin className="mt-0.5 size-3.5 shrink-0 text-industrial-blue-500" aria-hidden />
                        <span className="text-xs leading-5 text-slate-600">{place.label}</span>
                    </button>
                </li>
            ))}
        </ul>
    );
}

export default function LocationPicker({ latitude, longitude, onChange, disabled = false }: LocationPickerProps) {
    const hasMarker = latitude != null && longitude != null;
    const center: [number, number] = hasMarker ? [latitude, longitude] : DEFAULT_CENTER;

    const [query, setQuery] = useState("");
    const [focus, setFocus] = useState<FocusTarget | null>(null);
    const { places, isSearching, error } = usePlaceSearch(disabled ? "" : query);

    // Titik yang tersimpan bisa datang setelah peta terpasang (profil dimuat
    // belakangan). Geser satu kali ke titik itu, lalu serahkan kendali zoom
    // sepenuhnya ke pengguna.
    const initialFocusDone = useRef(false);

    useEffect(() => {
        if (initialFocusDone.current || latitude == null || longitude == null) return;

        initialFocusDone.current = true;
        setFocus({ point: { latitude, longitude }, bounds: null });
    }, [latitude, longitude]);

    // Memilih hasil pencarian menggeser peta saja, tidak memindahkan titiknya.
    // Titik lokasi usaha tampil publik (FR-057), jadi yang tersimpan harus yang
    // sengaja ditandai pengguna, bukan pusat wilayah yang kebetulan cocok namanya.
    function selectPlace(place: GeocodePlace) {
        setFocus({ point: place.point, bounds: place.bounds });
        setQuery(place.label);
    }

    return (
        <div className="space-y-3">
            <div>
                <label htmlFor="location_search" className="sr-only">
                    Cari nama tempat
                </label>

                <div className="relative">
                    <LuSearch className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400" aria-hidden />

                    <input id="location_search" type="search" value={query} disabled={disabled} onChange={(event) => setQuery(event.target.value)} className={inputClassName} placeholder="Cari kota, jalan, atau nama tempat" autoComplete="off" />

                    {query ? (
                        <button
                            type="button"
                            onClick={() => setQuery("")}
                            disabled={disabled}
                            className="absolute right-2.5 top-1/2 -translate-y-1/2 cursor-pointer rounded-lg p-1.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-600 disabled:cursor-not-allowed"
                            aria-label="Kosongkan pencarian"
                        >
                            <LuX className="size-3.5" aria-hidden />
                        </button>
                    ) : null}
                </div>

                <PlaceResults places={places} isSearching={isSearching} error={error} query={query} onSelect={selectPlace} />
            </div>

            <div className={cn("h-72 w-full overflow-hidden rounded-2xl border border-slate-200", disabled && "opacity-70")}>
                <MapContainer center={center} zoom={hasMarker ? MARKER_ZOOM : DEFAULT_ZOOM} scrollWheelZoom={!disabled} className="h-full w-full">
                    <TileLayer attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>' url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />

                    <ClickHandler onChange={onChange} disabled={disabled} />
                    <MapController focus={focus} />

                    {hasMarker ? <Marker position={[latitude, longitude]} /> : null}
                </MapContainer>
            </div>
        </div>
    );
}
