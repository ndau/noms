// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ndau/noms/go/config"
	"github.com/ndau/noms/go/d"
	"github.com/ndau/noms/go/types"
	"github.com/ndau/noms/go/util/profile"
)

func nomsBlobPut(filePath string, dbPath string, concurrency int) int {
	info, err := os.Stat(filePath)
	if err != nil {
		d.CheckError(errors.New("couldn't stat file"))
	}

	defer profile.MaybeStartProfile().Stop()

	fileSize := info.Size()
	chunkSize := fileSize / int64(concurrency)
	if chunkSize < (1 << 20) {
		chunkSize = 1 << 20
	}

	readers := make([]io.Reader, fileSize/chunkSize)
	for i := 0; i < len(readers); i++ {
		r, err := os.Open(filePath)
		d.CheckErrorNoUsage(err)
		defer r.Close()
		r.Seek(int64(i)*chunkSize, 0)
		limit := chunkSize
		if i == len(readers)-1 {
			limit += fileSize % chunkSize // adjust size of last slice to include the final bytes.
		}
		lr := io.LimitReader(r, limit)
		readers[i] = lr
	}

	cfg := config.NewResolver()
	db, err := cfg.GetDatabase(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open database: %s\n", err)
		return 1
	}
	defer db.Close()

	blob := types.NewBlob(db, readers...)
	ref := db.WriteValue(blob)
	db.Flush()
	fmt.Printf("#%s\n", ref.TargetHash())
	return 0
}
