// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"errors"

	"v.io/v23/context"
	"v.io/v23/discovery"
	"v.io/v23/query/engine"
	"v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/public"
	"v.io/v23/vdl"
	"v.io/v23/vom"
)

// Matcher is the interface for matching advertisements against a query.
type Matcher interface {
	// Match returns true if the matcher matches the advertisement.
	Match(ad *discovery.Advertisement) (bool, error)

	// TargetKey returns the key if a single key is being queried; otherwise
	// an empty string is returned.
	TargetKey() string

	// TargetInterfaceName returns the interface name if a single interface name
	// is being queried; otherwise an empty string is returned.
	TargetInterfaceName() string
}

// trueMatcher matches any advertisement.
type trueMatcher struct{}

func (m trueMatcher) Match(*discovery.Advertisement) (bool, error) { return true, nil }
func (m trueMatcher) TargetKey() string                            { return "" }
func (m trueMatcher) TargetInterfaceName() string                  { return "" }

// dDS implements a datasource for syncQL.
type dDS struct {
	ctx  *context.T
	k    string
	v    *vom.RawBytes
	done bool
}

func (ds *dDS) GetContext() *context.T                          { return ds.ctx }
func (ds *dDS) GetTable(string, bool) (datasource.Table, error) { return ds, nil }

func (ds *dDS) GetIndexFields() []datasource.Index                                { return nil }
func (ds *dDS) Scan(...datasource.IndexRanges) (datasource.KeyValueStream, error) { return ds, nil }
func (ds *dDS) Delete(string) (bool, error)                                       { return false, nil }

func (ds *dDS) Advance() bool {
	if ds.done {
		return false
	}
	ds.done = true
	return true
}
func (ds *dDS) KeyValue() (string, *vom.RawBytes) { return ds.k, ds.v }
func (ds *dDS) Err() error                        { return nil }
func (ds *dDS) Cancel()                           { ds.done = true }

func (ds *dDS) addKeyValue(k string, v *vom.RawBytes) {
	ds.k, ds.v = k, v
	ds.done = false
}

// dummyDS implements a datasource for extracting the target columns from the query.
type dummyDS struct {
	ctx                  *context.T
	targetKey            string
	targetInterfaceName  string
	hasTargetAttachments bool
}

func (ds *dummyDS) GetContext() *context.T                          { return ds.ctx }
func (ds *dummyDS) GetTable(string, bool) (datasource.Table, error) { return ds, nil }

func (ds *dummyDS) GetIndexFields() []datasource.Index {
	// Mimic having indices on InterfaceName and Attachments so that we can
	// get the target ranges on these columns from the query.
	return []datasource.Index{
		datasource.Index{FieldName: "v.InterfaceName", Kind: vdl.String},
		// Pretend to be a string type index since no other type is supported.
		// It's OK to see if attachments are being queried.
		datasource.Index{FieldName: "v.Attachments", Kind: vdl.String},
	}
}

func (ds *dummyDS) Scan(indices ...datasource.IndexRanges) (datasource.KeyValueStream, error) {
	ds.targetKey = getTargetValue(indices[0])           // 0 is for key.
	ds.targetInterfaceName = getTargetValue(indices[1]) // 1 is for v.InterfaceName.
	ds.hasTargetAttachments = !indices[2].NilAllowed    // 2 is for v.Attachments
	return nil, nil
}

func getTargetValue(index datasource.IndexRanges) string {
	if !index.NilAllowed && len(*index.StringRanges) == 1 {
		// If limit is equal to start plus a zero byte, a single interface name is being queried.
		strRange := (*index.StringRanges)[0]
		if len(strRange.Start) > 0 && strRange.Limit == strRange.Start+"\000" {
			return strRange.Start
		}
	}
	return ""
}

func (ds *dummyDS) Delete(string) (bool, error) { return false, nil }

// queryMatcher matches advertisements against the given query.
type queryMatcher struct {
	ds                  *dDS
	pstmt               public.PreparedStatement
	targetKey           string
	targetInterfaceName string
}

func (m *queryMatcher) Match(ad *discovery.Advertisement) (bool, error) {
	v, err := vom.RawBytesFromValue(ad)
	if err != nil {
		return false, err
	}

	m.ds.addKeyValue(ad.Id.String(), v)
	_, r, err := m.pstmt.Exec()
	if err != nil {
		return false, err
	}

	// Note that the datasource has only one row and so we can know whether it is
	// matched or not just with Advance() call.
	if r.Advance() {
		r.Cancel()
		return true, nil
	}
	return false, r.Err()
}

func (m *queryMatcher) TargetKey() string           { return m.targetKey }
func (m *queryMatcher) TargetInterfaceName() string { return m.targetInterfaceName }

func NewMatcher(ctx *context.T, query string) (Matcher, error) {
	if len(query) == 0 {
		return trueMatcher{}, nil
	}

	query = "SELECT v FROM d WHERE " + query

	// Extract the target columns and check any semantic error in the query.
	dummy := &dummyDS{ctx: ctx}
	_, _, err := engine.Create(dummy).Exec(query)
	if err != nil {
		return nil, err
	}
	if dummy.hasTargetAttachments {
		return nil, NewErrBadQuery(ctx, errors.New("v.Attachments cannot be queried"))
	}

	// Prepare the query engine.
	ds := &dDS{ctx: ctx}
	pstmt, err := engine.Create(ds).PrepareStatement(query)
	if err != nil {
		// Should not happen; just for safety.
		return nil, err
	}
	matcher := &queryMatcher{
		ds:                  ds,
		pstmt:               pstmt,
		targetKey:           dummy.targetKey,
		targetInterfaceName: dummy.targetInterfaceName,
	}
	return matcher, nil
}
