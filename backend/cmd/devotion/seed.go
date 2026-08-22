package main

import (
	"context"
	"flag"
)

// runSeedRegions seeds provinces and cities from the normalized regions copy,
// or refreshes from wilayah.id with --refresh. Implemented in the masterdata
// branch (T019).
func runSeedRegions(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed:regions", flag.ExitOnError)
	_ = fs.Bool("refresh", false, "ambil ulang dari jaringan lalu tulis salinan JSON")
	_ = fs.Parse(args)
	return errNotImplemented
}

// runSeedMasterData seeds the baseline product and machine master data.
// Implemented in the masterdata branch (T019).
func runSeedMasterData(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed:master-data", flag.ExitOnError)
	_ = fs.Parse(args)
	return errNotImplemented
}
