// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query_functions

import (
	"strings"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/conversions"
	"v.io/v23/query/engine/internal/query_parser"
	"v.io/v23/query/syncql"
	"v.io/v23/vom"
)

type queryFunc func(ds.Database, int64, []*query_parser.Operand) (*query_parser.Operand, error)
type checkArgsFunc func(ds.Database, int64, []*query_parser.Operand) error

type function struct {
	argTypes      []query_parser.OperandType // TypNil allows any.
	hasVarArgs    bool
	varArgsType   query_parser.OperandType // ignored if !hasVarArgs, TypNil allows any.
	returnType    query_parser.OperandType
	funcAddr      queryFunc
	checkArgsAddr checkArgsFunc
}

var functions map[string]function
var lowercaseFunctions map[string]string // map of lowercase(funcName)->funcName

func init() {
	functions = make(map[string]function)

	// Time Functions
	functions["Time"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypTime, timeFunc, nil}
	functions["Now"] = function{[]query_parser.OperandType{}, false, query_parser.TypNil, query_parser.TypTime, now, nil}
	functions["Year"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, year, secondArgLocationCheck}
	functions["Month"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, month, secondArgLocationCheck}
	functions["Day"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, day, secondArgLocationCheck}
	functions["Hour"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, hour, secondArgLocationCheck}
	functions["Minute"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, minute, secondArgLocationCheck}
	functions["Second"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, second, secondArgLocationCheck}
	functions["Nanosecond"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, nanosecond, secondArgLocationCheck}
	functions["Weekday"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, weekday, secondArgLocationCheck}
	functions["YearDay"] = function{[]query_parser.OperandType{query_parser.TypTime, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, yearDay, secondArgLocationCheck}

	// String Functions
	functions["Atoi"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, atoi, nil}
	functions["Atof"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypFloat, atof, nil}
	functions["HtmlEscape"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, htmlEscapeFunc, nil}
	functions["HtmlUnescape"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, htmlUnescapeFunc, nil}
	functions["Lowercase"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, lowerCase, nil}
	functions["Split"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypObject, split, nil}
	functions["Type"] = function{[]query_parser.OperandType{query_parser.TypObject}, false, query_parser.TypNil, query_parser.TypStr, typeFunc, typeFuncFieldCheck}
	functions["Uppercase"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, upperCase, nil}
	functions["RuneCount"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, runeCount, nil}
	functions["Sprintf"] = function{[]query_parser.OperandType{query_parser.TypStr}, true, query_parser.TypNil, query_parser.TypStr, sprintf, nil}
	functions["Str"] = function{[]query_parser.OperandType{query_parser.TypNil}, false, query_parser.TypNil, query_parser.TypStr, str, nil}
	functions["StrCat"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypStr}, true, query_parser.TypStr, query_parser.TypStr, strCat, nil}
	functions["StrIndex"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, strIndex, nil}
	functions["StrRepeat"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypInt}, false, query_parser.TypNil, query_parser.TypStr, strRepeat, nil}
	functions["StrReplace"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypStr, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, strReplace, nil}
	functions["StrLastIndex"] = function{[]query_parser.OperandType{query_parser.TypStr, query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypInt, strLastIndex, nil}
	functions["Trim"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, trim, nil}
	functions["TrimLeft"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, trimLeft, nil}
	functions["TrimRight"] = function{[]query_parser.OperandType{query_parser.TypStr}, false, query_parser.TypNil, query_parser.TypStr, trimRight, nil}

	// Math functions
	functions["Ceiling"] = function{[]query_parser.OperandType{query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, ceilingFunc, nil}
	functions["Floor"] = function{[]query_parser.OperandType{query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, floorFunc, nil}
	functions["Inf"] = function{[]query_parser.OperandType{query_parser.TypInt}, false, query_parser.TypNil, query_parser.TypFloat, infFunc, nil}
	functions["IsInf"] = function{[]query_parser.OperandType{query_parser.TypFloat, query_parser.TypInt}, false, query_parser.TypNil, query_parser.TypBool, isInfFunc, nil}
	functions["IsNaN"] = function{[]query_parser.OperandType{query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypBool, isNanFunc, nil}
	functions["Log"] = function{[]query_parser.OperandType{query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, logFunc, nil}
	functions["Log10"] = function{[]query_parser.OperandType{query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, log10Func, nil}
	functions["NaN"] = function{[]query_parser.OperandType{}, false, query_parser.TypNil, query_parser.TypFloat, nanFunc, nil}
	functions["Pow"] = function{[]query_parser.OperandType{query_parser.TypFloat, query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, powFunc, nil}
	functions["Pow10"] = function{[]query_parser.OperandType{query_parser.TypInt}, false, query_parser.TypNil, query_parser.TypFloat, pow10Func, nil}
	functions["Mod"] = function{[]query_parser.OperandType{query_parser.TypFloat, query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, modFunc, nil}
	functions["Truncate"] = function{[]query_parser.OperandType{query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, truncateFunc, nil}
	functions["Remainder"] = function{[]query_parser.OperandType{query_parser.TypFloat, query_parser.TypFloat}, false, query_parser.TypNil, query_parser.TypFloat, remainderFunc, nil}

	// TODO(jkline): Make len work with more types.
	functions["Len"] = function{[]query_parser.OperandType{query_parser.TypObject}, false, query_parser.TypNil, query_parser.TypInt, lenFunc, nil}

	// Build lowercaseFuncName->funcName
	lowercaseFunctions = make(map[string]string)
	for f := range functions {
		lowercaseFunctions[strings.ToLower(f)] = f
	}
}

// Check that function exists and that the number of args passed matches the spec.
// Call query_functions.CheckFunction.  This will check for, to the extent possible, correct types.
// Furthermore, it may execute the function if the function takes no args or
// takes only literal args (or an arg that is a function that is also executed
// early).  CheckFunction will fill in arg types, return types and may fill in
// Computed and RetValue.
func CheckFunction(db ds.Database, f *query_parser.Function) error {
	if entry, err := lookupFuncName(db, f); err != nil {
		return err
	} else {
		f.ArgTypes = entry.argTypes
		f.RetType = entry.returnType
		if !entry.hasVarArgs && len(f.Args) != len(entry.argTypes) {
			return syncql.NewErrFunctionArgCount(db.GetContext(), f.Off, f.Name, int64(len(f.ArgTypes)), int64(len(f.Args)))
		}
		if entry.hasVarArgs && len(f.Args) < len(entry.argTypes) {
			return syncql.NewErrFunctionAtLeastArgCount(db.GetContext(), f.Off, f.Name, int64(len(f.ArgTypes)), int64(len(f.Args)))
		}
		// Standard check for types of fixed and var args
		if err = argsStandardCheck(db, f.Off, entry, f.Args); err != nil {
			return err
		}
		// Check if the function can be executed now.
		// If any arg is not a literal and not a function that has been already executed,
		// then okToExecuteNow will be set to false.
		okToExecuteNow := true
		for _, arg := range f.Args {
			switch arg.Type {
			case query_parser.TypBigInt, query_parser.TypBigRat, query_parser.TypBool, query_parser.TypFloat, query_parser.TypInt, query_parser.TypStr, query_parser.TypTime, query_parser.TypUint:
				// do nothing
			case query_parser.TypFunction:
				if !arg.Function.Computed {
					okToExecuteNow = false
					break
				}
			default:
				okToExecuteNow = false
				break
			}
		}
		// If all of the functions args are literals or already computed functions,
		// execute this function now and save the result.
		if okToExecuteNow {
			op, err := ExecFunction(db, f, f.Args)
			if err != nil {
				return err
			}
			f.Computed = true
			f.RetValue = op
			return nil
		} else {
			// We can't execute now, but give the function a chance to complain
			// about the arguments that it can check now.
			return FuncCheck(db, f, f.Args)
		}
	}
}

func lookupFuncName(db ds.Database, f *query_parser.Function) (*function, error) {
	if entry, ok := functions[f.Name]; !ok {
		// No such function, is the case wrong?
		if correctCase, ok := lowercaseFunctions[strings.ToLower(f.Name)]; !ok {
			return nil, syncql.NewErrFunctionNotFound(db.GetContext(), f.Off, f.Name)
		} else {
			// the case is wrong
			return nil, syncql.NewErrDidYouMeanFunction(db.GetContext(), f.Off, correctCase)
		}
	} else {
		return &entry, nil
	}
}

func FuncCheck(db ds.Database, f *query_parser.Function, args []*query_parser.Operand) error {
	if entry, err := lookupFuncName(db, f); err != nil {
		return err
	} else {
		if entry.checkArgsAddr != nil {
			if err := entry.checkArgsAddr(db, f.Off, args); err != nil {
				return err
			}
		}
	}
	return nil
}

func ExecFunction(db ds.Database, f *query_parser.Function, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if entry, err := lookupFuncName(db, f); err != nil {
		return nil, err
	} else {
		retValue, err := entry.funcAddr(db, f.Off, args)
		if err != nil {
			return nil, err
		} else {
			return retValue, nil
		}
	}
}

func ConvertFunctionRetValueToRawBytes(o *query_parser.Operand) *vom.RawBytes {
	if o == nil {
		return vom.RawBytesOf(nil)
	}
	switch o.Type {
	case query_parser.TypBool:
		return vom.RawBytesOf(o.Bool)
	case query_parser.TypFloat:
		return vom.RawBytesOf(o.Float)
	case query_parser.TypInt:
		return vom.RawBytesOf(o.Int)
	case query_parser.TypStr:
		return vom.RawBytesOf(o.Str)
	case query_parser.TypTime:
		return vom.RawBytesOf(o.Time)
	case query_parser.TypObject:
		return vom.RawBytesOf(o.Object)
	case query_parser.TypUint:
		return vom.RawBytesOf(o.Uint)
	default:
		// Other types can't be converted and *shouldn't* be returned
		// from a function.  This case will result in a nil for this
		// column in the row.
		return vom.RawBytesOf(nil)
	}
}

func makeStrOp(off int64, s string) *query_parser.Operand {
	var o query_parser.Operand
	o.Off = off
	o.Type = query_parser.TypStr
	o.Str = s
	return &o
}

func makeBoolOp(off int64, b bool) *query_parser.Operand {
	var o query_parser.Operand
	o.Off = off
	o.Type = query_parser.TypBool
	o.Bool = b
	return &o
}

func makeIntOp(off int64, i int64) *query_parser.Operand {
	var o query_parser.Operand
	o.Off = off
	o.Type = query_parser.TypInt
	o.Int = i
	return &o
}

func makeFloatOp(off int64, r float64) *query_parser.Operand {
	var o query_parser.Operand
	o.Off = off
	o.Type = query_parser.TypFloat
	o.Float = r
	return &o
}

func checkArg(db ds.Database, off int64, argType query_parser.OperandType, arg *query_parser.Operand) error {
	// We can't check unless the arg is a literal or an already computed function,
	var operandToConvert *query_parser.Operand
	switch arg.Type {
	case query_parser.TypBigInt, query_parser.TypBigRat, query_parser.TypBool, query_parser.TypFloat, query_parser.TypInt, query_parser.TypStr, query_parser.TypTime, query_parser.TypUint:
		operandToConvert = arg
	case query_parser.TypFunction:
		if arg.Function.Computed {
			operandToConvert = arg.Function.RetValue
		} else {
			return nil // arg is not yet resolved, we can't check
		}
	default:
		return nil // arg is not yet resolved, we can't check
	}
	// make sure it can be converted to argType.
	var err error
	switch argType {
	case query_parser.TypBigInt:
		_, err = conversions.ConvertValueToBigInt(operandToConvert)
		if err != nil {
			err = syncql.NewErrBigIntConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypBigRat:
		_, err = conversions.ConvertValueToBigRat(operandToConvert)
		if err != nil {
			err = syncql.NewErrBigRatConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypBool:
		_, err = conversions.ConvertValueToBool(operandToConvert)
		if err != nil {
			err = syncql.NewErrBoolConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypFloat:
		_, err = conversions.ConvertValueToFloat(operandToConvert)
		if err != nil {
			err = syncql.NewErrFloatConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypInt:
		_, err = conversions.ConvertValueToInt(operandToConvert)
		if err != nil {
			err = syncql.NewErrIntConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypStr:
		_, err = conversions.ConvertValueToString(operandToConvert)
		if err != nil {
			err = syncql.NewErrStringConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypTime:
		_, err = conversions.ConvertValueToTime(operandToConvert)
		if err != nil {
			err = syncql.NewErrTimeConversionError(db.GetContext(), arg.Off, err)
		}
	case query_parser.TypUint:
		_, err = conversions.ConvertValueToUint(operandToConvert)
		if err != nil {
			err = syncql.NewErrUintConversionError(db.GetContext(), arg.Off, err)
		}
	}
	return err
}

// Check types of fixed args.  For functions that take varargs, check that the type of
// any varargs matches the type specified.
func argsStandardCheck(db ds.Database, off int64, f *function, args []*query_parser.Operand) error {
	// Check types of required args.
	for i := 0; i < len(f.argTypes); i++ {
		if err := checkArg(db, off, f.argTypes[i], args[i]); err != nil {
			return err
		}
	}
	// Check types of varargs.
	if f.hasVarArgs {
		for i := len(f.argTypes); i < len(args); i++ {
			if f.varArgsType != query_parser.TypNil {
				if err := checkArg(db, off, f.varArgsType, args[i]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
