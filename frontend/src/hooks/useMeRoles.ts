import { useAuth } from "@hooks/useAuth";
import { useMemo } from "react";

interface Roles {
    subcontractor: boolean;
    buyer: boolean;
    is_admin: boolean;
}

export function useMeRoles() {
    const { user } = useAuth();

    const roles = useMemo<Roles>(() => {
        if (!user) {
            return {
                subcontractor: false,
                buyer: false,
                is_admin: false,
            };
        }

        return {
            subcontractor: user.roles?.subcontractor ?? false,
            buyer: user.roles?.buyer ?? false,
            is_admin: user.is_admin ?? false,
        };
    }, [user]);

    const hasRole = (role: keyof Roles): boolean => {
        return roles[role] ?? false;
    };

    const hasAnyRole = (roleList: Array<keyof Roles>): boolean => {
        return roleList.some((role) => hasRole(role));
    };

    const hasAllRoles = (roleList: Array<keyof Roles>): boolean => {
        return roleList.every((role) => hasRole(role));
    };

    return {
        ...roles,
        hasRole,
        hasAnyRole,
        hasAllRoles,
    };
}
