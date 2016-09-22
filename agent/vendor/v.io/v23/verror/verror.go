// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package verror implements an error reporting mechanism that works across
// programming environments, and a set of common errors.
//
// Each error has an identifier string, which is used for equality checks.
// E.g. a Javascript client can check if a Go server returned a NoExist error by
// checking the string identifier.  Error identifier strings start with the VDL
// package path to ensure uniqueness, e.g. "v.io/v23/verror.NoExist".
//
// Each error contains an action, which is the suggested action for a typical
// client to perform upon receiving the error.  E.g. some action codes represent
// whether to retry the operation after receiving the error.
//
// Each error also contains a list of typed parameters, and an error message.
// The error message is created by looking up a format string keyed on the error
// identifier, and applying the parameters to the format string.  This enables
// error messages to be generated in different languages.
//
// Examples
//
// To define a new error identifier, for example "someNewError", client code is
// expected to declare a variable like this:
//      var someNewError = verror.Register("my/package/name.someNewError", NoRetry,
//                                         "{1} {2} English text for new error")
// Text for other languages can be added to the default i18n Catalogue.
//
// If the error should cause a client to retry, consider replacing "NoRetry" with
// one of the other Action codes below.
//
// Errors are given parameters when used.  Conventionally, the first parameter
// is the name of the component (typically server or binary name), and the
// second is the name of the operation (such as an RPC or subcommand) that
// encountered the error.  Other parameters typically identify the object(s) on
// which the error occurred.  This convention is normally applied by New(),
// which fetches the language, component name and operation name from the
// context.T:
//      err = verror.New(someNewError, ctx, "object_on_which_error_occurred")
//
// The ExplicitNew() call can be used to specify these things explicitly:
//      err = verror.ExplicitNew(someNewError, i18n.GetLangID(ctx),
//              "my_component", "op_name", "procedure_name", "object_name")
// If the language, component and/or operation name are unknown, use i18n.NoLangID
// or the empty string, respectively.
//
// Because of the convention for the first two parameters, messages in the
// catalogue typically look like this (at least for left-to-right languages):
//      {1} {2} The new error {_}
// The tokens {1}, {2}, etc.  refer to the first and second positional parameters
// respectively, while {_} is replaced by the positional parameters not
// explicitly referred to elsewhere in the message.  Thus, given the parameters
// above, this would lead to the output:
//      my_component op_name The new error object_name
//
// If a substring is of the form {:<number>}, {<number>:}, {:<number>:}, {:_},
// {_:}, or {:_:}, and the corresponding parameters are not the empty string,
// the parameter is preceded by ": " or followed by ":" or both,
// respectively.  For example, if the format:
//      {3:} foo {2} bar{:_} ({3})
// is used with the cat.Format example above, it yields:
//      3rd: foo 2nd bar: 1st 4th (3rd)
//
// The Convert() and ExplicitConvert() calls are like New() and ExplicitNew(),
// but convert existing errors (with their parameters, if applicable)
// to verror errors with given language, component name, and operation name,
// if non-empty values for these are provided.  They also add a PC to a
// list of PC values to assist developers hunting for the error.
//
// If the context.T specified with New() or Convert() is nil, a default
// context is used, set by SetDefaultContext().  This can be used in standalone
// programmes, or in anciliary threads not associated with an RPC.  The user
// might do the following to get the language from the environment, and the
// programme name from Args[0]:
//     ctx := runtime.NewContext()
//     ctx = i18n.WithLangID(ctx, i18n.LangIDFromEnv())
//     ctx = verror.WithComponentName(ctx, os.Args[0])
//     verror.SetDefaultContext(ctx)
// A standalone tool might set the operation name to be a subcommand name, if
// any.  If the default context has not been set, the error generated has no
// language, component and operation values; they will be filled in by the
// first Convert() call that does have these values.
package verror

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"v.io/v23/context"
	"v.io/v23/i18n"
	"v.io/v23/vdl"
	"v.io/v23/vtrace"
)

// ID is a unique identifier for errors.
type ID string

// An ActionCode represents the action expected to be performed by a typical client
// receiving an error that perhaps it does not understand.
type ActionCode uint32

// Codes for ActionCode.
const (
	// Retry actions are encoded in the bottom few bits.
	RetryActionMask ActionCode = 3

	NoRetry         ActionCode = 0 // Do not retry.
	RetryConnection ActionCode = 1 // Renew high-level connection/context.
	RetryRefetch    ActionCode = 2 // Refetch and retry (e.g., out of date HTTP ETag)
	RetryBackoff    ActionCode = 3 // Backoff and retry a finite number of times.
)

// RetryAction returns the part of the ActionCode that indicates retry behaviour.
func (ac ActionCode) RetryAction() ActionCode {
	return ac & RetryActionMask
}

// An IDAction combines a unique identifier ID for errors with an ActionCode.
// The ID allows stable error checking across different error messages and
// different address spaces.  By convention the format for the identifier is
// "PKGPATH.NAME" - e.g. ErrIDFoo defined in the "v23/verror" package has id
// "v23/verror.ErrIDFoo".  It is unwise ever to create two IDActions that
// associate different ActionCodes with the same ID.
type IDAction struct {
	ID     ID
	Action ActionCode
}

// Register returns a IDAction with the given ID and Action fields, and
// inserts a message into the default i18n Catalogue in US English.
// Other languages can be added by adding to the Catalogue.
func Register(id ID, action ActionCode, englishText string) IDAction {
	i18n.Cat().SetWithBase(defaultLangID(i18n.NoLangID), i18n.MsgID(id), englishText)
	return IDAction{id, action}
}

// E is the in-memory representation of a verror error.
//
// The wire representation is defined as vdl.WireError; values of E
// type are automatically converted to/from vdl.WireError by VDL and VOM.
type E struct {
	ID         ID            // The identity of the error.
	Action     ActionCode    // Default action to take on error.
	Msg        string        // Error message; empty if no language known.
	ParamList  []interface{} // The variadic parameters given to ExplicitNew().
	stackPCs   []uintptr     // PCs of creators of E
	chainedPCs []uintptr     // PCs of a chained E
}

// TypeOf(verror.E{}) should give vdl.WireError.
func (E) __VDLReflect(struct {
	Name string `vdl:"v.io/v23/vdl.WireError"`
}) {
}

func (x E) VDLEqual(yiface interface{}) bool {
	y := yiface.(E)
	switch {
	case x.ID != y.ID:
		return false
	case x.Action != y.Action:
		return false
	case x.Error() != y.Error():
		// NOTE: We compare the result of Error() rather than comparing the Msg
		// fields, since the Msg field isn't always set; Msg=="" is equivalent to
		// Msg==v.io/v23/verror.Unknown.
		//
		// TODO(toddw): Investigate the root cause of this.
		return false
	case len(x.ParamList) != len(y.ParamList):
		return false
	}
	for i := range x.ParamList {
		if !vdl.DeepEqual(x.ParamList[i], y.ParamList[i]) {
			return false
		}
	}
	return true
}

var (
	ttErrorElem     = vdl.ErrorType.Elem()
	ttWireRetryCode = ttErrorElem.Field(1).Type
	ttListAny       = vdl.ListType(vdl.AnyType)
)

func (x E) VDLWrite(enc vdl.Encoder) error {
	if err := enc.StartValue(ttErrorElem); err != nil {
		return err
	}
	if x.ID != "" {
		if err := enc.NextFieldValueString(0, vdl.StringType, string(x.ID)); err != nil {
			return err
		}
	}
	if x.Action != NoRetry {
		var actionStr string
		switch x.Action {
		case RetryConnection:
			actionStr = "RetryConnection"
		case RetryRefetch:
			actionStr = "RetryRefetch"
		case RetryBackoff:
			actionStr = "RetryBackoff"
		default:
			return fmt.Errorf("action %d not in enum WireRetryCode", x.Action)
		}
		if err := enc.NextFieldValueString(1, ttWireRetryCode, actionStr); err != nil {
			return err
		}
	}
	if x.Msg != "" {
		if err := enc.NextFieldValueString(2, vdl.StringType, x.Msg); err != nil {
			return err
		}
	}
	if len(x.ParamList) != 0 {
		if err := enc.NextField(3); err != nil {
			return err
		}
		if err := enc.StartValue(ttListAny); err != nil {
			return err
		}
		if err := enc.SetLenHint(len(x.ParamList)); err != nil {
			return err
		}
		for i := 0; i < len(x.ParamList); i++ {
			if err := enc.NextEntry(false); err != nil {
				return err
			}
			if x.ParamList[i] == nil {
				if err := enc.NilValue(vdl.AnyType); err != nil {
					return err
				}
			} else {
				if err := vdl.Write(enc, x.ParamList[i]); err != nil {
					return err
				}
			}
		}
		if err := enc.NextEntry(true); err != nil {
			return err
		}
		if err := enc.FinishValue(); err != nil {
			return err
		}
	}
	if err := enc.NextField(-1); err != nil {
		return err
	}
	return enc.FinishValue()
}

func (x *E) VDLRead(dec vdl.Decoder) error {
	*x = E{}
	if err := dec.StartValue(ttErrorElem); err != nil {
		return err
	}
	decType := dec.Type()
	for {
		index, err := dec.NextField()
		switch {
		case err != nil:
			return err
		case index == -1:
			return dec.FinishValue()
		}
		if decType != ttErrorElem {
			index = ttErrorElem.FieldIndexByName(decType.Field(index).Name)
			if index == -1 {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				continue
			}
		}
	errorFieldSwitch:
		switch index {
		case 0:
			id, err := dec.ReadValueString()
			if err != nil {
				return err
			}
			x.ID = ID(id)
		case 1:
			code, err := dec.ReadValueString()
			if err != nil {
				return err
			}
			switch code {
			case "NoRetry":
				x.Action = NoRetry
			case "RetryConnection":
				x.Action = RetryConnection
			case "RetryRefetch":
				x.Action = RetryRefetch
			case "RetryBackoff":
				x.Action = RetryBackoff
			default:
				return fmt.Errorf("label %s not in enum WireRetryCode", code)
			}
		case 2:
			msg, err := dec.ReadValueString()
			if err != nil {
				return err
			}
			x.Msg = msg
		case 3:
			if err := dec.StartValue(ttListAny); err != nil {
				return err
			}
			switch len := dec.LenHint(); {
			case len > 0:
				x.ParamList = make([]interface{}, 0, len)
			default:
				x.ParamList = nil
			}
			for {
				switch done, err := dec.NextEntry(); {
				case err != nil:
					return err
				case done:
					if err := dec.FinishValue(); err != nil {
						return err
					}
					break errorFieldSwitch
				}
				var elem interface{}
				if err := vdl.Read(dec, &elem); err != nil {
					return err
				}
				x.ParamList = append(x.ParamList, elem)
			}
		}
	}
}

const maxPCs = 40 // Maximum number of PC values we'll include in a stack trace.

// A SubErrs is a special type that allows clients to include a list of
// subordinate errors to an error's parameter list.  Clients can add a SubErrs
// to the parameter list directly, via New() of include one in an existing
// error using AddSubErrs().  Each element of the slice has a name, an error,
// and an integer that encodes options such as verror.Print as bits set within
// it.  By convention, clients are expected to use name of the form "X=Y" to
// distinguish their subordinate errors from those of other abstraction layers.
// For example, a layer reporting on errors in individual blessings in an RPC
// might use strings like "blessing=<blessing_name>".
type SubErrs []SubErr

type SubErrOpts uint32

// A SubErr represents a (string, error, int32) triple,  It is the element type for SubErrs.
type SubErr struct {
	Name    string
	Err     error
	Options SubErrOpts
}

const (
	// Print, when set in SubErr.Options, tells Error() to print this SubErr.
	Print SubErrOpts = 0x1
)

func assertIsE(err error) (E, bool) {
	if e, ok := err.(E); ok {
		return e, true
	}
	if e, ok := err.(*E); ok && e != nil {
		return *e, true
	}
	return E{}, false
}

// ErrorID returns the ID of the given err, or Unknown if the err has no ID.
// If err is nil then ErrorID returns "".
func ErrorID(err error) ID {
	if err == nil {
		return ""
	}
	if e, ok := assertIsE(err); ok {
		return e.ID
	}
	return ErrUnknown.ID
}

// Action returns the action of the given err, or NoRetry if the err has no Action.
func Action(err error) ActionCode {
	if err == nil {
		return NoRetry
	}
	if e, ok := assertIsE(err); ok {
		return e.Action
	}
	return NoRetry
}

// PCs represents a list of PC locations
type PCs []uintptr

// Stack returns the list of PC locations on the stack when this error was
// first generated within this address space, or an empty list if err is not an
// E.
func Stack(err error) PCs {
	if err != nil {
		if e, ok := assertIsE(err); ok {
			stackIntPtr := make([]uintptr, len(e.stackPCs))
			copy(stackIntPtr, e.stackPCs)
			if e.chainedPCs != nil {
				stackIntPtr = append(stackIntPtr, uintptr(0))
				stackIntPtr = append(stackIntPtr, e.chainedPCs...)
			}
			return stackIntPtr
		}
	}
	return nil
}

func (st PCs) String() string {
	buf := bytes.NewBufferString("")
	StackToText(buf, st)
	return buf.String()
}

// stackToTextIndent emits on w a text representation of stack, which is typically
// obtained from Stack() and represents the source location(s) where an
// error was generated or passed through in the local address space.
// indent is added to a prefix of each line printed.
func stackToTextIndent(w io.Writer, stack []uintptr, indent string) (err error) {
	for i := 0; i != len(stack) && err == nil; i++ {
		if stack[i] == 0 {
			_, err = fmt.Fprintf(w, "%s----- chained verror -----\n", indent)
		} else {
			fnc := runtime.FuncForPC(stack[i])
			file, line := fnc.FileLine(stack[i])
			_, err = fmt.Fprintf(w, "%s%s:%d: %s\n", indent, file, line, fnc.Name())
		}
	}
	return err
}

// StackToText emits on w a text representation of stack, which is typically
// obtained from Stack() and represents the source location(s) where an
// error was generated or passed through in the local address space.
func StackToText(w io.Writer, stack []uintptr) error {
	return stackToTextIndent(w, stack, "")
}

// defaultLangID returns langID is it is not i18n.NoLangID, and the default
// value of US English otherwise.
func defaultLangID(langID i18n.LangID) i18n.LangID {
	if langID == i18n.NoLangID {
		langID = "en-US"
	}
	return langID
}

func isDefaultIDAction(id ID, action ActionCode) bool {
	return id == "" && action == 0
}

// makeInternal is like ExplicitNew(), but takes a slice of PC values as an argument,
// rather than constructing one from the caller's PC.
func makeInternal(idAction IDAction, langID i18n.LangID, componentName string, opName string, stack []uintptr, v ...interface{}) E {
	msg := ""
	var chainedPCs []uintptr
	for _, par := range v {
		if err, ok := par.(error); ok {
			if _, ok := assertIsE(err); ok {
				chainedPCs = Stack(err)
				break
			}
		}
	}
	params := append([]interface{}{componentName, opName}, v...)
	if langID != i18n.NoLangID {
		id := idAction.ID
		if id == "" {
			id = ErrUnknown.ID
		}
		msg = i18n.Cat().Format(langID, i18n.MsgID(id), params...)
	}
	return E{idAction.ID, idAction.Action, msg, params, stack, chainedPCs}
}

// ExplicitNew returns an error with the given ID, with an error string in the chosen
// language.  The component and operation name are included the first and second
// parameters of the error.  Other parameters are taken from v[].  The
// parameters are formatted into the message according to i18n.Cat().Format.
// The caller's PC is added to the error's stack.
// If the parameter list contains an instance of verror.E, then the stack of
// the first, and only the first, occurrence of such an instance, will be chained
// to the stack of this newly created error.
func ExplicitNew(idAction IDAction, langID i18n.LangID, componentName string, opName string, v ...interface{}) error {
	stack := make([]uintptr, maxPCs)
	stack = stack[:runtime.Callers(2, stack)]
	return makeInternal(idAction, langID, componentName, opName, stack, v...)
}

// A componentKey is used as a key for context.T's Value() map.
type componentKey struct{}

// WithComponentName returns a context based on ctx that has the
// componentName that New() and Convert() can use.
func WithComponentName(ctx *context.T, componentName string) *context.T {
	return context.WithValue(ctx, componentKey{}, componentName)
}

// New is like ExplicitNew(), but obtains the language, component name, and operation
// name from the specified context.T.  ctx may be nil.
func New(idAction IDAction, ctx *context.T, v ...interface{}) error {
	langID, componentName, opName := dataFromContext(ctx)
	stack := make([]uintptr, maxPCs)
	stack = stack[:runtime.Callers(2, stack)]
	return makeInternal(idAction, langID, componentName, opName, stack, v...)
}

// isEmptyString() returns whether v is an empty string.
func isEmptyString(v interface{}) bool {
	str, isAString := v.(string)
	return isAString && str == ""
}

// convertInternal is like ExplicitConvert(), but takes a slice of PC values as an argument,
// rather than constructing one from the caller's PC.
func convertInternal(idAction IDAction, langID i18n.LangID, componentName string, opName string, stack []uintptr, err error) E {
	// If err is already a verror.E, we wish to:
	//  - retain all set parameters.
	//  - if not yet set, set parameters 0 and 1 from componentName and
	//    opName.
	//  - if langID is set, and we have the appropriate language's format
	//    in our catalogue, use it.  Otherwise, retain a message (assuming
	//    additional parameters were not set) even if the language is not
	//    correct.
	if e, ok := assertIsE(err); !ok {
		return makeInternal(idAction, langID, componentName, opName, stack, err.Error())
	} else {
		oldParams := e.ParamList

		// Convert all embedded E errors, recursively.
		for i := range oldParams {
			if subErr, isE := oldParams[i].(E); isE {
				oldParams[i] = convertInternal(idAction, langID, componentName, opName, stack, subErr)
			} else if subErrs, isSubErrs := oldParams[i].(SubErrs); isSubErrs {
				for j := range subErrs {
					if subErr, isE := subErrs[j].Err.(E); isE {
						subErrs[j].Err = convertInternal(idAction, langID, componentName, opName, stack, subErr)
					}
				}
				oldParams[i] = subErrs
			}
		}

		// Create a non-empty format string if we have the language in the catalogue.
		var formatStr string
		if langID != i18n.NoLangID {
			id := e.ID
			if id == "" {
				id = ErrUnknown.ID
			}
			formatStr = i18n.Cat().Lookup(langID, i18n.MsgID(id))
		}

		// Ignore the caller-supplied component and operation if we already have them.
		if componentName != "" && len(oldParams) >= 1 && !isEmptyString(oldParams[0]) {
			componentName = ""
		}
		if opName != "" && len(oldParams) >= 2 && !isEmptyString(oldParams[1]) {
			opName = ""
		}

		var msg string
		var newParams []interface{}
		if componentName == "" && opName == "" {
			if formatStr == "" {
				return e // Nothing to change.
			} else { // Parameter list does not change.
				newParams = e.ParamList
				msg = i18n.FormatParams(formatStr, newParams...)
			}
		} else { // Replace at least one of the first two parameters.
			newLen := len(oldParams)
			if newLen < 2 {
				newLen = 2
			}
			newParams = make([]interface{}, newLen)
			copy(newParams, oldParams)
			if componentName != "" {
				newParams[0] = componentName
			}
			if opName != "" {
				newParams[1] = opName
			}
			if formatStr != "" {
				msg = i18n.FormatParams(formatStr, newParams...)
			}
		}
		return E{e.ID, e.Action, msg, newParams, e.stackPCs, nil}
	}
}

// ExplicitConvert converts a regular err into an E error, setting its IDAction to idAction.  If
// err is already an E, it returns err or an equivalent value without changing its type, but
// potentially changing the language, component or operation if langID!=i18n.NoLangID,
// componentName!="" or opName!="" respectively.  The caller's PC is added to the
// error's stack.
func ExplicitConvert(idAction IDAction, langID i18n.LangID, componentName string, opName string, err error) error {
	if err == nil {
		return nil
	} else {
		var stack []uintptr
		if _, isE := assertIsE(err); !isE { // Walk the stack only if convertInternal will allocate an E.
			stack = make([]uintptr, maxPCs)
			stack = stack[:runtime.Callers(2, stack)]
		}
		return convertInternal(idAction, langID, componentName, opName, stack, err)
	}
}

// defaultCtx is the context used when a nil context.T is passed to New() or Convert().
var (
	defaultCtx     *context.T
	defaultCtxLock sync.RWMutex // Protects defaultCtx.
)

// SetDefaultContext sets the default context used when a nil context.T is
// passed to New() or Convert().  It is typically used in standalone
// programmes that have no RPC context, or in servers for the context of
// ancillary threads not associated with any particular RPC.
func SetDefaultContext(ctx *context.T) {
	defaultCtxLock.Lock()
	defaultCtx = ctx
	defaultCtxLock.Unlock()
}

// Convert is like ExplicitConvert(), but obtains the language, component and operation names
// from the specified context.T.  ctx may be nil.
func Convert(idAction IDAction, ctx *context.T, err error) error {
	if err == nil {
		return nil
	}
	langID, componentName, opName := dataFromContext(ctx)
	var stack []uintptr
	if _, isE := assertIsE(err); !isE { // Walk the stack only if convertInternal will allocate an E.
		stack = make([]uintptr, maxPCs)
		stack = stack[:runtime.Callers(2, stack)]
	}
	return convertInternal(idAction, langID, componentName, opName, stack, err)
}

// dataFromContext reads the languageID, component name, and operation name
// from the context, using defaults as appropriate.
func dataFromContext(ctx *context.T) (langID i18n.LangID, componentName string, opName string) {
	// Use a default context if ctx is nil.  defaultCtx may also be nil, so
	// further nil checks are required below.
	if ctx == nil {
		defaultCtxLock.RLock()
		ctx = defaultCtx
		defaultCtxLock.RUnlock()
	}
	if ctx != nil {
		langID = i18n.GetLangID(ctx)
		value := ctx.Value(componentKey{})
		componentName, _ = value.(string)
		opName = vtrace.GetSpan(ctx).Name()
	}
	if componentName == "" {
		componentName = filepath.Base(os.Args[0])
	}
	return defaultLangID(langID), componentName, opName
}

// Error returns the error message; if it has not been formatted for a specific
// language, a default message containing the error ID and parameters is
// generated.  This method is required to fulfil the error interface.
func (e E) Error() string {
	msg := e.Msg
	if isDefaultIDAction(e.ID, e.Action) && msg == "" {
		msg = i18n.Cat().Format(i18n.NoLangID, i18n.MsgID(ErrUnknown.ID), e.ParamList...)
	} else if msg == "" {
		msg = i18n.Cat().Format(i18n.NoLangID, i18n.MsgID(e.ID), e.ParamList...)
	}
	return msg
}

// String is the default printing function for SubErrs.
func (subErrs SubErrs) String() (result string) {
	if len(subErrs) > 0 {
		sep := ""
		for _, s := range subErrs {
			if (s.Options & Print) != 0 {
				result += fmt.Sprintf("%s[%s: %s]", sep, s.Name, s.Err.Error())
				sep = ", "
			}
		}
	}
	return result
}

// params returns the variadic arguments to ExplicitNew().  We do not export it
// to discourage its abuse as a general-purpose way to pass alternate return
// values.
func params(err error) []interface{} {
	if e, ok := assertIsE(err); ok {
		return e.ParamList
	}
	return nil
}

// subErrorIndex returns index of the first SubErrs in e.ParamList
// or len(e.ParamList) if there is no such parameter.
func (e E) subErrorIndex() (i int) {
	for i = range e.ParamList {
		if _, isSubErrs := e.ParamList[i].(SubErrs); isSubErrs {
			return i
		}
	}
	return len(e.ParamList)
}

// subErrors returns a copy of the subordinate errors accumulated into err, or
// nil if there are no such errors.  We do not export it to discourage its
// abuse as a general-purpose way to pass alternate return values.
func subErrors(err error) (r SubErrs) {
	if e, ok := assertIsE(err); ok {
		size := 0
		for i := range e.ParamList {
			if subErrsParam, isSubErrs := e.ParamList[i].(SubErrs); isSubErrs {
				size += len(subErrsParam)
			}
		}
		if size != 0 {
			r = make(SubErrs, 0, size)
			for i := range e.ParamList {
				if subErrsParam, isSubErrs := e.ParamList[i].(SubErrs); isSubErrs {
					r = append(r, subErrsParam...)
				}
			}
		}
	}
	return r
}

// addSubErrsInternal returns a copy of err with supplied errors appended as
// subordinate errors.  Requires that errors[i].Err!=nil for 0<=i<len(errors).
func addSubErrsInternal(err error, langID i18n.LangID, componentName string, opName string, stack []uintptr, errors SubErrs) error {
	var pe *E
	var e E
	var ok bool
	if pe, ok = err.(*E); ok {
		e = *pe
	} else if e, ok = err.(E); !ok {
		panic("non-verror.E passed to verror.AddSubErrs")
	}
	var subErrs SubErrs
	index := e.subErrorIndex()
	copy(e.ParamList, e.ParamList)
	if index == len(e.ParamList) {
		e.ParamList = append(e.ParamList, subErrs)
	} else {
		copy(subErrs, e.ParamList[index].(SubErrs))
	}
	for _, subErr := range errors {
		if _, ok := assertIsE(subErr.Err); ok {
			subErrs = append(subErrs, subErr)
		} else {
			subErr.Err = convertInternal(IDAction{}, langID, componentName, opName, stack, subErr.Err)
			subErrs = append(subErrs, subErr)
		}
	}
	e.ParamList[index] = subErrs
	if langID != i18n.NoLangID {
		e.Msg = i18n.Cat().Format(langID, i18n.MsgID(e.ID), e.ParamList...)
	}
	return e
}

// ExplicitAddSubErrs returns a copy of err with supplied errors appended as
// subordinate errors.  Requires that errors[i].Err!=nil for 0<=i<len(errors).
func ExplicitAddSubErrs(err error, langID i18n.LangID, componentName string, opName string, errors ...SubErr) error {
	stack := make([]uintptr, maxPCs)
	stack = stack[:runtime.Callers(2, stack)]
	return addSubErrsInternal(err, langID, componentName, opName, stack, errors)
}

// AddSubErrs is like ExplicitAddSubErrs, but uses the provided context
// to obtain the langID, componentName, and opName values.
func AddSubErrs(err error, ctx *context.T, errors ...SubErr) error {
	stack := make([]uintptr, maxPCs)
	stack = stack[:runtime.Callers(2, stack)]
	langID, componentName, opName := dataFromContext(ctx)
	return addSubErrsInternal(err, langID, componentName, opName, stack, errors)
}

// debugStringInternal returns a more verbose string representation of an
// error, perhaps more thorough than one might present to an end user, but
// useful for debugging by a developer.  It prefixes all lines output with
// "prefix" and "name" (if non-empty) and adds intent to prefix wen recursing.
func debugStringInternal(err error, prefix string, name string) string {
	str := prefix
	if len(name) > 0 {
		str += name + " "
	}
	str += err.Error()
	// Append err's stack, indented a little.
	prefix += "  "
	buf := bytes.NewBufferString("")
	stackToTextIndent(buf, Stack(err), prefix)
	str += "\n" + buf.String()
	// Print all the subordinate errors, even the ones that were not
	// printed by Error(), indented a bit further.
	prefix += "  "
	if subErrs := subErrors(err); len(subErrs) > 0 {
		for _, s := range subErrs {
			str += debugStringInternal(s.Err, prefix, s.Name)
		}
	}
	return str
}

// DebugString returns a more verbose string representation of an error,
// perhaps more thorough than one might present to an end user, but useful for
// debugging by a developer.
func DebugString(err error) string {
	return debugStringInternal(err, "", "")
}
