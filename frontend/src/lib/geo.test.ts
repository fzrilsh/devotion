import { formatDistanceKm, haversineKm, toCoordinate } from "./geo";

describe("toCoordinate", () => {
    it("menolak koordinat kosong maupun di luar rentang", () => {
        expect(toCoordinate(null, 106.8)).toBeNull();
        expect(toCoordinate(-6.2, null)).toBeNull();
        expect(toCoordinate(95, 106.8)).toBeNull();
        expect(toCoordinate(-6.2, 200)).toBeNull();
        expect(toCoordinate(-6.2, 106.8)).toEqual({ latitude: -6.2, longitude: 106.8 });
    });
});

describe("haversineKm FR-064", () => {
    it("jarak ke titik yang sama adalah nol", () => {
        const jakarta = { latitude: -6.2088, longitude: 106.8456 };
        expect(haversineKm(jakarta, jakarta)).toBeCloseTo(0, 6);
    });

    it("Jakarta ke Bandung sekitar 120 km", () => {
        const km = haversineKm({ latitude: -6.2088, longitude: 106.8456 }, { latitude: -6.9175, longitude: 107.6191 });
        expect(km).toBeGreaterThan(110);
        expect(km).toBeLessThan(130);
    });

    it("simetris pada kedua arah", () => {
        const a = { latitude: -6.2088, longitude: 106.8456 };
        const b = { latitude: -7.2575, longitude: 112.7521 };
        expect(haversineKm(a, b)).toBeCloseTo(haversineKm(b, a), 9);
    });
});

describe("formatDistanceKm", () => {
    it("menyebut kurang dari 1 km untuk jarak sangat dekat", () => {
        expect(formatDistanceKm(0.4)).toBe("kurang dari 1 km");
    });

    it("memakai koma sebagai pemisah desimal", () => {
        expect(formatDistanceKm(4.25)).toBe("4,3 km");
    });

    it("membulatkan jarak jauh ke kilometer bulat", () => {
        expect(formatDistanceKm(662.4)).toContain("662");
    });
});
