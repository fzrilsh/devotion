import { BrowserRouter, Route, Routes } from "react-router-dom";

import GuestRoute from "@routes/GuestRoute";
import ProtectedRoute from "@routes/ProtectedRoute";

import Home from "@pages/Home";
import NotFound from "@pages/NotFound";
import PublicProfile from "@pages/Profile/PublicProfile";
import About from "@pages/Static/About";
import Help from "@pages/Static/Help";
import Privacy from "@pages/Static/Privacy";
import Terms from "@pages/Static/Terms";

import Login from "@pages/Auth/Login";
import Register from "@pages/Auth/Register";
import ForgotPassword from "@pages/Auth/ForgotPassword";
import ResetPassword from "@pages/Auth/ResetPassword";
import VerifyEmail from "@pages/Auth/VerifyEmail";
import VerifyPhone from "@pages/Auth/VerifyPhone";

import AppLayout from "@components/layout/AppLayout";
import MyProfile from "@pages/Profile/MyProfile";
import Verification from "@pages/Verification";
import Notifications from "@pages/Notifications";
import NotificationPreferences from "@pages/Notifications/Preferences";

import WorkOrderList from "@pages/WorkOrders/List";
import WorkOrderDetail from "@pages/WorkOrders/Detail";

import Listing from "@pages/Listing";
import ListingCalendar from "@pages/Listing/Calendar";
import IncomingRequests from "@pages/Requests/Incoming";
import IncomingRequestDetail from "@pages/Requests/IncomingDetail";

import Search from "@pages/Search";
import CreateQuotaRequest from "@pages/Requests/Create";
import SentRequests from "@pages/Requests/Sent";
import SentRequestDetail from "@pages/Requests/SentDetail";

import AdminDashboard from "@pages/Admin/Dashboard";
import AdminVerificationQueue from "@pages/Admin/VerificationQueue";
import AdminMasterItems from "@pages/Admin/MasterItems";
import AdminProposals from "@pages/Admin/Proposals";
import AdminLateOrders from "@pages/Admin/LateOrders";
import AdminDisputes from "@pages/Admin/Disputes";
import AdminReviewsModeration from "@pages/Admin/ReviewsModeration";
import AdminWhatsApp from "@pages/Admin/WhatsApp";
import AdminSystem from "@pages/Admin/System";
import AdminOrderDetail from "@pages/Admin/OrderDetail";
import PageTitle from "@components/common/PageTitle";

export default function App() {
    return (
        <BrowserRouter>
            <PageTitle />
            <Routes>
                <Route path="/" element={<Home />} />
                <Route path="/profile/:profileId" element={<PublicProfile />} />
                <Route path="/tentang" element={<About />} />
                <Route path="/bantuan" element={<Help />} />
                <Route path="/syarat-ketentuan" element={<Terms />} />
                <Route path="/kebijakan-privasi" element={<Privacy />} />

                <Route path="/auth" element={<GuestRoute />}>
                    <Route path="register" element={<Register />} />
                    <Route path="login" element={<Login />} />
                    <Route path="verify-email" element={<VerifyEmail />} />
                    <Route path="verify-phone" element={<VerifyPhone />} />
                    <Route path="forgot-password" element={<ForgotPassword />} />
                    <Route path="reset-password" element={<ResetPassword />} />
                </Route>

                <Route element={<ProtectedRoute />}>
                    <Route element={<AppLayout />}>
                        <Route path="/profile/me" element={<MyProfile />} />
                        <Route path="/verification" element={<Verification />} />
                        <Route path="/notifications" element={<Notifications />} />
                        <Route path="/notifications/preferences" element={<NotificationPreferences />} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute allowedRoles={["buyer", "subcontractor", "is_admin"]} />}>
                    <Route element={<AppLayout />}>
                        <Route path="/orders" element={<WorkOrderList />} />
                        <Route path="/orders/:workOrderId" element={<WorkOrderDetail />} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute allowedRoles={["subcontractor", "is_admin"]} redirectTo="/profile/me" />}>
                    <Route element={<AppLayout />}>
                        <Route path="/listing" element={<Listing />} />
                        <Route path="/listing/calendar" element={<ListingCalendar />} />
                        <Route path="/requests/incoming" element={<IncomingRequests />} />
                        <Route path="/requests/incoming/:requestId" element={<IncomingRequestDetail />} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute allowedRoles={["buyer", "is_admin"]} redirectTo="/profile/me" />}>
                    <Route element={<AppLayout />}>
                        <Route path="/search" element={<Search />} />
                        <Route path="/quota-requests/new" element={<CreateQuotaRequest />} />
                        <Route path="/quota-requests" element={<SentRequests />} />
                        <Route path="/quota-requests/:requestId" element={<SentRequestDetail />} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute adminOnly />}>
                    <Route path="/admin" element={<AppLayout />}>
                        <Route index element={<AdminDashboard />} />
                        <Route path="verification" element={<AdminVerificationQueue />} />
                        <Route path="master/items" element={<AdminMasterItems />} />
                        <Route path="proposals" element={<AdminProposals />} />
                        <Route path="late-orders" element={<AdminLateOrders />} />
                        <Route path="orders/:workOrderId" element={<AdminOrderDetail />} />
                        <Route path="disputes" element={<AdminDisputes />} />
                        <Route path="reviews" element={<AdminReviewsModeration />} />
                        <Route path="whatsapp" element={<AdminWhatsApp />} />
                        <Route path="system" element={<AdminSystem />} />
                    </Route>
                </Route>
                
                <Route path="*" element={<NotFound />} />
            </Routes>
        </BrowserRouter>
    );
}
