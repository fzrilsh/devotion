import "leaflet/dist/leaflet.css";

import { usePlaceSearch } from "@hooks/usePlaceSearch";
import type { Coordinate } from "@lib/geo";
import type { GeocodeBounds, GeocodePlace } from "@lib/geocode";
import { MIN_QUERY_LENGTH } from "@lib/geocode";
import { cn } from "@lib/utils";
import L from "leaflet";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { LuLoaderCircle, LuLocateFixed, LuMapPin, LuSearch, LuX } from "react-icons/lu";
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

        map.setView([focus.point.latitude, focus.point.longitude], Math.max(map.getZoom(), MARKER_ZOOM));
    }, [map, focus]);

    return null;
}

// useLeafletControl membuat satu L.Control kosong di posisi yang diminta, lalu
// mengembalikan elemen DOM-nya. Konten sebenarnya dirender lewat React portal
// dari komponen pemanggil, jadi kontrol tetap bisa memakai hook & state biasa
// walau secara DOM ia hidup di luar pohon <MapContainer>.
function useLeafletControl(position: L.ControlPosition, className = "leaflet-control") {
    const map = useMap();
    const [container, setContainer] = useState<HTMLDivElement | null>(null);

    useEffect(() => {
        const control = new L.Control({ position });

        control.onAdd = () => {
            const el = L.DomUtil.create("div", className) as HTMLDivElement;
            // Tanpa ini, klik atau scroll di dalam kontrol akan diteruskan ke
            // peta di bawahnya — misalnya klik pada hasil pencarian akan ikut
            // menandai titik lewat ClickHandler.
            L.DomEvent.disableClickPropagation(el);
            L.DomEvent.disableScrollPropagation(el);
            setContainer(el);
            return el;
        };

        control.addTo(map);

        return () => {
            control.remove();
            setContainer(null);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [map, position]);

    return container;
}

function PlaceResults({ places, isSearching, error, query, onSelect }: { places: GeocodePlace[]; isSearching: boolean; error: string; query: string; onSelect: (place: GeocodePlace) => void }) {
    const trimmed = query.trim();

    if (error) {
        return (
            <p className="text-xs text-amber-600" role="status">
                {error}
            </p>
        );
    }

    if (trimmed.length > 0 && trimmed.length < MIN_QUERY_LENGTH) {
        return <p className="text-xs text-slate-400">Tuliskan minimal {MIN_QUERY_LENGTH} karakter.</p>;
    }

    if (isSearching) {
        return (
            <p className="flex items-center gap-1.5 text-xs text-slate-400" role="status">
                <LuLoaderCircle className="size-3.5 animate-spin" aria-hidden />
                Mencari tempat...
            </p>
        );
    }

    if (places.length === 0) {
        if (trimmed.length < MIN_QUERY_LENGTH) return null;

        return (
            <p className="text-xs text-slate-400" role="status">
                Tempat tidak ditemukan. Coba nama lain, atau tandai titiknya langsung pada peta.
            </p>
        );
    }

    return (
        <ul className="max-h-52 divide-y divide-slate-100 overflow-y-auto rounded-lg border border-slate-100" aria-label="Hasil pencarian tempat">
            {places.map((place) => (
                <li key={place.id}>
                    <button type="button" onClick={() => onSelect(place)} className="flex w-full cursor-pointer items-start gap-2 px-3 py-2.5 text-left transition hover:bg-industrial-blue-500/5">
                        <LuMapPin className="mt-0.5 size-3.5 shrink-0 text-industrial-blue-500" aria-hidden />
                        <span className="text-xs leading-5 text-slate-600">{place.label}</span>
                    </button>
                </li>
            ))}
        </ul>
    );
}

// SearchControl menempatkan pencarian tempat sebagai kontrol peta ala
// leaflet-control-geocoder: ikon saja saat tertutup, kotak pencarian penuh
// saat diklik. Posisinya kiri bawah supaya tidak menimpa kontrol zoom di kiri
// atas maupun tombol lokasi di kanan atas. Memilih hasil hanya menggeser peta
// (lihat komentar pada selectPlace di komponen utama), titik lokasi usaha tetap
// harus ditandai sengaja oleh pengguna (FR-057).
function SearchControl({ query, setQuery, places, isSearching, error, onSelect, disabled }: { query: string; setQuery: (value: string) => void; places: GeocodePlace[]; isSearching: boolean; error: string; onSelect: (place: GeocodePlace) => void; disabled: boolean }) {
    // Kelas dasar sengaja tidak lewat "leaflet-bar", bentuknya pil/kartu
    // sendiri, bukan deret tombol kotak seperti kontrol zoom.
    const container = useLeafletControl("bottomleft", "leaflet-control");
    const [expanded, setExpanded] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (expanded) inputRef.current?.focus();
    }, [expanded]);

    function handleSelect(place: GeocodePlace) {
        onSelect(place);
        setExpanded(false);
    }

    if (!container) return null;

    return createPortal(
        expanded ? (
            // flex-col-reverse: kotak pencarian tetap di bawah (dekat sudut
            // kontrolnya) dan daftar hasil tumbuh ke atas, jadi panel tidak
            // pernah keluar dari batas bawah peta. Urutan DOM tetap input lalu
            // hasil supaya pembaca layar dan Tab membacanya berurutan.
            <div className="flex w-[min(20rem,80vw)] flex-col-reverse gap-2 rounded-2xl border border-slate-200 bg-white p-2.5 shadow-lg">
                <div className="relative">
                    <LuSearch className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400" aria-hidden />

                    <input
                        ref={inputRef}
                        type="search"
                        value={query}
                        disabled={disabled}
                        onChange={(event) => setQuery(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === "Escape") setExpanded(false);
                        }}
                        placeholder="Cari kota, jalan, atau nama tempat"
                        autoComplete="off"
                        className="w-full rounded-full border border-slate-200 bg-slate-50 py-2.5 pl-10 pr-9 text-sm text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-industrial-blue-500 focus:bg-white"
                    />

                    <button type="button" onClick={() => setExpanded(false)} className="absolute right-2 top-1/2 -translate-y-1/2 cursor-pointer rounded-full p-1.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-600" aria-label="Tutup pencarian">
                        <LuX className="size-3.5" aria-hidden />
                    </button>
                </div>

                <PlaceResults places={places} isSearching={isSearching} error={error} query={query} onSelect={handleSelect} />
            </div>
        ) : (
            <button
                type="button"
                onClick={() => setExpanded(true)}
                disabled={disabled}
                title="Cari tempat"
                aria-label="Cari tempat"
                className="flex items-center gap-2 rounded-full border border-slate-200 bg-white py-2 pl-3 pr-3.5 text-xs font-medium text-slate-600 shadow-md transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-300"
            >
                <LuSearch className="size-3.5" aria-hidden />
                Cari lokasi
            </button>
        ),
        container,
    );
}

// LocateControl langsung menandai titik dari GPS pengguna — tidak seperti hasil
// pencarian, ini "to the point": begitu izin lokasi diberikan, titiknya langsung
// tersimpan lewat onChange, bukan cuma memindahkan pandangan peta.
function LocateControl({ onLocate, disabled }: { onLocate: (latitude: number, longitude: number) => void; disabled: boolean }) {
    const map = useMap();
    const container = useLeafletControl("topright", "leaflet-control");
    const [isLocating, setIsLocating] = useState(false);
    const [locateError, setLocateError] = useState("");

    function handleLocate() {
        if (disabled || isLocating) return;

        if (!navigator.geolocation) {
            setLocateError("Perangkat ini tidak mendukung deteksi lokasi.");
            return;
        }

        setLocateError("");
        setIsLocating(true);

        navigator.geolocation.getCurrentPosition(
            (position) => {
                const latitude = Number(position.coords.latitude.toFixed(6));
                const longitude = Number(position.coords.longitude.toFixed(6));

                map.flyTo([latitude, longitude], MARKER_ZOOM);
                onLocate(latitude, longitude);
                setIsLocating(false);
            },
            (geoError) => {
                setLocateError(geoError.code === geoError.PERMISSION_DENIED ? "Izin lokasi ditolak. Aktifkan lewat pengaturan browser." : "Lokasi tidak dapat dideteksi. Coba lagi atau tandai manual di peta.");
                setIsLocating(false);
            },
            { enableHighAccuracy: true, timeout: 10000 },
        );
    }

    if (!container) return null;

    return createPortal(
        <div className="relative">
            <button
                type="button"
                onClick={handleLocate}
                disabled={disabled || isLocating}
                title="Gunakan lokasi saya"
                aria-label="Gunakan lokasi saya"
                className="flex size-9 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-600 shadow-md transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-300"
            >
                {isLocating ? <LuLoaderCircle className="size-4 animate-spin" aria-hidden /> : <LuLocateFixed className="size-4" aria-hidden />}
            </button>

            {locateError ? (
                <div className="absolute right-0 top-full mt-2 w-48 rounded-lg border border-amber-200 bg-amber-50 px-2 py-1.5 text-[11px] leading-4 text-amber-700 shadow-sm" role="status">
                    {locateError}
                </div>
            ) : null}
        </div>,
        container,
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
        <div className={cn("h-72 w-full overflow-hidden rounded-2xl border border-slate-200", disabled && "opacity-70")}>
            <MapContainer center={center} zoom={hasMarker ? MARKER_ZOOM : DEFAULT_ZOOM} scrollWheelZoom={!disabled} className="h-full w-full">
                <TileLayer attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>' url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />

                <ClickHandler onChange={onChange} disabled={disabled} />
                <MapController focus={focus} />

                <SearchControl query={query} setQuery={setQuery} places={places} isSearching={isSearching} error={error} onSelect={selectPlace} disabled={disabled} />
                <LocateControl onLocate={onChange} disabled={disabled} />

                {hasMarker ? <Marker position={[latitude, longitude]} /> : null}
            </MapContainer>
        </div>
    );
}
