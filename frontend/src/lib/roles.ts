export function getDefaultRedirectPath(roles: { subcontractor?: boolean; buyer?: boolean; is_admin?: boolean }): string {
    if (roles.is_admin) {
        return "/admin";
    }
    if (roles.subcontractor) {
        return "/listing";
    }
    if (roles.buyer) {
        return "/search";
    }
    return "/";
}
