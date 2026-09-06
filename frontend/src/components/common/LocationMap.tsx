import "leaflet/dist/leaflet.css";

import { defaultMarkerIcon } from "@lib/leafletIcons";
import { MapContainer, Marker, Popup, TileLayer } from "react-leaflet";

type LocationMapProps = {
    latitude: number;
    longitude: number;
    label?: string;
};

export default function LocationMap({ latitude, longitude, label = "Lokasi" }: LocationMapProps) {
    return (
        <div className="h-80 w-full overflow-hidden rounded-2xl border border-slate-200 z-10">
            <MapContainer center={[latitude, longitude]} zoom={15} scrollWheelZoom={false} className="h-full w-full z-10">
                <TileLayer attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>' url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />

                <Marker position={[latitude, longitude]} icon={defaultMarkerIcon}>
                    <Popup>{label}</Popup>
                </Marker>
            </MapContainer>
        </div>
    );
}
