// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query_functions

import (
	"math"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/conversions"
	"v.io/v23/query/engine/internal/query_parser"
)

func ceilingFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Ceil(f.Float)), nil
}

func floorFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Floor(f.Float)), nil
}

func nanFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	return makeFloatOp(off, math.NaN()), nil
}

func isNanFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	return makeBoolOp(off, math.IsNaN(f.Float)), nil
}

func infFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	i, err := conversions.ConvertValueToInt(args[0])
	if err != nil {
		return nil, err
	}
	var sign int
	if i.Int < 0 {
		sign = -1
	} else if i.Int == 0 {
		sign = 0
	} else {
		sign = 1
	}

	return makeFloatOp(off, math.Inf(sign)), nil
}

func isInfFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	i, err := conversions.ConvertValueToInt(args[1])
	if err != nil {
		return nil, err
	}
	var sign int
	if i.Int < 0 {
		sign = -1
	} else if i.Int == 0 {
		sign = 0
	} else {
		sign = 1
	}

	return makeBoolOp(off, math.IsInf(f.Float, sign)), nil
}

func logFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Log(f.Float)), nil
}

func log10Func(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Log10(f.Float)), nil
}

func powFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	x, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	y, err := conversions.ConvertValueToFloat(args[1])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Pow(x.Float, y.Float)), nil
}

func pow10Func(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	x, err := conversions.ConvertValueToInt(args[0])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Pow10(int(x.Int))), nil
}

func modFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	x, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}

	y, err := conversions.ConvertValueToFloat(args[1])
	if err != nil {
		return nil, err
	}

	return makeFloatOp(off, math.Mod(x.Float, y.Float)), nil
}

func truncateFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	f, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}
	return makeFloatOp(off, math.Trunc(f.Float)), nil
}

func remainderFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	x, err := conversions.ConvertValueToFloat(args[0])
	if err != nil {
		return nil, err
	}

	y, err := conversions.ConvertValueToFloat(args[1])
	if err != nil {
		return nil, err
	}

	return makeFloatOp(off, math.Remainder(x.Float, y.Float)), nil
}
