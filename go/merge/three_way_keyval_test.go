// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package merge

import (
	"testing"

	"github.com/ndau/noms/go/types"
	"github.com/stretchr/testify/suite"
)

func TestThreeWayMapMerge(t *testing.T) {
	suite.Run(t, &ThreeWayMapMergeSuite{})
}

func TestThreeWayStructMerge(t *testing.T) {
	suite.Run(t, &ThreeWayStructMergeSuite{})
}

type kvs []interface{}

func (kv kvs) items() []interface{} {
	return kv
}

func (kv kvs) remove(k interface{}) kvs {
	out := make(kvs, 0, len(kv))
	for i := 0; i < len(kv); i += 2 {
		if kv[i] != k {
			out = append(out, kv[i], kv[i+1])
		}
	}
	return out
}

func (kv kvs) set(k, v interface{}) kvs {
	out := make(kvs, len(kv))
	for i := 0; i < len(kv); i += 2 {
		out[i], out[i+1] = kv[i], kv[i+1]
		if kv[i] == k {
			out[i+1] = v
		}
	}
	return out
}

var (
	aa1      = kvs{"a1", "a-one", "a2", "a-two", "a3", "a-three", "a4", "a-four"}
	aa1a     = kvs{"a1", "a-one", "a2", "a-two", "a3", "a-three-diff", "a4", "a-four", "a6", "a-six"}
	aa1b     = kvs{"a1", "a-one", "a3", "a-three-diff", "a4", "a-four", "a5", "a-five"}
	aaMerged = kvs{"a1", "a-one", "a3", "a-three-diff", "a4", "a-four", "a5", "a-five", "a6", "a-six"}

	mm1       = kvs{}
	mm1a      = kvs{"k1", kvs{"a", 0}}
	mm1b      = kvs{"k1", kvs{"b", 1}}
	mm1Merged = kvs{"k1", kvs{"a", 0, "b", 1}}

	mm2       = kvs{"k2", aa1, "k3", "k-three"}
	mm2a      = kvs{"k1", kvs{"a", 0}, "k2", aa1a, "k3", "k-three", "k4", "k-four"}
	mm2b      = kvs{"k1", kvs{"b", 1}, "k2", aa1b}
	mm2Merged = kvs{"k1", kvs{"a", 0, "b", 1}, "k2", aaMerged, "k4", "k-four"}
)

type ThreeWayKeyValMergeSuite struct {
	ThreeWayMergeSuite
}

type ThreeWayMapMergeSuite struct {
	ThreeWayKeyValMergeSuite
}

func (s *ThreeWayMapMergeSuite) SetupSuite() {
	s.create = func(seq seq) (val types.Value) {
		if seq != nil {
			keyValues := valsToTypesValues(s.create, seq.items()...)
			val = types.NewMap(s.vs, keyValues...)
		}
		return
	}
	s.typeStr = "Map"
}

type ThreeWayStructMergeSuite struct {
	ThreeWayKeyValMergeSuite
}

func (s *ThreeWayStructMergeSuite) SetupSuite() {
	s.create = func(seq seq) (val types.Value) {
		if seq != nil {
			kv := seq.items()
			fields := types.StructData{}
			for i := 0; i < len(kv); i += 2 {
				fields[kv[i].(string)] = valToTypesValue(s.create, kv[i+1])
			}
			val = types.NewStruct("TestStruct", fields)
		}
		return
	}
	s.typeStr = "Struct"
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_DoNothing() {
	s.tryThreeWayMerge(nil, nil, aa1, aa1)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NoRecursion() {
	s.tryThreeWayMerge(aa1a, aa1b, aa1, aaMerged)
	s.tryThreeWayMerge(aa1b, aa1a, aa1, aaMerged)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveCreate() {
	s.tryThreeWayMerge(mm1a, mm1b, mm1, mm1Merged)
	s.tryThreeWayMerge(mm1b, mm1a, mm1, mm1Merged)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveCreateNil() {
	s.tryThreeWayMerge(mm1a, mm1b, nil, mm1Merged)
	s.tryThreeWayMerge(mm1b, mm1a, nil, mm1Merged)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveMerge() {
	s.tryThreeWayMerge(mm2a, mm2b, mm2, mm2Merged)
	s.tryThreeWayMerge(mm2b, mm2a, mm2, mm2Merged)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RefMerge() {
	strRef := s.vs.WriteValue(types.NewStruct("Foo", types.StructData{"life": types.Number(42)}))

	m := kvs{"r2", s.vs.WriteValue(s.create(aa1))}
	ma := kvs{"r1", strRef, "r2", s.vs.WriteValue(s.create(aa1a))}
	mb := kvs{"r1", strRef, "r2", s.vs.WriteValue(s.create(aa1b))}
	mMerged := kvs{"r1", strRef, "r2", s.vs.WriteValue(s.create(aaMerged))}

	s.tryThreeWayMerge(ma, mb, m, mMerged)
	s.tryThreeWayMerge(mb, ma, m, mMerged)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveMultiLevelMerge() {
	m := kvs{"mm1", mm1, "mm2", s.vs.WriteValue(s.create(mm2))}
	ma := kvs{"mm1", mm1a, "mm2", s.vs.WriteValue(s.create(mm2a))}
	mb := kvs{"mm1", mm1b, "mm2", s.vs.WriteValue(s.create(mm2b))}
	mMerged := kvs{"mm1", mm1Merged, "mm2", s.vs.WriteValue(s.create(mm2Merged))}

	s.tryThreeWayMerge(ma, mb, m, mMerged)
	s.tryThreeWayMerge(mb, ma, m, mMerged)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_CustomMerge() {
	p := kvs{"k1", "k-one", "k2", "k-two", "mm1", mm1, "s1", "s-one"}
	a := kvs{"k1", "k-won", "k2", "k-too", "mm1", mm1, "s1", "s-one", "n1", kvs{"a", "1"}}
	b := kvs{"k2", "k-two", "mm1", "mm-one", "s1", "s-one", "n1", kvs{"a", "2"}}
	exp := kvs{"k2", "k-too", "mm1", "mm-one", "s1", "s-one", "n1", kvs{"a", "1"}}

	expectedConflictPaths := [][]string{{"k1"}, {"n1", "a"}}
	conflictPaths := []types.Path{}
	resolve := func(aChange, bChange types.DiffChangeType, aVal, bVal types.Value, p types.Path) (change types.DiffChangeType, merged types.Value, ok bool) {
		conflictPaths = append(conflictPaths, p)
		if _, ok := aVal.(types.Map); ok || bChange == types.DiffChangeRemoved {
			return bChange, bVal, true
		}
		return aChange, aVal, true
	}

	merged, err := ThreeWay(s.create(a), s.create(b), s.create(p), s.vs, resolve, nil)
	if s.NoError(err) {
		expected := s.create(exp)
		s.True(expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
	}
	if s.Len(conflictPaths, len(expectedConflictPaths), "Wrong number of conflicts!") {
		for i := 0; i < len(conflictPaths); i++ {
			for j, c := range conflictPaths[i] {
				s.Contains(c.String(), expectedConflictPaths[i][j])
			}
		}
	}
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_MergeOurs() {
	p := kvs{"k1", "k-one"}
	a := kvs{"k1", "k-won"}
	b := kvs{"k1", "k-too", "k2", "k-two"}
	exp := kvs{"k1", "k-won", "k2", "k-two"}

	merged, err := ThreeWay(s.create(a), s.create(b), s.create(p), s.vs, Ours, nil)
	if s.NoError(err) {
		expected := s.create(exp)
		s.True(expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
	}
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_MergeTheirs() {
	p := kvs{"k1", "k-one"}
	a := kvs{"k1", "k-won"}
	b := kvs{"k1", "k-too", "k2", "k-two"}
	exp := kvs{"k1", "k-too", "k2", "k-two"}

	merged, err := ThreeWay(s.create(a), s.create(b), s.create(p), s.vs, Theirs, nil)
	if s.NoError(err) {
		expected := s.create(exp)
		s.True(expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
	}
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NilConflict() {
	s.tryThreeWayConflict(nil, s.create(mm2b), s.create(mm2), "Cannot merge nil Value with")
	s.tryThreeWayConflict(s.create(mm2a), nil, s.create(mm2), "with nil Value.")
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_ImmediateConflict() {
	s.tryThreeWayConflict(types.NewSet(s.vs), s.create(mm2b), s.create(mm2), "Cannot merge Set<> with "+s.typeStr)
	s.tryThreeWayConflict(s.create(mm2b), types.NewSet(s.vs), s.create(mm2), "Cannot merge "+s.typeStr)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RefConflict() {
	strRef := s.vs.WriteValue(types.NewStruct("Foo", types.StructData{"life": types.Number(42)}))
	numRef := s.vs.WriteValue(types.Number(7))

	m := kvs{"r2", strRef}
	ma := kvs{"r1", strRef, "r2", strRef}
	mb := kvs{"r1", numRef, "r2", strRef}

	s.tryThreeWayConflict(s.create(ma), s.create(mb), s.create(m), "Cannot merge Struct Foo")
	s.tryThreeWayConflict(s.create(mb), s.create(ma), s.create(m), "Cannot merge Number and Struct Foo")
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NestedConflict() {
	a := mm2a.set("k2", types.NewSet(s.vs))
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), types.EncodedValue(types.NewSet(s.vs)))
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), types.EncodedValue(s.create(aa1b)))
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NestedConflictingOperation() {
	a := mm2a.remove("k2")
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), `removed "k2"`)
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), `modded "k2"`)
}
