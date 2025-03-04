// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ndau/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func isEmptyStruct(s types.Struct) bool {
	return s.Equals(types.EmptyStruct)
}

func TestCreateCommitMetaStructBasic(t *testing.T) {
	assert := assert.New(t)

	meta, err := CreateCommitMetaStruct(nil, "", "", nil, nil)
	assert.NoError(err)
	assert.False(isEmptyStruct(meta))
	assert.Equal("Struct Meta {\n  date: String,\n}", types.TypeOf(meta).Describe())
}

func TestCreateCommitMetaStructFromFlags(t *testing.T) {
	assert := assert.New(t)

	setCommitMetaFlags(time.Now().UTC().Format(CommitMetaDateFormat), "this is a message", "k1=v1,k2=v2,k3=v3")
	defer resetCommitMetaFlags()

	meta, err := CreateCommitMetaStruct(nil, "", "", nil, nil)
	assert.NoError(err)
	assert.Equal("Struct Meta {\n  date: String,\n  k1: String,\n  k2: String,\n  k3: String,\n  message: String,\n}",
		types.TypeOf(meta).Describe())
	assert.Equal(types.String(commitMetaDate), meta.Get("date"))
	assert.Equal(types.String(commitMetaMessage), meta.Get("message"))
	assert.Equal(types.String("v1"), meta.Get("k1"))
	assert.Equal(types.String("v2"), meta.Get("k2"))
	assert.Equal(types.String("v3"), meta.Get("k3"))
}

func TestCreateCommitMetaStructFromArgs(t *testing.T) {
	assert := assert.New(t)

	dateArg := time.Now().UTC().Format(CommitMetaDateFormat)
	messageArg := "this is a message"
	keyValueArg := map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"}
	meta, err := CreateCommitMetaStruct(nil, dateArg, messageArg, keyValueArg, nil)
	assert.NoError(err)
	assert.Equal("Struct Meta {\n  date: String,\n  k1: String,\n  k2: String,\n  k3: String,\n  message: String,\n}",
		types.TypeOf(meta).Describe())
	assert.Equal(types.String(dateArg), meta.Get("date"))
	assert.Equal(types.String(messageArg), meta.Get("message"))
	assert.Equal(types.String("v1"), meta.Get("k1"))
	assert.Equal(types.String("v2"), meta.Get("k2"))
	assert.Equal(types.String("v3"), meta.Get("k3"))
}

func TestCreateCommitMetaStructFromFlagsAndArgs(t *testing.T) {
	assert := assert.New(t)

	setCommitMetaFlags(time.Now().UTC().Format(CommitMetaDateFormat), "this is a message", "k1=v1p1,k2=v2p2,k4=v4p4")
	defer resetCommitMetaFlags()

	dateArg := time.Now().UTC().Add(time.Hour * -24).Format(CommitMetaDateFormat)
	messageArg := "this is a message"
	keyValueArg := map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"}

	// args passed in should win over the ones in the flags
	meta, err := CreateCommitMetaStruct(nil, dateArg, messageArg, keyValueArg, nil)
	assert.NoError(err)
	assert.Equal("Struct Meta {\n  date: String,\n  k1: String,\n  k2: String,\n  k3: String,\n  k4: String,\n  message: String,\n}",
		types.TypeOf(meta).Describe())
	assert.Equal(types.String(dateArg), meta.Get("date"))
	assert.Equal(types.String(messageArg), meta.Get("message"))
	assert.Equal(types.String("v1"), meta.Get("k1"))
	assert.Equal(types.String("v2"), meta.Get("k2"))
	assert.Equal(types.String("v3"), meta.Get("k3"))
	assert.Equal(types.String("v4p4"), meta.Get("k4"))
}

func TestCreateCommitMetaStructBadDate(t *testing.T) {
	assert := assert.New(t)

	testBadDates := func(cliDateString, argDateString string) {
		setCommitMetaFlags(cliDateString, "", "")
		defer resetCommitMetaFlags()

		meta, err := CreateCommitMetaStruct(nil, argDateString, "", nil, nil)
		assert.Error(err)
		assert.True(strings.HasPrefix(err.Error(), "Unable to parse date: "))
		assert.True(isEmptyStruct(meta))
	}
	testBadDateMultipleWays := func(dateString string) {
		testBadDates(dateString, "")
		testBadDates("", dateString)
		testBadDates(dateString, dateString)
	}

	testBadDateMultipleWays(time.Now().UTC().Format("Jan _2 15:04:05 2006"))
	testBadDateMultipleWays(time.Now().UTC().Format("Mon Jan _2 15:04:05 2006"))
	testBadDateMultipleWays(time.Now().UTC().Format("2006-01-02T15:04:05"))
}

func TestCreateCommitMetaStructBadMetaStrings(t *testing.T) {
	assert := assert.New(t)

	testBadMetaSeparator := func(k, v, sep string) {
		setCommitMetaFlags("", "", fmt.Sprintf("%s%s%s", k, sep, v))
		defer resetCommitMetaFlags()

		meta, err := CreateCommitMetaStruct(nil, "", "", nil, nil)
		assert.Error(err)
		assert.True(strings.HasPrefix(err.Error(), "Unable to parse meta value: "))
		assert.True(isEmptyStruct(meta))
	}

	testBadMetaKeys := func(k, v string) {
		testBadMetaSeparator(k, v, ":")
		testBadMetaSeparator(k, v, "-")

		setCommitMetaFlags("", "", fmt.Sprintf("%s=%s", k, v))

		meta, err := CreateCommitMetaStruct(nil, "", "", nil, nil)
		assert.Error(err)
		assert.True(strings.HasPrefix(err.Error(), "Invalid meta key: "))
		assert.True(isEmptyStruct(meta))

		resetCommitMetaFlags()

		metaValues := map[string]string{k: v}
		meta, err = CreateCommitMetaStruct(nil, "", "", metaValues, nil)
		assert.Error(err)
		assert.True(strings.HasPrefix(err.Error(), "Invalid meta key: "))
		assert.True(isEmptyStruct(meta))
	}

	// Valid names must start with `a-zA-Z` and after that `a-zA-Z0-9_`.
	testBadMetaKeys("_name", "value")
	testBadMetaKeys("99problems", "now 100")
	testBadMetaKeys("one-hundred-bottles", "take one down")
	testBadMetaKeys("👀", "who watches the watchers?")
	testBadMetaKeys("key:", "value")
}

func setCommitMetaFlags(date, message, kvStrings string) {
	commitMetaDate = date
	commitMetaMessage = message
	commitMetaKeyValueStrings = kvStrings
}

func resetCommitMetaFlags() {
	setCommitMetaFlags("", "", "")
}
