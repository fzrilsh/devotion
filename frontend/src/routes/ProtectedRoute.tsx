import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "@hooks/useAuth";

export default function ProtectedRoute() {
    const { isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return <div>Memuat...</div>;
    }

    return isAuthenticated ? <Outlet /> : <Navigate to="/auth/login" replace />;
}
