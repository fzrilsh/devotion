// Package masterdata owns the reference data every other module reads against:
// the province and city list seeded from wilayah.id, and the baseline catalog of
// product and machine types. It exposes four public read endpoints (the region
// and catalog lists carry security:[] in the contract) and the logic the
// seed:regions and seed:master-data subcommands drive.
//
// The region codes from the source use a dotted form for cities (32.73);
// NormalizeCityCode strips the dot before anything is stored, because the
// city_code_format and city_belongs_to_province constraints, and the ^[0-9]{4}$
// pattern in the contract, all assume the dot-free four-digit form. That single
// normalization is the step whose absence fails silently until every row is
// rejected, so it lives in one named function with a direct test.
package masterdata

import "strings"

// NormalizeCityCode removes the dot separator the wilayah.id source uses for
// city codes, turning "32.73" into "3273". The source always uses exactly two
// digits after the dot, so the conversion is reversible if ever needed. A code
// with no dot is returned unchanged, so re-normalizing an already-normalized
// code is a no-op and the seed stays idempotent.
func NormalizeCityCode(code string) string {
	return strings.ReplaceAll(strings.TrimSpace(code), ".", "")
}
