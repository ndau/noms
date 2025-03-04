// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/attic-labs/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"

	"github.com/ndau/noms/go/datas"
	"github.com/ndau/noms/go/hash"
	"github.com/ndau/noms/go/nbs"
	"github.com/ndau/noms/go/types"
	"github.com/ndau/noms/go/util/profile"
)

var (
	dir    = kingpin.Flag("dir", "Write to an NBS store in the given directory").String()
	table  = kingpin.Flag("table", "Write to an NBS store in AWS, using this table").String()
	bucket = kingpin.Flag("bucket", "Write to an NBS store in AWS, using this bucket").String()
	dbName = kingpin.Flag("db", "Write to an NBS store in AWS, using this db name").String()
)

const memTableSize = 128 * humanize.MiByte

func main() {
	profile.RegisterProfileFlags(kingpin.CommandLine)
	kingpin.Parse()

	var store *nbs.NomsBlockStore
	if *dir != "" {
		store = nbs.NewLocalStore(*dir, memTableSize)
		*dbName = *dir
	} else if *table != "" && *bucket != "" && *dbName != "" {
		sess := session.Must(session.NewSession(aws.NewConfig().WithRegion("us-west-2")))
		store = nbs.NewAWSStore(*table, *dbName, *bucket, s3.New(sess), dynamodb.New(sess), memTableSize)
	} else {
		log.Fatalf("Must set either --dir or ALL of --table, --bucket and --db\n")
	}

	db := datas.NewDatabase(store)
	defer db.Close()

	defer profile.MaybeStartProfile().Stop()

	height := types.NewRef(db.Datasets()).Height()
	fmt.Println("Store is of height", height)
	fmt.Println("| Height |   Nodes | Children | Branching | Groups | Reads | Pruned |")
	fmt.Println("+--------+---------+----------+-----------+--------+-------+--------+")
	chartFmt := "| %6d | %7d | %8d | %9d | %6d | %5d | %6d |\n"

	var optimal, sum int
	visited := map[hash.Hash]bool{}

	current := hash.HashSlice{store.Root()}
	for numNodes := 1; numNodes > 0; numNodes = len(current) {
		// Start by reading the values of the current level of the graph
		currentValues := make(map[hash.Hash]types.Value, len(current))
		readValues := db.ReadManyValues(current)
		for i, v := range readValues {
			h := current[i]
			currentValues[h] = v
			visited[h] = true
		}

		// Iterate all the Values at the current level of the graph IN ORDER (as specified in |current|) and gather up their embedded refs. We'll build two different lists of hash.Hashes during this process:
		// 1) An ordered list of ALL the children of the current level.
		// 2) An ordered list of the child nodes that contain refs to chunks we haven't yet visited. This *excludes* already-visted nodes and nodes without children.
		// We'll use 1) to get an estimate of how good the locality is among the children of the current level, and then 2) to descend to the next level of the graph.
		orderedChildren := hash.HashSlice{}
		nextLevel := hash.HashSlice{}
		for _, h := range current {
			currentValues[h].WalkRefs(func(r types.Ref) {
				target := r.TargetHash()
				orderedChildren = append(orderedChildren, target)
				if !visited[target] && r.Height() > 1 {
					nextLevel = append(nextLevel, target)
				}
			})
		}

		// Estimate locality among the members of |orderedChildren| by splitting into groups that are roughly |branchFactor| in size and calling CalcReads on each group. With perfect locality, we'd expect that each group could be read in a single physical read.
		numChildren := len(orderedChildren)
		branchFactor := numChildren / numNodes
		numGroups := numNodes
		if numChildren%numNodes != 0 {
			numGroups++
		}
		wg := &sync.WaitGroup{}
		reads := make([]int, numGroups)
		for i := 0; i < numGroups; i++ {
			wg.Add(1)
			if i+1 == numGroups { // last group
				go func(i int) {
					defer wg.Done()
					reads[i], _ = store.CalcReads(orderedChildren[i*branchFactor:].HashSet(), 0)
				}(i)
				continue
			}
			go func(i int) {
				defer wg.Done()
				reads[i], _ = store.CalcReads(orderedChildren[i*branchFactor:(i+1)*branchFactor].HashSet(), 0)
			}(i)
		}

		wg.Wait()

		sumOfReads := sumInts(reads)
		fmt.Printf(chartFmt, height, numNodes, numChildren, branchFactor, numGroups, sumOfReads, numChildren-len(nextLevel))

		sum += sumOfReads
		optimal += numGroups
		height--
		current = nextLevel
	}

	fmt.Printf("\nVisited %d chunk groups\n", optimal)
	fmt.Printf("Reading DB %s requires %.01fx optimal number of reads\n", *dbName, float64(sum)/float64(optimal))
}

func sumInts(nums []int) (sum int) {
	for _, n := range nums {
		sum += n
	}
	return
}
