# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Added reusable form components under `components/form`: `FormField` (text input and select), `PasswordField` (show/hide toggle), and `RoleOption` (radio-style role card).
- Added `DashboardLayout`, a generic sidebar layout for authenticated dashboards (fixed sidebar on desktop, drawer on mobile, sticky header with an account menu and logout), and `AdminLayout` with the admin navigation items.
- Added `AppLayout`, a sidebar layout for non-admin authenticated users with navigation items that follow the account roles (buyer, subcontractor, or both). All protected non-admin routes are now nested under it.
- Added placeholder content for all admin pages (dashboard, verification queue, master items, proposals, late orders, disputes, reviews moderation, WhatsApp) and nested the admin routes under `AdminLayout` in `App.tsx`.
- Added placeholder content for the verification, notifications, notification preferences, work orders, listing, listing calendar, requests (incoming, sent, create, detail), and search pages.
- Implemented the public business profile page (`/profile/:profileId`): identity header with a verified badge, reputation cards that follow the `enough_data` rule (FR-073), capacity listing details (products, machines, weekly capacity), and a friendly not-found state.
- Added `VerifiedRoute`, a route guard for pages that only make sense while verification is incomplete. Accounts with both email and phone verified are redirected away from `/verification` to their profile.
- Connected the Forgot Password and Reset Password pages to the API: requesting a recovery code (`/auth/recover/request`) from the forgot page, then confirming the code and new password (`/auth/recover/confirm`) on the reset page, with the email carried through router state and a redirect back to forgot-password when it is missing.
- Connected the Verify Phone page to the API (`/auth/verify-phone`, `/auth/resend-code`): real OTP submission, resend countdown, masked phone number from the account, and redirect to the role's default page on success.
- Added `LocationPicker`, a click-to-pin Leaflet map used on the My Profile edit form to set latitude and longitude, replacing the manual coordinate inputs. The view mode shows the saved point on a read-only map.
- Redesigned the My Profile page: an identity card with avatar initials, location, verification status badges, and a reputation panel (rating plus completion rate honoring the `enough_data` rule). The update payload now sends only `ProfileUpdateRequest` fields (`province_code` stays a form-only city filter), fixing the 422 "Format permintaan tidak sah" rejection.
- Reworked `DashboardLayout`: the header's right side now holds a notification bell with an unread badge (from `/notifications`) and the account identity linking to the profile; a profile card sits above the logout button in the sidebar footer. The Notifications sidebar item was removed in favor of the bell.
- Connected the Notifications page to the API: an event-icon list with per-type styling, an all/unread filter, relative timestamps in WIB, mark-as-read per item (which also refreshes the header badge), links to the related work order, and "Muat lebih banyak" that forwards the opaque `next_cursor` untouched.
- Connected the Notification Preferences page to the API (`GET`/`PUT /notifications/preferences`): email and WhatsApp channel toggles for non-transactional notifications, with a note that transactional notifications cannot be disabled (FR-054).
- Implemented the Admin Dashboard: four queue summary cards (verification queue, pending item proposals, open disputes, late orders) plus the five newest entries of each queue with links to the corresponding admin pages, fed by `/admin/verification`, `/admin/proposals`, `/admin/disputes`, and `/admin/late-orders`.
- Split the My Profile page by account type: admins now see an account-only profile (no business profile), while buyer/subcontractor accounts keep the business profile. Both versions show an account section with email and phone verification status and a button that opens the existing Verify Email / Verify Phone pages for unverified channels.
- Allowed logged-in users to open `/auth/verify-email` and `/auth/verify-phone` from their profile (GuestRoute no longer bounces them out; already-verified channels redirect to `/profile/me`). The Verify Email page falls back to the account email when no router state is present, and successful verification returns to the profile instead of the auth flow. `/verification` now sits under the general ProtectedRoute and `VerifiedRoute` was removed.
- Connected the Listing page to the API (`/listing/me`, `/listing/me/visibility`, `/master/products`, `/master/machines`, `/master/proposals`): a create/edit form with product chips, machine checkboxes with unit counts, weekly capacity and readiness lead inputs, inline item proposals for the master catalog, and a publish toggle with a visibility banner.
- Connected the Listing Calendar page to the API (`GET`/`PUT /listing/me/periods`): a 12-week paged grid starting from the current Monday in WIB, per-week capacity inputs with a marked-full toggle, weeks with active allocations locked against edits, and a sticky save bar that only sends changed weeks.
- Connected the Work Orders pages to the API (`/orders`): the list page has status filter chips for all seven statuses, a buyer/subcontractor role filter, and opaque-cursor pagination; the detail page renders action buttons from `allowed_transitions` and `self_cancellable` (FR-039) with panels for cancellation, payment statements (direction and date, no amounts, FR-040/042), dispute reports, and reviews, plus the auto-confirm countdown banner, allocations, payment history, and the status timeline.
- Connected the Search page to the API (`/search`): a criteria form (product, machine, quantity, deadline, maximum lead days, city/province/national scope from the account region), candidate cards with the 4-criteria score and per-criterion explanations, verified and stale-calendar badges, reputation honoring `enough_data`, and a sticky selection bar that hands the chosen listings to the quota request form. Cursor pagination forwards `next_cursor` untouched.
- Connected the quota request pages to the API (`/quota-requests`, `/candidates/{id}/offers`, `/offers/{id}/counter`, `/offers/{id}/accept`): the create form receives candidates through router state from search and blocks self-requests; the sent list shows per-request candidate summaries; the detail page compares candidates with per-candidate statuses, the full offer chain ordered by round, a counter-offer panel, and accept-offer which navigates to the created work order.
- Connected the Identity Verification page to the API (`POST /files`, `GET`/`POST /verification`): a submission form with an 8-32 character identity number, two upload slots (identity document and business location photo, 5 MB limit with client-side size check), pending/approved/rejected banners driven by the account's verification status, and a submission history list. `apiClient` now skips the JSON Content-Type for `FormData` bodies so fetch sets the multipart boundary itself.
- Connected the Incoming Requests pages to the API (`/quota-requests/incoming`): the list page filters by candidate status and links into the detail page; the detail page shows the request header, the full offer chain, and panels to send an offer (total price and readiness lead), counter a buyer's offer, or reject with a reason. Added `request_id` to the `RequestCandidate` schema in `openapi.yaml` (regenerated type updated to match) because incoming items otherwise cannot link to their request detail.
- Connected all remaining admin pages to the API. Verification Queue (`/admin/verification` + decision) filters by status, links the identity document and location photo, and requires a rejection reason. Item Proposals (`/admin/proposals` + decision) approves into the master catalog or rejects with a reason. Master Items (`/admin/master/items`) adds, renames, and activates/deactivates catalog items. Disputes (`/admin/disputes` + mediate/resolve) starts mediation and resolves with a continued/confirmed/cancelled result, reversing allocations on cancellation (FR-071/072). Late Orders (`/admin/late-orders`) lists orders past their readiness deadline. Reviews Moderation (`/profile/{id}/reviews` + hide) looks up a profile's reviews by ID and hides violating ones with a reason (FR-050). WhatsApp (`/admin/whatsapp`) shows session status, the last error, and the relinking QR code with a 30-second auto refresh, never the service number (FR-082).
- WhatsApp now converts the session QR payload into a scannable PNG using `qrcode`, while accepting image data URLs from the API.
- Updated dashboard and admin sidebars to use the logo asset, added the missing `/admin/orders` destination, and prevented business-profile requests before a non-admin account is loaded.
- Aligned admin mutation payloads with the generated OpenAPI `paths` types, encoded admin path identifiers, and built master-item query parameters with `URLSearchParams`.
- Normalized admin proposal and dispute list responses before rendering and skipped the business-profile query for admin accounts, which do not have a business profile.
- Rendered the WhatsApp QR payload into a scannable QR image with the `qrcode` package, while keeping support for image data URLs.

### Changed

- Redesigned the Register page for clarity and accessibility: simplified flat branding panel, role selection via a proper radiogroup, plain password fields with a visible minimum-length rule, an explicit terms agreement checkbox, friendlier Indonesian validation messages, and a clearer primary CTA with a spinner loading state.
- Restyled the Login, Register, Forgot Password, and Reset Password pages to a shared two-panel design: a deep-navy brand panel with abstract shapes on desktop and a compact form panel, keeping the existing color palette and all form logic.
- Restyled the My Profile page to fit the dashboard layout: card-based sections, and the save button now reflects the pending state of the update mutation.
- Redesigned the home Final CTA section: a two-column card with abstract shapes matching the auth pages, a feature highlight panel, rounded buttons with `react-router-dom` links, and a hover arrow on the secondary action.
- Restyled `DashboardLayout` to a light, blurred sidebar: accent-tinted active items, collapsible sub-menus, and a logout button pinned to the sidebar footer. The header account area is simplified to a static identity block. Admin navigation now groups master data and order supervision into sub-menus.
- Redesigned the public footer: a four-column layout (brand with the capacity-exchange description and the no-funds-held note, in-page navigation, platform links, company links) with a bottom bar for copyright and legal links.

### Fixed

- The Verify Email page no longer shows the raw "Belum masuk" problem title. A 401 now explains that the session was not read and offers a "Masuk kembali" link, and a 410 explains the code expired with a hint to resend.
- Registration now continues to the Verify Email page with the registered email instead of dropping the user back at login, and a successful email verification continues to Verify Phone.

### Removed

- Removed the unused duplicate `pages/Search/SearchPage.tsx`; the search page lives at `pages/Search.tsx`.
- Relaxed the phone number schema to accept `08`, `62`, and `+62` formats, and made the password confirmation error message clearer.
- Initialized the React 19 + TypeScript + Vite frontend.
- Added Tailwind CSS 4 through the Vite plugin.
- Added the initial source structure for components, pages, hooks, routes, API, schemas, utilities, and assets.
- Added UI dependencies: `motion`, `react-icons`, `clsx`, `tailwind-merge`, and `react-router-dom`.
- Added the initial application shell, entry point, shared utility, and starter assets.
- Added development scripts for `dev`, `build`, `lint`, and `preview`.
- Enabled the React Compiler and ESLint configuration for TypeScript and React.
