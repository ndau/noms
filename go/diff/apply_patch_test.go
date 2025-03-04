// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package diff

import (
	"testing"

	"github.com/ndau/noms/go/chunks"
	"github.com/ndau/noms/go/d"
	"github.com/ndau/noms/go/marshal"
	"github.com/ndau/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestCommonPrefixCount(t *testing.T) {
	assert := assert.New(t)

	testCases := [][]interface{}{
		{".value[#94a2oa20oka0jdv5lha03vuvvumul1vb].sizes[#316j9oc39b09fbc2qf3klenm6p1o1d7h]", 0},
		{".value[#94a2oa20oka0jdv5lha03vuvvumul1vb].sizes[#77eavttned7llu1pkvhaei9a9qgcagir]", 3},
		{".value[#94a2oa20oka0jdv5lha03vuvvumul1vb].sizes[#hboaq9581drq4g9jf62d3s06al3us49s]", 3},
		{".value[#94a2oa20oka0jdv5lha03vuvvumul1vb].sizes[#l0hpa7sbr7qutrcfn5173kar4j2847m1]", 3},
		{".value[#9vj5m3049mav94bttcujhgfdfqcavsbn].sizes[#33f6tb4h8agh57s2bqlmi9vbhlkbtmct]", 1},
		{".value[#9vj5m3049mav94bttcujhgfdfqcavsbn].sizes[#a43ne9a8kotcqph4up5pqqdmr1e1qcsl]", 3},
		{".value[#9vj5m3049mav94bttcujhgfdfqcavsbn].sizes[#ppqg6pem2sb64h2i2ptnh8ckj8gogj9h]", 3},
		{".value[#9vj5m3049mav94bttcujhgfdfqcavsbn].sizes[#s7r2vpnqlk20sd72mg8ijerg9cmauaqo]", 3},
		{".value[#bpspmmlc41pk0r144a7682oah0tmge1e].sizes[#9vuc1gg3c3eude5v3j5deqopjsobe3no]", 1},
		{".value[#bpspmmlc41pk0r144a7682oah0tmge1e].sizes[#qo3gfdsf14v3dh0oer82vn1bg4o8nlsc]", 3},
		{".value[#bpspmmlc41pk0r144a7682oah0tmge1e].sizes[#rlidki5ipbjdofsm2rq3a66v908m5fpl]", 3},
		{".value[#bpspmmlc41pk0r144a7682oah0tmge1e].sizes[#st1n96rh89c2vgo090dt9lknd5ip4kck]", 3},
		{".value[#hjh5hpn55591k0gjvgckc14erli968ao].sizes[#267889uv3mtih6fij3fhio2jiqtl6nho]", 1},
		{".value[#hjh5hpn55591k0gjvgckc14erli968ao].sizes[#7ncb7guoip9e400bm2lcvr0dda29o9jn]", 3},
		{".value[#hjh5hpn55591k0gjvgckc14erli968ao].sizes[#afscb0on7rt8bq6eutup8juusmid7i96]", 3},
		{".value[#hjh5hpn55591k0gjvgckc14erli968ao].sizes[#drqe4lr0vdfdtmvejsjun1l3mfv6ums5]", 3},
	}

	var lastPath types.Path

	for i, tc := range testCases {
		path, expected := tc[0].(string), tc[1].(int)
		p, err := types.ParsePath(path)
		assert.NoError(err)
		assert.Equal(expected, commonPrefixCount(lastPath, p), "failed for paths[%d]: %s", i, path)
		lastPath = p
	}
}

type testFunc func(parent types.Value) types.Value
type testKey struct {
	X, Y int
}

var (
	vm map[string]types.Value
)

func vfk(keys ...string) []types.Value {
	var values []types.Value
	for _, k := range keys {
		values = append(values, vm[k])
	}
	return values
}

func testValues(vrw types.ValueReadWriter) map[string]types.Value {
	if vm == nil {
		vm = map[string]types.Value{
			"k1":      types.String("k1"),
			"k2":      types.String("k2"),
			"k3":      types.String("k3"),
			"s1":      types.String("string1"),
			"s2":      types.String("string2"),
			"s3":      types.String("string3"),
			"s4":      types.String("string4"),
			"n1":      types.Number(1),
			"n2":      types.Number(2),
			"n3":      types.Number(3.3),
			"n4":      types.Number(4.4),
			"b1":      mustMarshal(true),
			"b2":      mustMarshal(false),
			"l1":      mustMarshal([]string{}),
			"l2":      mustMarshal([]string{"one", "two", "three", "four"}),
			"l3":      mustMarshal([]string{"two", "three", "four", "five"}),
			"l4":      mustMarshal([]string{"two", "three", "four"}),
			"l5":      mustMarshal([]string{"one", "two", "three", "four", "five"}),
			"l6":      mustMarshal([]string{"one", "four"}),
			"struct1": types.NewStruct("test1", types.StructData{"f1": types.Number(1), "f2": types.Number(2)}),
			"struct2": types.NewStruct("test1", types.StructData{"f1": types.Number(11111), "f2": types.Number(2)}),
			"struct3": types.NewStruct("test1", types.StructData{"f1": types.Number(1), "f2": types.Number(2), "f3": types.Number(3)}),
			"struct4": types.NewStruct("test1", types.StructData{"f2": types.Number(2)}),
			"m1":      mustMarshal(map[string]int{}),
			"m2":      mustMarshal(map[string]int{"k1": 1, "k2": 2, "k3": 3}),
			"m3":      mustMarshal(map[string]int{"k2": 2, "k3": 3, "k4": 4}),
			"m4":      mustMarshal(map[string]int{"k1": 1, "k3": 3}),
			"m5":      mustMarshal(map[string]int{"k1": 1, "k2": 2222, "k3": 3}),
			"ms1":     mustMarshal(map[testKey]int{{1, 1}: 1, {2, 2}: 2, {3, 3}: 3}),
			"ms2":     mustMarshal(map[testKey]int{{1, 1}: 1, {4, 4}: 4, {5, 5}: 5}),
		}

		vm["mh1"] = types.NewMap(vrw, vfk("k1", "struct1", "k2", "l1")...)
		vm["mh2"] = types.NewMap(vrw, vfk("k1", "n1", "k2", "l2", "k3", "l3")...)
		vm["set1"] = types.NewSet(vrw)
		vm["set2"] = types.NewSet(vrw, vfk("s1", "s2")...)
		vm["set3"] = types.NewSet(vrw, vfk("s1", "s2", "s3")...)
		vm["set1"] = types.NewSet(vrw, vfk("s2")...)
		vm["seth1"] = types.NewSet(vrw, vfk("struct1", "struct2", "struct3")...)
		vm["seth2"] = types.NewSet(vrw, vfk("struct2", "struct3")...)
		vm["setj3"] = types.NewSet(vrw, vfk("struct1")...)
		vm["mk1"] = types.NewMap(vrw, vfk("struct1", "s1", "struct2", "s2")...)
		vm["mk2"] = types.NewMap(vrw, vfk("struct1", "s3", "struct4", "s4")...)
	}
	return vm
}

func newTestValueStore() *types.ValueStore {
	st := &chunks.TestStorage{}
	return types.NewValueStore(st.NewView())
}

func getPatch(g1, g2 types.Value) Patch {
	dChan := make(chan Difference)
	sChan := make(chan struct{})
	go func() {
		Diff(g1, g2, dChan, sChan, true)
		close(dChan)
	}()

	patch := Patch{}
	for dif := range dChan {
		patch = append(patch, dif)
	}
	return patch
}

func checkApplyPatch(assert *assert.Assertions, g1, expectedG2 types.Value, k1, k2 string) {
	patch := getPatch(g1, expectedG2)
	g2 := Apply(g1, patch)
	assert.True(expectedG2.Equals(g2), "failed to apply diffs for k1: %s and k2: %s", k1, k2)
}

func TestPatches(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	cnt := 0
	for k1, g1 := range testValues(vs) {
		for k2, expectedG2 := range testValues(vs) {
			if k1 != k2 {
				cnt++
				checkApplyPatch(assert, g1, expectedG2, k1, k2)
			}
		}
	}
}

func TestNestedLists(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	ol1 := mustMarshal([]string{"one", "two", "three", "four"})
	nl1 := mustMarshal([]string{"two", "three"})
	ol2 := mustMarshal([]int{2, 3})
	nl2 := mustMarshal([]int{1, 2, 3, 4})
	nl3 := mustMarshal([]bool{true, false, true})
	g1 := types.NewList(vs, ol1, ol2)
	g2 := types.NewList(vs, nl1, nl2, nl3)
	checkApplyPatch(assert, g1, g2, "g1", "g2")
}

func TestUpdateNode(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	doTest := func(pp types.PathPart, parent, ov, nv, exp types.Value, f testFunc) {
		stack := &patchStack{}
		se := &stackElem{path: []types.PathPart{pp}, pathPart: pp, changeType: types.DiffChangeModified, oldValue: ov, newValue: nv}
		updated := stack.updateNode(se, parent)
		testVal := f(updated)
		assert.True(exp.Equals(testVal), "%s != %s", nv, testVal)
	}

	var pp types.PathPart
	oldVal := types.String("Yo")
	newVal := types.String("YooHoo")

	s1 := types.NewStruct("TestStruct", types.StructData{"f1": types.Number(1), "f2": oldVal})
	pp = types.FieldPath{Name: "f2"}
	doTest(pp, s1, oldVal, newVal, newVal, func(parent types.Value) types.Value {
		return parent.(types.Struct).Get("f2")
	})

	l1 := types.NewList(vs, types.String("one"), oldVal, types.String("three"))
	pp = types.IndexPath{Index: types.Number(1)}
	doTest(pp, l1, oldVal, newVal, newVal, func(parent types.Value) types.Value {
		return parent.(types.List).Get(1)
	})

	m1 := types.NewMap(vs, types.String("k1"), types.Number(1), types.String("k2"), oldVal)
	pp = types.IndexPath{Index: types.String("k2")}
	doTest(pp, m1, oldVal, newVal, newVal, func(parent types.Value) types.Value {
		return parent.(types.Map).Get(types.String("k2"))
	})

	k1 := types.NewStruct("Sizes", types.StructData{"height": types.Number(200), "width": types.Number(300)})
	vs.WriteValue(k1)
	m1 = types.NewMap(vs, k1, oldVal)
	pp = types.HashIndexPath{Hash: k1.Hash()}
	doTest(pp, m1, oldVal, newVal, newVal, func(parent types.Value) types.Value {
		return parent.(types.Map).Get(k1)
	})

	set1 := types.NewSet(vs, oldVal, k1)
	pp = types.IndexPath{Index: oldVal}
	exp := types.NewSet(vs, newVal, k1)
	doTest(pp, set1, oldVal, newVal, exp, func(parent types.Value) types.Value {
		return parent
	})

	k2 := types.NewStruct("Sizes", types.StructData{"height": types.Number(300), "width": types.Number(500)})
	set1 = types.NewSet(vs, oldVal, k1)
	pp = types.HashIndexPath{Hash: k1.Hash()}
	exp = types.NewSet(vs, oldVal, k2)
	doTest(pp, set1, k1, k2, exp, func(parent types.Value) types.Value {
		return parent
	})
}

func checkApplyDiffs(a *assert.Assertions, n1, n2 types.Value, leftRight bool) {
	dChan := make(chan Difference)
	sChan := make(chan struct{})
	go func() {
		Diff(n1, n2, dChan, sChan, leftRight)
		close(dChan)
	}()

	difs := Patch{}
	for dif := range dChan {
		difs = append(difs, dif)
	}

	res := Apply(n1, difs)
	a.True(n2.Equals(res))
}

func tryApplyDiff(a *assert.Assertions, a1, a2 interface{}) {
	n1 := mustMarshal(a1)
	n2 := mustMarshal(a2)

	checkApplyDiffs(a, n1, n2, true)
	checkApplyDiffs(a, n1, n2, false)
	checkApplyDiffs(a, n2, n1, true)
	checkApplyDiffs(a, n2, n1, false)
}

func TestUpdateList(t *testing.T) {
	a := assert.New(t)

	// insert at beginning
	a1 := []interface{}{"five", "ten", "fifteen"}
	a2 := []interface{}{"one", "two", "three", "five", "ten", "fifteen"}
	tryApplyDiff(a, a1, a2)

	// append at end
	a1 = []interface{}{"five", "ten", "fifteen"}
	a2 = []interface{}{"five", "ten", "fifteen", "twenty", "twenty-five"}
	tryApplyDiff(a, a1, a2)

	// insert interleaved
	a1 = []interface{}{"one", "three", "five", "seven"}
	a2 = []interface{}{"one", "two", "three", "four", "five", "six", "seven"}
	tryApplyDiff(a, a1, a2)

	// delete from beginning and append to end
	a1 = []interface{}{"one", "two", "three", "four", "five"}
	a2 = []interface{}{"four", "five", "six", "seven"}
	tryApplyDiff(a, a1, a2)

	// replace entries at beginning
	a1 = []interface{}{"one", "two", "three", "four", "five"}
	a2 = []interface{}{"3.5", "four", "five"}
	tryApplyDiff(a, a1, a2)

	// replace entries at end
	a1 = []interface{}{"one", "two", "three"}
	a2 = []interface{}{"one", "four"}
	tryApplyDiff(a, a1, a2)

	// insert at beginning, replace at end
	a1 = []interface{}{"five", "ten", "fifteen"}
	a2 = []interface{}{"one", "two", "five", "eight", "eleven", "sixteen", "twenty"}
	tryApplyDiff(a, a1, a2)

	// remove everything
	a1 = []interface{}{"five", "ten", "fifteen"}
	a2 = []interface{}{}
	tryApplyDiff(a, a1, a2)
}

func TestUpdateMap(t *testing.T) {
	a := assert.New(t)

	// insertions, deletions, and replacements
	a1 := map[string]int{"five": 5, "ten": 10, "fifteen": 15, "twenty": 20}
	a2 := map[string]int{"one": 1, "two": 2, "three": 3, "five": 5, "ten": 10, "fifteen": 15, "twenty": 2020}
	tryApplyDiff(a, a1, a2)

	// delete everything
	a1 = map[string]int{"five": 5, "ten": 10, "fifteen": 15, "twenty": 20}
	a2 = map[string]int{}
	tryApplyDiff(a, a1, a2)
}

func TestUpdateStruct(t *testing.T) {
	a := assert.New(t)

	a1 := types.NewStruct("tStruct", types.StructData{
		"f1": types.Number(1),
		"f2": types.String("two"),
		"f3": mustMarshal([]string{"one", "two", "three"}),
	})
	a2 := types.NewStruct("tStruct", types.StructData{
		"f1": types.Number(2),
		"f2": types.String("twotwo"),
		"f3": mustMarshal([]interface{}{0, "one", 1, "two", 2, "three", 3}),
	})
	checkApplyDiffs(a, a1, a2, true)
	checkApplyDiffs(a, a1, a2, false)

	a2 = types.NewStruct("tStruct", types.StructData{
		"f1": types.Number(2),
		"f2": types.String("two"),
		"f3": mustMarshal([]interface{}{0, "one", 1, "two", 2, "three", 3}),
		"f4": types.Bool(true),
	})
	checkApplyDiffs(a, a1, a2, true)
	checkApplyDiffs(a, a1, a2, false)
}

func TestUpdateSet(t *testing.T) {
	a := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	a1 := types.NewSet(vs, types.Number(1), types.String("two"), mustMarshal([]string{"one", "two", "three"}))
	a2 := types.NewSet(vs, types.Number(3), types.String("three"), mustMarshal([]string{"one", "two", "three", "four"}))

	checkApplyDiffs(a, a1, a2, true)
	checkApplyDiffs(a, a1, a2, false)
	checkApplyDiffs(a, a2, a1, true)
	checkApplyDiffs(a, a2, a1, false)
}

func mustMarshal(v interface{}) types.Value {
	vs := newTestValueStore()
	defer vs.Close()

	v1, err := marshal.Marshal(vs, v)
	d.Chk.NoError(err)
	return v1
}
