// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package chunks

import (
	"github.com/stretchr/testify/suite"

	"github.com/ndau/noms/go/constants"
	"github.com/ndau/noms/go/hash"
)

type ChunkStoreTestSuite struct {
	suite.Suite
	Factory Factory
}

func (suite *ChunkStoreTestSuite) TestChunkStorePut() {
	store := suite.Factory.CreateStore("ns")
	input := "abc"
	c := NewChunk([]byte(input))
	store.Put(c)
	h := c.Hash()

	// Reading it via the API should work.
	assertInputInStore(input, h, store, suite.Assert())
}

func (suite *ChunkStoreTestSuite) TestChunkStoreRoot() {
	store := suite.Factory.CreateStore("ns")
	oldRoot := store.Root()
	suite.True(oldRoot.IsEmpty())

	bogusRoot := hash.Parse("8habda5skfek1265pc5d5l1orptn5dr0")
	newRoot := hash.Parse("8la6qjbh81v85r6q67lqbfrkmpds14lg")

	// Try to update root with bogus oldRoot
	result := store.Commit(newRoot, bogusRoot)
	suite.False(result)

	// Now do a valid root update
	result = store.Commit(newRoot, oldRoot)
	suite.True(result)
}

func (suite *ChunkStoreTestSuite) TestChunkStoreCommitPut() {
	name := "ns"
	store := suite.Factory.CreateStore(name)
	input := "abc"
	c := NewChunk([]byte(input))
	store.Put(c)
	h := c.Hash()

	// Reading it via the API should work...
	assertInputInStore(input, h, store, suite.Assert())
	// ...but it shouldn't be persisted yet
	assertInputNotInStore(input, h, suite.Factory.CreateStore(name), suite.Assert())

	store.Commit(h, store.Root()) // Commit persists Chunks
	assertInputInStore(input, h, store, suite.Assert())
	assertInputInStore(input, h, suite.Factory.CreateStore(name), suite.Assert())
}

func (suite *ChunkStoreTestSuite) TestChunkStoreGetNonExisting() {
	store := suite.Factory.CreateStore("ns")
	h := hash.Parse("11111111111111111111111111111111")
	c := store.Get(h)
	suite.True(c.IsEmpty())
}

func (suite *ChunkStoreTestSuite) TestChunkStoreVersion() {
	store := suite.Factory.CreateStore("ns")
	oldRoot := store.Root()
	suite.True(oldRoot.IsEmpty())
	newRoot := hash.Parse("11111222223333344444555556666677")
	suite.True(store.Commit(newRoot, oldRoot))

	suite.Equal(constants.NomsVersion, store.Version())
}

func (suite *ChunkStoreTestSuite) TestChunkStoreCommitUnchangedRoot() {
	store1, store2 := suite.Factory.CreateStore("ns"), suite.Factory.CreateStore("ns")
	input := "abc"
	c := NewChunk([]byte(input))
	store1.Put(c)
	h := c.Hash()

	// Reading c from store1 via the API should work...
	assertInputInStore(input, h, store1, suite.Assert())
	// ...but not store2.
	assertInputNotInStore(input, h, store2, suite.Assert())

	store1.Commit(store1.Root(), store1.Root())
	store2.Rebase()
	// Now, reading c from store2 via the API should work...
	assertInputInStore(input, h, store2, suite.Assert())
}
