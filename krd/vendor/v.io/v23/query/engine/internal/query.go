// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal

import (
	"reflect"
	"strconv"
	"sync"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/query_checker"
	"v.io/v23/query/engine/internal/query_functions"
	"v.io/v23/query/engine/internal/query_parser"
	"v.io/v23/query/engine/public"
	"v.io/v23/query/syncql"
	"v.io/v23/vdl"
	"v.io/v23/vom"
)

type queryEngineImpl struct {
	db                      ds.Database
	mutexNextID             sync.Mutex
	nextID                  int64
	mutexPreparedStatements sync.Mutex
	preparedStatements      map[int64]*query_parser.Statement
}

type preparedStatementImpl struct {
	qe *queryEngineImpl
	id int64 // key to AST stored in queryEngineImpl.
}

type paramInfo struct {
	paramValues []*vdl.Value
	cursor      int64
}

func Create(db ds.Database) public.QueryEngine {
	return &queryEngineImpl{db: db, nextID: 0, preparedStatements: map[int64]*query_parser.Statement{}}
}

func (qe *queryEngineImpl) Exec(q string) ([]string, syncql.ResultStream, error) {
	return Exec(qe.db, q)
}

func (qe *queryEngineImpl) GetPreparedStatement(handle int64) (public.PreparedStatement, error) {
	qe.mutexPreparedStatements.Lock()
	_, ok := qe.preparedStatements[handle]
	qe.mutexPreparedStatements.Unlock()
	if ok {
		return &preparedStatementImpl{qe, handle}, nil
	} else {
		return nil, syncql.NewErrPreparedStatementNotFound(qe.db.GetContext())
	}
}

func (qe *queryEngineImpl) PrepareStatement(q string) (public.PreparedStatement, error) {
	s, err := query_parser.Parse(qe.db, q)
	if err != nil {
		return nil, err
	}
	qe.mutexNextID.Lock()
	id := qe.nextID
	qe.nextID++
	qe.mutexNextID.Unlock()
	qe.mutexPreparedStatements.Lock()
	qe.preparedStatements[id] = s
	qe.mutexPreparedStatements.Unlock()
	return &preparedStatementImpl{qe, id}, nil
}

func (p *preparedStatementImpl) Exec(paramValues ...*vom.RawBytes) ([]string, syncql.ResultStream, error) {
	// Find the AST
	p.qe.mutexPreparedStatements.Lock()
	s := p.qe.preparedStatements[p.id]
	p.qe.mutexPreparedStatements.Unlock()

	vvs := make([]*vdl.Value, len(paramValues))
	for i := range paramValues {
		if err := paramValues[i].ToValue(&vvs[i]); err != nil {
			return nil, nil, err
		}
	}

	// Copy the AST and substitute any parameters with actual values.
	// Note: Not all of the AST is copied as most parts are immutable.
	sCopy, err := (*s).CopyAndSubstitute(p.qe.db, vvs)
	if err != nil {
		return nil, nil, err
	}

	// Sematnically check the copied AST and then execute it.
	return checkAndExec(p.qe.db, &sCopy)
}

func (p *preparedStatementImpl) Handle() int64 {
	return p.id
}

func (p *preparedStatementImpl) Close() {
	p.qe.mutexPreparedStatements.Lock()
	delete(p.qe.preparedStatements, p.id)
	p.qe.mutexPreparedStatements.Unlock()
}

func Exec(db ds.Database, q string) ([]string, syncql.ResultStream, error) {
	s, err := query_parser.Parse(db, q)
	if err != nil {
		return nil, nil, err
	}
	return checkAndExec(db, s)
}

func checkAndExec(db ds.Database, s *query_parser.Statement) ([]string, syncql.ResultStream, error) {
	if err := query_checker.Check(db, s); err != nil {
		return nil, nil, err
	}
	switch (*s).(type) {
	case query_parser.SelectStatement, query_parser.DeleteStatement:
		return execStatement(db, s)
	default:
		return nil, nil, syncql.NewErrExecOfUnknownStatementType(db.GetContext(), (*s).Offset(), reflect.TypeOf(*s).Name())
	}
}

// Given a key, a value and a SelectClause, return the projection.
// This function is only called if Eval returned true on the WhereClause expression.
func ComposeProjection(db ds.Database, k string, v *vdl.Value, s *query_parser.SelectClause) []*vom.RawBytes {
	var projection []*vom.RawBytes
	for _, selector := range s.Selectors {
		switch selector.Type {
		case query_parser.TypSelField:
			// If field not found, nil is returned (as per specification).
			f := ResolveField(db, k, v, selector.Field)
			projection = append(projection, vom.RawBytesOf(f))
		case query_parser.TypSelFunc:
			if selector.Function.Computed {
				projection = append(projection, query_functions.ConvertFunctionRetValueToRawBytes(selector.Function.RetValue))
			} else {
				// need to exec function
				// If error executing function, return nil (as per specification).
				retValue, err := resolveArgsAndExecFunction(db, k, v, selector.Function)
				if err != nil {
					retValue = nil
				}
				projection = append(projection, query_functions.ConvertFunctionRetValueToRawBytes(retValue))
			}
		}
	}
	return projection
}

// For testing purposes, given a SelectStatement, k and v;
// return nil if row not selected, else return the projection (type []*vdl.Value).
// Note: limit and offset clauses are ignored for this function as they make no sense
// for a single row.
func ExecSelectSingleRow(db ds.Database, k string, v *vdl.Value, s *query_parser.SelectStatement) []*vom.RawBytes {
	if !Eval(db, k, v, s.Where.Expr) {
		rs := []*vom.RawBytes{}
		return rs
	}
	return ComposeProjection(db, k, v, s.Select)
}

func getColumnHeadings(s *query_parser.SelectStatement) []string {
	columnHeaders := []string{}
	for _, selector := range s.Select.Selectors {
		columnName := ""
		if selector.As != nil {
			columnName = selector.As.AltName.Value
		} else {
			switch selector.Type {
			case query_parser.TypSelField:
				sep := ""
				for _, segment := range selector.Field.Segments {
					columnName = columnName + sep + segment.Value
					for _, key := range segment.Keys {
						columnName += getSegmentKeyAsHeading(key)
					}
					sep = "."
				}
			case query_parser.TypSelFunc:
				columnName = selector.Function.Name
			}
		}
		columnHeaders = append(columnHeaders, columnName)
	}
	return columnHeaders
}

// TODO(jkline): Should we really include key/index of a map/set/array/list in the header?
// The column names can get quite long.  Perhaps just "[]" at the end of the segment
// would be better.  The author of the query can always use the As clause to specify a
// better heading.  Note: for functions, just the function name is included in the header.
// When a decision is made, it's best to be consistent for functions and key/indexes.
func getSegmentKeyAsHeading(segKey *query_parser.Operand) string {
	val := "["
	switch segKey.Type {
	case query_parser.TypBigInt:
		val += segKey.BigInt.String()
	case query_parser.TypBigRat:
		val += segKey.BigRat.String()
	case query_parser.TypField:
		sep := ""
		for _, segment := range segKey.Column.Segments {
			val += sep + segment.Value
			for _, key := range segment.Keys {
				val += getSegmentKeyAsHeading(key)
			}
			sep = "."
		}
	case query_parser.TypBool:
		val += strconv.FormatBool(segKey.Bool)
	case query_parser.TypInt:
		val += strconv.FormatInt(segKey.Int, 10)
	case query_parser.TypFloat:
		val += strconv.FormatFloat(segKey.Float, 'f', -1, 64)
	case query_parser.TypFunction:
		val += segKey.Function.Name
	case query_parser.TypStr:
		val += segKey.Str
	case query_parser.TypTime:
		val += segKey.Time.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
	case query_parser.TypNil:
		val += "<nil>"
	case query_parser.TypObject:
		val += "<object>"
	default:
		val += "<?>"
	}
	val += "]"
	return val
}

func getIndexRanges(db ds.Database, tableName string, tableOff int64, indexFields []ds.Index, w *query_parser.WhereClause) ([]ds.IndexRanges, error) {
	indexes := []ds.IndexRanges{}

	// Get IndexRanges for k
	kField := &query_parser.Field{Segments: []query_parser.Segment{query_parser.Segment{Value: "k"}}}
	idxRanges := *query_checker.CompileIndexRanges(kField, vdl.String, w)
	indexes = append(indexes, idxRanges)

	// Get IndexRanges for secondary indexes.
	for _, idx := range indexFields {
		if idx.Kind != vdl.String {
			return nil, syncql.NewErrIndexKindNotSupported(db.GetContext(), tableOff, idx.Kind.String(), idx.FieldName, tableName)
		}
		var err error
		var idxField *query_parser.Field
		// Construct a Field from the string.  Use the parser as it knows best.
		if idxField, err = query_parser.ParseIndexField(db, idx.FieldName, tableName); err != nil {
			return nil, err
		}
		idxRanges := *query_checker.CompileIndexRanges(idxField, idx.Kind, w)
		indexes = append(indexes, idxRanges)
	}
	return indexes, nil
}

func execStatement(db ds.Database, s *query_parser.Statement) ([]string, syncql.ResultStream, error) {
	switch st := (*s).(type) {

	// Select
	case query_parser.SelectStatement:
		indexes, err := getIndexRanges(db, st.From.Table.Name, st.From.Table.Off, st.From.Table.DBTable.GetIndexFields(), st.Where)
		if err != nil {
			return nil, nil, err
		}

		keyValueStream, err := st.From.Table.DBTable.Scan(indexes...)
		if err != nil {
			return nil, nil, syncql.NewErrScanError(db.GetContext(), st.Off, err)
		}
		var resultStream selectResultStreamImpl
		resultStream.db = db
		resultStream.selectStatement = &st
		resultStream.keyValueStream = keyValueStream
		return getColumnHeadings(&st), &resultStream, nil

	// Delete
	case query_parser.DeleteStatement:
		indexes, err := getIndexRanges(db, st.From.Table.Name, st.From.Table.Off, st.From.Table.DBTable.GetIndexFields(), st.Where)
		if err != nil {
			return nil, nil, err
		}

		keyValueStream, err := st.From.Table.DBTable.Scan(indexes...)
		if err != nil {
			return nil, nil, syncql.NewErrScanError(db.GetContext(), st.Off, err)
		}

		deleteCount := int64(0)
		for keyValueStream.Advance() {
			if st.Limit != nil && deleteCount >= st.Limit.Limit.Value {
				defer keyValueStream.Cancel()
				break
			}
			k, v := keyValueStream.KeyValue()
			// EvalWhereUsingOnlyKey
			// INCLUDE: the row should be included in the results
			// EXCLUDE: the row should NOT be included
			// FETCH_VALUE: the value and/or type of the value are required to make determination.
			rv := EvalWhereUsingOnlyKey(db, st.Where, k)
			var match bool
			switch rv {
			case INCLUDE:
				match = true
			case EXCLUDE:
				match = false
			case FETCH_VALUE:
				match = Eval(db, k, vdl.ValueOf(v), st.Where.Expr)
			}
			if match {
				b, err := st.From.Table.DBTable.Delete(k)
				// May not have delete permission to delete this k/v pair.
				// Continue, but don't increment delete count.
				if err == nil && b {
					deleteCount++
				}
			}
		}
		if err := keyValueStream.Err(); err != nil {
			return nil, nil, syncql.NewErrKeyValueStreamError(db.GetContext(), st.Off, err)
		}

		var resultStream deleteResultStreamImpl
		resultStream.db = db
		resultStream.deleteStatement = &st
		resultStream.deleteCursor = 0
		resultStream.deleteCount = deleteCount
		return []string{"Count"}, &resultStream, nil
	}
	return nil, nil, syncql.NewErrOperationNotSupported(db.GetContext(), "")
}
