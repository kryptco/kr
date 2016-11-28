// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package i18n implements internationalization of formatted message strings in
// different languages.
//
// Typical usage:
//   cat := i18n.Cat()   // get default Catalogue
//   outputString = cat.Format(language, msgID, "1st", "2nd", "3rd", "4th")
//
// i18n.Catalogue maps language names and message identifiers to message format
// strings.  The intent is to provide a primitive form of Sprintf(), where the
// format string can depend upon the language.
//
// i18n.MsgID is a string that identitifies a set of message format strings that
// have the same meaning, but may be available in multiple languages.
//
// i18n.Lang is a string that identifies a language.
//
// A message format string is a string containing substrings of the form
// {<number>} which are replaced by the corresponding position parameter
// (numbered from 1), or {_}, which is replaced by all otherwise unused
// parameters.  If a substring is of the form {:<number>}, {<number>:},
// {:<number>:}, {:_}, {_:}, or {:_:}, and the corresponding parameters are not
// the empty string, the parameter is preceded by ": " or followed by ":" or
// both, respectively.  For example, if the format:
//   {3:} foo {2} bar{:_} ({3})
// is used with the cat.Format example above, it yields:
//   3rd: foo 2nd bar: 1st 4th (3rd)
//
// The positional parameters may have any type, and are printed in their default
// formatting.  If particular formatting is desired, the parameter should be
// converted to a string first.  In principle, the default formating for a
// parameter may depend on LangID.
package i18n

import "bufio"
import "fmt"
import "io"
import "os"
import "strconv"
import "strings"
import "sync"
import "v.io/v23/context"

// MsgID identifies a message, without specifying its language.
type MsgID string

// LangID represents the name of a language or locale.
// By convention it should be an IETF language tag:
//   http://en.wikipedia.org/wiki/IETF_language_tag
type LangID string

// NoLangID is the empty LangID.
const NoLangID LangID = ""

// Catalogue maps (LangID, MsgID) pairs to message format strings.
type Catalogue struct {
	lock    sync.RWMutex // Protects remaining fields.
	formats map[LangID]map[MsgID]string
}

// *defaultCatalogue is the default Catalogue of the process.
// It is initialized via oneTimeInit in Cat().
var (
	defaultCatalogue *Catalogue
	oneTimeInit      sync.Once
)

// Cat returns the default Catalogue.
func Cat() (result *Catalogue) {
	oneTimeInit.Do(func() { defaultCatalogue = new(Catalogue) })
	return defaultCatalogue
}

// Format applies FormatParams to the result of Lookup(langID, msgId) and the
// parameters v.  If Lookup fails, the result is the text of the MsgID, and if
// there are any positional parameters, a colon followed by those parameters.
func (cat *Catalogue) Format(langID LangID, msgID MsgID, v ...interface{}) string {
	formatStr := cat.Lookup(langID, msgID)
	if formatStr == "" {
		formatStr = string(msgID)
		if len(v) != 0 {
			formatStr += "{:_}"
		}
	}
	return FormatParams(formatStr, v...)
}

// langIDKey is used as a key for context.T's Value() map.
type langIDKey struct{}

// GetLangID returns the LangID associated with a context.T,
// or the empty LangID if there is none.
func GetLangID(ctx *context.T) (langID LangID) {
	if ctx != nil {
		v := ctx.Value(langIDKey{})
		langID, _ = v.(LangID)
	}
	return langID
}

// WithLangID returns a context based on ctx that has the
// language ID langID.
func WithLangID(ctx *context.T, langID LangID) *context.T {
	return context.WithValue(ctx, langIDKey{}, langID)
}

// Lookup returns the format corresponding to a particular language and MsgID.
// If no such message is known, any message for BaseLangID(langID) is
// retrievied.  If no such message exists, empty string is returned.
func (cat *Catalogue) Lookup(langID LangID, msgID MsgID) (result string) {
	cat.lock.RLock()
	result = cat.formats[langID][msgID]
	if result == "" {
		result = cat.formats[BaseLangID(langID)][msgID]
	}
	cat.lock.RUnlock()
	return result
}

// FormatParams returns a copy of format with instances of "{1}", "{2}", ...
// replaced by the default string representation of v[0], v[1], ...
// The last instance of the string "{_}" is replaced with a space-separated
// list of positional parameters unused by other {...} sequences.
// Missing parameters are replaced with "?".
func FormatParams(formatStr string, v ...interface{}) (result string) {
	prefix := ""                   // The text before {_}, if any.
	underbar := false              // Whether {_} appears in formatStr.
	underbarLeadingColon := false  // true if {:_}
	underbarTrailingColon := false // true if {_:}
	used := make([]bool, len(v))   // used[i] indicates whether v[i] has been used.
	for i := 0; i != len(formatStr); {
		if braceIndex := skipNotIn(formatStr, i, "{"); braceIndex == len(formatStr) {
			// No more positional parameters.
			result += formatStr[i:]
			i = len(formatStr)
		} else {
			digitsIndex := braceIndex + 1
			leadingColon := (digitsIndex < len(formatStr) && formatStr[digitsIndex] == ':')
			if leadingColon {
				digitsIndex++
			}
			if strings.HasPrefix(formatStr[digitsIndex:], "_}") || strings.HasPrefix(formatStr[digitsIndex:], "_:}") {
				underbar = true
				underbarLeadingColon = leadingColon
				prefix += result + formatStr[i:braceIndex]
				result = ""
				underbarTrailingColon = false
				i = digitsIndex + 2
				if formatStr[digitsIndex+1] == ':' {
					underbarTrailingColon = true
					i++
				}
			} else if endIndex := skipIn(formatStr, digitsIndex, "0123456789"); endIndex != len(formatStr) &&
				endIndex != digitsIndex && (formatStr[endIndex] == '}' || strings.HasPrefix(formatStr[endIndex:], ":}")) {

				// Well-formed {digits}.
				n, _ := strconv.Atoi(formatStr[digitsIndex:endIndex])
				formattedParameter := "?" // Used if no such positional parmeter.
				if 1 <= n && n < len(v)+1 {
					formattedParameter = fmt.Sprint(v[n-1])
					used[n-1] = true
				}
				result += formatStr[i:braceIndex]
				if leadingColon && formattedParameter != "" {
					result += ": "
				}
				result += formattedParameter
				i = endIndex + 1
				if formatStr[endIndex] == ':' {
					if formattedParameter != "" {
						result += ":"
					}
					i++
				}
			} else { // No digits, or no '}'; add the '{' to result.
				result += formatStr[i : braceIndex+1]
				i = braceIndex + 1
			}
		}
	}
	if underbar { // insert unused parameters
		first := true
		paramStr := ""
		for i := 0; i != len(v); i++ {
			if !used[i] {
				if !first {
					paramStr += " "
				}
				first = false
				paramStr += fmt.Sprint(v[i])
			}
		}
		if paramStr != "" {
			if underbarLeadingColon {
				paramStr = ": " + paramStr
			}
			if underbarTrailingColon {
				paramStr += ":"
			}
		}
		result = prefix + paramStr + result
	}
	return result
}

// setUnlocked is like Set(), but does not acquire locks.
func (cat *Catalogue) setUnlocked(langID LangID, msgID MsgID, newFormat string) (oldFormat string) {
	idToFmt := cat.formats[langID]
	if idToFmt == nil && newFormat != "" {
		if cat.formats == nil {
			cat.formats = make(map[LangID]map[MsgID]string)
		}
		idToFmt = make(map[MsgID]string)
		cat.formats[langID] = idToFmt
	}
	oldFormat = idToFmt[msgID]
	if newFormat != "" {
		idToFmt[msgID] = newFormat
	} else {
		delete(idToFmt, msgID)
		if len(idToFmt) == 0 {
			delete(cat.formats, langID)
		}
	}
	return oldFormat
}

// Set sets the format corresponding to msgID in the specified language to
// formatStr.  If formatStr is empty, the corresponding entry is removed.  Any
// previous string is returned.
func (cat *Catalogue) Set(langID LangID, msgID MsgID, newFormat string) (oldFormat string) {
	cat.lock.Lock()
	oldFormat = cat.setUnlocked(langID, msgID, newFormat)
	cat.lock.Unlock()
	return oldFormat
}

// SetWithBase is like Set, but if newFormat != "", also sets the message for
// the base language ID if not already set.  Equivalent to:
//     baseLangID := BaseLangID(langID)
//     if newFormat != "" && baseLangID != langID && cat.Lookup(baseLangID, msgID) == "" {
//         cat.Set(baseLangID, msgID, newFormat)
//     }
//     return cat.Set(langID, msgID, newFormat)
func (cat *Catalogue) SetWithBase(langID LangID, msgID MsgID, newFormat string) (oldFormat string) {
	cat.lock.Lock()
	oldFormat = cat.setUnlocked(langID, msgID, newFormat)
	baseLangID := BaseLangID(langID)
	if newFormat != "" && baseLangID != langID && cat.formats[baseLangID][msgID] == "" {
		cat.setUnlocked(baseLangID, msgID, newFormat)
	}
	cat.lock.Unlock()
	return oldFormat
}

// skipIn returns the highest i where each byte in s[pos..i) exists and is in set.
func skipIn(s string, pos int, set string) int {
	for ; 0 <= pos && pos < len(s) && strings.IndexByte(set, s[pos]) != -1; pos++ {
	}
	return pos
}

// skipNotIn returns the highest i where each byte in s[pos..i) exists and is not in set.
func skipNotIn(s string, pos int, set string) int {
	for ; 0 <= pos && pos < len(s) && strings.IndexByte(set, s[pos]) == -1; pos++ {
	}
	return pos
}

// Merge merges the data in the lines from *r reader into *cat.
// Each line from *r is parsed with Scanf("%s %s %q"); that is,
// the first two fields are whitespace separated, and the third is quoted and escaped.
// If a line starts with a #, or cannot be parsed, the line is ignored.
// If the line contains at least three non-discarded fields, the first field is
// treated as LangID, the second as a i18n.MsgID, and the third as a format
// string in the specified language.
func (cat *Catalogue) Merge(r io.Reader) error {
	bufReader := bufio.NewReader(r)
	lineStr, err := bufReader.ReadString('\n')
	for len(lineStr) != 0 {
		var langID LangID
		var msgID MsgID
		var formatStr string
		var fields int
		fields, err = fmt.Sscanf(lineStr, "%s %s %q", &langID, &msgID, &formatStr)
		if fields == 3 && !strings.HasPrefix(string(langID), "#") {
			cat.SetWithBase(langID, msgID, formatStr)
		}
		lineStr, err = bufReader.ReadString('\n')
	}
	if err == io.EOF { // EOF is expected
		err = nil
	}
	return err
}

// MergeFromFile calls Merge() on the contents of the named file.
func (cat *Catalogue) MergeFromFile(filename string) (err error) {
	var f *os.File
	if f, err = os.Open(filename); err == nil {
		err = cat.Merge(f)
		f.Close()
	}
	return err
}

// Output emits the contents of *cat to *w in the format expected by Merge().
func (cat *Catalogue) Output(w io.Writer) error {
	cat.lock.RLock()
	defer cat.lock.RUnlock()
	for langID, idToFmt := range cat.formats {
		for msgID, formatStr := range idToFmt {
			_, err := fmt.Fprintf(w, "%s %s %q\n", langID, msgID, formatStr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// NormalizeLangID normalizes a LangID.  Currently, the only normalization
// performed is to translate underbars into hyphens.
func NormalizeLangID(langID string) LangID {
	result := ""
	for _, ch := range langID {
		if ch == '_' {
			ch = '-'
		}
		result += string(ch)
	}
	return LangID(result)
}

// BaseLangID returns a base language identifier.  It is the first hyphen-separated
// segment of an IETF Language ID.
func BaseLangID(langID LangID) LangID {
	return langID[:skipNotIn(string(langID), 0, "-")]
}

// LangIDFromEnv returns a language ID for messages based on the programme's
// environment variables.  This is suitable only for code not running in the
// context of an RPC; code in an RPC context should use language information
// from the RPC context.
func LangIDFromEnv() LangID {
	// The order of precedence of these environment variables is taken from
	// the POSIX definitions in IEEE Std 1003.1-2001.
	langID := os.Getenv("LC_ALL")
	if langID == "" {
		langID = os.Getenv("LC_MESSAGES")
	}
	if langID == "" {
		langID = os.Getenv("LANG")
	}
	if langID == "C" || langID == "" {
		langID = "en-US"
	}
	return NormalizeLangID(langID)
}
