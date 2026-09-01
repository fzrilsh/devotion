import { createBrowserRouter, createRoutesFromElements, Route, RouterProvider } from "react-router-dom";

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
    const router = createBrowserRouter(
        createRoutesFromElements(
            <Route element={<PageTitle />}>
                <Route path="/" element={<Home />} handle={{ title: "Beranda" }} />
                <Route path="/profile/:profileId" element={<PublicProfile />} handle={{ title: "Profil Pengguna" }} />
                <Route path="/tentang" element={<About />} handle={{ title: "Tentang Kami" }} />
                <Route path="/bantuan" element={<Help />} handle={{ title: "Bantuan" }} />
                <Route path="/syarat-ketentuan" element={<Terms />} handle={{ title: "Syarat & Ketentuan" }} />
                <Route path="/kebijakan-privasi" element={<Privacy />} handle={{ title: "Kebijakan Privasi" }} />

                <Route path="/auth" element={<GuestRoute />}>
                    <Route path="register" element={<Register />} handle={{ title: "Daftar" }} />
                    <Route path="login" element={<Login />} handle={{ title: "Masuk" }} />
                    <Route path="verify-email" element={<VerifyEmail />} handle={{ title: "Verifikasi Email" }} />
                    <Route path="verify-phone" element={<VerifyPhone />} handle={{ title: "Verifikasi Nomor Telepon" }} />
                    <Route path="forgot-password" element={<ForgotPassword />} handle={{ title: "Lupa Kata Sandi" }} />
                    <Route path="reset-password" element={<ResetPassword />} handle={{ title: "Atur Ulang Kata Sandi" }} />
                </Route>

                <Route element={<ProtectedRoute />}>
                    <Route element={<AppLayout />}>
                        <Route path="/profile/me" element={<MyProfile />} handle={{ title: "Profil Saya" }} />
                        <Route path="/verification" element={<Verification />} handle={{ title: "Verifikasi Akun" }} />
                        <Route path="/notifications" element={<Notifications />} handle={{ title: "Notifikasi" }} />
                        <Route path="/notifications/preferences" element={<NotificationPreferences />} handle={{ title: "Preferensi Notifikasi" }} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute allowedRoles={["buyer", "subcontractor", "is_admin"]} />}>
                    <Route element={<AppLayout />}>
                        <Route path="/orders" element={<WorkOrderList />} handle={{ title: "Pesanan" }} />
                        <Route path="/orders/:workOrderId" element={<WorkOrderDetail />} handle={{ title: "Detail Pesanan" }} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute allowedRoles={["subcontractor", "is_admin"]} redirectTo="/profile/me" />}>
                    <Route element={<AppLayout />}>
                        <Route path="/listing" element={<Listing />} handle={{ title: "Listing Jasa" }} />
                        <Route path="/listing/calendar" element={<ListingCalendar />} handle={{ title: "Kalender Listing" }} />
                        <Route path="/requests/incoming" element={<IncomingRequests />} handle={{ title: "Permintaan Masuk" }} />
                        <Route path="/requests/incoming/:requestId" element={<IncomingRequestDetail />} handle={{ title: "Detail Permintaan Masuk" }} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute allowedRoles={["buyer", "is_admin"]} redirectTo="/profile/me" />}>
                    <Route element={<AppLayout />}>
                        <Route path="/search" element={<Search />} handle={{ title: "Cari Jasa" }} />
                        <Route path="/quota-requests/new" element={<CreateQuotaRequest />} handle={{ title: "Buat Permintaan Kuota" }} />
                        <Route path="/quota-requests" element={<SentRequests />} handle={{ title: "Permintaan Saya" }} />
                        <Route path="/quota-requests/:requestId" element={<SentRequestDetail />} handle={{ title: "Detail Permintaan Terkirim" }} />
                    </Route>
                </Route>

                <Route element={<ProtectedRoute adminOnly />}>
                    <Route path="/admin" element={<AppLayout />}>
                        <Route index element={<AdminDashboard />} handle={{ title: "Dashboard Admin" }} />
                        <Route path="verification" element={<AdminVerificationQueue />} handle={{ title: "Verifikasi Pengguna" }} />
                        <Route path="master/items" element={<AdminMasterItems />} handle={{ title: "Master Item" }} />
                        <Route path="proposals" element={<AdminProposals />} handle={{ title: "Proposal" }} />
                        <Route path="late-orders" element={<AdminLateOrders />} handle={{ title: "Pesanan Terlambat" }} />
                        <Route path="orders/:workOrderId" element={<AdminOrderDetail />} handle={{ title: "Detail Pesanan Admin" }} />
                        <Route path="disputes" element={<AdminDisputes />} handle={{ title: "Sengketa" }} />
                        <Route path="reviews" element={<AdminReviewsModeration />} handle={{ title: "Moderasi Ulasan" }} />
                        <Route path="whatsapp" element={<AdminWhatsApp />} handle={{ title: "WhatsApp" }} />
                        <Route path="system" element={<AdminSystem />} handle={{ title: "Pengaturan Sistem" }} />
                    </Route>
                </Route>
                
                <Route path="*" element={<NotFound />} />
            </Route>,
        ),
    );

    return <RouterProvider router={router} />;
}
