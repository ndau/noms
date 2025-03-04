// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"runtime"
	"strconv"

	"github.com/attic-labs/kingpin"

	"github.com/ndau/noms/cmd/util"
	"github.com/ndau/noms/go/d"
)

func nomsBlob(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	blob := noms.Command("blob", "Interact with blobs.")

	blobPut := blob.Command("put", "imports a blob to a dataset")
	concurrency := blobPut.Flag("concurrency", "number of concurrent HTTP calls to retrieve remote resources").Default(strconv.Itoa(runtime.NumCPU())).Int()
	putFile := blobPut.Arg("file", "a file to import").Required().String()
	putDB := blobPut.Arg("dbSpec", "the database to import into").String()

	blobGet := blob.Command("export", "exports a blob from a dataset")
	getDs := blobGet.Arg("dataset", "the dataset to export").Required().String()
	getPath := blobGet.Arg("file", "an optional file to save the blob to").String()

	return blob, func(input string) int {
		switch input {
		case blobPut.FullCommand():
			return nomsBlobPut(*putFile, *putDB, *concurrency)
		case blobGet.FullCommand():
			return nomsBlobGet(*getDs, *getPath)
		}
		d.Panic("notreached")
		return 1
	}
}
