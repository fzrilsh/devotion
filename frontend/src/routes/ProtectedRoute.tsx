import { Navigate, Outlet } from "react-router-dom";
import { motion } from "motion/react";
import { useAuth } from "@hooks/useAuth";
import { useMeRoles } from "@hooks/useMeRoles";
import Loading from "@components/common/Loading";
import { fadeIn } from "@data/animation";

interface ProtectedRouteProps {
    allowedRoles?: Array<"subcontractor" | "buyer" | "is_admin">;
    redirectTo?: string;
    adminOnly?: boolean;
}

export default function ProtectedRoute({ allowedRoles = ["subcontractor", "buyer", "is_admin"], redirectTo = "/", adminOnly = false }: ProtectedRouteProps) {
    const { isAuthenticated, isLoading } = useAuth();
    const { hasAnyRole } = useMeRoles();

    const effectiveAllowedRoles: Array<"subcontractor" | "buyer" | "is_admin"> = adminOnly ? ["is_admin"] : allowedRoles;

    if (isLoading) {
        return <Loading />;
    }

    if (!isAuthenticated) {
        return (
            <motion.div {...fadeIn}>
                <Navigate to="/auth/login" replace />
            </motion.div>
        );
    }

    if (!hasAnyRole(effectiveAllowedRoles)) {
        return (
            <motion.div {...fadeIn}>
                <Navigate to={redirectTo} replace />
            </motion.div>
        );
    }

    return (
        <motion.div {...fadeIn}>
            <Outlet />
        </motion.div>
    );
}
