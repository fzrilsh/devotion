import "leaflet/dist/leaflet.css";

import { useEffect } from "react";
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

function ClickHandler({ onChange, disabled }: { onChange: (latitude: number, longitude: number) => void; disabled?: boolean }) {
    useMapEvents({
        click(event) {
            if (disabled) return;
            onChange(Number(event.latlng.lat.toFixed(6)), Number(event.latlng.lng.toFixed(6)));
        },
    });

    return null;
}

function Recenter({ position, hasMarker }: { position: [number, number]; hasMarker: boolean }) {
    const map = useMap();

    useEffect(() => {
        map.setView(position, hasMarker ? MARKER_ZOOM : DEFAULT_ZOOM);
    }, [map, position, hasMarker]);

    return null;
}

export default function LocationPicker({ latitude, longitude, onChange, disabled = false }: LocationPickerProps) {
    const hasMarker = latitude != null && longitude != null;
    const center: [number, number] = hasMarker ? [latitude, longitude] : DEFAULT_CENTER;

    return (
        <div className="h-72 w-full overflow-hidden rounded-2xl border border-slate-200">
            <MapContainer center={center} zoom={hasMarker ? MARKER_ZOOM : DEFAULT_ZOOM} scrollWheelZoom={!disabled} className="h-full w-full">
                <TileLayer attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>' url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />

                <ClickHandler onChange={onChange} disabled={disabled} />
                <Recenter position={center} hasMarker={hasMarker} />

                {hasMarker ? <Marker position={[latitude, longitude]} /> : null}
            </MapContainer>
        </div>
    );
}
