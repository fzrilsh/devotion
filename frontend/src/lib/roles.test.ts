import { getDefaultRedirectPath } from "./roles";

describe("getDefaultRedirectPath", () => {
    it("mengarahkan admin ke panel admin", () => {
        expect(getDefaultRedirectPath({ is_admin: true })).toBe("/admin");
    });

    it("admin menang bila punya peran lain sekaligus", () => {
        expect(getDefaultRedirectPath({ is_admin: true, subcontractor: true, buyer: true })).toBe("/admin");
    });

    it("mengarahkan subkontraktor ke listing", () => {
        expect(getDefaultRedirectPath({ subcontractor: true })).toBe("/listing");
    });

    it("mengarahkan buyer ke pencarian", () => {
        expect(getDefaultRedirectPath({ buyer: true })).toBe("/search");
    });

    it("subkontraktor menang bila punya dua peran", () => {
        expect(getDefaultRedirectPath({ subcontractor: true, buyer: true })).toBe("/listing");
    });

    it("kembali ke beranda bila tidak ada peran", () => {
        expect(getDefaultRedirectPath({})).toBe("/");
    });
});
