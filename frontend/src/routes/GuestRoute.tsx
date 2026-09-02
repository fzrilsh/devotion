import { Navigate, Outlet, useLocation } from "react-router-dom";
import { motion } from "motion/react";
import { useAuth } from "@hooks/useAuth";
import Loading from "@components/common/Loading";
import { fadeIn } from "@data/animation";
import { getDefaultRedirectPath } from "@lib/roles";

const verificationPaths = ["/auth/verify-email", "/auth/verify-phone"];

export default function GuestRoute() {
    const { isAuthenticated, isLoading, user } = useAuth();
    const { pathname } = useLocation();

    if (isLoading) {
        return <Loading />;
    }

    if (isAuthenticated) {
        if (verificationPaths.includes(pathname)) {
            const alreadyVerified = (pathname === "/auth/verify-email" && user?.email_verified) || (pathname === "/auth/verify-phone" && user?.phone_verified);

            if (alreadyVerified) {
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
