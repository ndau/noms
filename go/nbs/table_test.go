// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	"sync"

	"github.com/ndau/noms/go/chunks"
	"github.com/ndau/noms/go/d"
	"github.com/ndau/noms/go/hash"
	"github.com/stretchr/testify/assert"
)

func buildTable(chunks [][]byte) ([]byte, addr) {
	totalData := uint64(0)
	for _, chunk := range chunks {
		totalData += uint64(len(chunk))
	}
	capacity := maxTableSize(uint64(len(chunks)), totalData)

	buff := make([]byte, capacity)

	tw := newTableWriter(buff, nil)

	for _, chunk := range chunks {
		tw.addChunk(computeAddr(chunk), chunk)
	}

	length, blockHash := tw.finish()
	return buff[:length], blockHash
}

func TestSimple(t *testing.T) {
	assert := assert.New(t)

	chunks := [][]byte{
		[]byte("hello2"),
		[]byte("goodbye2"),
		[]byte("badbye2"),
	}

	tableData, _ := buildTable(chunks)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), fileBlockSize)

	assertChunksInReader(chunks, tr, assert)

	assert.Equal(string(chunks[0]), string(tr.get(computeAddr(chunks[0]), &Stats{})))
	assert.Equal(string(chunks[1]), string(tr.get(computeAddr(chunks[1]), &Stats{})))
	assert.Equal(string(chunks[2]), string(tr.get(computeAddr(chunks[2]), &Stats{})))

	notPresent := [][]byte{
		[]byte("yo"),
		[]byte("do"),
		[]byte("so much to do"),
	}

	assertChunksNotInReader(notPresent, tr, assert)

	assert.NotEqual(string(notPresent[0]), string(tr.get(computeAddr(notPresent[0]), &Stats{})))
	assert.NotEqual(string(notPresent[1]), string(tr.get(computeAddr(notPresent[1]), &Stats{})))
	assert.NotEqual(string(notPresent[2]), string(tr.get(computeAddr(notPresent[2]), &Stats{})))
}

func assertChunksInReader(chunks [][]byte, r chunkReader, assert *assert.Assertions) {
	for _, c := range chunks {
		assert.True(r.has(computeAddr(c)))
	}
}

func assertChunksNotInReader(chunks [][]byte, r chunkReader, assert *assert.Assertions) {
	for _, c := range chunks {
		assert.False(r.has(computeAddr(c)))
	}
}

func TestHasMany(t *testing.T) {
	assert := assert.New(t)

	chunks := [][]byte{
		[]byte("hello2"),
		[]byte("goodbye2"),
		[]byte("badbye2"),
	}

	tableData, _ := buildTable(chunks)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), fileBlockSize)

	addrs := addrSlice{computeAddr(chunks[0]), computeAddr(chunks[1]), computeAddr(chunks[2])}
	hasAddrs := []hasRecord{
		{&addrs[0], binary.BigEndian.Uint64(addrs[0][:addrPrefixSize]), 0, false},
		{&addrs[1], binary.BigEndian.Uint64(addrs[1][:addrPrefixSize]), 1, false},
		{&addrs[2], binary.BigEndian.Uint64(addrs[2][:addrPrefixSize]), 2, false},
	}
	sort.Sort(hasRecordByPrefix(hasAddrs))

	tr.hasMany(hasAddrs)
	for _, ha := range hasAddrs {
		assert.True(ha.has, "Nothing for prefix %d", ha.prefix)
	}
}

func TestHasManySequentialPrefix(t *testing.T) {
	assert := assert.New(t)

	// Use bogus addrs so we can generate the case of sequentially non-unique prefixes in the index
	// Note that these are already sorted
	addrStrings := []string{
		"0rfgadopg6h3fk7d253ivbjsij4qo3nv",
		"0rfgadopg6h3fk7d253ivbjsij4qo4nv",
		"0rfgadopg6h3fk7d253ivbjsij4qo9nv",
	}

	addrs := make([]addr, len(addrStrings))
	for i, s := range addrStrings {
		addrs[i] = addr(hash.Parse(s))
	}

	bogusData := []byte("bogus") // doesn't matter what this is. hasMany() won't check chunkRecords
	totalData := uint64(len(bogusData) * len(addrs))

	capacity := maxTableSize(uint64(len(addrs)), totalData)
	buff := make([]byte, capacity)
	tw := newTableWriter(buff, nil)

	for _, a := range addrs {
		tw.addChunk(a, bogusData)
	}

	length, _ := tw.finish()
	buff = buff[:length]

	tr := newTableReader(parseTableIndex(buff), tableReaderAtFromBytes(buff), fileBlockSize)

	hasAddrs := make([]hasRecord, 2)
	// Leave out the first address
	hasAddrs[0] = hasRecord{&addrs[1], addrs[1].Prefix(), 1, false}
	hasAddrs[1] = hasRecord{&addrs[2], addrs[2].Prefix(), 2, false}

	tr.hasMany(hasAddrs)

	for _, ha := range hasAddrs {
		assert.True(ha.has, fmt.Sprintf("Nothing for prefix %x\n", ha.prefix))
	}
}

func TestGetMany(t *testing.T) {
	assert := assert.New(t)

	data := [][]byte{
		[]byte("hello2"),
		[]byte("goodbye2"),
		[]byte("badbye2"),
	}

	tableData, _ := buildTable(data)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), fileBlockSize)

	addrs := addrSlice{computeAddr(data[0]), computeAddr(data[1]), computeAddr(data[2])}
	getBatch := []getRecord{
		{&addrs[0], binary.BigEndian.Uint64(addrs[0][:addrPrefixSize]), false},
		{&addrs[1], binary.BigEndian.Uint64(addrs[1][:addrPrefixSize]), false},
		{&addrs[2], binary.BigEndian.Uint64(addrs[2][:addrPrefixSize]), false},
	}
	sort.Sort(getRecordByPrefix(getBatch))

	wg := &sync.WaitGroup{}

	chunkChan := make(chan *chunks.Chunk, len(getBatch))
	tr.getMany(getBatch, chunkChan, wg, &Stats{})
	wg.Wait()
	close(chunkChan)

	gotCount := 0
	for range chunkChan {
		gotCount++
	}

	assert.True(gotCount == len(getBatch))
}

func TestCalcReads(t *testing.T) {
	assert := assert.New(t)

	chunks := [][]byte{
		[]byte("hello2"),
		[]byte("goodbye2"),
		[]byte("badbye2"),
	}

	tableData, _ := buildTable(chunks)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), 0)
	addrs := addrSlice{computeAddr(chunks[0]), computeAddr(chunks[1]), computeAddr(chunks[2])}
	getBatch := []getRecord{
		{&addrs[0], binary.BigEndian.Uint64(addrs[0][:addrPrefixSize]), false},
		{&addrs[1], binary.BigEndian.Uint64(addrs[1][:addrPrefixSize]), false},
		{&addrs[2], binary.BigEndian.Uint64(addrs[2][:addrPrefixSize]), false},
	}

	gb2 := []getRecord{getBatch[0], getBatch[2]}
	sort.Sort(getRecordByPrefix(getBatch))

	reads, remaining := tr.calcReads(getBatch, 0)
	assert.False(remaining)
	assert.Equal(1, reads)

	sort.Sort(getRecordByPrefix(gb2))
	reads, remaining = tr.calcReads(gb2, 0)
	assert.False(remaining)
	assert.Equal(2, reads)
}

func TestExtract(t *testing.T) {
	assert := assert.New(t)

	chunks := [][]byte{
		[]byte("hello2"),
		[]byte("goodbye2"),
		[]byte("badbye2"),
	}

	tableData, _ := buildTable(chunks)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), fileBlockSize)

	addrs := addrSlice{computeAddr(chunks[0]), computeAddr(chunks[1]), computeAddr(chunks[2])}

	chunkChan := make(chan extractRecord)
	go func() { tr.extract(chunkChan); close(chunkChan) }()
	i := 0
	for rec := range chunkChan {
		assert.NotNil(rec.data, "Nothing for", addrs[i])
		assert.Equal(addrs[i], rec.a)
		assert.Equal(chunks[i], rec.data)
		i++
	}
}

func Test65k(t *testing.T) {
	assert := assert.New(t)

	count := 1 << 16
	chunks := make([][]byte, count)

	dataFn := func(i int) []byte {
		return []byte(fmt.Sprintf("data%d", i*2))
	}

	for i := 0; i < count; i++ {
		chunks[i] = dataFn(i)
	}

	tableData, _ := buildTable(chunks)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), fileBlockSize)

	for i := 0; i < count; i++ {
		data := dataFn(i)
		h := computeAddr(data)
		assert.True(tr.has(computeAddr(data)))
		assert.Equal(string(data), string(tr.get(h, &Stats{})))
	}

	for i := count; i < count*2; i++ {
		data := dataFn(i)
		h := computeAddr(data)
		assert.False(tr.has(computeAddr(data)))
		assert.NotEqual(string(data), string(tr.get(h, &Stats{})))
	}
}

// Ensure all addresses share the first 7 bytes. Useful for easily generating tests which have
// "prefix" collisions.
func computeAddrCommonPrefix(data []byte) addr {
	a := computeAddrDefault(data)
	a[0] = 0x01
	a[1] = 0x23
	a[2] = 0x45
	a[3] = 0x67
	a[4] = 0x89
	a[5] = 0xab
	a[6] = 0xcd
	return a
}

func doTestNGetMany(t *testing.T, count int) {
	assert := assert.New(t)

	data := make([][]byte, count)

	dataFn := func(i int) []byte {
		return []byte(fmt.Sprintf("data%d", i*2))
	}

	for i := 0; i < count; i++ {
		data[i] = dataFn(i)
	}

	tableData, _ := buildTable(data)
	tr := newTableReader(parseTableIndex(tableData), tableReaderAtFromBytes(tableData), fileBlockSize)

	getBatch := make([]getRecord, len(data))
	for i := 0; i < count; i++ {
		a := computeAddr(dataFn(i))
		getBatch[i] = getRecord{&a, a.Prefix(), false}
	}

	sort.Sort(getRecordByPrefix(getBatch))

	wg := &sync.WaitGroup{}
	chunkChan := make(chan *chunks.Chunk, len(getBatch))
	tr.getMany(getBatch, chunkChan, wg, &Stats{})
	wg.Wait()
	close(chunkChan)

	gotCount := 0
	for range chunkChan {
		gotCount++
	}

	assert.True(gotCount == len(getBatch))
}

func Test65kGetMany(t *testing.T) {
	doTestNGetMany(t, 1<<16)
}

func Test2kGetManyCommonPrefix(t *testing.T) {
	computeAddr = computeAddrCommonPrefix
	defer func() {
		computeAddr = computeAddrDefault
	}()

	doTestNGetMany(t, 1<<11)
}

func TestEmpty(t *testing.T) {
	assert := assert.New(t)

	buff := make([]byte, footerSize)
	tw := newTableWriter(buff, nil)
	length, _ := tw.finish()
	assert.Equal(length, footerSize)

	d.PanicIfError(nil)
}
