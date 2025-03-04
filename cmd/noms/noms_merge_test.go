// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/ndau/noms/go/datas"
	"github.com/ndau/noms/go/spec"
	"github.com/ndau/noms/go/types"
	"github.com/ndau/noms/go/util/clienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type nomsMergeTestSuite struct {
	clienttest.ClientTestSuite
}

func TestNomsMerge(t *testing.T) {
	suite.Run(t, &nomsMergeTestSuite{})
}

func (s *nomsMergeTestSuite) TearDownTest() {
	s.NoError(os.RemoveAll(s.DBDir))
}

func (s *nomsMergeTestSuite) TestNomsMerge_Success() {
	left, right := "left", "right"
	parentSpec := s.spec("parent")
	defer parentSpec.Close()
	leftSpec := s.spec(left)
	defer leftSpec.Close()
	rightSpec := s.spec(right)
	defer rightSpec.Close()

	p := s.setupMergeDataset(
		parentSpec,
		types.StructData{
			"num": types.Number(42),
			"str": types.String("foobar"),
			"lst": types.NewList(parentSpec.GetDatabase(), types.Number(1), types.String("foo")),
			"map": types.NewMap(parentSpec.GetDatabase(), types.Number(1), types.String("foo"),
				types.String("foo"), types.Number(1)),
		},
		types.NewSet(parentSpec.GetDatabase()))

	l := s.setupMergeDataset(
		leftSpec,
		types.StructData{
			"num": types.Number(42),
			"str": types.String("foobaz"),
			"lst": types.NewList(leftSpec.GetDatabase(), types.Number(1), types.String("foo")),
			"map": types.NewMap(leftSpec.GetDatabase(), types.Number(1), types.String("foo"),
				types.String("foo"), types.Number(1)),
		},
		types.NewSet(leftSpec.GetDatabase(), p))

	r := s.setupMergeDataset(
		rightSpec,
		types.StructData{
			"num": types.Number(42),
			"str": types.String("foobar"),
			"lst": types.NewList(rightSpec.GetDatabase(), types.Number(1), types.String("foo")),
			"map": types.NewMap(rightSpec.GetDatabase(), types.Number(1), types.String("foo"),
				types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
		},
		types.NewSet(rightSpec.GetDatabase(), p))

	expected := types.NewStruct("", types.StructData{
		"num": types.Number(42),
		"str": types.String("foobaz"),
		"lst": types.NewList(parentSpec.GetDatabase(), types.Number(1), types.String("foo")),
		"map": types.NewMap(parentSpec.GetDatabase(), types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
	})

	stdout, stderr, err := s.Run(main, []string{"merge", s.DBDir, left, right})
	if err == nil {
		s.Equal("", stderr)
		s.validateOutput(stdout, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) spec(name string) spec.Spec {
	sp, err := spec.ForDataset(spec.CreateValueSpecString("nbs", s.DBDir, name))
	s.NoError(err)
	return sp
}

func (s *nomsMergeTestSuite) setupMergeDataset(sp spec.Spec, data types.StructData, p types.Set) types.Ref {
	ds := sp.GetDataset()
	ds, err := sp.GetDatabase().Commit(ds, types.NewStruct("", data), datas.CommitOptions{Parents: p})
	s.NoError(err)
	return ds.HeadRef()
}

func (s *nomsMergeTestSuite) validateOutput(outHash string, expected types.Struct, parents ...types.Value) {
	outHash = strings.TrimSpace(outHash)
	sp, err := spec.ForPath(spec.CreateValueSpecString("nbs", s.DBDir, fmt.Sprintf("#%s", outHash)))
	db := sp.GetDatabase()
	if s.NoError(err) {
		defer sp.Close()
		commit := sp.GetValue().(types.Struct)
		s.True(commit.Get(datas.ParentsField).Equals(types.NewSet(db, parents...)))
		merged := commit.Get("value")
		s.True(expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_Left() {
	left, right := "left", "right"
	parentSpec := s.spec("parent")
	defer parentSpec.Close()
	leftSpec := s.spec(left)
	defer leftSpec.Close()
	rightSpec := s.spec(right)
	defer rightSpec.Close()

	p := s.setupMergeDataset(parentSpec, types.StructData{"num": types.Number(42)}, types.NewSet(parentSpec.GetDatabase()))
	l := s.setupMergeDataset(leftSpec, types.StructData{"num": types.Number(43)}, types.NewSet(leftSpec.GetDatabase(), p))
	r := s.setupMergeDataset(rightSpec, types.StructData{"num": types.Number(44)}, types.NewSet(rightSpec.GetDatabase(), p))

	expected := types.NewStruct("", types.StructData{"num": types.Number(43)})

	stdout, stderr, err := s.Run(main, []string{"merge", "--policy=l", s.DBDir, left, right})
	if err == nil {
		s.Equal("", stderr)
		s.validateOutput(stdout, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_Right() {
	left, right := "left", "right"
	parentSpec := s.spec("parent")
	defer parentSpec.Close()
	leftSpec := s.spec(left)
	defer leftSpec.Close()
	rightSpec := s.spec(right)
	defer rightSpec.Close()

	p := s.setupMergeDataset(parentSpec, types.StructData{"num": types.Number(42)}, types.NewSet(parentSpec.GetDatabase()))
	l := s.setupMergeDataset(leftSpec, types.StructData{"num": types.Number(43)}, types.NewSet(leftSpec.GetDatabase(), p))
	r := s.setupMergeDataset(rightSpec, types.StructData{"num": types.Number(44)}, types.NewSet(rightSpec.GetDatabase(), p))

	expected := types.NewStruct("", types.StructData{"num": types.Number(44)})

	stdout, stderr, err := s.Run(main, []string{"merge", "--policy=r", s.DBDir, left, right})
	if err == nil {
		s.Equal("", stderr)
		s.validateOutput(stdout, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_Conflict() {
	left, right := "left", "right"
	parentSpec := s.spec("parent")
	defer parentSpec.Close()
	leftSpec := s.spec(left)
	defer leftSpec.Close()
	rightSpec := s.spec(right)
	defer rightSpec.Close()
	p := s.setupMergeDataset(parentSpec, types.StructData{"num": types.Number(42)}, types.NewSet(parentSpec.GetDatabase()))
	s.setupMergeDataset(leftSpec, types.StructData{"num": types.Number(43)}, types.NewSet(leftSpec.GetDatabase(), p))
	s.setupMergeDataset(rightSpec, types.StructData{"num": types.Number(44)}, types.NewSet(rightSpec.GetDatabase(), p))

	s.Panics(func() { s.MustRun(main, []string{"merge", s.DBDir, left, right}) })
}

func (s *nomsMergeTestSuite) TestBadInput() {
	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	l, r := "left", "right"
	type c struct {
		args []string
		err  string
	}
	cases := []c{
		{[]string{sp.String(), l + "!!", r}, "error: Invalid dataset " + l + "!!, must match [a-zA-Z0-9\\-_/]+\n"},
		{[]string{sp.String(), l + "2", r}, "error: Dataset " + l + "2 has no data\n"},
		{[]string{sp.String(), l, r + "2"}, "error: Dataset " + r + "2 has no data\n"},
	}

	db := sp.GetDatabase()

	prep := func(dsName string) {
		ds := db.GetDataset(dsName)
		db.CommitValue(ds, types.NewMap(db, types.String("foo"), types.String("bar")))
	}
	prep(l)
	prep(r)

	for _, c := range cases {
		stdout, stderr, err := s.Run(main, append([]string{"merge"}, c.args...))
		s.Empty(stdout, "Expected non-empty stdout for case: %#v", c.args)
		if !s.NotNil(err, "Unexpected success for case: %#v\n", c.args) {
			continue
		}
		if mainErr, ok := err.(clienttest.ExitError); ok {
			s.Equal(1, mainErr.Code)
			s.Equal(c.err, stderr, "Unexpected error output for case: %#v\n", c.args)
		} else {
			s.Fail("Run() recovered non-error panic", "err: %#v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
		}
	}
}

func TestNomsMergeCliResolve(t *testing.T) {
	type c struct {
		input            string
		aChange, bChange types.DiffChangeType
		aVal, bVal       types.Value
		expectedChange   types.DiffChangeType
		expected         types.Value
		success          bool
	}

	cases := []c{
		{"l\n", types.DiffChangeAdded, types.DiffChangeAdded, types.String("foo"), types.String("bar"), types.DiffChangeAdded, types.String("foo"), true},
		{"r\n", types.DiffChangeAdded, types.DiffChangeAdded, types.String("foo"), types.String("bar"), types.DiffChangeAdded, types.String("bar"), true},
		{"l\n", types.DiffChangeAdded, types.DiffChangeAdded, types.Number(7), types.String("bar"), types.DiffChangeAdded, types.Number(7), true},
		{"r\n", types.DiffChangeModified, types.DiffChangeModified, types.Number(7), types.String("bar"), types.DiffChangeModified, types.String("bar"), true},
	}

	for _, c := range cases {
		input := bytes.NewBufferString(c.input)

		changeType, newVal, ok := cliResolve(input, ioutil.Discard, c.aChange, c.bChange, c.aVal, c.bVal, types.Path{})
		if !c.success {
			assert.False(t, ok)
		} else if assert.True(t, ok) {
			assert.Equal(t, c.expectedChange, changeType)
			assert.True(t, c.expected.Equals(newVal))
		}
	}
}
