import { Navigate, Outlet } from "react-router-dom";
import { motion } from "motion/react";
import { useAuth } from "@hooks/useAuth";
import Loading from "@components/common/Loading";
import { fadeIn } from "@data/animation";
import { getDefaultRedirectPath } from "@lib/roles";

export default function GuestRoute() {
    const { isAuthenticated, isLoading, user } = useAuth();

    if (isLoading) {
        return <Loading />;
    }

    if (isAuthenticated) {
        return (
            <motion.div {...fadeIn}>
                <Navigate to={user ? getDefaultRedirectPath({ subcontractor: user.roles?.subcontractor, buyer: user.roles?.buyer, is_admin: user.is_admin }) : "/"} replace />
            </motion.div>
        );
    }

    return (
        <motion.div {...fadeIn}>
            <Outlet />
        </motion.div>
    );
}
