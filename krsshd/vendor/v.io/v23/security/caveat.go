// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"v.io/v23/context"
	"v.io/v23/uniqueid"
	"v.io/v23/vdl"
	"v.io/v23/verror"
	"v.io/v23/vom"
)

var (
	errCaveatRegisteredTwice          = verror.Register(pkgPath+".errCaveatRegisteredTwice", verror.NoRetry, "{1:}{2:}Caveat with UUID {3} registered twice. Once with ({4}, fn={5}) from {6}, once with ({7}, fn={8}) from {9}{:_}")
	errBadCaveatDescriptorType        = verror.Register(pkgPath+".errBadCaveatDescriptorType", verror.NoRetry, "{1:}{2:}invalid caveat descriptor: vdl.Type({3}) cannot be converted to a Go type{:_}")
	errBadCaveatDescriptorKind        = verror.Register(pkgPath+".errBadCaveatDescriptorKind", verror.NoRetry, "{1:}{2:}invalid caveat validator: must be {3}, not {4}{:_}")
	errBadCaveatOutputNum             = verror.Register(pkgPath+".errBadCaveatOutputNum", verror.NoRetry, "{1:}{2:}invalid caveat validator: expected {3} outputs, not {4}{:_}")
	errBadCaveatOutput                = verror.Register(pkgPath+".errBadCaveatOutput", verror.NoRetry, "{1:}{2:}invalid caveat validator: output must be {3}, not {4}{:_}")
	errBadCaveatInputs                = verror.Register(pkgPath+".errBadCaveatInputs", verror.NoRetry, "{1:}{2:}invalid caveat validator: expected {3} inputs, not {4}{:_}")
	errBadCaveat1stArg                = verror.Register(pkgPath+".errBadCaveat1stArg", verror.NoRetry, "{1:}{2:}invalid caveat validator: first argument must be {3}, not {4}{:_}")
	errBadCaveat2ndArg                = verror.Register(pkgPath+".errBadCaveat2ndArg", verror.NoRetry, "{1:}{2:}invalid caveat validator: second argument must be {3}, not {4}{:_}")
	errBadCaveat3rdArg                = verror.Register(pkgPath+".errBadCaveat3rdArg", verror.NoRetry, "{1:}{2:}invalid caveat validator: third argument must be {3}, not {4}{:_}")
	errBadCaveatRestriction           = verror.Register(pkgPath+".errBadCaveatRestriction", verror.NoRetry, "{1:}{2:}could not validate embedded restriction({3}): {4}{:_}")
	errCantUnmarshalDischargeKey      = verror.Register(pkgPath+".errCantUnmarshalDischargeKey", verror.NoRetry, "{1:}{2:}invalid {3}: failed to unmarshal discharger's public key: {4}{:_}")
	errInapproriateDischargeSignature = verror.Register(pkgPath+".errInapproriateDischargeSignature", verror.NoRetry, "{1:}{2:}signature on discharge for caveat {3} was not intended for discharges(purpose={4}){:_}")
	errBadDischargeSignature          = verror.Register(pkgPath+".errBadDischargeSignature", verror.NoRetry, "{1:}{2:}signature verification on discharge for caveat {3} failed{:_}")

	dischargeSignatureCache = &sigCache{m: make(map[[sha256.Size]byte]bool)}
)

type registryEntry struct {
	desc        CaveatDescriptor
	validatorFn reflect.Value
	paramType   reflect.Type
	registerer  string
}

// Instance of unconstrained use caveat, to be used by UnconstrainedCaveat().
var unconstrainedUseCaveat Caveat

func init() {
	var err error
	unconstrainedUseCaveat, err = NewCaveat(ConstCaveat, true)
	if err != nil {
		panic(fmt.Sprintf("Error in NewCaveat: %v", err))
	}
}

// caveatRegistry is used to implement a singleton global registry that maps
// the unique id of a caveat to its validation function.
//
// It is safe to invoke methods on caveatRegistry concurrently.
type caveatRegistry struct {
	mu     sync.RWMutex
	byUUID map[uniqueid.Id]registryEntry
}

var registry = &caveatRegistry{byUUID: make(map[uniqueid.Id]registryEntry)}

func (r *caveatRegistry) register(d CaveatDescriptor, validator interface{}) error {
	_, file, line, _ := runtime.Caller(2) // one for r.register, one for RegisterCaveatValidator
	registerer := fmt.Sprintf("%s:%d", file, line)
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, exists := r.byUUID[d.Id]; exists {
		return verror.New(errCaveatRegisteredTwice, nil, d.Id, e.desc.ParamType, e.validatorFn.Interface(), e.registerer, d.ParamType, validator, registerer)
	}
	fn := reflect.ValueOf(validator)
	param := vdl.TypeToReflect(d.ParamType)
	if param == nil {
		return verror.New(errBadCaveatDescriptorType, nil, d.ParamType)
	}
	var (
		rtErr  = reflect.TypeOf((*error)(nil)).Elem()
		rtCtx  = reflect.TypeOf((*context.T)(nil))
		rtCall = reflect.TypeOf((*Call)(nil)).Elem()
	)
	if got, want := fn.Kind(), reflect.Func; got != want {
		return verror.New(errBadCaveatDescriptorKind, nil, want, got)
	}
	if got, want := fn.Type().NumOut(), 1; got != want {
		return verror.New(errBadCaveatOutputNum, nil, want, got)
	}
	if got, want := fn.Type().Out(0), rtErr; got != want {
		return verror.New(errBadCaveatOutput, nil, want, got)
	}
	if got, want := fn.Type().NumIn(), 3; got != want {
		return verror.New(errBadCaveatInputs, nil, want, got)
	}
	if got, want := fn.Type().In(0), rtCtx; got != want {
		return verror.New(errBadCaveat1stArg, nil, want, got)
	}
	if got, want := fn.Type().In(1), rtCall; got != want {
		return verror.New(errBadCaveat2ndArg, nil, want, got)
	}
	if got, want := fn.Type().In(2), param; got != want {
		return verror.New(errBadCaveat3rdArg, nil, want, got)
	}
	r.byUUID[d.Id] = registryEntry{d, fn, param, registerer}
	return nil
}

func (r *caveatRegistry) lookup(uid uniqueid.Id) (registryEntry, bool) {
	r.mu.RLock()
	entry, exists := r.byUUID[uid]
	r.mu.RUnlock()
	return entry, exists
}

func (r *caveatRegistry) validate(uid uniqueid.Id, ctx *context.T, call Call, paramvom []byte) error {
	entry, exists := r.lookup(uid)
	if !exists {
		return NewErrCaveatNotRegistered(ctx, uid)
	}
	param := reflect.New(entry.paramType).Interface()
	if err := vom.Decode(paramvom, param); err != nil {
		t, _ := vdl.TypeFromReflect(entry.paramType)
		return NewErrCaveatParamCoding(ctx, uid, t, err)
	}
	err := entry.validatorFn.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(call), reflect.ValueOf(param).Elem()})[0].Interface()
	if err == nil {
		return nil
	}
	return NewErrCaveatValidation(ctx, err.(error))
}

// RegisterCaveatValidator associates a CaveatDescriptor with the
// implementation of the validation function.
//
// The validation function must act as if the caveat was obtained from the
// remote end of the call. In particular, if the caveat is a third-party
// caveat then 'call.RemoteDischarges()' must be used to validate it.
//
// This function must be called at most once per c.ID, and will panic on duplicate
// registrations.
func RegisterCaveatValidator(c CaveatDescriptor, validator interface{}) {
	if err := registry.register(c, validator); err != nil {
		panic(err)
	}
}

// NewCaveat returns a Caveat that requires validation by the validation
// function correponding to c and uses the provided parameters.
func NewCaveat(c CaveatDescriptor, param interface{}) (Caveat, error) {
	got := vdl.TypeOf(param)
	// If the user inputs a vdl.Value, use the type of the vdl.Value instead.
	if vv, ok := param.(*vdl.Value); ok {
		got = vv.Type()
	}
	noAnyInParam := c.ParamType.Walk(vdl.WalkAll, func(t *vdl.Type) bool {
		return t.Kind() != vdl.Any
	})
	if !noAnyInParam {
		return Caveat{}, NewErrCaveatParamAny(nil, c.Id)
	}
	if want := c.ParamType; got != want {
		return Caveat{}, NewErrCaveatParamTypeMismatch(nil, c.Id, got, want)
	}
	bytes, err := vom.Encode(param)
	if err != nil {
		return Caveat{}, NewErrCaveatParamCoding(nil, c.Id, c.ParamType, err)
	}
	return Caveat{c.Id, bytes}, nil
}

// digest returns a hash of the contents of c.
func (c *Caveat) digest(hash Hash) []byte {
	return hash.sum(append(hash.sum(c.Id[:]), hash.sum(c.ParamVom)...))
}

// Validate tests if 'c' is satisfied under 'call', returning nil if it is or an
// error otherwise.
//
// It assumes that 'c' was found on a credential obtained from the remote end of
// the call. In particular, if 'c' is a third-party caveat then it uses the
// call.RemoteDischarges() to validate it.
func (c *Caveat) Validate(ctx *context.T, call Call) error {
	return registry.validate(c.Id, ctx, call, c.ParamVom)
}

// ThirdPartyDetails returns nil if c is not a third party caveat, or details about
// the third party otherwise.
func (c *Caveat) ThirdPartyDetails() ThirdPartyCaveat {
	if c.Id == PublicKeyThirdPartyCaveat.Id {
		var param publicKeyThirdPartyCaveatParam
		if err := vom.Decode(c.ParamVom, &param); err != nil {
			// TODO(jsimsa): Decide what (if any) logging mechanism to use.
			// vlog.Errorf("Error decoding PublicKeyThirdPartyCaveat: %v", err)
		}
		return &param
	}
	return nil
}

func (c Caveat) String() string {
	var param interface{}
	if err := vom.Decode(c.ParamVom, &param); err == nil {
		return fmt.Sprintf("%v(%T=%v)", c.Id, param, param)
	}
	return fmt.Sprintf("%v(%d bytes of param)", c.Id, len(c.ParamVom))
}

// UnconstrainedUse returns a Caveat implementation that never fails to
// validate. This is useful only for providing unconstrained
// blessings/discharges to another principal.
func UnconstrainedUse() Caveat {
	return unconstrainedUseCaveat
}

// NewExpiryCaveat returns a Caveat that validates iff the current time is before t.
func NewExpiryCaveat(t time.Time) (Caveat, error) {
	return NewCaveat(ExpiryCaveat, t)
}

// NewMethodCaveat returns a Caveat that validates iff the method being invoked by
// the peer is listed in an argument to this function.
func NewMethodCaveat(method string, additionalMethods ...string) (Caveat, error) {
	return NewCaveat(MethodCaveat, append(additionalMethods, method))
}

// NewPublicKeyCaveat returns a third-party caveat, i.e., the returned
// Caveat will be valid only when a discharge signed by discharger
// is issued.
//
// Location specifies the expected address at which the third-party
// service is found (and which issues discharges).
//
// The discharger will validate all provided caveats (caveat,
// additionalCaveats) before issuing a discharge.
func NewPublicKeyCaveat(discharger PublicKey, location string, requirements ThirdPartyRequirements, caveat Caveat, additionalCaveats ...Caveat) (Caveat, error) {
	param := publicKeyThirdPartyCaveatParam{
		Caveats:                append(additionalCaveats, caveat),
		DischargerLocation:     location,
		DischargerRequirements: requirements,
	}
	var err error
	if param.DischargerKey, err = discharger.MarshalBinary(); err != nil {
		return Caveat{}, err
	}
	if _, err := rand.Read(param.Nonce[:]); err != nil {
		return Caveat{}, err
	}
	c, err := NewCaveat(PublicKeyThirdPartyCaveat, param)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *publicKeyThirdPartyCaveatParam) ID() string {
	key, err := c.discharger(nil)
	if err != nil {
		// TODO(jsimsa): Decide what (if any) logging mechanism to use.
		// vlog.Error(err)
		return ""
	}
	hash := key.hash()
	bytes := append(hash.sum(c.Nonce[:]), hash.sum(c.DischargerKey)...)
	for _, cav := range c.Caveats {
		bytes = append(bytes, cav.digest(hash)...)
	}
	return base64.StdEncoding.EncodeToString(hash.sum(bytes))
}

func (c *publicKeyThirdPartyCaveatParam) Location() string { return c.DischargerLocation }
func (c *publicKeyThirdPartyCaveatParam) Requirements() ThirdPartyRequirements {
	return c.DischargerRequirements
}

func (c *publicKeyThirdPartyCaveatParam) Dischargeable(ctx *context.T, call Call) error {
	// Validate the caveats embedded within this third-party caveat.
	for _, cav := range c.Caveats {
		if err := cav.Validate(ctx, call); err != nil {
			return verror.New(errBadCaveatRestriction, ctx, cav, err)
		}
	}
	return nil
}

func (c *publicKeyThirdPartyCaveatParam) discharger(ctx *context.T) (PublicKey, error) {
	key, err := UnmarshalPublicKey(c.DischargerKey)
	if err != nil {
		return nil, verror.New(errCantUnmarshalDischargeKey, ctx, fmt.Sprintf("%T", *c), err)
	}
	return key, nil
}

func (c publicKeyThirdPartyCaveatParam) String() string {
	return fmt.Sprintf("%v@%v [%+v]", c.ID(), c.Location(), c.Requirements())
}

func (d *PublicKeyDischarge) digest(hash Hash) []byte {
	msg := hash.sum([]byte(d.ThirdPartyCaveatId))
	for _, cav := range d.Caveats {
		msg = append(msg, cav.digest(hash)...)
	}
	return hash.sum(msg)
}

func (d *PublicKeyDischarge) verify(ctx *context.T, key PublicKey) error {
	if !bytes.Equal(d.Signature.Purpose, dischargePurpose) {
		return verror.New(errInapproriateDischargeSignature, ctx, d.ThirdPartyCaveatId, d.Signature.Purpose)
	}
	digest := d.digest(key.hash())
	cachekey, err := d.signatureCacheKey(digest, key, d.Signature)
	if err == nil && dischargeSignatureCache.verify(cachekey) {
		return nil
	}
	if !d.Signature.Verify(key, digest) {
		return verror.New(errBadDischargeSignature, ctx, d.ThirdPartyCaveatId)
	}
	dischargeSignatureCache.cache([][]byte{cachekey})
	return nil
}

func (d *PublicKeyDischarge) signatureCacheKey(digest []byte, key PublicKey, signature Signature) ([]byte, error) {
	// Every "argument" to signature verification must make it into the cache key.
	keybytes, err := key.MarshalBinary()
	if err != nil {
		return nil, err
	}
	keyhash := key.hash().sum(keybytes)
	sighash := signature.digest(key.hash())
	return append(keyhash[:], append(sighash, digest...)...), nil
}

func (d *PublicKeyDischarge) sign(signer Signer) error {
	var err error
	d.Signature, err = signer.Sign(dischargePurpose, d.digest(signer.PublicKey().hash()))
	return err
}

func (d *PublicKeyDischarge) String() string {
	return fmt.Sprint(*d)
}
