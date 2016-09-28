// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query_parser

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode/utf8"

	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/pattern"
	"v.io/v23/query/syncql"
	"v.io/v23/vdl"
)

type TokenType int

const (
	TokCHAR TokenType = 1 + iota
	TokCOMMA
	TokEOF
	TokEQUAL
	TokFLOAT
	TokIDENT
	TokINT
	TokLEFTANGLEBRACKET
	TokLEFTBRACKET
	TokLEFTPAREN
	TokMINUS
	TokParameter // allowed only in prepared statements
	TokPERIOD
	TokRIGHTANGLEBRACKET
	TokRIGHTBRACKET
	TokRIGHTPAREN
	TokSTRING
	TokERROR
)

const (
	MaxStatementLen = 10000
)

type Token struct {
	Tok   TokenType
	Value string
	Off   int64
}

type Node struct {
	Off int64
}

type Statement interface {
	Offset() int64
	String() string
	CopyAndSubstitute(db ds.Database, paramValues []*vdl.Value) (Statement, error)
}

type Segment struct {
	Value string
	Keys  []*Operand // Used as key(s) or index(es) to dereference map/set/array/list.
	Node
}

type Field struct {
	Segments []Segment
	Node
}

type BinaryOperatorType int

const (
	And BinaryOperatorType = 1 + iota
	Equal
	GreaterThan
	GreaterThanOrEqual
	Is
	IsNot
	LessThan
	LessThanOrEqual
	Like
	NotEqual
	NotLike
	Or
)

type BinaryOperator struct {
	Type BinaryOperatorType
	Node
}

type OperandType int

const (
	TypBigInt OperandType = 1 + iota // Only as a result of Resolve/Coerce Operand
	TypBigRat                        // Only as a result of Resolve/Coerce Operand
	TypBool
	TypExpr
	TypField
	TypFloat
	TypFunction
	TypInt
	TypNil
	TypParameter
	TypStr
	TypTime
	TypObject // Only as the result of a ResolveOperand
	TypUint   // Only as the result of a ResolveOperand
)

type Operand struct {
	Type     OperandType
	BigInt   *big.Int
	BigRat   *big.Rat
	Bool     bool
	Column   *Field
	Float    float64
	Function *Function
	Int      int64
	Str      string
	Time     time.Time
	Prefix   string           // Computed by checker for Like expressions
	Pattern  *pattern.Pattern // Computed by checker for Like expressions
	Uint     uint64
	Expr     *Expression
	Object   *vdl.Value
	Node
}

type Function struct {
	Name     string
	Args     []*Operand
	ArgTypes []OperandType // Filled in by checker.
	RetType  OperandType   // Filled in by checker.
	Computed bool          // Checker sets to true and sets RetValue if function takes no args
	RetValue *Operand
	Node
}

type Expression struct {
	Operand1 *Operand
	Operator *BinaryOperator
	Operand2 *Operand
	Node
}

type SelectorType int

const (
	TypSelField SelectorType = 1 + iota
	TypSelFunc
)

// Selector: entries in the select clause.
// Entries can be functions for fields.
// The AS name, if present, will ONLY be used in the
// returned column header.
type Selector struct {
	Type     SelectorType
	Field    *Field
	Function *Function
	As       *AsClause // If not nil, used in returned column header.
	Node
}

type AsClause struct {
	AltName Name
	Node
}

type Name struct {
	Value string
	Node
}

type SelectClause struct {
	Selectors []Selector
	Node
}

type FromClause struct {
	Table TableEntry
	Node
}

type TableEntry struct {
	Name    string
	DBTable ds.Table // Checker gets table from db and sets this.
	Node
}

type WhereClause struct {
	Expr *Expression
	Node
}

type CharValue struct {
	Value rune
	Node
}

type Int64Value struct {
	Value int64
	Node
}

type EscapeClause struct {
	EscapeChar *CharValue
	Node
}

type LimitClause struct {
	Limit *Int64Value
	Node
}

type ResultsOffsetClause struct {
	ResultsOffset *Int64Value
	Node
}

type SelectStatement struct {
	Select        *SelectClause
	From          *FromClause
	Where         *WhereClause
	Escape        *EscapeClause
	Limit         *LimitClause
	ResultsOffset *ResultsOffsetClause
	Node
}

type DeleteStatement struct {
	From   *FromClause
	Where  *WhereClause
	Escape *EscapeClause
	Limit  *LimitClause
	Node
}

func scanToken(s *scanner.Scanner) *Token {
	// TODO(jkline): Replace golang text/scanner.
	var token Token
	tok := s.Scan()
	token.Value = s.TokenText()
	token.Off = int64(s.Position.Offset)

	if s.ErrorCount > 0 {
		token.Tok = TokERROR
		return &token
	}

	switch tok {
	case '.':
		token.Tok = TokPERIOD
	case ',':
		token.Tok = TokCOMMA
	case '-':
		token.Tok = TokMINUS
	case '(':
		token.Tok = TokLEFTPAREN
	case ')':
		token.Tok = TokRIGHTPAREN
	case '=':
		token.Tok = TokEQUAL
	case '<':
		token.Tok = TokLEFTANGLEBRACKET
	case '>':
		token.Tok = TokRIGHTANGLEBRACKET
	case '[':
		token.Tok = TokLEFTBRACKET
	case ']':
		token.Tok = TokRIGHTBRACKET
	case '?':
		token.Tok = TokParameter
	case scanner.EOF:
		token.Tok = TokEOF
	case scanner.Ident:
		token.Tok = TokIDENT
	case scanner.Int:
		token.Tok = TokINT
	case scanner.Float:
		token.Tok = TokFLOAT
	case scanner.Char:
		token.Tok = TokCHAR
		token.Value = token.Value[1 : len(token.Value)-1]
	case scanner.String:
		token.Tok = TokSTRING
		token.Value = token.Value[1 : len(token.Value)-1]
	}
	return &token
}

// Text/scanner reports errors to stderr that are not errors in the query language.
// For example, to get the value where the key is "\",
// One would write the string:
// "select v where k = \"\\\""
// This will result in the scanner spewing "literal not terminated" to stderr if we don't
// set the Error field in Scanner.  As such, Error is set to the following function which
// eats errors.  In the longer term, there is still a TODO to replace text/scanner with
// our own scanner.
func scannerError(s *scanner.Scanner, msg string) {
	// Do nothing.
}

// Parse a statement.  Return it or an error.
func Parse(db ds.Database, src string) (*Statement, error) {
	if len(src) > MaxStatementLen {
		return nil, syncql.NewErrMaxStatementLenExceeded(db.GetContext(), int64(0), MaxStatementLen, int64(len(src)))
	}
	r := strings.NewReader(src)
	var s scanner.Scanner
	s.Init(r)
	s.Error = scannerError

	token := scanToken(&s) // eat the select
	if token.Tok == TokEOF {
		return nil, syncql.NewErrNoStatementFound(db.GetContext(), token.Off)
	}
	if token.Tok != TokIDENT {
		return nil, syncql.NewErrExpectedIdentifier(db.GetContext(), token.Off, token.Value)
	}
	switch strings.ToLower(token.Value) {
	case "select":
		var st Statement
		var err error
		st, token, err = selectStatement(db, &s, token)
		return &st, err
	case "delete":
		var st Statement
		var err error
		st, token, err = deleteStatement(db, &s, token)
		return &st, err
	default:
		return nil, syncql.NewErrUnknownIdentifier(db.GetContext(), token.Off, token.Value)
	}
}

// Parse select.
func selectStatement(db ds.Database, s *scanner.Scanner, token *Token) (Statement, *Token, error) {
	var st SelectStatement
	st.Off = token.Off

	// parse SelectClause
	var err error
	st.Select, token, err = parseSelectClause(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	st.From, token, err = parseFromClause(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	st.Where, token, err = parseWhereClause(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	st.Escape, st.Limit, st.ResultsOffset, token, err = parseEscapeLimitResultsOffsetClauses(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	// There can be nothing remaining for the current statement
	if token.Tok != TokEOF {
		return nil, nil, syncql.NewErrUnexpected(db.GetContext(), token.Off, token.Value)
	}

	return st, token, nil
}

// Parse delete.
func deleteStatement(db ds.Database, s *scanner.Scanner, token *Token) (Statement, *Token, error) {
	var st DeleteStatement
	st.Off = token.Off

	token = scanToken(s) // eat the delete
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}

	// parse FromClause
	var err error
	st.From, token, err = parseFromClause(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	// parse WhereClause
	st.Where, token, err = parseWhereClause(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	st.Escape, st.Limit, token, err = parseEscapeLimitClauses(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	// There can be nothing remaining for the current statement
	if token.Tok != TokEOF {
		return nil, nil, syncql.NewErrUnexpected(db.GetContext(), token.Off, token.Value)
	}

	return st, token, nil
}

// Parse the select clause (fields). Return *SelectClause, next token (or error).
func parseSelectClause(db ds.Database, s *scanner.Scanner, token *Token) (*SelectClause, *Token, error) {
	// must be at least one selector or it is an error
	// field seclectors may be in dot notation
	// selectors are separated by commas
	var selectClause SelectClause
	selectClause.Off = token.Off
	token = scanToken(s) // eat the select
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	var err error
	// scan first selector
	if token, err = parseSelector(db, s, &selectClause, token); err != nil {
		return nil, nil, err
	}

	// More selectors?
	for token.Tok == TokCOMMA {
		token = scanToken(s)
		if token, err = parseSelector(db, s, &selectClause, token); err != nil {
			return nil, nil, err
		}
	}

	return &selectClause, token, nil
}

// Parse a selector. Return next token (or error).
func parseSelector(db ds.Database, s *scanner.Scanner, selectClause *SelectClause, token *Token) (*Token, error) {
	if token.Tok != TokIDENT {
		return nil, syncql.NewErrExpectedIdentifier(db.GetContext(), token.Off, token.Value)
	}

	var selector Selector
	selector.Off = token.Off
	selector.Type = TypSelField
	var field Field
	selector.Field = &field
	selector.Field.Off = token.Off
	selector.Field = &field

	var segment *Segment
	var err error
	if segment, token, err = parseSegment(db, s, token); err != nil {
		return nil, err
	}
	selector.Field.Segments = append(selector.Field.Segments, *segment)

	// It might be a function.
	if token.Tok == TokLEFTPAREN {
		// Segments with a key(s) specified cannot be function calls.
		if len(segment.Keys) != 0 {
			return nil, syncql.NewErrUnexpected(db.GetContext(), token.Off, token.Value)
		}
		// switch selector to a function
		selector.Type = TypSelFunc
		var err error
		if selector.Function, token, err = parseFunction(db, s, segment.Value, segment.Off, token); err != nil {
			return nil, err
		}
		selector.Field = nil

	} else {
		for token.Tok != TokEOF && token.Tok == TokPERIOD {
			token = scanToken(s)
			if token.Tok != TokIDENT {
				return nil, syncql.NewErrExpectedIdentifier(db.GetContext(), token.Off, token.Value)
			}
			if segment, token, err = parseSegment(db, s, token); err != nil {
				return nil, err
			}
			selector.Field.Segments = append(selector.Field.Segments, *segment)
		}
	}

	// Check for AS
	if token.Tok == TokIDENT && strings.ToLower(token.Value) == "as" {
		var asClause AsClause
		asClause.Off = token.Off
		token = scanToken(s)
		if token.Tok != TokIDENT {
			return nil, syncql.NewErrExpectedIdentifier(db.GetContext(), token.Off, token.Value)
		}
		asClause.AltName.Value = token.Value
		asClause.AltName.Off = token.Off
		selector.As = &asClause
		token = scanToken(s)
	}

	selectClause.Selectors = append(selectClause.Selectors, selector)
	return token, nil
}

// Parse a segment. Return the segment and the next token (or error).
// Check for a key (i.e., [<key>] following the segment).
func parseSegment(db ds.Database, s *scanner.Scanner, token *Token) (*Segment, *Token, error) {
	var segment Segment
	segment.Value = token.Value
	segment.Off = token.Off
	token = scanToken(s)

	for token.Tok == TokLEFTBRACKET {
		// A key to the segment is specified.
		token = scanToken(s)
		var key *Operand
		var err error
		key, token, err = parseOperand(db, s, token)
		if err != nil {
			return nil, nil, err
		}
		segment.Keys = append(segment.Keys, key)
		if token.Tok != TokRIGHTBRACKET {
			return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, "]")
		}
		token = scanToken(s)
	}
	return &segment, token, nil
}

// Parse the from clause, Return FromClause and next Token or error.
func parseFromClause(db ds.Database, s *scanner.Scanner, token *Token) (*FromClause, *Token, error) {
	if strings.ToLower(token.Value) != "from" {
		return nil, nil, syncql.NewErrExpectedFrom(db.GetContext(), token.Off, token.Value)
	}
	var fromClause FromClause
	fromClause.Off = token.Off
	token = scanToken(s) // eat from
	// must be a table specified
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	if token.Tok != TokIDENT {
		return nil, nil, syncql.NewErrExpectedIdentifier(db.GetContext(), token.Off, token.Value)
	}
	fromClause.Table.Off = token.Off
	fromClause.Table.Name = token.Value
	token = scanToken(s)
	return &fromClause, token, nil
}

// Parse the where clause (if any).  Return WhereClause (could be nil) and and next Token or error.
func parseWhereClause(db ds.Database, s *scanner.Scanner, token *Token) (*WhereClause, *Token, error) {
	// parse Optional where clause
	if token.Tok != TokEOF {
		if strings.ToLower(token.Value) != "where" {
			return nil, token, nil
		}
		var where WhereClause
		where.Off = token.Off
		token = scanToken(s)
		// parse expression
		var err error
		where.Expr, token, err = parseExpression(db, s, token)
		if err != nil {
			return nil, nil, err
		}
		return &where, token, nil
	} else {
		return nil, token, nil
	}
}

// Parse a parenthesized expression.  Return expression and next token (or error)
func parseParenthesizedExpression(db ds.Database, s *scanner.Scanner, token *Token) (*Expression, *Token, error) {
	// Only called when token == TokLEFTPAREN
	token = scanToken(s) // eat '('
	var expr *Expression
	var err error
	expr, token, err = parseExpression(db, s, token)
	if err != nil {
		return nil, nil, err
	}
	// Expect right paren
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	if token.Tok != TokRIGHTPAREN {
		return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, ")")
	}
	token = scanToken(s) // eat ')'
	return expr, token, nil
}

// Parse an expression.  Return expression and next token (or error)
func parseExpression(db ds.Database, s *scanner.Scanner, token *Token) (*Expression, *Token, error) {
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}

	var err error
	var expr *Expression

	if token.Tok == TokLEFTPAREN {
		expr, token, err = parseParenthesizedExpression(db, s, token)
	} else {
		// We expect a like/equal expression
		expr, token, err = parseLikeEqualExpression(db, s, token)
	}
	if err != nil {
		return nil, nil, err
	}

	for token.Tok != TokEOF && token.Tok != TokRIGHTPAREN {
		// There is more.  If not 'and', 'or' or ')', the where is over.
		if strings.ToLower(token.Value) != "and" && strings.ToLower(token.Value) != "or" {
			return expr, token, nil
		}
		var newExpression Expression
		var operand1 Operand
		operand1.Type = TypExpr
		operand1.Expr = expr
		operand1.Off = operand1.Expr.Off
		newExpression.Operand1 = &operand1
		newExpression.Off = operand1.Off

		newExpression.Operator, token, err = parseLogicalOperator(db, s, token)
		if err != nil {
			return nil, nil, err
		}

		expr = &newExpression
		// Need to set operand2.
		var operand2 Operand
		expr.Operand2 = &operand2
		if token.Tok == TokLEFTPAREN {
			expr.Operand2.Type = TypExpr
			expr.Operand2.Expr, token, err = parseParenthesizedExpression(db, s, token)
		} else {
			expr.Operand2.Type = TypExpr
			expr.Operand2.Expr, token, err = parseLikeEqualExpression(db, s, token)
		}
		if err != nil {
			return nil, nil, err
		}
		expr.Operand2.Off = expr.Operand2.Expr.Off
	}

	return expr, token, nil
}

// Parse a binary expression.  Return expression and next token (or error)
func parseLikeEqualExpression(db ds.Database, s *scanner.Scanner, token *Token) (*Expression, *Token, error) {
	var expression Expression
	expression.Off = token.Off

	// operand 1
	var operand1 *Operand
	var err error
	operand1, token, err = parseOperand(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	// operator
	var operator *BinaryOperator
	operator, token, err = parseBinaryOperator(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	// operand 2
	var operand2 *Operand
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	operand2, token, err = parseOperand(db, s, token)
	if err != nil {
		return nil, nil, err
	}

	expression.Operand1 = operand1
	expression.Operator = operator
	expression.Operand2 = operand2

	return &expression, token, nil
}

func parseFunction(db ds.Database, s *scanner.Scanner, funcName string, funcOffset int64, token *Token) (*Function, *Token, error) {
	var function Function
	function.Name = funcName
	function.Off = funcOffset
	token = scanToken(s) // eat left paren
	for token.Tok != TokRIGHTPAREN {
		if token.Tok == TokEOF {
			return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, ")")
		}
		var arg *Operand
		var err error
		arg, token, err = parseOperand(db, s, token)
		if err != nil {
			return nil, nil, err
		}
		function.Args = append(function.Args, arg)
		// A comma or right paren is expected, but a right paren cannot come after a comma.
		if token.Tok == TokCOMMA {
			token = scanToken(s)
			if token.Tok == TokRIGHTPAREN {
				// right paren cannot come after a comma
				return nil, nil, syncql.NewErrExpectedOperand(db.GetContext(), token.Off, token.Value)
			}
		} else if token.Tok != TokRIGHTPAREN {
			return nil, nil, syncql.NewErrUnexpected(db.GetContext(), token.Off, token.Value)
		}
	}
	token = scanToken(s) // eat right paren
	return &function, token, nil
}

// Parse an operand (field or literal) and return it and the next Token (or error)
func parseOperand(db ds.Database, s *scanner.Scanner, token *Token) (*Operand, *Token, error) {
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	var operand Operand
	operand.Off = token.Off
	switch token.Tok {
	case TokIDENT:
		operand.Type = TypField
		var field Field
		field.Off = token.Off
		var segment *Segment
		var err error
		if segment, token, err = parseSegment(db, s, token); err != nil {
			return nil, nil, err
		}
		field.Segments = append(field.Segments, *segment)

		// If the next token is not a period, check for true/false/nil.
		// If true/false or nil, change this operand to a bool or nil, respectively.
		// Also, check for function call.  If so, change to a function operand.
		if token.Tok != TokPERIOD && (strings.ToLower(segment.Value) == "true" || strings.ToLower(segment.Value) == "false") {
			operand.Type = TypBool
			operand.Bool = strings.ToLower(segment.Value) == "true"
		} else if token.Tok != TokPERIOD && strings.ToLower(segment.Value) == "nil" {
			operand.Type = TypNil
		} else if token.Tok == TokLEFTPAREN {
			// Segments with a key specified cannot be function calls.
			if len(segment.Keys) != 0 {
				return nil, nil, syncql.NewErrUnexpected(db.GetContext(), token.Off, token.Value)
			}
			operand.Type = TypFunction
			var err error
			if operand.Function, token, err = parseFunction(db, s, segment.Value, segment.Off, token); err != nil {
				return nil, nil, err
			}
		} else { // This is a field (column) operand.
			// If the next token is a period, collect the rest of the segments in the column.
			for token.Tok != TokEOF && token.Tok == TokPERIOD {
				token = scanToken(s)
				if token.Tok != TokIDENT {
					return nil, nil, syncql.NewErrExpectedIdentifier(db.GetContext(), token.Off, token.Value)
				}
				if segment, token, err = parseSegment(db, s, token); err != nil {
					return nil, nil, err
				}
				field.Segments = append(field.Segments, *segment)
			}
			operand.Column = &field
		}
	case TokINT:
		operand.Type = TypInt
		i, err := strconv.ParseInt(token.Value, 0, 64)
		if err != nil {
			return nil, nil, syncql.NewErrCouldNotConvert(db.GetContext(), token.Off, token.Value, "int64")
		}
		operand.Int = i
		token = scanToken(s)
	case TokFLOAT:
		operand.Type = TypFloat
		f, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			return nil, nil, syncql.NewErrCouldNotConvert(db.GetContext(), token.Off, token.Value, "float64")
		}
		operand.Float = f
		token = scanToken(s)
	case TokCHAR:
		operand.Type = TypInt
		ch, _ := utf8.DecodeRuneInString(token.Value)
		operand.Int = int64(ch)
		token = scanToken(s)
	case TokSTRING:
		operand.Type = TypStr
		operand.Str = token.Value
		token = scanToken(s)
	case TokMINUS:
		// Could be negative int or negative float
		off := token.Off
		token = scanToken(s)
		switch token.Tok {
		case TokINT:
			operand.Type = TypInt
			i, err := strconv.ParseInt("-"+token.Value, 0, 64)
			if err != nil {
				return nil, nil, syncql.NewErrCouldNotConvert(db.GetContext(), off, "-"+token.Value, "int64")
			}
			operand.Int = i
		case TokFLOAT:
			operand.Type = TypFloat
			f, err := strconv.ParseFloat("-"+token.Value, 64)
			if err != nil {
				return nil, nil, syncql.NewErrCouldNotConvert(db.GetContext(), off, "-"+token.Value, "float64")
			}
			operand.Float = f
		default:
			return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, "int or float")
		}
		token = scanToken(s)
	case TokParameter:
		operand.Type = TypParameter
		token = scanToken(s)
	default:
		return nil, nil, syncql.NewErrExpectedOperand(db.GetContext(), token.Off, token.Value)
	}
	return &operand, token, nil
}

// Parse binary operator and return it and the next Token (or error)
func parseBinaryOperator(db ds.Database, s *scanner.Scanner, token *Token) (*BinaryOperator, *Token, error) {
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	var operator BinaryOperator
	operator.Off = token.Off
	if token.Tok == TokIDENT {
		switch strings.ToLower(token.Value) {
		case "equal":
			operator.Type = Equal
			token = scanToken(s)
		case "is":
			operator.Type = Is
			token = scanToken(s)
			// if the next token is "not", change to IsNot
			if token.Tok != TokEOF && strings.ToLower(token.Value) == "not" {
				operator.Type = IsNot
				token = scanToken(s)
			}
		case "like":
			operator.Type = Like
			token = scanToken(s)
		case "not":
			token = scanToken(s)
			if token.Tok == TokEOF || (strings.ToLower(token.Value) != "equal" && strings.ToLower(token.Value) != "like") {
				return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, "'equal' or 'like'")
			}
			switch strings.ToLower(token.Value) {
			case "equal":
				operator.Type = NotEqual
			default: //case "like":
				operator.Type = NotLike
			}
			token = scanToken(s)
		default:
			return nil, nil, syncql.NewErrExpectedOperator(db.GetContext(), token.Off, token.Value)
		}
	} else {
		switch token.Tok {
		case TokEQUAL:
			operator.Type = Equal
			token = scanToken(s)
		case TokLEFTANGLEBRACKET:
			// Can be '<', '<=', '<>'.
			token = scanToken(s)
			switch token.Tok {
			case TokRIGHTANGLEBRACKET:
				operator.Type = NotEqual
				token = scanToken(s)
			case TokEQUAL:
				operator.Type = LessThanOrEqual
				token = scanToken(s)
			default:
				operator.Type = LessThan
			}
		case TokRIGHTANGLEBRACKET:
			// Can be '>', '>='
			token = scanToken(s)
			switch token.Tok {
			case TokEQUAL:
				operator.Type = GreaterThanOrEqual
				token = scanToken(s)
			default:
				operator.Type = GreaterThan
			}
		default:
			return nil, nil, syncql.NewErrExpectedOperator(db.GetContext(), token.Off, token.Value)
		}
	}

	return &operator, token, nil
}

// Parse logical operator and return it and the next Token (or error)
func parseLogicalOperator(db ds.Database, s *scanner.Scanner, token *Token) (*BinaryOperator, *Token, error) {
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	var operator BinaryOperator
	operator.Off = token.Off
	switch strings.ToLower(token.Value) {
	case "and":
		operator.Type = And
	case "or":
		operator.Type = Or
	default:
		return nil, nil, syncql.NewErrExpectedOperator(db.GetContext(), token.Off, token.Value)
	}

	token = scanToken(s)
	return &operator, token, nil
}

// Parse and return EscapeClause, LimitClause and ResultsOffsetClause (any or all can be nil) and next token (or error)
func parseEscapeLimitResultsOffsetClauses(db ds.Database, s *scanner.Scanner, token *Token) (*EscapeClause, *LimitClause, *ResultsOffsetClause, *Token, error) {
	var err error
	var ec *EscapeClause
	var lc *LimitClause
	var oc *ResultsOffsetClause
	for token.Tok != TokEOF {
		// Note: Can be in any order.  If more than one, the last one wins
		if token.Tok == TokIDENT && strings.ToLower(token.Value) == "escape" {
			ec, token, err = parseEscapeClause(db, s, token)
		} else if token.Tok == TokIDENT && strings.ToLower(token.Value) == "limit" {
			lc, token, err = parseLimitClause(db, s, token)
		} else if token.Tok == TokIDENT && strings.ToLower(token.Value) == "offset" {
			oc, token, err = parseResultsOffsetClause(db, s, token)
		} else {
			return ec, lc, oc, token, nil
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	return ec, lc, oc, token, nil
}

// Parse and return the EscapeClause and LimitClause (any or all can be nil) and next token (or error)
func parseEscapeLimitClauses(db ds.Database, s *scanner.Scanner, token *Token) (*EscapeClause, *LimitClause, *Token, error) {
	var err error
	var ec *EscapeClause
	var lc *LimitClause
	for token.Tok != TokEOF {
		// Note: Can be in any order.  If more than one, the last one wins
		if token.Tok == TokIDENT && strings.ToLower(token.Value) == "escape" {
			ec, token, err = parseEscapeClause(db, s, token)
		} else if token.Tok == TokIDENT && strings.ToLower(token.Value) == "limit" {
			lc, token, err = parseLimitClause(db, s, token)
		} else {
			return ec, lc, token, nil
		}
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return ec, lc, token, nil
}

// Parse the escape clause.  Return the EscapeClause and the next Token (or error).
func parseEscapeClause(db ds.Database, s *scanner.Scanner, token *Token) (*EscapeClause, *Token, error) {
	var ec EscapeClause
	ec.Off = token.Off
	token = scanToken(s)
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	if token.Tok != TokCHAR {
		return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, "char literal")
	}
	var v CharValue
	v.Off = token.Off
	v.Value, _ = utf8.DecodeRuneInString(token.Value)
	ec.EscapeChar = &v
	token = scanToken(s)
	return &ec, token, nil
}

// Parse the limit clause.  Return the LimitClause and the next Token (or error).
func parseLimitClause(db ds.Database, s *scanner.Scanner, token *Token) (*LimitClause, *Token, error) {
	var lc LimitClause
	lc.Off = token.Off
	token = scanToken(s)
	var err error
	lc.Limit, token, err = parseNonNegInt64(db, s, token)
	if err != nil {
		return nil, nil, err
	}
	return &lc, token, nil
}

// Parse the results offset clause.  Return the ResultsOffsetClause and the next Token (or error).
func parseResultsOffsetClause(db ds.Database, s *scanner.Scanner, token *Token) (*ResultsOffsetClause, *Token, error) {
	var oc ResultsOffsetClause
	oc.Off = token.Off
	token = scanToken(s)
	var err error
	oc.ResultsOffset, token, err = parseNonNegInt64(db, s, token)
	if err != nil {
		return nil, nil, err
	}
	return &oc, token, nil
}

// Parse and return an Int64Value and next token (or error).
// This function is called by parseLimitClause and parseResultsOffsetClause.  The integer
// values for both of these clauses cannot be negative.
func parseNonNegInt64(db ds.Database, s *scanner.Scanner, token *Token) (*Int64Value, *Token, error) {
	// We expect an integer literal
	// Since we're looking for integers >= 0, don't allow TokMINUS.
	if token.Tok == TokEOF {
		return nil, nil, syncql.NewErrUnexpectedEndOfStatement(db.GetContext(), token.Off)
	}
	if token.Tok != TokINT {
		return nil, nil, syncql.NewErrExpected(db.GetContext(), token.Off, "positive integer literal")
	}
	var v Int64Value
	v.Off = token.Off
	var err error
	v.Value, err = strconv.ParseInt(token.Value, 0, 64)
	if err != nil {
		// The token value has already been checked, so this can't happen.
		return nil, nil, syncql.NewErrCouldNotConvert(db.GetContext(), token.Off, token.Value, "int64")
	}
	token = scanToken(s)
	return &v, token, nil
}

func (st SelectStatement) Offset() int64 {
	return st.Off
}

func (st DeleteStatement) Offset() int64 {
	return st.Off
}

// Pretty string of select statement.
func (st SelectStatement) String() string {
	val := fmt.Sprintf("Off(%d):", st.Off)
	if st.Select != nil {
		val += st.Select.String()
	}
	if st.From != nil {
		val += " " + st.From.String()
	}
	if st.Where != nil {
		val += " " + st.Where.String()
	}
	if st.Escape != nil {
		val += " " + st.Escape.String()
	}
	if st.Limit != nil {
		val += " " + st.Limit.String()
	}
	if st.ResultsOffset != nil {
		val += " " + st.ResultsOffset.String()
	}
	return val
}

// Pretty string of delete statement.
func (st DeleteStatement) String() string {
	val := fmt.Sprintf("Off(%d):", st.Off)
	val += "DELETE"
	if st.From != nil {
		val += " " + st.From.String()
	}
	if st.Where != nil {
		val += " " + st.Where.String()
	}
	if st.Escape != nil {
		val += " " + st.Escape.String()
	}
	if st.Limit != nil {
		val += " " + st.Limit.String()
	}
	return val
}

func (st SelectStatement) CopyAndSubstitute(db ds.Database, paramValues []*vdl.Value) (Statement, error) {
	var copy SelectStatement
	copy.Off = st.Off
	copy.Select = st.Select
	copy.From = st.From
	if st.Where != nil {
		var err error
		if copy.Where, err = st.Where.CopyAndSubstitute(db, paramValues); err != nil {
			return nil, err
		}
	} else {
		// There is no where clause.  If any paramValues suppied, we have too many.
		if len(paramValues) > 0 {
			return nil, syncql.NewErrTooManyParamValuesSpecified(db.GetContext(), copy.Off)
		}
	}
	copy.Escape = st.Escape
	copy.Limit = st.Limit
	copy.ResultsOffset = st.ResultsOffset
	return copy, nil
}

func (st DeleteStatement) CopyAndSubstitute(db ds.Database, paramValues []*vdl.Value) (Statement, error) {
	var copy DeleteStatement
	copy.Off = st.Off
	copy.From = st.From
	if st.Where != nil {
		var err error
		if copy.Where, err = st.Where.CopyAndSubstitute(db, paramValues); err != nil {
			return nil, err
		}
	} else {
		// There is no where clause.  If any paramValues suppied, we have too many.
		if len(paramValues) > 0 {
			return nil, syncql.NewErrTooManyParamValuesSpecified(db.GetContext(), copy.Off)
		}
	}
	copy.Escape = st.Escape
	copy.Limit = st.Limit
	return copy, nil
}

func (sel SelectClause) String() string {
	val := fmt.Sprintf(" Off(%d):SELECT Columns(", sel.Off)
	sep := ""
	for _, selector := range sel.Selectors {
		val += sep + selector.String()
		sep = ","
	}
	val += ")"
	return val
}

func (s Selector) String() string {
	val := fmt.Sprintf(" Off(%d):", s.Off)
	switch s.Type {
	case TypSelField:
		val += s.Field.String()
	case TypSelFunc:
		val += s.Function.String()
	}
	if s.As != nil {
		val += s.As.String()
	}
	return val
}

func (a AsClause) String() string {
	val := fmt.Sprintf(" Off(%d):", a.Off)
	val += a.AltName.String()
	return val
}

func (n Name) String() string {
	val := fmt.Sprintf(" Off(%d):", n.Off)
	val += n.Value
	return val
}

func (f Field) String() string {
	val := fmt.Sprintf(" Off(%d):", f.Off)
	for i := range f.Segments {
		if i != 0 {
			val += "."
		}
		val += f.Segments[i].String()
	}
	return val
}

func (f Function) String() string {
	val := fmt.Sprintf("Off(%d):", f.Off)
	val += f.Name
	val += "("
	sep := ""
	for _, a := range f.Args {
		val += sep + a.String()
		sep = ","
	}
	val += ")"
	return val
}

func (s Segment) String() string {
	val := fmt.Sprintf(" Off(%d):%s", s.Off, s.Value)
	for _, k := range s.Keys {
		val += "[" + k.String() + "]"
	}
	return val
}

func (f FromClause) String() string {
	return fmt.Sprintf("Off(%d):FROM %s", f.Off, f.Table.String())
}

func (t TableEntry) String() string {
	return fmt.Sprintf("Off(%d):%s", t.Off, t.Name)
}

func (w WhereClause) String() string {
	return fmt.Sprintf(" Off(%d):WHERE %s", w.Off, w.Expr.String())
}

func (e EscapeClause) String() string {
	return fmt.Sprintf(" Off(%d):ESCAPE %s", e.Off, e.EscapeChar.String())
}

func (i Int64Value) String() string {
	return fmt.Sprintf(" Off(%d): %d", i.Off, i.Value)
}

func (c CharValue) String() string {
	return fmt.Sprintf(" Off(%d): %c", c.Off, c.Value)
}

func (l LimitClause) String() string {
	return fmt.Sprintf(" Off(%d):LIMIT %s", l.Off, l.Limit.String())
}

func (l ResultsOffsetClause) String() string {
	return fmt.Sprintf(" Off(%d):OFFSET %s", l.Off, l.ResultsOffset.String())
}

func (o Operand) String() string {
	val := fmt.Sprintf("Off(%d):", o.Off)
	switch o.Type {
	case TypBigInt:
		val += "(BigInt)"
		val += o.BigInt.String()
	case TypBigRat:
		val += "(BigRat)"
		val += o.BigRat.String()
	case TypField:
		val += "(field)"
		val += o.Column.String()
	case TypBool:
		val += "(bool)"
		val += strconv.FormatBool(o.Bool)
	case TypInt:
		val += "(int)"
		val += strconv.FormatInt(o.Int, 10)
	case TypFloat:
		val += "(float)"
		val += strconv.FormatFloat(o.Float, 'f', -1, 64)
	case TypFunction:
		val += "(function)"
		val += o.Function.String()
	case TypStr:
		val += "(string)"
		val += o.Str
	case TypExpr:
		val += "(expr)"
		val += o.Expr.String()
	case TypTime:
		val += "(time)"
		val += o.Time.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
	case TypNil:
		val += "<nil>"
	case TypObject:
		val += "(object)"
		val += fmt.Sprintf("%v", o.Object)
	case TypParameter:
		val += "?"
	default:
		val += "<operand-type-undefined>"

	}
	return val
}

func (o BinaryOperator) String() string {
	val := fmt.Sprintf("Off(%d):", o.Off)
	switch o.Type {
	case And:
		val += "AND"
	case Equal:
		val += "="
	case GreaterThan:
		val += ">"
	case GreaterThanOrEqual:
		val += ">="
	case Is:
		val += "IS"
	case IsNot:
		val += "IS NOT"
	case LessThan:
		val += "<"
	case LessThanOrEqual:
		val += "<="
	case Like:
		val += "LIKE"
	case NotEqual:
		val += "<>"
	case NotLike:
		val += "NOT LIKE"
	case Or:
		val += "OR"
	default:
		val += "<operator-undefined>"
	}
	return val
}

func (e Expression) String() string {
	return fmt.Sprintf("(Off(%d):%s %s %s)", e.Off, e.Operand1.String(), e.Operator.String(), e.Operand2.String())
}

// paramInfo is used to keep track of supplied values for parameter markers.
type paramInfo struct {
	paramValues []*vdl.Value
	cursor      int // Index into paramValues pointing to next value to substitute.
}

func (w WhereClause) CopyAndSubstitute(db ds.Database, paramValues []*vdl.Value) (*WhereClause, error) {
	var copy WhereClause
	copy.Off = w.Off
	var err error
	pi := paramInfo{paramValues: paramValues, cursor: 0}

	if copy.Expr, err = w.Expr.CopyAndSubstitute(db, &pi); err != nil {
		return nil, err
	}

	// Did any of the supplied values go unused?
	if pi.cursor < len(paramValues) {
		return nil, syncql.NewErrTooManyParamValuesSpecified(db.GetContext(), w.Off)
	}

	return &copy, nil
}

func (e Expression) CopyAndSubstitute(db ds.Database, pi *paramInfo) (*Expression, error) {
	var copy Expression
	copy.Off = e.Off
	copy.Operator = e.Operator
	var err error
	if copy.Operand1, err = e.Operand1.CopyAndSubstitute(db, pi); err != nil {
		return nil, err
	}
	if copy.Operand2, err = e.Operand2.CopyAndSubstitute(db, pi); err != nil {
		return nil, err
	}
	return &copy, nil
}

func (o Operand) CopyAndSubstitute(db ds.Database, pi *paramInfo) (*Operand, error) {
	switch o.Type {
	case TypExpr:
		var copy Operand
		copy.Type = TypExpr
		copy.Off = o.Off
		var err error
		if copy.Expr, err = o.Expr.CopyAndSubstitute(db, pi); err != nil {
			return nil, err
		}
		return &copy, nil
	case TypFunction:
		var copy Operand
		copy.Type = TypFunction
		copy.Off = o.Off
		var err error
		if copy.Function, err = o.Function.CopyAndSubstitute(db, pi); err != nil {
			return nil, err
		}
		return &copy, nil
	case TypParameter:
		if pi.cursor >= len(pi.paramValues) {
			// not enough paramater values specified
			return nil, syncql.NewErrNotEnoughParamValuesSpecified(db.GetContext(), o.Off)
		}
		if cpOp, err := ConvertValueToAnOperand(pi.paramValues[pi.cursor], o.Off); err == nil {
			pi.cursor++
			return cpOp, nil
		} else {
			return nil, err
		}
	default:
		// No need to copy the operand.
		return &o, nil
	}
}

func (f Function) CopyAndSubstitute(db ds.Database, pi *paramInfo) (*Function, error) {
	var copy Function
	copy.Name = f.Name
	copy.Off = f.Off

	for _, a := range f.Args {
		if newArg, err := a.CopyAndSubstitute(db, pi); err == nil {
			copy.Args = append(copy.Args, newArg)
		} else {
			return nil, err
		}
	}

	copy.RetType = f.RetType
	copy.Computed = f.Computed
	if copy.Computed {
		var err error
		if copy.RetValue, err = f.RetValue.CopyAndSubstitute(db, pi); err != nil {
			return nil, err
		}
	}
	return &copy, nil
}

func ConvertValueToAnOperand(value *vdl.Value, off int64) (*Operand, error) {
	var op Operand
	op.Off = off

	switch value.Kind() {
	case vdl.Bool:
		op.Type = TypBool
		op.Bool = value.Bool()
	case vdl.Enum:
		op.Type = TypStr
		op.Str = value.EnumLabel()
	case vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64:
		op.Type = TypInt
		op.Int = value.Int()
	case vdl.Byte, vdl.Uint16, vdl.Uint32, vdl.Uint64:
		op.Type = TypInt
		op.Int = int64(value.Uint())
	case vdl.Float32, vdl.Float64:
		op.Type = TypFloat
		op.Float = value.Float()
	case vdl.String:
		op.Type = TypStr
		op.Str = value.RawString()
	default: // OpObject for structs, arrays, maps, ...
		if value.Kind() == vdl.Struct && value.Type().Name() == "time.Time" {
			op.Type = TypTime
			if err := vdl.Convert(&op.Time, value); err != nil {
				return nil, err
			}
		} else {
			op.Type = TypObject
			op.Object = value
		}
	}
	return &op, nil
}

// ParseIndexField is used to parse datasource supplied index fields.  It creates a new
// scanner with the contents of the field name and only succeeds if the result of
// parsing the contents of the scan is a field and there is nothing left over.
// Note: This function is NOT involved in the parsing of the AST.  Offsets of 0 are
//       returned on error as these errors are unrelated to the input query.  They
//       are configuration errors in the datasource.
func ParseIndexField(db ds.Database, fieldName, tableName string) (*Field, error) {
	// Set up a scanner and call the parser's parseOperand function.
	r := strings.NewReader(fieldName)
	var s scanner.Scanner
	s.Init(r)
	s.Error = scannerError

	token := scanToken(&s)
	if token.Tok == TokEOF {
		return nil, syncql.NewErrInvalidIndexField(db.GetContext(), 0, fieldName, tableName)
	}
	var op *Operand
	var err error
	op, token, err = parseOperand(db, &s, token)
	if err != nil {
		return nil, syncql.NewErrInvalidIndexField(db.GetContext(), 0, fieldName, tableName)
	}
	if op.Type != TypField {
		return nil, syncql.NewErrInvalidIndexField(db.GetContext(), 0, fieldName, tableName)
	}
	if token.Tok != TokEOF {
		return nil, syncql.NewErrInvalidIndexField(db.GetContext(), 0, fieldName, tableName)
	}
	// Look at last segment.  If a key or index is supplied, it can't be used as an index.
	if len(op.Column.Segments[len(op.Column.Segments)-1].Keys) != 0 {
		return nil, syncql.NewErrInvalidIndexField(db.GetContext(), 0, fieldName, tableName)
	}

	return op.Column, nil
}
