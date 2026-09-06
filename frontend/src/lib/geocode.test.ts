import { parsePlaces } from "./geocode";

describe("parsePlaces", () => {
    it("memetakan hasil Nominatim beserta kotak batasnya", () => {
        const places = parsePlaces([
            {
                place_id: 12345,
                display_name: "Kota Bandung, Jawa Barat, Indonesia",
                lat: "-6.9175",
                lon: "107.6191",
                boundingbox: ["-6.98", "-6.85", "107.55", "107.71"],
            },
        ]);

        expect(places).toHaveLength(1);
        expect(places[0].id).toBe("12345");
        expect(places[0].label).toBe("Kota Bandung, Jawa Barat, Indonesia");
        expect(places[0].point).toEqual({ latitude: -6.9175, longitude: 107.6191 });
        expect(places[0].bounds).toEqual([
            [-6.98, 107.55],
            [-6.85, 107.71],
        ]);
    });

    it("membuang baris tanpa koordinat sah atau tanpa nama", () => {
        expect(parsePlaces([{ display_name: "Tanpa koordinat" }])).toEqual([]);
        expect(parsePlaces([{ display_name: "Di luar rentang", lat: "95", lon: "107.6" }])).toEqual([]);
        expect(parsePlaces([{ display_name: "   ", lat: "-6.2", lon: "106.8" }])).toEqual([]);
    });

    it("kotak batas yang tidak lengkap menjadi null, titiknya tetap dipakai", () => {
        const places = parsePlaces([{ display_name: "Jalan Braga", lat: "-6.9175", lon: "107.6191", boundingbox: ["-6.98", "-6.85"] }]);

        expect(places).toHaveLength(1);
        expect(places[0].bounds).toBeNull();
    });

    it("respons yang bukan array tidak melempar", () => {
        expect(parsePlaces(null)).toEqual([]);
        expect(parsePlaces({ error: "rate limited" })).toEqual([]);
    });
});
