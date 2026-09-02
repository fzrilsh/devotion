package platform

import "time"

// dateLayout is the wire format for a bare calendar date, YYYY-MM-DD.
const dateLayout = "2006-01-02"

// ParseDate parses a bare YYYY-MM-DD as midnight in Asia/Jakarta, the single
// zone every week boundary lives in. Parsing here (not UTC) means a parsed
// week_start compares equal to WeekStart of itself, so a Monday date stays a
// Monday instead of shifting a day when localized.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, s, jakarta)
}

// FormatDate renders t as a bare YYYY-MM-DD, the inverse of ParseDate and the
// only correct serializer for a JSON field the contract declares as
// `format: date`. It localizes to Asia/Jakarta first, so a date column decoded
// as UTC midnight still names its own day, and a timestamptz names the day a
// reader in WIB would see.
//
// Use this on the wire and FormatDateID only in text a person reads (the
// `detail` string of an error, the body of a notification). A field that takes
// ParseDate on the way in must give FormatDate back on the way out, or the same
// field is two different formats depending on direction.
func FormatDate(t time.Time) string {
	return t.In(jakarta).Format(dateLayout)
}

// monthsID maps a month to its Indonesian name, indexed by time.Month (January
// == 1). Index 0 is unused so the lookup needs no offset.
var monthsID = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// FormatDateID renders t as an Indonesian long date, "24 Agustus 2026". It
// localizes to Asia/Jakarta first so a timestamptz taken near midnight names the
// day a reader in WIB would see, and notifications quote the same string a period
// row carries. Day is not zero-padded, matching how the date reads aloud.
func FormatDateID(t time.Time) string {
	t = t.In(jakarta)
	y, m, d := t.Date()
	return itoa(d) + " " + monthsID[m] + " " + itoa(y)
}

// itoa formats a non-negative int without importing strconv for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
