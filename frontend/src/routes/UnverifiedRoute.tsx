import { Navigate, Outlet } from "react-router-dom";
import { motion } from "motion/react";
import { useAuth } from "@hooks/useAuth";
import Loading from "@components/common/Loading";
import { fadeIn } from "@data/animation";

// Verification pages are only useful while an account has an
// unverified channel. Fully verified accounts are redirected away so a
// direct URL cannot show a form that is no longer usable.
export default function UnverifiedRoute() {
    const { user, isAuthenticated, isLoading } = useAuth();

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

    if (user?.email_verified && user?.phone_verified) {
        return (
            <motion.div {...fadeIn}>
                <Navigate to="/profile/me" replace />
            </motion.div>
        );
    }

    return (
        <motion.div {...fadeIn}>
            <Outlet />
        </motion.div>
    );
}
