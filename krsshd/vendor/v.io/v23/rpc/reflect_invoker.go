// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"reflect"
	"sort"
	"sync"

	"v.io/v23/context"
	"v.io/v23/glob"
	"v.io/v23/vdl"
	"v.io/v23/vdlroot/signature"
	"v.io/v23/verror"
)

// Describer may be implemented by an underlying object served by the
// ReflectInvoker, in order to describe the interfaces that the object
// implements.  This describes all data in signature.Interface that the
// ReflectInvoker cannot obtain through reflection; basically everything except
// the method names and types.
//
// Note that a single object may implement multiple interfaces; to describe such
// an object, simply return more than one elem in the returned list.
type Describer interface {
	// Describe the underlying object.  The implementation must be idempotent
	// across different instances of the same underlying type; the ReflectInvoker
	// calls this once per type and caches the results.
	Describe__() []InterfaceDesc
}

// InterfaceDesc describes an interface; it is similar to signature.Interface,
// without the information that can be obtained via reflection.
type InterfaceDesc struct {
	Name    string
	PkgPath string
	Doc     string
	Embeds  []EmbedDesc
	Methods []MethodDesc
}

// EmbedDesc describes an embedded interface; it is similar to signature.Embed,
// without the information that can be obtained via reflection.
type EmbedDesc struct {
	Name    string
	PkgPath string
	Doc     string
}

// MethodDesc describes an interface method; it is similar to signature.Method,
// without the information that can be obtained via reflection.
type MethodDesc struct {
	Name      string
	Doc       string
	InArgs    []ArgDesc    // Input arguments
	OutArgs   []ArgDesc    // Output arguments
	InStream  ArgDesc      // Input stream (client to server)
	OutStream ArgDesc      // Output stream (server to client)
	Tags      []*vdl.Value // Method tags
}

// ArgDesc describes an argument; it is similar to signature.Arg, without the
// information that can be obtained via reflection.
type ArgDesc struct {
	Name string
	Doc  string
}

type reflectInvoker struct {
	rcvr    reflect.Value
	methods map[string]methodInfo // used by Prepare and Invoke
	sig     []signature.Interface // used by Signature and MethodSignature
}

var _ Invoker = (*reflectInvoker)(nil)

// methodInfo holds the runtime information necessary for Prepare and Invoke.
type methodInfo struct {
	rvFunc   reflect.Value  // Function representing the method.
	rtInArgs []reflect.Type // In arg types, not including receiver and call.

	// rtStreamCall holds the type of the typesafe streaming call, if any.
	// rvStreamCallInit is the associated Init function.
	rtStreamCall     reflect.Type
	rvStreamCallInit reflect.Value

	tags []*vdl.Value // Tags from the signature.
}

// ReflectInvoker returns an Invoker implementation that uses reflection to make
// each compatible exported method in obj available.  E.g.:
//
//   type impl struct{}
//   func (impl) NonStreaming(ctx *context.T, call rpc.ServerCall, ...) (...)
//   func (impl) Streaming(ctx *context.T, call *MyCall, ...) (...)
//
// The first in-arg must be context.T.  The second in-arg must be a call; for
// non-streaming methods it must be rpc.ServerCall, and for streaming methods it
// must be a pointer to a struct that implements rpc.StreamServerCall, and also
// adds typesafe streaming wrappers.  Here's an example that streams int32 from
// client to server, and string from server to client:
//
//   type MyCall struct { rpc.StreamServerCall }
//
//   // Init initializes MyCall via rpc.StreamServerCall.
//   func (*MyCall) Init(rpc.StreamServerCall) {...}
//
//   // RecvStream returns the receiver side of the server stream.
//   func (*MyCall) RecvStream() interface {
//     Advance() bool
//     Value() int32
//     Err() error
//   } {...}
//
//   // SendStream returns the sender side of the server stream.
//   func (*MyCall) SendStream() interface {
//     Send(item string) error
//   } {...}
//
// We require the streaming call arg to have this structure so that we can
// capture the streaming in and out arg types via reflection.  We require it to
// be a concrete type with an Init func so that we can create new instances,
// also via reflection.
//
// As a temporary special-case, we also allow generic streaming methods:
//
//   func (impl) Generic(ctx *context.T, call rpc.StreamServerCall, ...) (...)
//
// The problem with allowing this form is that via reflection we can no longer
// determine whether the server performs streaming, or what the streaming in and
// out types are.
// TODO(toddw): Remove this special-case.
//
// The ReflectInvoker silently ignores unexported methods, and exported methods
// whose first argument doesn't implement rpc.ServerCall.  All other methods
// must follow the above rules; bad method types cause an error to be returned.
//
// If obj implements the Describer interface, we'll use it to describe portions
// of the object signature that cannot be retrieved via reflection;
// e.g. method tags, documentation, variable names, etc.
func ReflectInvoker(obj interface{}) (Invoker, error) {
	rt := reflect.TypeOf(obj)
	info := reflectCache.lookup(rt)
	if info == nil {
		// Concurrent calls may cause reflectCache.set to be called multiple times.
		// This race is benign; the info for a given type never changes.
		var err error
		if info, err = newReflectInfo(obj); err != nil {
			return nil, err
		}
		reflectCache.set(rt, info)
	}
	return reflectInvoker{reflect.ValueOf(obj), info.methods, info.sig}, nil
}

// ReflectInvokerOrDie is the same as ReflectInvoker, but panics on all errors.
func ReflectInvokerOrDie(obj interface{}) Invoker {
	invoker, err := ReflectInvoker(obj)
	if err != nil {
		panic(err)
	}
	return invoker
}

// Prepare implements the Invoker.Prepare method.
func (ri reflectInvoker) Prepare(ctx *context.T, method string, _ int) ([]interface{}, []*vdl.Value, error) {
	info, ok := ri.methods[method]
	if !ok {
		return nil, nil, verror.New(verror.ErrUnknownMethod, nil, method)
	}
	// Return the tags and new in-arg objects.
	var argptrs []interface{}
	if len(info.rtInArgs) > 0 {
		argptrs = make([]interface{}, len(info.rtInArgs))
		for ix, rtInArg := range info.rtInArgs {
			argptrs[ix] = reflect.New(rtInArg).Interface()
		}
	}

	return argptrs, info.tags, nil
}

// Invoke implements the Invoker.Invoke method.
func (ri reflectInvoker) Invoke(ctx *context.T, call StreamServerCall, method string, argptrs []interface{}) ([]interface{}, error) {
	info, ok := ri.methods[method]
	if !ok {
		return nil, verror.New(verror.ErrUnknownMethod, ctx, method)
	}
	// Create the reflect.Value args for the invocation.  The receiver of the
	// method is always first, followed by the required ctx and call args.
	rvArgs := make([]reflect.Value, len(argptrs)+3)
	rvArgs[0] = ri.rcvr
	rvArgs[1] = reflect.ValueOf(ctx)
	if info.rtStreamCall == nil {
		// There isn't a typesafe streaming call, just use the call.
		rvArgs[2] = reflect.ValueOf(call)
	} else {
		// There is a typesafe streaming call with type rtStreamCall.  We perform
		// the equivalent of the following:
		//   ctx := new(rtStreamCall)
		//   ctx.Init(call)
		rvStreamCall := reflect.New(info.rtStreamCall)
		info.rvStreamCallInit.Call([]reflect.Value{rvStreamCall, reflect.ValueOf(call)})
		rvArgs[2] = rvStreamCall
	}
	// Positional user args follow.
	for ix, argptr := range argptrs {
		rvArgs[ix+3] = reflect.ValueOf(argptr).Elem()
	}
	// Invoke the method, and handle the final error out-arg.
	rvResults := info.rvFunc.Call(rvArgs)
	if len(rvResults) == 0 {
		return nil, abortedf(errNoFinalErrorOutArg)
	}
	rvErr := rvResults[len(rvResults)-1]
	rvResults = rvResults[:len(rvResults)-1]
	if rvErr.Type() != rtError {
		return nil, abortedf(errNoFinalErrorOutArg)
	}
	if iErr := rvErr.Interface(); iErr != nil {
		return nil, iErr.(error)
	}
	// Convert the rest of the results into interface{}.
	if len(rvResults) == 0 {
		return nil, nil
	}
	results := make([]interface{}, len(rvResults))
	for ix, r := range rvResults {
		results[ix] = r.Interface()
	}
	return results, nil
}

// Signature implements the Invoker.Signature method.
func (ri reflectInvoker) Signature(ctx *context.T, call ServerCall) ([]signature.Interface, error) {
	return signature.CopyInterfaces(ri.sig), nil
}

// MethodSignature implements the Invoker.MethodSignature method.
func (ri reflectInvoker) MethodSignature(ctx *context.T, call ServerCall, method string) (signature.Method, error) {
	// Return the first method in any interface with the given method name.
	for _, iface := range ri.sig {
		if msig, ok := iface.FindMethod(method); ok {
			return signature.CopyMethod(msig), nil
		}
	}
	return signature.Method{}, verror.New(verror.ErrUnknownMethod, ctx, method)
}

// Globber implements the rpc.Globber interface.
func (ri reflectInvoker) Globber() *GlobState {
	return determineGlobState(ri.rcvr.Interface())
}

// reflectRegistry is a locked map from reflect.Type to reflection info, which
// is expensive to compute.  The only instance is reflectCache, which is a
// global cache to speed up repeated lookups.  There is no GC; the total set of
// types in a single address space is expected to be bounded and small.
type reflectRegistry struct {
	sync.RWMutex
	infoMap map[reflect.Type]*reflectInfo
}

type reflectInfo struct {
	methods map[string]methodInfo
	sig     []signature.Interface
}

func (reg *reflectRegistry) lookup(rt reflect.Type) *reflectInfo {
	reg.RLock()
	info := reg.infoMap[rt]
	reg.RUnlock()
	return info
}

// set the entry for (rt, info).  Is a no-op if rt already exists in the map.
func (reg *reflectRegistry) set(rt reflect.Type, info *reflectInfo) {
	reg.Lock()
	if exist := reg.infoMap[rt]; exist == nil {
		reg.infoMap[rt] = info
	}
	reg.Unlock()
}

var reflectCache = &reflectRegistry{infoMap: make(map[reflect.Type]*reflectInfo)}

// newReflectInfo returns reflection information that is expensive to compute.
// Although it is passed an object rather than a type, it guarantees that the
// returned information is always the same for all instances of a given type.
func newReflectInfo(obj interface{}) (*reflectInfo, error) {
	if obj == nil {
		return nil, verror.New(errReflectInvokerNil, nil)
	}
	// First make methodInfos, based on reflect.Type, which also captures the name
	// and in, out and streaming types of each method in methodSigs.  This
	// information is guaranteed to be correct, since it's based on reflection on
	// the underlying object.
	rt := reflect.TypeOf(obj)
	methodInfos, methodSigs, err := makeMethods(rt)
	switch {
	case err != nil:
		return nil, err
	case len(methodInfos) == 0 && determineGlobState(obj) == nil:
		if m := TypeCheckMethods(obj); len(m) > 0 {
			return nil, verror.New(errNoCompatibleMethods, nil, rt, TypeCheckMethods(obj))
		}
		return nil, verror.New(errNoCompatibleMethods, nil, rt, "no exported methods")
	}
	// Now attach method tags to each methodInfo.  Since this is based on the desc
	// provided by the user, there's no guarantee it's "correct", but if the same
	// method is described by multiple interfaces, we check the tags are the same.
	desc := describe(obj)
	if err := attachMethodTags(methodInfos, desc); verror.ErrorID(err) == verror.ErrAborted.ID {
		return nil, verror.New(errTagError, nil, rt, err)
	}
	// Finally create the signature.  This combines the desc provided by the user
	// with the methodSigs computed via reflection.  We ensure that the method
	// names and types computed via reflection always remains in the final sig;
	// the desc is merely used to augment the signature.
	sig := makeSig(desc, methodSigs)
	return &reflectInfo{methodInfos, sig}, nil
}

// determineGlobState determines whether and how obj implements Glob.  Returns
// nil iff obj doesn't implement Glob, based solely on the type of obj.
func determineGlobState(obj interface{}) *GlobState {
	if x, ok := obj.(Globber); ok {
		return x.Globber()
	}
	return NewGlobState(obj)
}

func describe(obj interface{}) []InterfaceDesc {
	if d, ok := obj.(Describer); ok {
		// Describe__ must not vary across instances of the same underlying type.
		return d.Describe__()
	}
	return nil
}

func makeMethods(rt reflect.Type) (map[string]methodInfo, map[string]signature.Method, error) {
	infos := make(map[string]methodInfo, rt.NumMethod())
	sigs := make(map[string]signature.Method, rt.NumMethod())
	for mx := 0; mx < rt.NumMethod(); mx++ {
		method := rt.Method(mx)
		// Silently skip incompatible methods, except for Aborted errors.
		var sig signature.Method
		if err := typeCheckMethod(method, &sig); err != nil {
			if verror.ErrorID(err) == verror.ErrAborted.ID {
				return nil, nil, verror.New(errAbortedDetail, nil, rt.String(), method.Name, err)
			}
			continue
		}
		infos[method.Name] = makeMethodInfo(method)
		sigs[method.Name] = sig
	}
	return infos, sigs, nil
}

func makeMethodInfo(method reflect.Method) methodInfo {
	info := methodInfo{rvFunc: method.Func}
	mtype := method.Type
	for ix := 3; ix < mtype.NumIn(); ix++ { // Skip receiver, ctx and call
		info.rtInArgs = append(info.rtInArgs, mtype.In(ix))
	}
	// Initialize info for typesafe streaming calls.  Note that we've already
	// type-checked the method.  We memoize the stream type and Init function, so
	// that we can create and initialize the stream type in Invoke.
	if rt := mtype.In(2); rt != rtStreamServerCall && rt != rtServerCall && rt.Kind() == reflect.Ptr {
		info.rtStreamCall = rt.Elem()
		mInit, _ := rt.MethodByName("Init")
		info.rvStreamCallInit = mInit.Func
	}
	return info
}

func abortedf(embeddedErr verror.IDAction, v ...interface{}) error {
	return verror.New(verror.ErrAborted, nil, verror.New(embeddedErr, nil, v...))
}

const (
	pkgPath    = "v.io/v23/rpc"
	useCall    = "  Use either rpc.ServerCall for non-streaming methods, or use a non-interface typesafe call for streaming methods."
	forgotWrap = useCall + "  Perhaps you forgot to wrap your server with the VDL-generated server stub."
)

var (
	rtPtrToContext           = reflect.TypeOf((*context.T)(nil))
	rtStreamServerCall       = reflect.TypeOf((*StreamServerCall)(nil)).Elem()
	rtServerCall             = reflect.TypeOf((*ServerCall)(nil)).Elem()
	rtGlobServerCall         = reflect.TypeOf((*GlobServerCall)(nil)).Elem()
	rtGlobChildrenServerCall = reflect.TypeOf((*GlobChildrenServerCall)(nil)).Elem()
	rtBool                   = reflect.TypeOf(bool(false))
	rtError                  = reflect.TypeOf((*error)(nil)).Elem()
	rtPtrToGlobState         = reflect.TypeOf((*GlobState)(nil))
	rtSliceOfInterfaceDesc   = reflect.TypeOf([]InterfaceDesc{})
	rtPtrToGlobGlob          = reflect.TypeOf((*glob.Glob)(nil))
	rtPtrToGlobElement       = reflect.TypeOf((*glob.Element)(nil))

	// ReflectInvoker will panic iff the error is Aborted, otherwise it will
	// silently ignore the error.

	// These errors are not embedded in other errors.
	errReflectInvokerNil   = verror.Register(pkgPath+".errReflectInvokerNil", verror.NoRetry, "{1:}{2:}rpc: ReflectInvoker(nil) is invalid{:_}")
	errNoCompatibleMethods = verror.Register(pkgPath+".errNoCompatibleMethods", verror.NoRetry, "{1:}{2:}rpc: type {3} has no compatible methods{:_}")
	errTagError            = verror.Register(pkgPath+".errTagError", verror.NoRetry, "{1:}{2:}rpc: type {3} tag error{:_}")
	errAbortedDetail       = verror.Register(pkgPath+".errAbortedDetail", verror.NoRetry, "{1:}{2:}rpc: type {3}.{4}{:_}")

	// These errors are embedded in verror.ErrInternal:
	errReservedMethod = verror.Register(pkgPath+".errReservedMethod", verror.NoRetry, "{1:}{2:}Reserved method{:_}")

	// These errors are embedded in verror.ErrBadArg:
	errMethodNotExported = verror.Register(pkgPath+".errMethodNotExported", verror.NoRetry, "{1:}{2:}Method not exported{:_}")
	errNonRPCMethod      = verror.Register(pkgPath+".errNonRPCMethod", verror.NoRetry, "{1:}{2:}Non-rpc method, at least 2 in-args are required, with first arg *context.T."+useCall+"{:_}")

	// These errors are expected to be embedded in verror.Aborted, via abortedf():
	errInStreamServerCall = verror.Register(pkgPath+".errInStreamServerCall", verror.NoRetry, "{1:}{2:}Call arg rpc.StreamServerCall is invalid; cannot determine streaming types."+forgotWrap+"{:_}")
	errNoFinalErrorOutArg = verror.Register(pkgPath+".errNoFinalErrorOutArg", verror.NoRetry, "{1:}{2:}Invalid out-args (final out-arg must be error){:_}")
	errBadDescribe        = verror.Register(pkgPath+".errBadDescribe", verror.NoRetry, "{1:}{2:}Describe__ must have signature Describe__() []rpc.InterfaceDesc{:_}")
	errBadGlobber         = verror.Register(pkgPath+".errBadGlobber", verror.NoRetry, "{1:}{2:}Globber must have signature Globber() *rpc.GlobState{:_}")
	errBadGlob            = verror.Register(pkgPath+".errBadGlob", verror.NoRetry, "{1:}{2:}Glob__ must have signature Glob__(ctx *context.T, call GlobServerCall, g *glob.Glob) error{:_}")
	errBadGlobChildren    = verror.Register(pkgPath+".errBadGlobChildren", verror.NoRetry, "{1:}{2:}GlobChildren__ must have signature GlobChildren__(ctx *context.T, call GlobChildrenServerCall, matcher *glob.Element) error{:_}")

	errNeedStreamingCall       = verror.Register(pkgPath+".errNeedStreamingCall", verror.NoRetry, "{1:}{2:}Call arg %s is invalid streaming call; must be pointer to a struct representing the typesafe streaming call."+forgotWrap+"{:_}")
	errNeedInitMethod          = verror.Register(pkgPath+".errNeedInitMethod", verror.NoRetry, "{1:}{2:}Call arg %s is invalid streaming call; must have Init method."+forgotWrap+"{:_}")
	errNeedSigFunc             = verror.Register(pkgPath+".errNeedNeedSigFunc", verror.NoRetry, "{1:}{2:}Call arg %s is invalid streaming call; Init must have signature func (*) Init(rpc.StreamServerCall)."+forgotWrap+"{:_}")
	errNeedStreamMethod        = verror.Register(pkgPath+".errNeedStreamMethod", verror.NoRetry, "{1:}{2:}Call arg %s is invalid streaming call; must have at least one of RecvStream or SendStream methods."+forgotWrap+"{:_}")
	errInvalidInStream         = verror.Register(pkgPath+".errInvalidInStream", verror.NoRetry, "{1:}{2:}Invalid in-stream type{:_}")
	errInvalidOutStream        = verror.Register(pkgPath+".errInvalidOutStream", verror.NoRetry, "{1:}{2:}Invalid out-stream type{:_}")
	errNeedRecvStreamSignature = verror.Register(pkgPath+".errNeedRecvStreamSignature", verror.NoRetry, "{1:}{2:}Call arg %s is invalid streaming call; RecvStream must have signature func (*) RecvStream() interface{ Advance() bool; Value() _; Err() error }."+forgotWrap+"{:_}")
	errNeedSendStreamSignature = verror.Register(pkgPath+".errNeedSendStreamSignature", verror.NoRetry, "{1:}{2:}Call arg %s is invalid streaming call; SendStream must have signature func (*) SendStream() interface{ Send(_) error }."+forgotWrap+"{:_}")
	errInvalidInArg            = verror.Register(pkgPath+".errInvalidInArg", verror.NoRetry, "{1:}{2:}Invalid in-arg {3} type{:_}")
	errInvalidOutArg           = verror.Register(pkgPath+".errInvalidOutArg", verror.NoRetry, "{1:}{2:}Invalid out-arg {3} type{:_}")
	errDifferentTags           = verror.Register(pkgPath+".errDifferentTags", verror.NoRetry, "{1:}{2:}different tags {3} and {4}{:_}")
	errUnknown                 = verror.Register(pkgPath+".errUnknown", verror.NoRetry, "{1:}{2:}method {3}{:_}")
)

func typeCheckMethod(method reflect.Method, sig *signature.Method) error {
	if err := typeCheckReservedMethod(method); err != nil {
		return err
	}
	// Unexported methods always have a non-empty pkg path.
	if method.PkgPath != "" {
		return verror.New(verror.ErrBadArg, nil, verror.New(errMethodNotExported, nil))
	}
	sig.Name = method.Name
	mtype := method.Type
	// Method must have at least 3 in args (receiver, ctx, call).
	if in := mtype.NumIn(); in < 3 || mtype.In(1) != rtPtrToContext {
		return verror.New(verror.ErrBadArg, nil, verror.New(errNonRPCMethod, nil))
	}
	switch in2 := mtype.In(2); {
	case in2 == rtStreamServerCall:
		// If the second call arg is rpc.StreamServerCall, we do not know whether
		// the method performs streaming, or what the stream types are.
		sig.InStream = &signature.Arg{Type: vdl.AnyType}
		sig.OutStream = &signature.Arg{Type: vdl.AnyType}
		// We can either disallow rpc.StreamServerCall, at the expense of more boilerplate
		// for users that don't use the VDL but want to perform streaming.  Or we
		// can allow it, but won't be able to determine whether the server uses the
		// stream, or what the streaming types are.
		//
		// At the moment we allow it; we can easily disallow by enabling this error.
		//   return abortedf(errInStreamServerCall)
	case in2 == rtServerCall:
		// Non-streaming method.
	case in2.Implements(rtServerCall):
		// Streaming method, validate call argument.
		if err := typeCheckStreamingCall(in2, sig); err != nil {
			return err
		}
	default:
		return verror.New(verror.ErrBadArg, nil, verror.New(errNonRPCMethod, nil))
	}
	return typeCheckMethodArgs(mtype, sig)
}

func typeCheckReservedMethod(method reflect.Method) error {
	switch method.Name {
	case "Describe__":
		// Describe__() []InterfaceDesc
		if t := method.Type; t.NumIn() != 1 || t.NumOut() != 1 ||
			t.Out(0) != rtSliceOfInterfaceDesc {
			return abortedf(errBadDescribe)
		}
		return verror.New(verror.ErrInternal, nil, verror.New(errReservedMethod, nil))
	case "Globber":
		// Globber() *GlobState
		if t := method.Type; t.NumIn() != 1 || t.NumOut() != 1 ||
			t.Out(0) != rtPtrToGlobState {
			return abortedf(errBadGlobber)
		}
		return verror.New(verror.ErrInternal, nil, verror.New(errReservedMethod, nil))
	case "Glob__":
		// Glob__(ctx *context.T, call GlobServerCall, g *glob.Glob) error
		if t := method.Type; t.NumIn() != 4 || t.NumOut() != 1 ||
			t.In(1) != rtPtrToContext || t.In(2) != rtGlobServerCall || t.In(3) != rtPtrToGlobGlob ||
			t.Out(0) != rtError {
			return abortedf(errBadGlob)
		}
		return verror.New(verror.ErrInternal, nil, verror.New(errReservedMethod, nil))
	case "GlobChildren__":
		// GlobChildren__(ctx *context.T, call GlobChildrenServerCall, matcher *glob.Element) error
		if t := method.Type; t.NumIn() != 4 || t.NumOut() != 1 ||
			t.In(1) != rtPtrToContext || t.In(2) != rtGlobChildrenServerCall || t.In(3) != rtPtrToGlobElement ||
			t.Out(0) != rtError {
			return abortedf(errBadGlobChildren)
		}
		return verror.New(verror.ErrInternal, nil, verror.New(errReservedMethod, nil))
	}
	return nil
}

func typeCheckStreamingCall(rtCall reflect.Type, sig *signature.Method) error {
	// The call must be a pointer to a struct.
	if rtCall.Kind() != reflect.Ptr || rtCall.Elem().Kind() != reflect.Struct {
		return abortedf(errNeedStreamingCall, rtCall)
	}
	// Must have Init(rpc.StreamServerCall) method.
	mInit, hasInit := rtCall.MethodByName("Init")
	if !hasInit {
		return abortedf(errNeedInitMethod, rtCall)
	}
	if t := mInit.Type; t.NumIn() != 2 || t.In(0).Kind() != reflect.Ptr || t.In(1) != rtStreamServerCall || t.NumOut() != 0 {
		return abortedf(errNeedSigFunc, rtCall)
	}
	// Must have either RecvStream or SendStream method, or both.
	mRecvStream, hasRecvStream := rtCall.MethodByName("RecvStream")
	mSendStream, hasSendStream := rtCall.MethodByName("SendStream")
	if !hasRecvStream && !hasSendStream {
		return abortedf(errNeedStreamMethod, rtCall)
	}
	if hasRecvStream {
		// func (*) RecvStream() interface{ Advance() bool; Value() _; Err() error }
		tRecv := mRecvStream.Type
		if tRecv.NumIn() != 1 || tRecv.In(0).Kind() != reflect.Ptr ||
			tRecv.NumOut() != 1 || tRecv.Out(0).Kind() != reflect.Interface {
			return abortedf(errNeedRecvStreamSignature, rtCall)
		}
		mA, hasA := tRecv.Out(0).MethodByName("Advance")
		mV, hasV := tRecv.Out(0).MethodByName("Value")
		mE, hasE := tRecv.Out(0).MethodByName("Err")
		tA, tV, tE := mA.Type, mV.Type, mE.Type
		if !hasA || !hasV || !hasE ||
			tA.NumIn() != 0 || tA.NumOut() != 1 || tA.Out(0) != rtBool ||
			tV.NumIn() != 0 || tV.NumOut() != 1 || // tV.Out(0) is in-stream type
			tE.NumIn() != 0 || tE.NumOut() != 1 || tE.Out(0) != rtError {
			return abortedf(errNeedRecvStreamSignature, rtCall)
		}
		inType, err := vdl.TypeFromReflect(tV.Out(0))
		if err != nil {
			return abortedf(errInvalidInStream, err)
		}
		sig.InStream = &signature.Arg{Type: inType}
	}
	if hasSendStream {
		// func (*) SendStream() interface{ Send(_) error }
		tSend := mSendStream.Type
		if tSend.NumIn() != 1 || tSend.In(0).Kind() != reflect.Ptr ||
			tSend.NumOut() != 1 || tSend.Out(0).Kind() != reflect.Interface {
			return abortedf(errNeedSendStreamSignature, rtCall)
		}
		mS, hasS := tSend.Out(0).MethodByName("Send")
		tS := mS.Type
		if !hasS ||
			tS.NumIn() != 1 || // tS.In(0) is out-stream type
			tS.NumOut() != 1 || tS.Out(0) != rtError {
			return abortedf(errNeedSendStreamSignature, rtCall)
		}
		outType, err := vdl.TypeFromReflect(tS.In(0))
		if err != nil {
			return abortedf(errInvalidOutStream, err)
		}
		sig.OutStream = &signature.Arg{Type: outType}
	}
	return nil
}

func typeCheckMethodArgs(mtype reflect.Type, sig *signature.Method) error {
	// Start in-args from 3 to skip receiver, ctx and call arguments.
	for index := 3; index < mtype.NumIn(); index++ {
		vdlType, err := vdl.TypeFromReflect(mtype.In(index))
		if err != nil {
			return abortedf(errInvalidInArg, index, err)
		}
		(*sig).InArgs = append((*sig).InArgs, signature.Arg{Type: vdlType})
	}
	// The out-args must contain a final error argument, which is handled
	// specially by the framework.
	if mtype.NumOut() == 0 || mtype.Out(mtype.NumOut()-1) != rtError {
		return abortedf(errNoFinalErrorOutArg)
	}
	for index := 0; index < mtype.NumOut()-1; index++ {
		vdlType, err := vdl.TypeFromReflect(mtype.Out(index))
		if err != nil {
			return abortedf(errInvalidOutArg, index, err)
		}
		(*sig).OutArgs = append((*sig).OutArgs, signature.Arg{Type: vdlType})
	}
	return nil
}

func makeSig(desc []InterfaceDesc, methods map[string]signature.Method) []signature.Interface {
	var sig []signature.Interface
	used := make(map[string]bool, len(methods))
	// Loop through the user-provided desc, attaching descriptions to the actual
	// method types to create our final signatures.  Ignore user-provided
	// descriptions of interfaces or methods that don't exist.
	for _, descIface := range desc {
		var sigMethods []signature.Method
		for _, descMethod := range descIface.Methods {
			sigMethod, ok := methods[descMethod.Name]
			if ok {
				// The method name and all types are already populated in sigMethod;
				// fill in the rest of the description.
				sigMethod.Doc = descMethod.Doc
				sigMethod.InArgs = makeArgSigs(sigMethod.InArgs, descMethod.InArgs)
				sigMethod.OutArgs = makeArgSigs(sigMethod.OutArgs, descMethod.OutArgs)
				sigMethod.InStream = fillArgSig(sigMethod.InStream, descMethod.InStream)
				sigMethod.OutStream = fillArgSig(sigMethod.OutStream, descMethod.OutStream)
				sigMethod.Tags = descMethod.Tags
				sigMethods = append(sigMethods, sigMethod)
				used[sigMethod.Name] = true
			}
		}
		if len(sigMethods) > 0 {
			sort.Sort(signature.SortableMethods(sigMethods))
			sigIface := signature.Interface{
				Name:    descIface.Name,
				PkgPath: descIface.PkgPath,
				Doc:     descIface.Doc,
				Methods: sigMethods,
			}
			for _, descEmbed := range descIface.Embeds {
				sigEmbed := signature.Embed{
					Name:    descEmbed.Name,
					PkgPath: descEmbed.PkgPath,
					Doc:     descEmbed.Doc,
				}
				sigIface.Embeds = append(sigIface.Embeds, sigEmbed)
			}
			sig = append(sig, sigIface)
		}
	}
	// Add all unused methods into the catch-all empty interface.
	var unusedMethods []signature.Method
	for _, method := range methods {
		if !used[method.Name] {
			unusedMethods = append(unusedMethods, method)
		}
	}
	if len(unusedMethods) > 0 {
		const unusedDoc = "The empty interface contains methods not attached to any interface."
		sort.Sort(signature.SortableMethods(unusedMethods))
		sig = append(sig, signature.Interface{Doc: unusedDoc, Methods: unusedMethods})
	}
	return sig
}

func makeArgSigs(sigs []signature.Arg, descs []ArgDesc) []signature.Arg {
	result := make([]signature.Arg, len(sigs))
	for index, sig := range sigs {
		if index < len(descs) {
			sig = *fillArgSig(&sig, descs[index])
		}
		result[index] = sig
	}
	return result
}

func fillArgSig(sig *signature.Arg, desc ArgDesc) *signature.Arg {
	if sig == nil {
		return nil
	}
	ret := *sig
	ret.Name = desc.Name
	ret.Doc = desc.Doc
	return &ret
}

// extractTagsForMethod returns the tags associated with the given method name.
// If the desc lists the same method under multiple interfaces, we require all
// versions to have an identical list of tags.
func extractTagsForMethod(desc []InterfaceDesc, name string) ([]*vdl.Value, error) {
	seenFirst := false
	var first []*vdl.Value
	for _, descIface := range desc {
		for _, descMethod := range descIface.Methods {
			if name == descMethod.Name {
				switch tags := descMethod.Tags; {
				case !seenFirst:
					seenFirst = true
					first = tags
				case !equalTags(first, tags):
					return nil, abortedf(errDifferentTags, first, tags)
				}
			}
		}
	}
	return first, nil
}

func equalTags(a, b []*vdl.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !vdl.EqualValue(a[i], b[i]) {
			return false
		}
	}
	return true
}

// attachMethodTags sets methodInfo.tags to the tags that will be returned in
// Prepare.  This also performs type checking on the tags.
func attachMethodTags(infos map[string]methodInfo, desc []InterfaceDesc) error {
	for name, info := range infos {
		tags, err := extractTagsForMethod(desc, name)
		if err != nil {
			return abortedf(errUnknown, name, err)
		}
		info.tags = tags
		infos[name] = info
	}
	return nil
}

// TypeCheckMethods type checks each method in obj, and returns a map from
// method name to the type check result.  Nil errors indicate the method is
// invocable by the Invoker returned by ReflectInvoker(obj).  Non-nil errors
// contain details of the type mismatch - any error with the "Aborted" id will
// cause a panic in a ReflectInvoker() call.
//
// This is useful for debugging why a particular method isn't available via
// ReflectInvoker.
func TypeCheckMethods(obj interface{}) map[string]error {
	rt, desc := reflect.TypeOf(obj), describe(obj)
	var check map[string]error
	if rt != nil && rt.NumMethod() > 0 {
		check = make(map[string]error, rt.NumMethod())
		for mx := 0; mx < rt.NumMethod(); mx++ {
			method := rt.Method(mx)
			var sig signature.Method
			err := typeCheckMethod(method, &sig)
			if err == nil {
				_, err = extractTagsForMethod(desc, method.Name)
			}
			check[method.Name] = err
		}
	}
	return check
}
