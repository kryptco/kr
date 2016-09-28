// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal

import (
	"errors"
	"fmt"
	"reflect"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/conversions"
	"v.io/v23/query/engine/internal/query_checker"
	"v.io/v23/query/engine/internal/query_functions"
	"v.io/v23/query/engine/internal/query_parser"
	"v.io/v23/query/syncql"
	"v.io/v23/vdl"
)

func Eval(db ds.Database, k string, v *vdl.Value, e *query_parser.Expression) bool {
	if query_checker.IsLogicalOperator(e.Operator) {
		return evalLogicalOperators(db, k, v, e)
	} else {
		return evalComparisonOperators(db, k, v, e)
	}
}

func evalLogicalOperators(db ds.Database, k string, v *vdl.Value, e *query_parser.Expression) bool {
	switch e.Operator.Type {
	case query_parser.And:
		return Eval(db, k, v, e.Operand1.Expr) && Eval(db, k, v, e.Operand2.Expr)
	case query_parser.Or:
		return Eval(db, k, v, e.Operand1.Expr) || Eval(db, k, v, e.Operand2.Expr)
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func evalComparisonOperators(db ds.Database, k string, v *vdl.Value, e *query_parser.Expression) bool {
	lhsValue := resolveOperand(db, k, v, e.Operand1)
	// Check for an is nil expression (i.e., v[.<field>...] is nil).
	// These expressions evaluate to true if the field cannot be resolved.
	if e.Operator.Type == query_parser.Is && e.Operand2.Type == query_parser.TypNil {
		return lhsValue == nil
	}
	if e.Operator.Type == query_parser.IsNot && e.Operand2.Type == query_parser.TypNil {
		return lhsValue != nil
	}
	// For anything but "is[not] nil" (which is handled above), an unresolved operator
	// results in the expression evaluating to false.
	if lhsValue == nil {
		return false
	}
	rhsValue := resolveOperand(db, k, v, e.Operand2)
	if rhsValue == nil {
		return false
	}
	// coerce operands so they are comparable
	var err error
	lhsValue, rhsValue, err = coerceValues(lhsValue, rhsValue)
	if err != nil {
		return false // If operands can't be coerced to compare, expr evals to false.
	}
	// Do the compare
	switch lhsValue.Type {
	case query_parser.TypBigInt:
		return compareBigInts(lhsValue, rhsValue, e.Operator)
	case query_parser.TypBigRat:
		return compareBigRats(lhsValue, rhsValue, e.Operator)
	case query_parser.TypBool:
		return compareBools(lhsValue, rhsValue, e.Operator)
	case query_parser.TypFloat:
		return compareFloats(lhsValue, rhsValue, e.Operator)
	case query_parser.TypInt:
		return compareInts(lhsValue, rhsValue, e.Operator)
	case query_parser.TypStr:
		return compareStrings(lhsValue, rhsValue, e.Operator)
	case query_parser.TypUint:
		return compareUints(lhsValue, rhsValue, e.Operator)
	case query_parser.TypTime:
		return compareTimes(lhsValue, rhsValue, e.Operator)
	case query_parser.TypObject:
		return compareObjects(lhsValue, rhsValue, e.Operator)
	}
	return false
}

func coerceValues(lhsValue, rhsValue *query_parser.Operand) (*query_parser.Operand, *query_parser.Operand, error) {
	// TODO(jkline): explore using vdl for coercions ( https://vanadium.github.io/designdocs/vdl-spec.html#conversions ).
	var err error
	// If either operand is a string, convert the other to a string.
	if lhsValue.Type == query_parser.TypStr || rhsValue.Type == query_parser.TypStr {
		if lhsValue, err = conversions.ConvertValueToString(lhsValue); err != nil {
			return nil, nil, err
		}
		if rhsValue, err = conversions.ConvertValueToString(rhsValue); err != nil {
			return nil, nil, err
		}
		return lhsValue, rhsValue, nil
	}
	// If either operand is a big rat, convert both to a big rat.
	// Also, if one operand is a float and the other is a big int,
	// convert both to big rats.
	if lhsValue.Type == query_parser.TypBigRat || rhsValue.Type == query_parser.TypBigRat || (lhsValue.Type == query_parser.TypBigInt && rhsValue.Type == query_parser.TypFloat) || (lhsValue.Type == query_parser.TypFloat && rhsValue.Type == query_parser.TypBigInt) {
		if lhsValue, err = conversions.ConvertValueToBigRat(lhsValue); err != nil {
			return nil, nil, err
		}
		if rhsValue, err = conversions.ConvertValueToBigRat(rhsValue); err != nil {
			return nil, nil, err
		}
		return lhsValue, rhsValue, nil
	}
	// If either operand is a float, convert the other to a float.
	if lhsValue.Type == query_parser.TypFloat || rhsValue.Type == query_parser.TypFloat {
		if lhsValue, err = conversions.ConvertValueToFloat(lhsValue); err != nil {
			return nil, nil, err
		}
		if rhsValue, err = conversions.ConvertValueToFloat(rhsValue); err != nil {
			return nil, nil, err
		}
		return lhsValue, rhsValue, nil
	}
	// If either operand is a big int, convert both to a big int.
	// Also, if one operand is a uint64 and the other is an int64, convert both to big ints.
	if lhsValue.Type == query_parser.TypBigInt || rhsValue.Type == query_parser.TypBigInt || (lhsValue.Type == query_parser.TypUint && rhsValue.Type == query_parser.TypInt) || (lhsValue.Type == query_parser.TypInt && rhsValue.Type == query_parser.TypUint) {
		if lhsValue, err = conversions.ConvertValueToBigInt(lhsValue); err != nil {
			return nil, nil, err
		}
		if rhsValue, err = conversions.ConvertValueToBigInt(rhsValue); err != nil {
			return nil, nil, err
		}
		return lhsValue, rhsValue, nil
	}
	// If either operand is an int64, convert the other to int64.
	if lhsValue.Type == query_parser.TypInt || rhsValue.Type == query_parser.TypInt {
		if lhsValue, err = conversions.ConvertValueToInt(lhsValue); err != nil {
			return nil, nil, err
		}
		if rhsValue, err = conversions.ConvertValueToInt(rhsValue); err != nil {
			return nil, nil, err
		}
		return lhsValue, rhsValue, nil
	}
	// If either operand is an uint64, convert the other to uint64.
	if lhsValue.Type == query_parser.TypUint || rhsValue.Type == query_parser.TypUint {
		if lhsValue, err = conversions.ConvertValueToUint(lhsValue); err != nil {
			return nil, nil, err
		}
		if rhsValue, err = conversions.ConvertValueToUint(rhsValue); err != nil {
			return nil, nil, err
		}
		return lhsValue, rhsValue, nil
	}
	// Must be the same at this point.
	if lhsValue.Type != rhsValue.Type {
		return nil, nil, errors.New(fmt.Sprintf("Logic error: expected like types, got: %v, %v", lhsValue, rhsValue))
	}

	return lhsValue, rhsValue, nil
}

func compareBools(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.Bool == rhsValue.Bool
	case query_parser.NotEqual:
		return lhsValue.Bool != rhsValue.Bool
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareBigInts(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.BigInt.Cmp(rhsValue.BigInt) == 0
	case query_parser.NotEqual:
		return lhsValue.BigInt.Cmp(rhsValue.BigInt) != 0
	case query_parser.LessThan:
		return lhsValue.BigInt.Cmp(rhsValue.BigInt) < 0
	case query_parser.LessThanOrEqual:
		return lhsValue.BigInt.Cmp(rhsValue.BigInt) <= 0
	case query_parser.GreaterThan:
		return lhsValue.BigInt.Cmp(rhsValue.BigInt) > 0
	case query_parser.GreaterThanOrEqual:
		return lhsValue.BigInt.Cmp(rhsValue.BigInt) >= 0
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareBigRats(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.BigRat.Cmp(rhsValue.BigRat) == 0
	case query_parser.NotEqual:
		return lhsValue.BigRat.Cmp(rhsValue.BigRat) != 0
	case query_parser.LessThan:
		return lhsValue.BigRat.Cmp(rhsValue.BigRat) < 0
	case query_parser.LessThanOrEqual:
		return lhsValue.BigRat.Cmp(rhsValue.BigRat) <= 0
	case query_parser.GreaterThan:
		return lhsValue.BigRat.Cmp(rhsValue.BigRat) > 0
	case query_parser.GreaterThanOrEqual:
		return lhsValue.BigRat.Cmp(rhsValue.BigRat) >= 0
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareFloats(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.Float == rhsValue.Float
	case query_parser.NotEqual:
		return lhsValue.Float != rhsValue.Float
	case query_parser.LessThan:
		return lhsValue.Float < rhsValue.Float
	case query_parser.LessThanOrEqual:
		return lhsValue.Float <= rhsValue.Float
	case query_parser.GreaterThan:
		return lhsValue.Float > rhsValue.Float
	case query_parser.GreaterThanOrEqual:
		return lhsValue.Float >= rhsValue.Float
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareInts(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.Int == rhsValue.Int
	case query_parser.NotEqual:
		return lhsValue.Int != rhsValue.Int
	case query_parser.LessThan:
		return lhsValue.Int < rhsValue.Int
	case query_parser.LessThanOrEqual:
		return lhsValue.Int <= rhsValue.Int
	case query_parser.GreaterThan:
		return lhsValue.Int > rhsValue.Int
	case query_parser.GreaterThanOrEqual:
		return lhsValue.Int >= rhsValue.Int
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareUints(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.Uint == rhsValue.Uint
	case query_parser.NotEqual:
		return lhsValue.Uint != rhsValue.Uint
	case query_parser.LessThan:
		return lhsValue.Uint < rhsValue.Uint
	case query_parser.LessThanOrEqual:
		return lhsValue.Uint <= rhsValue.Uint
	case query_parser.GreaterThan:
		return lhsValue.Uint > rhsValue.Uint
	case query_parser.GreaterThanOrEqual:
		return lhsValue.Uint >= rhsValue.Uint
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareStrings(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.Str == rhsValue.Str
	case query_parser.NotEqual:
		return lhsValue.Str != rhsValue.Str
	case query_parser.LessThan:
		return lhsValue.Str < rhsValue.Str
	case query_parser.LessThanOrEqual:
		return lhsValue.Str <= rhsValue.Str
	case query_parser.GreaterThan:
		return lhsValue.Str > rhsValue.Str
	case query_parser.GreaterThanOrEqual:
		return lhsValue.Str >= rhsValue.Str
	case query_parser.Like:
		return rhsValue.Pattern.MatchString(lhsValue.Str)
	case query_parser.NotLike:
		return !rhsValue.Pattern.MatchString(lhsValue.Str)
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareTimes(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return lhsValue.Time.Equal(rhsValue.Time)
	case query_parser.NotEqual:
		return !lhsValue.Time.Equal(rhsValue.Time)
	case query_parser.LessThan:
		return lhsValue.Time.Before(rhsValue.Time)
	case query_parser.LessThanOrEqual:
		return lhsValue.Time.Before(rhsValue.Time) || lhsValue.Time.Equal(rhsValue.Time)
	case query_parser.GreaterThan:
		return lhsValue.Time.After(rhsValue.Time)
	case query_parser.GreaterThanOrEqual:
		return lhsValue.Time.After(rhsValue.Time) || lhsValue.Time.Equal(rhsValue.Time)
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return false
	}
}

func compareObjects(lhsValue, rhsValue *query_parser.Operand, oper *query_parser.BinaryOperator) bool {
	switch oper.Type {
	case query_parser.Equal:
		return reflect.DeepEqual(lhsValue.Object, rhsValue.Object)
	case query_parser.NotEqual:
		return !reflect.DeepEqual(lhsValue.Object, rhsValue.Object)
	default: // other operands are non-sensical
		return false
	}
}

func resolveArgsAndExecFunction(db ds.Database, k string, v *vdl.Value, f *query_parser.Function) (*query_parser.Operand, error) {
	// Resolve the function's arguments
	callingArgs := []*query_parser.Operand{}
	for _, arg := range f.Args {
		resolvedArg := resolveOperand(db, k, v, arg)
		if resolvedArg == nil {
			return nil, syncql.NewErrFunctionArgBad(db.GetContext(), arg.Off, f.Name, arg.String())
		}
		callingArgs = append(callingArgs, resolvedArg)
	}
	// Exec the function
	retValue, err := query_functions.ExecFunction(db, f, callingArgs)
	if err != nil {
		return nil, err
	}
	return retValue, nil
}

func resolveOperand(db ds.Database, k string, v *vdl.Value, o *query_parser.Operand) *query_parser.Operand {
	if o.Type == query_parser.TypFunction {
		// Note: if the function was computed at check time, the operand is replaced
		// in the parse tree with the return value.  As such, thre is no need to check
		// the computed field.
		if retValue, err := resolveArgsAndExecFunction(db, k, v, o.Function); err == nil {
			return retValue
		} else {
			// Per spec, function errors resolve to nil
			return nil
		}
	}
	if o.Type != query_parser.TypField {
		return o
	}
	value := ResolveField(db, k, v, o.Column)
	if value.IsNil() {
		return nil
	}

	if newOp, err := query_parser.ConvertValueToAnOperand(value, o.Off); err != nil {
		return nil
	} else {
		return newOp
	}
}

// Auto-dereference Any and Optional values
func autoDereference(o *vdl.Value) *vdl.Value {
	for o.Kind() == vdl.Any || o.Kind() == vdl.Optional {
		o = o.Elem()
		if o == nil {
			break
		}
	}
	if o == nil {
		o = vdl.ValueOf(nil)
	}
	return o
}

// Resolve object with the key(s) (if a key(s) was specified).  That is, resolve the object with the
// value of what is specified in brackets (for maps, sets. arrays and lists).
// If no key was specified, just return the object.
// If a key was specified, but the object is not map, set, array or list: return nil.
// If the key resolved to nil, return nil.
// If the key can't be converted to the required type, return nil.
// For arrays/lists, if the index is out of bounds, return nil.
// For maps, if key not found, return nil.
// For sets, if key found, return true, else return false.
func resolveWithKey(db ds.Database, k string, v *vdl.Value, object *vdl.Value, segment query_parser.Segment) *vdl.Value {
	for _, key := range segment.Keys {
		o := resolveOperand(db, k, v, key)
		if o == nil {
			return vdl.ValueOf(nil)
		}
		proposedKey := valueFromResolvedOperand(o)
		if proposedKey == nil {
			return vdl.ValueOf(nil)
		}
		switch object.Kind() {
		case vdl.Array, vdl.List:
			// convert key to int
			// vdl's Index function wants an int.
			// vdl can't make an int.
			// int is guaranteed to be at least 32-bits.
			// So have vdl make an int32 and then convert it to an int.
			index32 := vdl.IntValue(vdl.Int32Type, 0)
			if err := vdl.Convert(index32, proposedKey); err != nil {
				return vdl.ValueOf(nil)
			}
			index := int(index32.Int())
			if index < 0 || index >= object.Len() {
				return vdl.ValueOf(nil)
			}
			object = object.Index(index)
		case vdl.Map, vdl.Set:
			reqKeyType := object.Type().Key()
			keyVal := vdl.ZeroValue(reqKeyType)
			if err := vdl.Convert(keyVal, proposedKey); err != nil {
				return vdl.ValueOf(nil)
			}
			if object.Kind() == vdl.Map {
				rv := object.MapIndex(keyVal)
				if rv != nil {
					object = rv
				} else {
					return vdl.ValueOf(nil)
				}
			} else { // vdl.Set
				object = vdl.ValueOf(object.ContainsKey(keyVal))
			}
		default:
			return vdl.ValueOf(nil)
		}
	}
	return object
}

// Return the value of a non-nil *Operand that has been resolved by resolveOperand.
func valueFromResolvedOperand(o *query_parser.Operand) interface{} {
	// This switch contains the possible types returned from resolveOperand.
	switch o.Type {
	case query_parser.TypBigInt:
		return o.BigInt
	case query_parser.TypBigRat:
		return o.BigRat
	case query_parser.TypBool:
		return o.Bool
	case query_parser.TypFloat:
		return o.Float
	case query_parser.TypInt:
		return o.Int
	case query_parser.TypNil:
		return nil
	case query_parser.TypStr:
		return o.Str
	case query_parser.TypTime:
		return o.Time
	case query_parser.TypObject:
		return o.Object
	case query_parser.TypUint:
		return o.Uint
	}
	return nil
}

// Resolve a field.
func ResolveField(db ds.Database, k string, v *vdl.Value, f *query_parser.Field) *vdl.Value {
	if query_checker.IsKeyField(f) {
		return vdl.StringValue(nil, k)
	}
	// Auto-dereference Any and Optional values
	v = autoDereference(v)

	object := v
	segments := f.Segments
	// Does v contain a key?
	object = resolveWithKey(db, k, v, object, segments[0])

	// More segments?
	for i := 1; i < len(segments); i++ {
		// Auto-dereference Any and Optional values
		object = autoDereference(object)
		// object must be a struct in order to look for the next segment.
		if object.Kind() == vdl.Struct {
			if object = object.StructFieldByName(segments[i].Value); object == nil {
				return vdl.ValueOf(nil)
			}
			object = resolveWithKey(db, k, v, object, segments[i])
		} else if object.Kind() == vdl.Union {
			unionType := object.Type()
			idx, tempValue := object.UnionField()
			if segments[i].Value == unionType.Field(idx).Name {
				object = tempValue
			} else {
				return vdl.ValueOf(nil)
			}
			object = resolveWithKey(db, k, v, object, segments[i])
		} else {
			return vdl.ValueOf(nil)
		}
	}
	return object
}

// EvalWhereUsingOnlyKey return type.  See that function for details.
type EvalWithKeyResult int

const (
	INCLUDE EvalWithKeyResult = iota
	EXCLUDE
	FETCH_VALUE
)

// Evaluate the where clause to determine if the row should be selected, but do so using only
// the key.  Possible returns are:
// INCLUDE: the row should included in the results
// EXCLUDE: the row should NOT be included
// FETCH_VALUE: the value and/or type of the value are required to determine if row should be included.
// The above decision is accomplished by evaluating all expressions which compare the key
// with a string literal and substituing false for all other expressions.  If the result is true,
// INCLUDE is returned.
// If the result is false, but no other expressions were encountered, EXCLUDE is returned; else,
// FETCH_VALUE is returned indicating the value must be fetched in order to determine if the row
// should be included in the results.
func EvalWhereUsingOnlyKey(db ds.Database, w *query_parser.WhereClause, k string) EvalWithKeyResult {
	if w == nil { // all rows will be in result
		return INCLUDE
	}
	return evalExprUsingOnlyKey(db, w.Expr, k)
}

func evalExprUsingOnlyKey(db ds.Database, e *query_parser.Expression, k string) EvalWithKeyResult {
	switch e.Operator.Type {
	case query_parser.And:
		op1Result := evalExprUsingOnlyKey(db, e.Operand1.Expr, k)
		op2Result := evalExprUsingOnlyKey(db, e.Operand2.Expr, k)
		if op1Result == INCLUDE && op2Result == INCLUDE {
			return INCLUDE
		} else if op1Result == EXCLUDE || op2Result == EXCLUDE {
			// One of the operands evaluated to EXCLUDE.
			// As such, the value is not needed to reject the row.
			return EXCLUDE
		} else {
			return FETCH_VALUE
		}
	case query_parser.Or:
		op1Result := evalExprUsingOnlyKey(db, e.Operand1.Expr, k)
		op2Result := evalExprUsingOnlyKey(db, e.Operand2.Expr, k)
		if op1Result == INCLUDE || op2Result == INCLUDE {
			return INCLUDE
		} else if op1Result == EXCLUDE && op2Result == EXCLUDE {
			return EXCLUDE
		} else {
			return FETCH_VALUE
		}
	default:
		if !query_checker.ContainsKeyOperand(e) {
			return FETCH_VALUE
		} else {
			// at least one operand is a key
			// May still need to fetch the value (if
			// one of the operands is a value field or a function).
			if query_checker.ContainsFunctionOperand(e) || query_checker.ContainsValueFieldOperand(e) {
				return FETCH_VALUE
			} else if evalComparisonOperators(db, k, nil, e) {
				return INCLUDE
			} else {
				return EXCLUDE
			}
		}
	}
}
