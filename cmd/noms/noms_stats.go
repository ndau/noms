// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/kingpin"

	"github.com/ndau/noms/cmd/util"
	"github.com/ndau/noms/go/config"
	"github.com/ndau/noms/go/d"
)

func nomsStats(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	stats := noms.Command("stats", "Shows stats summary for a Noms Database.")
	database := stats.Arg("database", "See Spelling Objects at https://github.com/ndau/noms/blob/master/doc/spelling.md for details on the database argument.").Required().String()

	return stats, func(input string) int {
		cfg := config.NewResolver()
		store, err := cfg.GetDatabase(*database)
		d.CheckError(err)
		defer store.Close()

		fmt.Println(store.StatsSummary())
		return 0
	}
}
