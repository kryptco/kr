// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package datasource defines the interfaces a system must implement to support
// querying.
//
// The Database interface is used to get Table interfaces (by name).
// The Table interface is used to get a KeyValueStream (by key prefixes).
// The KeyValueStream interface is used to iterate over key-value pairs from a
// table.
// Note: Order, Index, GetIndexFields and the indexRanges arg to Scan
//       are being provided in beta form for use by discovery.  Currently only
//       indexes of type string are supported and an index must comprise exactly
//       one column.  This API will change when secondary indexes are fully supported.
package datasource

import (
	"fmt"
	"v.io/v23/context"
	"v.io/v23/vdl"
	"v.io/v23/vom"
)

type Database interface {
	// GetContext returns a context (used for creating error messages).
	GetContext() *context.T

	// GetTable returns an instance of the Table inteface for the table
	// specified by name.  If writeAccessReq is true, the Table needs
	// to support the Delete function.  If it cannot, the syncql.NotWritable
	// error should be returned.
	GetTable(name string, writeAccessReq bool) (Table, error)
}

type Table interface {

	// Return the fields on which there exist secondary indexes.
	// The possible ranges for these fields will be passed to Scan.
	// Example:
	// return []datasource.Index{
	//                datasource.Index{FieldName: "v.InterfaceName", Kind: vdl.String},
	//                datasource.Index{FieldName: "v.Address", Kind: vdl.String},
	// }
	// At present, the Kind MUST BE vdl.String
	GetIndexFields() []Index

	// Return a KeyValueStream where all k/v pairs fall within the range
	// of the index ranges passed in (the first of which is for the key).
	// Note: an empty string prefix (""), matches all keys.
	// The index ranges will be sorted (low to high).  The first index range
	// will be for the key.  After that will be ranges for any index returned
	// from GetIndexFields.  These will be returned in the same order as was
	// present in the return value for GetIndexFields.  Currently, only string indexes are
	// supported. Index ranges include the index field name (in order to differentiate among
	// multiple secondary indexes).  Again, the first will always be the "k" field.
	// If NilAllowed is true, nil values for the index field should be included in the
	// return k/v pairs from Scan.  If false, they should not be included.
	// It's best to honor all index ranges.  The datasource should honor the
	// the ranges by not passing in k/v pairs that the ranges exclude.  Future
	// optimzation may cause incorrect answers if this contract is not kept.
	Scan(indexRanges ...IndexRanges) (KeyValueStream, error)

	// Delete deletes the k/v pair for key k.
	// This will only be called if GetTable was called with writeAccessReq == true.
	// If Delete is not supported, GetTable should have returned an error.  If
	// Delete is called anyway (logic error), the syncql.OperationNotSupported error
	// should be returned.
	Delete(k string) (bool, error)
}

type KeyValueStream interface {
	// Advance stages an element so the client can retrieve it
	// with KeyValue.  Advance returns true iff there is an
	// element to retrieve.  The client must call Advance before
	// calling KeyValue.  The client must call Cancel if it does
	// not iterate through all elements (i.e. until Advance
	// returns false).  Advance may block if an element is not
	// immediately available.
	Advance() bool

	// KeyValue returns the element that was staged by Advance.
	// KeyValue may panic if Advance returned false or was not
	// called at all.  KeyValue does not block.
	KeyValue() (string, *vom.RawBytes)

	// Err returns a non-nil error iff the stream encountered
	// any errors.  Err does not block.
	Err() error

	// Cancel notifies the stream provider that it can stop
	// producing elements.  The client must call Cancel if it does
	// not iterate through all elements (i.e. until Advance
	// returns false).  Cancel is idempotent and can be called
	// concurrently with a goroutine that is iterating via
	// Advance/Value.  Cancel causes Advance to subsequently
	// return  false.  Cancel does not block.
	Cancel()
}

// Implement sort interface for StringFieldRanges.
func (stringFieldRanges StringFieldRanges) Len() int {
	return len(stringFieldRanges)
}

func (stringFieldRanges StringFieldRanges) Less(i, j int) bool {
	return stringFieldRanges[i].Start < stringFieldRanges[j].Start
}

func (stringFieldRanges StringFieldRanges) Swap(i, j int) {
	saveStart := stringFieldRanges[i].Start
	saveLimit := stringFieldRanges[i].Limit
	stringFieldRanges[i].Start = stringFieldRanges[j].Start
	stringFieldRanges[i].Limit = stringFieldRanges[j].Limit
	stringFieldRanges[j].Start = saveStart
	stringFieldRanges[j].Limit = saveLimit
}

type StringFieldRange struct {
	Start string
	Limit string
}

type Index struct {
	FieldName string
	Kind      vdl.Kind
}

type StringFieldRanges []StringFieldRange
type IndexRanges struct {
	FieldName    string
	Kind         vdl.Kind
	NilAllowed   bool // true if query could be true for a nil index value
	StringRanges *StringFieldRanges
	// TODO(jkline): add fields for other types of indexes.
}

// String() used in tests.
func (ir *IndexRanges) String() string {
	str := fmt.Sprintf("IndexRanges{FieldName: %s, Kind: %v, NilAllowed: %v, ", ir.FieldName, ir.Kind, ir.NilAllowed)
	for _, r := range *ir.StringRanges {
		str += fmt.Sprintf("{%s,%s}", r.Start, r.Limit)
	}
	str += "}"
	return str
}
