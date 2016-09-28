// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conversions

import (
	"errors"
	"math/big"
	"strings"

	"v.io/v23/query/engine/internal/query_parser"
)

func ConvertValueToString(o *query_parser.Operand) (*query_parser.Operand, error) {
	// Other types must be explicitly converted to string with the Str() function.
	var c query_parser.Operand
	c.Type = query_parser.TypStr
	c.Off = o.Off
	switch o.Type {
	case query_parser.TypStr:
		c.Str = o.Str
		c.Prefix = o.Prefix   // non-nil for rhs of like expressions
		c.Pattern = o.Pattern // non-nil for rhs of like expressions
	default:
		return nil, errors.New("Cannot convert operand to string.")
	}
	return &c, nil
}

func ConvertValueToTime(o *query_parser.Operand) (*query_parser.Operand, error) {
	switch o.Type {
	case query_parser.TypTime:
		return o, nil
	default:
		return nil, errors.New("Cannot convert operand to time.")
	}
}

func ConvertValueToBigRat(o *query_parser.Operand) (*query_parser.Operand, error) {
	// operand cannot be string literal.
	var c query_parser.Operand
	c.Type = query_parser.TypBigRat
	switch o.Type {
	case query_parser.TypBigInt:
		var b big.Rat
		c.BigRat = b.SetInt(o.BigInt)
	case query_parser.TypBigRat:
		c.BigRat = o.BigRat
	case query_parser.TypBool:
		return nil, errors.New("Cannot convert bool to big.Rat.")
	case query_parser.TypFloat:
		var b big.Rat
		c.BigRat = b.SetFloat64(o.Float)
	case query_parser.TypInt:
		c.BigRat = big.NewRat(o.Int, 1)
	case query_parser.TypUint:
		var bi big.Int
		bi.SetUint64(o.Uint)
		var br big.Rat
		c.BigRat = br.SetInt(&bi)
	case query_parser.TypObject:
		return nil, errors.New("Cannot convert object to big.Rat.")
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return nil, errors.New("Cannot convert operand to big.Rat.")
	}
	return &c, nil
}

func ConvertValueToFloat(o *query_parser.Operand) (*query_parser.Operand, error) {
	// Operand cannot be literal, big.Rat or big.Int
	var c query_parser.Operand
	c.Type = query_parser.TypFloat
	switch o.Type {
	case query_parser.TypBool:
		return nil, errors.New("Cannot convert bool to float64.")
	case query_parser.TypFloat:
		c.Float = o.Float
	case query_parser.TypInt:
		c.Float = float64(o.Int)
	case query_parser.TypUint:
		c.Float = float64(o.Uint)
	case query_parser.TypObject:
		return nil, errors.New("Cannot convert object to float64.")
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return nil, errors.New("Cannot convert operand to float64.")
	}
	return &c, nil
}

func ConvertValueToBool(o *query_parser.Operand) (*query_parser.Operand, error) {
	var c query_parser.Operand
	c.Type = query_parser.TypBool
	switch o.Type {
	case query_parser.TypBool:
		c.Bool = o.Bool
	case query_parser.TypStr:
		switch strings.ToLower(o.Str) {
		case "true":
			c.Bool = true
		case "false":
			c.Bool = false
		default:
			return nil, errors.New("Cannot convert object to bool.")
		}
	default:
		return nil, errors.New("Cannot convert operand to bool.")
	}
	return &c, nil
}

func ConvertValueToBigInt(o *query_parser.Operand) (*query_parser.Operand, error) {
	// Operand cannot be literal, big.Rat or float.
	var c query_parser.Operand
	c.Type = query_parser.TypBigInt
	switch o.Type {
	case query_parser.TypBigInt:
		c.BigInt = o.BigInt
	case query_parser.TypBool:
		return nil, errors.New("Cannot convert bool to big.Int.")
	case query_parser.TypInt:
		c.BigInt = big.NewInt(o.Int)
	case query_parser.TypUint:
		var b big.Int
		b.SetUint64(o.Uint)
		c.BigInt = &b
	case query_parser.TypObject:
		return nil, errors.New("Cannot convert object to big.Int.")
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return nil, errors.New("Cannot convert operand to big.Int.")
	}
	return &c, nil
}

func ConvertValueToInt(o *query_parser.Operand) (*query_parser.Operand, error) {
	// Operand cannot be literal, big.Rat or float or uint64.
	var c query_parser.Operand
	c.Type = query_parser.TypInt
	switch o.Type {
	case query_parser.TypBool:
		return nil, errors.New("Cannot convert bool to int64.")
	case query_parser.TypInt:
		c.Int = o.Int
	case query_parser.TypObject:
		return nil, errors.New("Cannot convert object to int64.")
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return nil, errors.New("Cannot convert operand to int64.")
	}
	return &c, nil
}

func ConvertValueToUint(o *query_parser.Operand) (*query_parser.Operand, error) {
	// Operand cannot be literal, big.Rat or float or int64.
	var c query_parser.Operand
	c.Type = query_parser.TypUint
	switch o.Type {
	case query_parser.TypBool:
		return nil, errors.New("Cannot convert bool to int64.")
	case query_parser.TypUint:
		c.Uint = o.Uint
	case query_parser.TypObject:
		return nil, errors.New("Cannot convert object to int64.")
	default:
		// TODO(jkline): Log this logic error and all other similar cases.
		return nil, errors.New("Cannot convert operand to int64.")
	}
	return &c, nil
}
