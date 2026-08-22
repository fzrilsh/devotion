// Package httpx holds the HTTP layer shared by every domain handler: the router,
// the RFC 9457 problem+json error type, the stable machine error codes, the
// middleware chain, and slog setup. Handlers build responses on top of this so
// the wire format and the error-code-to-status mapping live in exactly one
// place and cannot drift between endpoints.
package httpx

// Code is a stable machine-readable error code. The set is transcribed verbatim
// from openapi.yaml (the Problem.code enum); clients switch on it, so a value
// here must never diverge from the contract.
type Code string

// The full set of error codes from openapi.yaml. Kept in the contract's order.
const (
	CodeValidationFailed          Code = "VALIDATION_FAILED"
	CodeNotAuthenticated          Code = "NOT_AUTHENTICATED"
	CodeForbidden                 Code = "FORBIDDEN"
	CodeNotFound                  Code = "NOT_FOUND"
	CodeRateLimitExceeded         Code = "RATE_LIMIT_EXCEEDED"
	CodeEmailAlreadyRegistered    Code = "EMAIL_ALREADY_REGISTERED"
	CodeInvalidCredentials        Code = "INVALID_CREDENTIALS"
	CodeInvalidVerificationCode   Code = "INVALID_VERIFICATION_CODE"
	CodeVerificationCodeExpired   Code = "VERIFICATION_CODE_EXPIRED"
	CodeEmailNotVerified          Code = "EMAIL_NOT_VERIFIED"
	CodePhoneNotVerified          Code = "PHONE_NOT_VERIFIED"
	CodeIdentityNotVerified       Code = "IDENTITY_NOT_VERIFIED"
	CodeIdentityAlreadyVerified   Code = "IDENTITY_ALREADY_VERIFIED"
	CodeVerificationPending       Code = "VERIFICATION_PENDING"
	CodeFileTooLarge              Code = "FILE_TOO_LARGE"
	CodeUnsupportedFileType       Code = "UNSUPPORTED_FILE_TYPE"
	CodeStorageQuotaFull          Code = "STORAGE_QUOTA_FULL"
	CodeListingNotFound           Code = "LISTING_NOT_FOUND"
	CodeCapacityAlreadyAllocated  Code = "CAPACITY_ALREADY_ALLOCATED"
	CodePeriodAlreadyAllocated    Code = "PERIOD_ALREADY_ALLOCATED"
	CodeSelfRequest               Code = "SELF_REQUEST"
	CodeInsufficientCapacity      Code = "INSUFFICIENT_CAPACITY"
	CodeRequestExpired            Code = "REQUEST_EXPIRED"
	CodeRequestAlreadyAgreed      Code = "REQUEST_ALREADY_AGREED"
	CodeCapacityAlreadyTaken      Code = "CAPACITY_ALREADY_TAKEN"
	CodeInvalidStatusTransition   Code = "INVALID_STATUS_TRANSITION"
	CodeCancellationAfterProduction Code = "CANCELLATION_AFTER_PRODUCTION"
	CodeWorkOrderNotCompleted     Code = "WORK_ORDER_NOT_COMPLETED"
	CodeReviewAlreadySubmitted    Code = "REVIEW_ALREADY_SUBMITTED"
)

// codeMeta binds a code to its HTTP status and Indonesian title. Deriving both
// from the code keeps two handlers from returning the same code with different
// statuses. Statuses match openapi.yaml where the contract gives an explicit
// example; the verification and not-found codes without an inline example take
// the status their semantics imply (403 for "not verified" gates, 404 for
// missing resources, 409 for conflicting state, 410 for expired one-time codes).
type codeMeta struct {
	Status int
	Title  string
}

var codes = map[Code]codeMeta{
	CodeValidationFailed:            {422, "Masukan tidak sah"},
	CodeNotAuthenticated:            {401, "Belum masuk"},
	CodeForbidden:                   {403, "Tidak berwenang"},
	CodeNotFound:                    {404, "Tidak ditemukan"},
	CodeRateLimitExceeded:           {429, "Terlalu banyak percobaan"},
	CodeEmailAlreadyRegistered:      {409, "Data sudah terdaftar"},
	CodeInvalidCredentials:          {401, "Gagal masuk"},
	CodeInvalidVerificationCode:     {422, "Kode verifikasi salah"},
	CodeVerificationCodeExpired:     {410, "Kode verifikasi kedaluwarsa"},
	CodeEmailNotVerified:            {403, "Email belum diverifikasi"},
	CodePhoneNotVerified:            {403, "Nomor HP belum diverifikasi"},
	CodeIdentityNotVerified:         {403, "Identitas belum diverifikasi"},
	CodeIdentityAlreadyVerified:     {409, "Identitas sudah diverifikasi"},
	CodeVerificationPending:         {409, "Verifikasi masih diproses"},
	CodeFileTooLarge:                {413, "Berkas terlalu besar"},
	CodeUnsupportedFileType:         {415, "Tipe berkas tidak diizinkan"},
	CodeStorageQuotaFull:            {507, "Kuota penyimpanan penuh"},
	CodeListingNotFound:             {404, "Listing tidak ditemukan"},
	CodeCapacityAlreadyAllocated:    {409, "Kapasitas sudah dialokasikan"},
	CodePeriodAlreadyAllocated:      {409, "Periode sudah dialokasikan"},
	CodeSelfRequest:                 {409, "Kandidat tidak sah"},
	CodeInsufficientCapacity:        {409, "Kapasitas tidak mencukupi"},
	CodeRequestExpired:              {410, "Request kedaluwarsa"},
	CodeRequestAlreadyAgreed:        {409, "Request sudah disepakati"},
	CodeCapacityAlreadyTaken:        {409, "Kapasitas sudah diambil"},
	CodeInvalidStatusTransition:     {409, "Perubahan status tidak diizinkan"},
	CodeCancellationAfterProduction: {409, "Pembatalan tidak tersedia"},
	CodeWorkOrderNotCompleted:       {409, "Pesanan belum dikonfirmasi"},
	CodeReviewAlreadySubmitted:      {409, "Ulasan sudah dikirim"},
}

// StatusFor returns the HTTP status bound to code, or 500 if the code is
// unknown (which can only happen if a caller invents a code outside the set).
func StatusFor(code Code) int {
	if m, ok := codes[code]; ok {
		return m.Status
	}
	return 500
}

// TitleFor returns the Indonesian title bound to code, or a generic title.
func TitleFor(code Code) string {
	if m, ok := codes[code]; ok {
		return m.Title
	}
	return "Terjadi galat"
}
