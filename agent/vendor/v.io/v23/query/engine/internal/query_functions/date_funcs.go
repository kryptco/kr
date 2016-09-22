// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query_functions

// TODO(jkline): Probably rename this file to time_functions.go

import (
	"time"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/conversions"
	"v.io/v23/query/engine/internal/query_parser"
	"v.io/v23/query/syncql"
)

// If possible, check if arg is convertible to a location.  Fields and not yet computed
// functions cannot be checked and will just return nil.
func checkIfPossibleThatArgIsConvertibleToLocation(db ds.Database, arg *query_parser.Operand) error {
	var locStr *query_parser.Operand
	var err error
	switch arg.Type {
	case query_parser.TypBigInt, query_parser.TypBigRat, query_parser.TypBool, query_parser.TypFloat, query_parser.TypInt, query_parser.TypStr, query_parser.TypTime, query_parser.TypUint:
		if locStr, err = conversions.ConvertValueToString(arg); err != nil {
			if err != nil {
				return syncql.NewErrLocationConversionError(db.GetContext(), arg.Off, err)
			} else {
				return nil
			}
		}
	case query_parser.TypFunction:
		if arg.Function.Computed {
			if locStr, err = conversions.ConvertValueToString(arg.Function.RetValue); err != nil {
				if err != nil {
					return syncql.NewErrLocationConversionError(db.GetContext(), arg.Off, err)
				} else {
					return nil
				}
			}
		} else {
			// Arg is uncomputed function, can't make determination about arg.
			return nil
		}
	default:
		// Arg is not a literal or function, can't make determination about arg.
		return nil
	}
	_, err = time.LoadLocation(locStr.Str)
	if err != nil {
		return syncql.NewErrLocationConversionError(db.GetContext(), arg.Off, err)
	} else {
		return nil
	}
}

// Time(layout, value string)
// e.g., Time("Mon Jan 2 15:04:05 -0700 MST 2006", "Tue Aug 25 10:01:00 -0700 PDT 2015")
// e.g., Time("Jan 2 15:04 MST 2006", "Aug 25 10:01 PDT 2015")
func timeFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	layoutOp, err := conversions.ConvertValueToString(args[0])
	if err != nil {
		return nil, err
	}
	valueOp, err := conversions.ConvertValueToString(args[1])
	if err != nil {
		return nil, err
	}
	// Mon Jan 2 15:04:05 -0700 MST 2006
	tim, err := time.Parse(layoutOp.Str, valueOp.Str)
	if err != nil {
		return nil, err
	}
	return makeTimeOp(off, tim), nil
}

// now()
func now(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	return makeTimeOp(off, time.Now()), nil
}
func timeInLocation(db ds.Database, off int64, args []*query_parser.Operand) (time.Time, error) {
	var timeOp *query_parser.Operand
	var locOp *query_parser.Operand
	var err error
	if timeOp, err = conversions.ConvertValueToTime(args[0]); err != nil {
		return time.Time{}, err
	}
	if locOp, err = conversions.ConvertValueToString(args[1]); err != nil {
		return time.Time{}, err
	}
	var loc *time.Location
	if loc, err = time.LoadLocation(locOp.Str); err != nil {
		return time.Time{}, err
	}
	return timeOp.Time.In(loc), nil
}

// Year(v.InvoiceDate, "America/Los_Angeles")
func year(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Year())), nil
	}
}

// Month(v.InvoiceDate, "America/Los_Angeles")
func month(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Month())), nil
	}
}

// Day(v.InvoiceDate, "America/Los_Angeles")
func day(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Day())), nil
	}
}

// Hour(v.InvoiceDate, "America/Los_Angeles")
func hour(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Hour())), nil
	}
}

// Minute(v.InvoiceDate, "America/Los_Angeles")
func minute(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Minute())), nil
	}
}

// Second(v.InvoiceDate, "America/Los_Angeles")
func second(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Second())), nil
	}
}

// Nanosecond(v.InvoiceDate, "America/Los_Angeles")
func nanosecond(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Nanosecond())), nil
	}
}

// Weekday(v.InvoiceDate, "America/Los_Angeles")
func weekday(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.Weekday())), nil
	}
}

// YearDay(v.InvoiceDate, "America/Los_Angeles")
func yearDay(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	if tim, err := timeInLocation(db, off, args); err != nil {
		return nil, err
	} else {
		return makeIntOp(off, int64(tim.YearDay())), nil
	}
}

func makeTimeOp(off int64, tim time.Time) *query_parser.Operand {
	var o query_parser.Operand
	o.Off = off
	o.Type = query_parser.TypTime
	o.Time = tim
	return &o
}

func secondArgLocationCheck(db ds.Database, off int64, args []*query_parser.Operand) error {
	// At this point, for the args that can be evaluated before execution, it is known that
	// there are two args, a time followed by a string.
	// Just need to check that the 2nd arg is convertible to a location.
	if err := checkIfPossibleThatArgIsConvertibleToLocation(db, args[1]); err != nil {
		return err
	}
	return nil
}
