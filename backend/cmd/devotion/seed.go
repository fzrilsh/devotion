package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/masterdata"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// regionsDir is where the normalized regions.json snapshot lives. seed:regions
// reads it by default and --refresh rewrites it from wilayah.id.
const regionsDir = "docs/master-data"

// runSeedRegions seeds provinces and cities. By default it reads the committed
// regions.json snapshot; Prinsip V forbids depending on the external source
// while serving, so the network is touched only with --refresh, which fetches,
// normalizes, rewrites the snapshot, then seeds.
func runSeedRegions(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed:regions", flag.ExitOnError)
	refresh := fs.Bool("refresh", false, "ambil ulang dari jaringan lalu tulis salinan JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	svc, pool, err := newMasterdataService(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	var data masterdata.RegionData
	if *refresh {
		data, err = masterdata.FetchRegions(ctx)
		if err != nil {
			return err
		}
		if err := masterdata.WriteRegions(regionsDir, data); err != nil {
			return err
		}
	} else {
		data, err = masterdata.LoadRegions(regionsDir)
		if err != nil {
			return err
		}
	}
	if err := svc.SeedRegions(ctx, data); err != nil {
		return err
	}
	fmt.Printf("seed:regions selesai: %d provinsi, %d kota\n", len(data.Provinces), len(data.Cities))
	return nil
}

// runSeedMasterData seeds the baseline product and machine catalog.
func runSeedMasterData(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed:master-data", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	svc, pool, err := newMasterdataService(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := svc.SeedMasterData(ctx); err != nil {
		return err
	}
	fmt.Println("seed:master-data selesai")
	return nil
}

// newMasterdataService loads config, opens the pool, and builds a masterdata
// Service over a SystemClock. The caller closes the returned pool.
func newMasterdataService(ctx context.Context) (*masterdata.Service, *pgxpool.Pool, error) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	return masterdata.New(pool, platform.SystemClock{}, nil, nil), pool, nil
}
