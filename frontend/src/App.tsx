import ForgotPassword from "@pages/Auth/ForgotPassword";
import Login from "@pages/Auth/Login";
import Register from "@pages/Auth/Register";
import ResetPassword from "@pages/Auth/ResetPassword";
import VerifyEmail from "@pages/Auth/VerifyEmail";
import VerifyPhone from "@pages/Auth/VerifyPhone";
import Home from "@pages/Home";
import GuestRoute from "@routes/GuestRoute";
import ProtectedRoute from "@routes/ProtectedRoute";

import { BrowserRouter, Route, Routes } from "react-router-dom";
export default function App() {
    return (
        <BrowserRouter>
            <Routes>
                <Route path="/" element={<Home />} />
                <Route path="/auth" element={<GuestRoute />}>
                    <Route path="register" element={<Register />} />
                    <Route path="login" element={<Login />} />
                    <Route path="verify-email" element={<VerifyEmail />} />
                    <Route path="verify-phone" element={<VerifyPhone />} />
                    <Route path="forgot-password" element={<ForgotPassword />} />
                    <Route path="reset-password" element={<ResetPassword />} />
                </Route>
                <Route element={<ProtectedRoute />}>
                    <Route path="/dashboard" element={<Home />} />
                </Route>

                {/* <Route path="*" element={<NotFound />} /> */}
            </Routes>
        </BrowserRouter>
    );
}
