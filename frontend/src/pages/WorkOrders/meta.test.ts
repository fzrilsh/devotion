import type { WorkOrderStatus } from "@api/workOrders";
import { getWorkOrderSide, paymentDirectionForSide, transitionsForSide } from "./meta";

const BUYER = "11111111-1111-1111-1111-111111111111";
const SUB = "22222222-2222-2222-2222-222222222222";

const order = { buyer_profile_id: BUYER, subcontractor_profile_id: SUB };

describe("getWorkOrderSide", () => {
    it("mengenali pemberi order dan subkontraktor dari profile_id sendiri", () => {
        expect(getWorkOrderSide(order, BUYER)).toBe("buyer");
        expect(getWorkOrderSide(order, SUB)).toBe("subcontractor");
    });

    it("mengembalikan null untuk pihak luar dan untuk profil yang belum diketahui", () => {
        expect(getWorkOrderSide(order, "33333333-3333-3333-3333-333333333333")).toBeNull();
        expect(getWorkOrderSide(order, null)).toBeNull();
    });
});

// FR-039: tombol aksi dirender dari allowed_transitions. Penyaring per pihak di
// bawah hanya membuang transisi yang bukan urusan pihak yang melihat halaman, dan
// tidak pernah menambah transisi yang tidak dikirim backend.
describe("transitionsForSide_FR039_FR044", () => {
    const all: WorkOrderStatus[] = ["production", "completed", "shipped", "confirmed", "cancelled", "in_mediation"];

    it("pemberi order hanya melihat konfirmasi, pembatalan, dan sengketa", () => {
        expect(transitionsForSide(all, "buyer")).toEqual(["confirmed", "cancelled", "in_mediation"]);
    });

    it("subkontraktor hanya melihat perubahan status produksi, pembatalan, dan sengketa", () => {
        expect(transitionsForSide(all, "subcontractor")).toEqual(["production", "completed", "shipped", "cancelled", "in_mediation"]);
    });

    it("tidak menambah transisi yang tidak dikirim backend", () => {
        expect(transitionsForSide(["production"], "subcontractor")).toEqual(["production"]);
        expect(transitionsForSide(["production"], "buyer")).toEqual([]);
        expect(transitionsForSide([], "subcontractor")).toEqual([]);
    });

    it("pihak luar tidak melihat aksi apa pun", () => {
        expect(transitionsForSide(all, null)).toEqual([]);
    });
});

// FR-041: kedua pihak mencatat pernyataan pembayaran, masing-masing dari sisinya.
describe("paymentDirectionForSide_FR041", () => {
    it("pemberi order menyatakan sudah membayar, subkontraktor sudah menerima", () => {
        expect(paymentDirectionForSide.buyer).toBe("sent");
        expect(paymentDirectionForSide.subcontractor).toBe("received");
    });
});
