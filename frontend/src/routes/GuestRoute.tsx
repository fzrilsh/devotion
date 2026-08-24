import { Navigate, Outlet } from "react-router-dom";
import { motion } from "motion/react";
import { useAuth } from "@hooks/useAuth";
import Loading from "@components/common/Loading";
import { fadeIn } from "@data/animation";

export default function GuestRoute() {
    const { isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return <Loading />;
    }

    if (isAuthenticated) {
        return (
            <motion.div {...fadeIn}>
                <Navigate to="/dashboard" replace />
            </motion.div>
        );
    }

    return (
        <motion.div {...fadeIn}>
            <Outlet />
        </motion.div>
    );
}
