// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ndau/noms/go/d"
)

var EmptyStructType = MakeStructType("")
var EmptyStruct = newStruct("", nil, nil)

type StructData map[string]Value

type Struct struct {
	valueImpl
}

// readStruct reads the data provided by a decoder and moves the decoder forward.
func readStruct(dec *valueDecoder) Struct {
	start := dec.pos()
	skipStruct(dec)
	end := dec.pos()
	return Struct{valueImpl{dec.vrw, dec.byteSlice(start, end), nil}}
}

func skipStruct(dec *valueDecoder) {
	dec.skipKind()
	dec.skipString() // name
	count := dec.readCount()
	for i := uint64(0); i < count; i++ {
		dec.skipString()
		dec.skipValue()
	}
}

func isStructSameTypeForSure(dec *valueDecoder, t *Type) bool {
	desc := t.Desc.(StructDesc)
	dec.skipKind()
	if !dec.isStringSame(desc.Name) {
		return false
	}
	count := dec.readCount()
	if count != uint64(len(desc.fields)) {
		return false
	}
	for i := uint64(0); i < count; i++ {
		if desc.fields[i].Optional {
			return false
		}
		if !dec.isStringSame(desc.fields[i].Name) {
			return false
		}

		if !dec.isValueSameTypeForSure(desc.fields[i].Type) {
			return false
		}
	}
	return true
}

func walkStruct(r *refWalker, cb RefCallback) {
	r.skipKind()
	r.skipString() // name
	count := r.readCount()
	for i := uint64(0); i < count; i++ {
		r.skipString()
		r.walkValue(cb)
	}
}

func newStruct(name string, fieldNames []string, values []Value) Struct {
	var vrw ValueReadWriter
	w := newBinaryNomsWriter()
	StructKind.writeTo(&w)
	w.writeString(name)
	w.writeCount(uint64(len(fieldNames)))
	for i := 0; i < len(fieldNames); i++ {
		w.writeString(fieldNames[i])
		if vrw == nil {
			vrw = values[i].(valueReadWriter).valueReadWriter()
		}
		values[i].writeTo(&w)
	}
	return Struct{valueImpl{vrw, w.data(), nil}}
}

func NewStruct(name string, data StructData) Struct {
	verifyStructName(name)
	fieldNames := make([]string, len(data))
	values := make([]Value, len(data))

	i := 0
	for name := range data {
		verifyFieldName(name)
		fieldNames[i] = name
		i++
	}

	sort.Sort(sort.StringSlice(fieldNames))
	for i = 0; i < len(fieldNames); i++ {
		values[i] = data[fieldNames[i]]
	}

	return newStruct(name, fieldNames, values)
}

// StructTemplate allows creating a template for structs with a known shape
// (name and fields). If a lot of structs of the same shape are being created
// then using a StructTemplate makes that slightly more efficient.
type StructTemplate struct {
	name       string
	fieldNames []string
}

// MakeStructTemplate creates a new StructTemplate or panics if the name and
// fields are not valid.
func MakeStructTemplate(name string, fieldNames []string) (t StructTemplate) {
	t = StructTemplate{name, fieldNames}

	verifyStructName(name)
	if len(fieldNames) == 0 {
		return
	}
	verifyFieldName(fieldNames[0])
	for i := 1; i < len(fieldNames); i++ {
		verifyFieldName(fieldNames[i])
		d.PanicIfFalse(fieldNames[i] > fieldNames[i-1])
	}
	return
}

// NewStruct creates a new Struct from the StructTemplate. The order of the
// values must match the order of the field names of the StructTemplate.
func (st StructTemplate) NewStruct(values []Value) Struct {
	d.PanicIfFalse(len(st.fieldNames) == len(values))
	return newStruct(st.name, st.fieldNames, values)
}

func (s Struct) Empty() bool {
	return s.Len() == 0
}

// Value interface
func (s Struct) Value() Value {
	return s
}

func (s Struct) WalkValues(cb ValueCallback) {
	dec, count := s.decoderSkipToFields()
	for i := uint64(0); i < count; i++ {
		dec.skipString()
		cb(dec.readValue())
	}
}

func (s Struct) typeOf() *Type {
	dec := s.decoder()
	return readStructTypeOfValue(&dec)
}

func readStructTypeOfValue(dec *valueDecoder) *Type {
	dec.skipKind()
	name := dec.readString()
	count := dec.readCount()
	typeFields := make(structTypeFields, count)
	for i := uint64(0); i < count; i++ {
		typeFields[i] = StructField{
			Name:     dec.readString(),
			Optional: false,
			Type:     dec.readTypeOfValue(),
		}
	}
	return makeStructTypeQuickly(name, typeFields)
}

func (s Struct) decoderSkipToFields() (valueDecoder, uint64) {
	dec := s.decoder()
	dec.skipKind()
	dec.skipString()
	count := dec.readCount()
	return dec, count
}

// Len is the number of fields in the struct.
func (s Struct) Len() int {
	_, count := s.decoderSkipToFields()
	return int(count)
}

// Name is the name of the struct.
func (s Struct) Name() string {
	dec := s.decoder()
	dec.skipKind()
	return dec.readString()
}

// IterFields iterates over the fields, calling cb for every field in the
// struct.
func (s Struct) IterFields(cb func(name string, value Value) (stop bool)) {
	dec, count := s.decoderSkipToFields()
	for i := uint64(0); i < count; i++ {
		if cb(dec.readString(), dec.readValue()) {
			break
		}
	}
}

type structPartCallbacks interface {
	name(n string)
	count(c uint64)
	fieldName(n string)
	fieldValue(v Value)
	end()
}

func (s Struct) iterParts(cbs structPartCallbacks) {
	dec := s.decoder()
	dec.skipKind()
	cbs.name(dec.readString())
	count := dec.readCount()
	cbs.count(count)
	for i := uint64(0); i < count; i++ {
		cbs.fieldName(dec.readString())
		cbs.fieldValue(dec.readValue())
	}
	cbs.end()
}

// MaybeGet returns the value of a field in the struct. If the struct does not a have a field with
// the name name then this returns (nil, false).
func (s Struct) MaybeGet(n string) (v Value, found bool) {
	dec, count := s.decoderSkipToFields()
	for i := uint64(0); i < count; i++ {
		name := dec.readString()
		if name == n {
			found = true
			v = dec.readValue()
			return
		}
		if name > n {
			return
		}
		dec.skipValue()
	}

	return
}

// Get returns the value of a field in the struct. If the struct does not a have a field with the
// name name then this panics.
func (s Struct) Get(n string) Value {
	v, ok := s.MaybeGet(n)
	if !ok {
		d.Chk.Fail(fmt.Sprintf(`Struct has no field "%s"`, n))
	}
	return v
}

// Set returns a new struct where the field name has been set to value. If name is not an
// existing field in the struct or the type of value is different from the old value of the
// struct field a new struct type is created.
func (s Struct) Set(n string, v Value) Struct {
	verifyFieldName(n)

	prolog, head, tail, count, found := s.splitFieldsAt(n)

	w := binaryNomsWriter{make([]byte, len(s.buff)), 0}
	w.writeRaw(prolog)

	if !found {
		count++
	}
	w.writeCount(count)
	w.writeRaw(head)
	w.writeString(n)
	v.writeTo(&w)
	w.writeRaw(tail)

	return Struct{valueImpl{s.vrw, w.data(), nil}}
}

func (s Struct) SetName(name string) Struct {
	verifyStructName(name)

	w := binaryNomsWriter{make([]byte, len(s.buff)), 0}
	StructKind.writeTo(&w)
	w.writeString(name)

	dec := s.decoder()
	dec.skipKind()
	dec.skipString()

	w.writeRaw(dec.buff[dec.offset:])
	return Struct{valueImpl{s.vrw, w.data(), nil}}
}

// splitFieldsAt splits the buffer into two parts. The fields coming before the field we are looking for
// and the fields coming after it.
func (s Struct) splitFieldsAt(name string) (prolog, head, tail []byte, count uint64, found bool) {
	dec := s.decoder()
	dec.skipKind()
	dec.skipString()
	prolog = dec.buff[:dec.offset]
	count = dec.readCount()
	fieldsOffset := dec.offset

	for i := uint64(0); i < count; i++ {
		beforeCurrent := dec.offset
		fn := dec.readString()
		dec.skipValue()

		if fn == name {
			found = true
			head = dec.buff[fieldsOffset:beforeCurrent]
			tail = dec.buff[dec.offset:len(dec.buff)]
			break
		}

		if name < fn {
			head = dec.buff[fieldsOffset:beforeCurrent]
			tail = dec.buff[beforeCurrent:len(dec.buff)]
			break
		}
	}

	if head == nil && tail == nil {
		head = dec.buff[fieldsOffset:dec.offset]
	}

	return
}

// Delete returns a new struct where the field name has been removed.
// If name is not an existing field in the struct then the current struct is returned.
func (s Struct) Delete(n string) Struct {
	prolog, head, tail, count, found := s.splitFieldsAt(n)
	if !found {
		return s
	}

	w := binaryNomsWriter{make([]byte, len(s.buff)), 0}
	w.writeRaw(prolog)
	w.writeCount(count - 1)
	w.writeRaw(head)
	w.writeRaw(tail)

	return Struct{valueImpl{s.vrw, w.data(), nil}}
}

func (s Struct) Diff(last Struct, changes chan<- ValueChanged, closeChan <-chan struct{}) {
	if s.Equals(last) {
		return
	}
	dec1, dec2 := s.decoder(), last.decoder()
	dec1.skipKind()
	dec2.skipKind()
	dec1.skipString() // Ignore names
	dec2.skipString()
	count1, count2 := dec1.readCount(), dec2.readCount()
	i1, i2 := uint64(0), uint64(0)
	var fn1, fn2 string

	for i1 < count1 && i2 < count2 {
		if fn1 == "" {
			fn1 = dec1.readString()
		}
		if fn2 == "" {
			fn2 = dec2.readString()
		}
		var change ValueChanged
		if fn1 == fn2 {
			v1, v2 := dec1.readValue(), dec2.readValue()
			if !v1.Equals(v2) {
				change = ValueChanged{DiffChangeModified, String(fn1), v2, v1}
			}
			i1++
			i2++
			fn1, fn2 = "", ""
		} else if fn1 < fn2 {
			v1 := dec1.readValue()
			change = ValueChanged{DiffChangeAdded, String(fn1), nil, v1}
			i1++
			fn1 = ""
		} else {
			v2 := dec2.readValue()
			change = ValueChanged{DiffChangeRemoved, String(fn2), v2, nil}
			i2++
			fn2 = ""
		}

		if change != (ValueChanged{}) && !sendChange(changes, closeChan, change) {
			return
		}
	}

	for ; i1 < count1; i1++ {
		if fn1 == "" {
			fn1 = dec1.readString()
		}
		v1 := dec1.readValue()
		if !sendChange(changes, closeChan, ValueChanged{DiffChangeAdded, String(fn1), nil, v1}) {
			return
		}
	}

	for ; i2 < count2; i2++ {
		if fn2 == "" {
			fn2 = dec2.readString()
		}
		v2 := dec2.readValue()
		if !sendChange(changes, closeChan, ValueChanged{DiffChangeRemoved, String(fn2), v2, nil}) {
			return
		}
	}
}

var escapeChar = "Q"
var headFieldNamePattern = regexp.MustCompile("[a-zA-Z]")
var tailFieldNamePattern = regexp.MustCompile("[a-zA-Z0-9_]")
var spaceRegex = regexp.MustCompile("[ ]")
var escapeRegex = regexp.MustCompile(escapeChar)

var fieldNameComponentRe = regexp.MustCompile("^" + headFieldNamePattern.String() + tailFieldNamePattern.String() + "*")
var fieldNameRe = regexp.MustCompile(fieldNameComponentRe.String() + "$")

type encodingFunc func(string, *regexp.Regexp) string

func CamelCaseFieldName(input string) string {
	//strip invalid struct characters and leave spaces
	encode := func(s1 string, p *regexp.Regexp) string {
		if p.MatchString(s1) || spaceRegex.MatchString(s1) {
			return s1
		}
		return ""
	}

	strippedField := escapeField(input, encode)
	splitField := strings.Fields(strippedField)

	if len(splitField) == 0 {
		return ""
	}

	//Camelcase field
	output := strings.ToLower(splitField[0])
	if len(splitField) > 1 {
		for _, field := range splitField[1:] {
			output += strings.Title(strings.ToLower(field))
		}
	}
	//Because we are removing characters, we may generate an invalid field name
	//i.e. -- 1A B, we will remove the first bad chars and process until 1aB
	//1aB is invalid struct field name so we will return ""
	if !IsValidStructFieldName(output) {
		return ""
	}
	return output
}

func escapeField(input string, encode encodingFunc) string {
	output := ""
	pattern := headFieldNamePattern
	for _, ch := range input {
		output += encode(string([]rune{ch}), pattern)
		pattern = tailFieldNamePattern
	}
	return output
}

// EscapeStructField escapes names for use as noms structs with regards to non CSV imported data.
// Disallowed characters are encoded as 'Q<hex-encoded-utf8-bytes>'.
// Note that Q itself is also escaped since it is the escape character.
func EscapeStructField(input string) string {
	if !escapeRegex.MatchString(input) && IsValidStructFieldName(input) {
		return input
	}
	encode := func(s1 string, p *regexp.Regexp) string {
		if p.MatchString(s1) && s1 != escapeChar {
			return s1
		}

		var hs = fmt.Sprintf("%X", s1)
		var buf bytes.Buffer
		buf.WriteString(escapeChar)
		if len(hs) == 1 {
			buf.WriteString("0")
		}
		buf.WriteString(hs)
		return buf.String()
	}
	return escapeField(input, encode)
}

// IsValidStructFieldName returns whether the name is valid as a field name in a struct.
// Valid names must start with `a-zA-Z` and after that `a-zA-Z0-9_`.
func IsValidStructFieldName(name string) bool {
	for i, c := range name {
		if i == 0 {
			if !isAlpha(c) {
				return false
			}
		} else if !isAlphaNumOrUnderscore(c) {
			return false
		}
	}
	return len(name) != 0
}

func isAlpha(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isAlphaNumOrUnderscore(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

func verifyFields(fs structTypeFields) {
	for i, f := range fs {
		verifyFieldName(f.Name)
		if i > 0 && strings.Compare(fs[i-1].Name, f.Name) >= 0 {
			d.Chk.Fail("Field names must be unique and ordered alphabetically")
		}
	}
}

func verifyName(name, kind string) {
	if !IsValidStructFieldName(name) {
		d.Panic(`Invalid struct%s name: "%s"`, kind, name)
	}
}

func verifyFieldName(name string) {
	verifyName(name, " field")
}

func verifyStructName(name string) {
	if name != "" {
		verifyName(name, "")
	}
}
