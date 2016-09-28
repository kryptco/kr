// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query_checker

import (
	"sort"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/query_functions"
	"v.io/v23/query/engine/internal/query_parser"
	"v.io/v23/query/pattern"
	"v.io/v23/query/syncql"
	"v.io/v23/vdl"
)

const (
	MaxRangeLimit = ""
)

var (
	StringFieldRangeAll = ds.StringFieldRange{Start: "", Limit: MaxRangeLimit}
)

func Check(db ds.Database, s *query_parser.Statement) error {
	switch sel := (*s).(type) {
	case query_parser.SelectStatement:
		return checkSelectStatement(db, &sel)
	case query_parser.DeleteStatement:
		return checkDeleteStatement(db, &sel)
	default:
		return syncql.NewErrCheckOfUnknownStatementType(db.GetContext(), (*s).Offset())
	}
}

func checkSelectStatement(db ds.Database, s *query_parser.SelectStatement) error {
	if err := checkSelectClause(db, s.Select); err != nil {
		return err
	}
	if err := checkFromClause(db, s.From, false); err != nil {
		return err
	}
	if err := checkEscapeClause(db, s.Escape); err != nil {
		return err
	}
	if err := checkWhereClause(db, s.Where, s.Escape); err != nil {
		return err
	}
	if err := checkLimitClause(db, s.Limit); err != nil {
		return err
	}
	if err := checkResultsOffsetClause(db, s.ResultsOffset); err != nil {
		return err
	}
	return nil
}

func checkDeleteStatement(db ds.Database, s *query_parser.DeleteStatement) error {
	if err := checkFromClause(db, s.From, true); err != nil {
		return err
	}
	if err := checkEscapeClause(db, s.Escape); err != nil {
		return err
	}
	if err := checkWhereClause(db, s.Where, s.Escape); err != nil {
		return err
	}
	if err := checkLimitClause(db, s.Limit); err != nil {
		return err
	}
	return nil
}

// Check select clause.  Fields can be 'k' and v[{.<ident>}...]
func checkSelectClause(db ds.Database, s *query_parser.SelectClause) error {
	for _, selector := range s.Selectors {
		switch selector.Type {
		case query_parser.TypSelField:
			switch selector.Field.Segments[0].Value {
			case "k":
				if len(selector.Field.Segments) > 1 {
					return syncql.NewErrDotNotationDisallowedForKey(db.GetContext(), selector.Field.Segments[1].Off)
				}
			case "v":
				// Nothing to check.
			case "K":
				// Be nice and warn of mistakenly capped 'K'.
				return syncql.NewErrDidYouMeanLowercaseK(db.GetContext(), selector.Field.Segments[0].Off)
			case "V":
				// Be nice and warn of mistakenly capped 'V'.
				return syncql.NewErrDidYouMeanLowercaseV(db.GetContext(), selector.Field.Segments[0].Off)
			default:
				return syncql.NewErrInvalidSelectField(db.GetContext(), selector.Field.Segments[0].Off)
			}
		case query_parser.TypSelFunc:
			err := query_functions.CheckFunction(db, selector.Function)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Check from clause.  Table must exist in the database.
func checkFromClause(db ds.Database, f *query_parser.FromClause, writeAccessReq bool) error {
	var err error
	f.Table.DBTable, err = db.GetTable(f.Table.Name, writeAccessReq)
	if err != nil {
		return syncql.NewErrTableCantAccess(db.GetContext(), f.Table.Off, f.Table.Name, err)
	}
	return nil
}

// Check where clause.
func checkWhereClause(db ds.Database, w *query_parser.WhereClause, ec *query_parser.EscapeClause) error {
	if w == nil {
		return nil
	}
	return checkExpression(db, w.Expr, ec)
}

func checkExpression(db ds.Database, e *query_parser.Expression, ec *query_parser.EscapeClause) error {
	if err := checkOperand(db, e.Operand1, ec); err != nil {
		return err
	}
	if err := checkOperand(db, e.Operand2, ec); err != nil {
		return err
	}

	// Like expressions require operand2 to be a string literal that must be validated.
	if e.Operator.Type == query_parser.Like || e.Operator.Type == query_parser.NotLike {
		if e.Operand2.Type != query_parser.TypStr {
			return syncql.NewErrLikeExpressionsRequireRhsString(db.GetContext(), e.Operand2.Off)
		}
		// Compile the like pattern now to to check for errors.
		p, err := parseLikePattern(db, e.Operand2.Off, e.Operand2.Str, ec)
		if err != nil {
			return err
		}
		fixedPrefix, noWildcards := p.FixedPrefix()
		e.Operand2.Prefix = fixedPrefix
		// Optimization: If like/not like argument contains no wildcards, convert the expression to equals/not equals.
		if noWildcards {
			if e.Operator.Type == query_parser.Like {
				e.Operator.Type = query_parser.Equal
			} else { // not like
				e.Operator.Type = query_parser.NotEqual
			}
			// Since this is no longer a like expression, we need to unescape
			// any escaped chars.
			e.Operand2.Str = fixedPrefix
		}
		// Save the compiled pattern for later use in evaluation.
		e.Operand2.Pattern = p
	}

	// Is/IsNot expressions require operand1 to be a (value or function) and operand2 to be nil.
	if e.Operator.Type == query_parser.Is || e.Operator.Type == query_parser.IsNot {
		if !IsField(e.Operand1) && !IsFunction(e.Operand1) {
			return syncql.NewErrIsIsNotRequireLhsValue(db.GetContext(), e.Operand1.Off)
		}
		if e.Operand2.Type != query_parser.TypNil {
			return syncql.NewErrIsIsNotRequireRhsNil(db.GetContext(), e.Operand2.Off)
		}
	}

	// if an operand is k and the other operand is a literal, that literal must be a string
	// literal.
	if ContainsKeyOperand(e) && ((isLiteral(e.Operand1) && !isStringLiteral(e.Operand1)) ||
		(isLiteral(e.Operand2) && !isStringLiteral(e.Operand2))) {
		off := e.Operand1.Off
		if isLiteral(e.Operand2) {
			off = e.Operand2.Off
		}
		return syncql.NewErrKeyExpressionLiteral(db.GetContext(), off)
	}

	// If either operand is a bool, only = and <> operators are allowed.
	if (e.Operand1.Type == query_parser.TypBool || e.Operand2.Type == query_parser.TypBool) && e.Operator.Type != query_parser.Equal && e.Operator.Type != query_parser.NotEqual {
		return syncql.NewErrBoolInvalidExpression(db.GetContext(), e.Operator.Off)
	}

	return nil
}

func checkOperand(db ds.Database, o *query_parser.Operand, ec *query_parser.EscapeClause) error {
	switch o.Type {
	case query_parser.TypExpr:
		return checkExpression(db, o.Expr, ec)
	case query_parser.TypField:
		switch o.Column.Segments[0].Value {
		case "k":
			if len(o.Column.Segments) > 1 {
				return syncql.NewErrDotNotationDisallowedForKey(db.GetContext(), o.Column.Segments[1].Off)
			}
		case "v":
			// Nothing to do.
		case "K":
			// Be nice and warn of mistakenly capped 'K'.
			return syncql.NewErrDidYouMeanLowercaseK(db.GetContext(), o.Column.Segments[0].Off)
		case "V":
			// Be nice and warn of mistakenly capped 'V'.
			return syncql.NewErrDidYouMeanLowercaseV(db.GetContext(), o.Column.Segments[0].Off)
		default:
			return syncql.NewErrBadFieldInWhere(db.GetContext(), o.Column.Segments[0].Off)
		}
		return nil
	case query_parser.TypFunction:
		// Each of the functions args needs to be checked first.
		for _, arg := range o.Function.Args {
			if err := checkOperand(db, arg, ec); err != nil {
				return err
			}
		}
		// Call query_functions.CheckFunction.  This will check for correct number of args
		// and, to the extent possible, correct types.
		// Furthermore, it may execute the function if the function takes no args or
		// takes only literal args (or an arg that is a function that is also executed
		// early).  CheckFunction will fill in arg types, return types and may fill in
		// Computed and RetValue.
		err := query_functions.CheckFunction(db, o.Function)
		if err != nil {
			return err
		}
		// If function executed early, computed will be true and RetValue set.
		// Convert the operand to the RetValue
		if o.Function.Computed {
			*o = *o.Function.RetValue
		}
	}
	return nil
}

func parseLikePattern(db ds.Database, off int64, s string, ec *query_parser.EscapeClause) (*pattern.Pattern, error) {
	escChar := '\x00' // nul is ignored as an escape char
	if ec != nil {
		escChar = ec.EscapeChar.Value
	}
	p, err := pattern.ParseWithEscapeChar(s, escChar)
	if err != nil {
		return nil, syncql.NewErrInvalidLikePattern(db.GetContext(), off, err)
	}
	return p, nil
}

func IsLogicalOperator(o *query_parser.BinaryOperator) bool {
	return o.Type == query_parser.And || o.Type == query_parser.Or
}

func IsField(o *query_parser.Operand) bool {
	return o.Type == query_parser.TypField
}

func IsFunction(o *query_parser.Operand) bool {
	return o.Type == query_parser.TypFunction
}

func ContainsKeyOperand(expr *query_parser.Expression) bool {
	return IsKey(expr.Operand1) || IsKey(expr.Operand2)
}

func ContainsFieldOperand(f *query_parser.Field, expr *query_parser.Expression) bool {
	return IsExactField(f, expr.Operand1) || IsExactField(f, expr.Operand2)
}

func ContainsFunctionOperand(expr *query_parser.Expression) bool {
	return IsFunction(expr.Operand1) || IsFunction(expr.Operand2)
}

func ContainsValueFieldOperand(expr *query_parser.Expression) bool {
	return (expr.Operand1.Type == query_parser.TypField && IsValueField(expr.Operand1.Column)) ||
		(expr.Operand2.Type == query_parser.TypField && IsValueField(expr.Operand2.Column))

}

func isStringLiteral(o *query_parser.Operand) bool {
	return o.Type == query_parser.TypStr
}

func isLiteral(o *query_parser.Operand) bool {
	return o.Type == query_parser.TypBigInt ||
		o.Type == query_parser.TypBigRat || // currently, no way to specify as literal
		o.Type == query_parser.TypBool ||
		o.Type == query_parser.TypFloat ||
		o.Type == query_parser.TypInt ||
		o.Type == query_parser.TypStr ||
		o.Type == query_parser.TypTime || // currently, no way to specify as literal
		o.Type == query_parser.TypUint
}

func IsKey(o *query_parser.Operand) bool {
	return IsField(o) && IsKeyField(o.Column)
}

func IsExactField(f *query_parser.Field, o *query_parser.Operand) bool {
	if !IsField(o) {
		return false
	}
	oField := o.Column
	// Can't test for equality as offsets will be different.
	if len(f.Segments) != len(oField.Segments) {
		return false
	}
	for i := range f.Segments {
		if f.Segments[i].Value != oField.Segments[i].Value {
			return false
		}
	}
	return true
}

func IsKeyField(f *query_parser.Field) bool {
	return f.Segments[0].Value == "k"
}

func IsValueField(f *query_parser.Field) bool {
	return f.Segments[0].Value == "v"
}

func IsExpr(o *query_parser.Operand) bool {
	return o.Type == query_parser.TypExpr
}

func afterPrefix(prefix string) string {
	// Copied from syncbase.
	limit := []byte(prefix)
	for len(limit) > 0 {
		if limit[len(limit)-1] == 255 {
			limit = limit[:len(limit)-1] // chop off trailing \x00
		} else {
			limit[len(limit)-1] += 1 // add 1
			break                    // no carry
		}
	}
	return string(limit)
}

func computeStringFieldRangeForLike(prefix string) ds.StringFieldRange {
	if prefix == "" {
		return StringFieldRangeAll
	}
	return ds.StringFieldRange{Start: prefix, Limit: afterPrefix(prefix)}
}

func computeStringFieldRangesForNotLike(prefix string) *ds.StringFieldRanges {
	if prefix == "" {
		return &ds.StringFieldRanges{StringFieldRangeAll}
	}
	return &ds.StringFieldRanges{
		ds.StringFieldRange{Start: "", Limit: prefix},
		ds.StringFieldRange{Start: afterPrefix(prefix), Limit: ""},
	}
}

// The limit for a single value range is simply a zero byte appended.
func computeStringFieldRangeForSingleValue(start string) ds.StringFieldRange {
	limit := []byte(start)
	limit = append(limit, 0)
	return ds.StringFieldRange{Start: start, Limit: string(limit)}
}

// Compute a list of secondary index ranges to optionally be used by query's Table.Scan.
func CompileIndexRanges(idxField *query_parser.Field, kind vdl.Kind, where *query_parser.WhereClause) *ds.IndexRanges {
	var indexRanges ds.IndexRanges
	// Reconstruct field name from the segments in the field.
	sep := ""
	for _, seg := range idxField.Segments {
		indexRanges.FieldName += sep
		indexRanges.FieldName += seg.Value
		sep = "."
	}
	indexRanges.Kind = kind
	if where == nil {
		// Currently, only string is supported, so no need to check.
		indexRanges.StringRanges = &ds.StringFieldRanges{StringFieldRangeAll}
		indexRanges.NilAllowed = true
	} else {
		indexRanges.StringRanges = collectStringFieldRanges(idxField, where.Expr)
		indexRanges.NilAllowed = determineIfNilAllowed(idxField, where.Expr)
	}
	return &indexRanges
}

func computeRangeIntersection(lhs, rhs ds.StringFieldRange) *ds.StringFieldRange {
	// Detect if lhs.Start is contained within rhs or rhs.Start is contained within lhs.
	if (lhs.Start >= rhs.Start && compareStartToLimit(lhs.Start, rhs.Limit) < 0) ||
		(rhs.Start >= lhs.Start && compareStartToLimit(rhs.Start, lhs.Limit) < 0) {
		var start, limit string
		if lhs.Start < rhs.Start {
			start = rhs.Start
		} else {
			start = lhs.Start
		}
		if compareLimits(lhs.Limit, rhs.Limit) < 0 {
			limit = lhs.Limit
		} else {
			limit = rhs.Limit
		}
		return &ds.StringFieldRange{Start: start, Limit: limit}
	}
	return nil
}

func fieldRangeIntersection(lhs, rhs *ds.StringFieldRanges) *ds.StringFieldRanges {
	fieldRanges := &ds.StringFieldRanges{}
	lCur, rCur := 0, 0
	for lCur < len(*lhs) && rCur < len(*rhs) {
		// Any intersection at current cursors?
		if intersection := computeRangeIntersection((*lhs)[lCur], (*rhs)[rCur]); intersection != nil {
			// Add the intersection
			addStringFieldRange(*intersection, fieldRanges)
		}
		// increment the range with the lesser limit
		c := compareLimits((*lhs)[lCur].Limit, (*rhs)[rCur].Limit)
		switch c {
		case -1:
			lCur++
		case 1:
			rCur++
		default:
			lCur++
			rCur++
		}
	}
	return fieldRanges
}

func collectStringFieldRanges(idxField *query_parser.Field, expr *query_parser.Expression) *ds.StringFieldRanges {
	if IsExpr(expr.Operand1) { // then both operands must be expressions
		lhsStringFieldRanges := collectStringFieldRanges(idxField, expr.Operand1.Expr)
		rhsStringFieldRanges := collectStringFieldRanges(idxField, expr.Operand2.Expr)
		if expr.Operator.Type == query_parser.And {
			// intersection of lhsStringFieldRanges and rhsStringFieldRanges
			return fieldRangeIntersection(lhsStringFieldRanges, rhsStringFieldRanges)
		} else { // or
			// union of lhsStringFieldRanges and rhsStringFieldRanges
			for _, rhsStringFieldRange := range *rhsStringFieldRanges {
				addStringFieldRange(rhsStringFieldRange, lhsStringFieldRanges)
			}
			return lhsStringFieldRanges
		}
	} else if ContainsFieldOperand(idxField, expr) { // true if either operand is idxField
		if IsField(expr.Operand1) && IsField(expr.Operand2) {
			//<idx_field> <op> <idx_field>
			switch expr.Operator.Type {
			case query_parser.Equal, query_parser.GreaterThanOrEqual, query_parser.LessThanOrEqual:
				// True for all values of indexField
				return &ds.StringFieldRanges{StringFieldRangeAll}
			default: // query_parser.NotEqual, query_parser.GreaterThan, query_parser.LessThan:
				// False for all values of indexField
				return &ds.StringFieldRanges{}
			}
		} else if expr.Operator.Type == query_parser.Is {
			// <idx_field> is nil
			// False for entire range
			// TODO(jkline): Should the Scan contract return values where
			//               the index field is undefined?
			return &ds.StringFieldRanges{}
		} else if expr.Operator.Type == query_parser.IsNot {
			// k is not nil
			// True for all all values of indexField.
			return &ds.StringFieldRanges{StringFieldRangeAll}
		} else if isStringLiteral(expr.Operand2) {
			// indexField <op> <string-literal>
			switch expr.Operator.Type {
			case query_parser.Equal:
				return &ds.StringFieldRanges{computeStringFieldRangeForSingleValue(expr.Operand2.Str)}
			case query_parser.GreaterThan:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: string(append([]byte(expr.Operand2.Str), 0)), Limit: MaxRangeLimit}}
			case query_parser.GreaterThanOrEqual:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: expr.Operand2.Str, Limit: MaxRangeLimit}}
			case query_parser.Like:
				return &ds.StringFieldRanges{computeStringFieldRangeForLike(expr.Operand2.Prefix)}
			case query_parser.NotLike:
				return computeStringFieldRangesForNotLike(expr.Operand2.Prefix)
			case query_parser.LessThan:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: "", Limit: expr.Operand2.Str}}
			case query_parser.LessThanOrEqual:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: "", Limit: string(append([]byte(expr.Operand2.Str), 0))}}
			default: // case query_parser.NotEqual:
				return &ds.StringFieldRanges{
					ds.StringFieldRange{Start: "", Limit: expr.Operand2.Str},
					ds.StringFieldRange{Start: string(append([]byte(expr.Operand2.Str), 0)), Limit: MaxRangeLimit},
				}
			}
		} else if isStringLiteral(expr.Operand1) {
			//<string-literal> <op> k
			switch expr.Operator.Type {
			case query_parser.Equal:
				return &ds.StringFieldRanges{computeStringFieldRangeForSingleValue(expr.Operand1.Str)}
			case query_parser.GreaterThan:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: "", Limit: expr.Operand1.Str}}
			case query_parser.GreaterThanOrEqual:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: "", Limit: string(append([]byte(expr.Operand1.Str), 0))}}
			case query_parser.LessThan:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: string(append([]byte(expr.Operand1.Str), 0)), Limit: MaxRangeLimit}}
			case query_parser.LessThanOrEqual:
				return &ds.StringFieldRanges{ds.StringFieldRange{Start: expr.Operand1.Str, Limit: MaxRangeLimit}}
			default: // case query_parser.NotEqual:
				return &ds.StringFieldRanges{
					ds.StringFieldRange{Start: "", Limit: expr.Operand1.Str},
					ds.StringFieldRange{Start: string(append([]byte(expr.Operand1.Str), 0)), Limit: MaxRangeLimit},
				}
			}
		} else {
			// A function or a field s being compared to the indexField;
			// or, an indexField is being compared to a literal which
			// is not a string.  The latter could be considered an error,
			// but for now, just allow the full range.
			return &ds.StringFieldRanges{StringFieldRangeAll}
		}
	} else { // not a key compare, so it applies to the entire key range
		return &ds.StringFieldRanges{StringFieldRangeAll}
	}
}

func determineIfNilAllowed(idxField *query_parser.Field, expr *query_parser.Expression) bool {
	if IsExpr(expr.Operand1) { // then both operands must be expressions
		lhsNilAllowed := determineIfNilAllowed(idxField, expr.Operand1.Expr)
		rhsNilAllowed := determineIfNilAllowed(idxField, expr.Operand2.Expr)
		if expr.Operator.Type == query_parser.And {
			return lhsNilAllowed && rhsNilAllowed
		} else { // or
			return lhsNilAllowed || rhsNilAllowed
		}
	} else if ContainsFieldOperand(idxField, expr) { // true if either operand is idxField
		// The only way nil in the index field will evaluate to true is in the
		// Is Nil case.
		if expr.Operator.Type == query_parser.Is {
			// <idx_field> is nil
			return true
		} else {
			return false
		}
	} else { // not an index field expresion; as such, nil is allowed for the idx field
		return true
	}
}

// Helper function to compare start and limit byte arrays  taking into account that
// MaxRangeLimit is actually []byte{}.
func compareLimits(limitA, limitB string) int {
	if limitA == limitB {
		return 0
	} else if limitA == MaxRangeLimit {
		return 1
	} else if limitB == MaxRangeLimit {
		return -1
	} else if limitA < limitB {
		return -1
	} else {
		return 1
	}
}

func compareStartToLimit(startA, limitB string) int {
	if limitB == MaxRangeLimit {
		return -1
	} else if startA == limitB {
		return 0
	} else if startA < limitB {
		return -1
	} else {
		return 1
	}
}

func compareLimitToStart(limitA, startB string) int {
	if limitA == MaxRangeLimit {
		return -1
	} else if limitA == startB {
		return 0
	} else if limitA < startB {
		return -1
	} else {
		return 1
	}
}

func addStringFieldRange(fieldRange ds.StringFieldRange, fieldRanges *ds.StringFieldRanges) {
	handled := false
	// Is there an overlap with an existing range?
	for i, r := range *fieldRanges {
		// In the following if,
		// the first paren expr is true if the start of the range to be added is contained in r
		// the second paren expr is true if the limit of the range to be added is contained in r
		// the third paren expr is true if the range to be added entirely contains r
		if (fieldRange.Start >= r.Start && compareStartToLimit(fieldRange.Start, r.Limit) <= 0) ||
			(compareLimitToStart(fieldRange.Limit, r.Start) >= 0 && compareLimits(fieldRange.Limit, r.Limit) <= 0) ||
			(fieldRange.Start <= r.Start && compareLimits(fieldRange.Limit, r.Limit) >= 0) {

			// fieldRange overlaps with existing range at fieldRanges[i]
			// set newFieldRange to a range that ecompasses both
			var newFieldRange ds.StringFieldRange
			if fieldRange.Start < r.Start {
				newFieldRange.Start = fieldRange.Start
			} else {
				newFieldRange.Start = r.Start
			}
			if compareLimits(fieldRange.Limit, r.Limit) > 0 {
				newFieldRange.Limit = fieldRange.Limit
			} else {
				newFieldRange.Limit = r.Limit
			}
			// The new range may overlap with other ranges in fieldRanges
			// delete the current range and call addStringFieldRange again
			// This recursion will continue until no ranges overlap.
			*fieldRanges = append((*fieldRanges)[:i], (*fieldRanges)[i+1:]...)
			addStringFieldRange(newFieldRange, fieldRanges)
			handled = true // we don't want to add fieldRange below
			break
		}
	}
	// no overlap, just add it
	if !handled {
		*fieldRanges = append(*fieldRanges, fieldRange)
	}
	// sort before returning
	sort.Sort(*fieldRanges)
}

// Check escape clause. Escape char cannot be '\', ' ', or a wildcard.
// Return bool (true if escape char defined), escape char, error.
func checkEscapeClause(db ds.Database, e *query_parser.EscapeClause) error {
	if e == nil {
		return nil
	}
	switch ec := e.EscapeChar.Value; ec {
	case '\x00', '_', '%', ' ', '\\':
		return syncql.NewErrInvalidEscapeChar(db.GetContext(), e.EscapeChar.Off, string(ec))
	default:
		return nil
	}
}

// Check limit clause.  Limit must be >= 1.
// Note: The parser will not allow negative numbers here.
func checkLimitClause(db ds.Database, l *query_parser.LimitClause) error {
	if l == nil {
		return nil
	}
	if l.Limit.Value < 1 {
		return syncql.NewErrLimitMustBeGt0(db.GetContext(), l.Limit.Off)
	}
	return nil
}

// Check results offset clause.  Offset must be >= 0.
// Note: The parser will not allow negative numbers here, so this check is presently superfluous.
func checkResultsOffsetClause(db ds.Database, o *query_parser.ResultsOffsetClause) error {
	if o == nil {
		return nil
	}
	if o.ResultsOffset.Value < 0 {
		return syncql.NewErrOffsetMustBeGe0(db.GetContext(), o.ResultsOffset.Off)
	}
	return nil
}
