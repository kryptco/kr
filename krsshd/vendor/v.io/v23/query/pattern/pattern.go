// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pattern handles parsing and matching SQL LIKE-style glob patterns.
package pattern

import (
	"bytes"
	"fmt"
	"regexp"

	"v.io/v23/verror"
)

const (
	DefaultEscapeChar = '\\'
)

// Pattern is a parsed LIKE-style glob pattern.
type Pattern struct {
	// regular expression equivalent to the original like pattern
	regex *regexp.Regexp
	// fixed prefix that all pattern matches must start with
	fixedPrefix string
	// true if pattern contains no unescaped wildcards; in this case, fixedPrefix
	// is the entire unescaped expression
	noWildcards bool
}

// Parse parses a LIKE-style glob pattern assuming '\' as escape character.
// See ParseWithEscapeChar().
func Parse(pattern string) (*Pattern, error) {
	return ParseWithEscapeChar(pattern, DefaultEscapeChar)
}

// ParseWithEscapeChar parses a LIKE-style glob pattern.
// Supported wildcards are '_' (match any one character) and '%' (match zero or
// more characters). They can be escaped by escChar; escChar can also escape
// itself. '_' and '%' cannot be used as escChar; '\x00' escChar disables
// escaping.
func ParseWithEscapeChar(pattern string, escChar rune) (*Pattern, error) {
	if escChar == '%' || escChar == '_' {
		return nil, NewErrIllegalEscapeChar(nil)
	}

	// The LIKE-style pattern is converted to a regex, converting:
	// % to .*?
	// _ to .
	// Everything else that would be incorrectly interpreted as a regex is escaped.
	// The approach this function takes is to collect characters to be escaped
	// into toBeEscapedBuf. When a wildcard is encountered, first toBeEscapedBuf
	// is escaped and written to the regex buffer, next the wildcard is translated
	// to regex (either ".*?" or ".") and written to the regex buffer.
	// At the end, any remaining chars in toBeEscapedBuf are written.
	var buf bytes.Buffer            // buffer for return regex
	var toBeEscapedBuf bytes.Buffer // buffer to hold characters waiting to be escaped
	// Even though regexp.Regexp provides a LiteralPrefix() method, it doesn't
	// always return the longest fixed prefix, so we save it while parsing.
	var fixedPrefix string
	foundWildcard := false

	buf.WriteString("^") // '^<regex_str>$'
	escapedMode := false
	for _, c := range pattern {
		if escapedMode {
			switch c {
			case '%', '_', escChar:
				toBeEscapedBuf.WriteRune(c)
			default:
				return nil, NewErrInvalidEscape(nil, string(c))
			}
			escapedMode = false
		} else {
			switch c {
			case '%', '_':
				// Write out any chars waiting to be escaped, then write ".*?' or '.'.
				buf.WriteString(regexp.QuoteMeta(toBeEscapedBuf.String()))
				if !foundWildcard {
					// First wildcard found, fixedPrefix is the pattern up to it.
					fixedPrefix = toBeEscapedBuf.String()
					foundWildcard = true
				}
				toBeEscapedBuf.Reset()
				if c == '%' {
					buf.WriteString(".*?")
				} else {
					buf.WriteString(".")
				}
			case escChar:
				if escChar != '\x00' {
					escapedMode = true
				} else {
					// nul is never an escape char, treat same as default.
					toBeEscapedBuf.WriteRune(c)
				}
			default:
				toBeEscapedBuf.WriteRune(c)
			}
		}
	}
	if escapedMode {
		return nil, NewErrInvalidEscape(nil, "<end>")
	}
	// Write any remaining chars in toBeEscapedBuf.
	buf.WriteString(regexp.QuoteMeta(toBeEscapedBuf.String()))
	if !foundWildcard {
		// No wildcard found, fixedPrefix is the entire pattern.
		fixedPrefix = toBeEscapedBuf.String()
	}
	buf.WriteString("$") // '^<regex_str>$'

	regex := buf.String()
	compRegex, err := regexp.Compile(regex)
	if err != nil {
		// TODO(ivanpi): Should never happen. Panic here?
		return nil, verror.New(verror.ErrInternal, nil, fmt.Sprintf("failed to compile pattern %q (regular expression %q): %v", pattern, regex, err))
	}
	return &Pattern{
		regex:       compRegex,
		fixedPrefix: fixedPrefix,
		noWildcards: !foundWildcard,
	}, nil
}

// MatchString returns true iff the pattern matches the entire string.
func (p *Pattern) MatchString(s string) bool {
	return p.regex.MatchString(s)
}

// FixedPrefix returns the unescaped fixed prefix that all matching strings must
// start with, and whether the prefix is the whole pattern.
func (p *Pattern) FixedPrefix() (string, bool) {
	return p.fixedPrefix, p.noWildcards
}

// Escape escapes a literal string for inclusion in a LIKE-style pattern
// assuming '\' as escape character.
// See EscapeWithEscapeChar().
func Escape(s string) string {
	return EscapeWithEscapeChar(s, DefaultEscapeChar)
}

// EscapeWithEscapeChar escapes a literal string for inclusion in a LIKE-style
// pattern. It inserts escChar before each '_', '%', and escChar in the string.
func EscapeWithEscapeChar(s string, escChar rune) string {
	if escChar == '\x00' {
		panic(verror.New(verror.ErrBadArg, nil, "'\x00' disables escaping, cannot be used in EscapeWithEscapeChar"))
	}
	if escChar == '%' || escChar == '_' {
		panic(NewErrIllegalEscapeChar(nil))
	}
	var buf bytes.Buffer
	for _, c := range s {
		if c == '%' || c == '_' || c == escChar {
			buf.WriteRune(escChar)
		}
		buf.WriteRune(c)
	}
	return buf.String()
}
