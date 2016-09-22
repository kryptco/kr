// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

// TODO(toddw): Add tests.

import "v.io/v23/vdl"

// TODO(toddw): Provide type management routines

// Bootstrap mappings between type, id and kind.
var (
	bootstrapWireTypes map[*vdl.Type]struct{}
	bootstrapIdToType  map[TypeId]*vdl.Type
	bootstrapTypeToId  map[*vdl.Type]TypeId
	bootstrapKindToId  map[vdl.Kind]TypeId

	typeIDType        = vdl.TypeOf(TypeId(0))
	wireTypeType      = vdl.TypeOf((*wireType)(nil))
	wireNamedType     = vdl.TypeOf(wireNamed{})
	wireEnumType      = vdl.TypeOf(wireEnum{})
	wireArrayType     = vdl.TypeOf(wireArray{})
	wireListType      = vdl.TypeOf(wireList{})
	wireSetType       = vdl.TypeOf(wireSet{})
	wireMapType       = vdl.TypeOf(wireMap{})
	wireFieldType     = vdl.TypeOf(wireField{})
	wireFieldListType = vdl.TypeOf([]wireField{})
	wireStructType    = vdl.TypeOf(wireStruct{})
	wireUnionType     = vdl.TypeOf(wireUnion{})
	wireOptionalType  = vdl.TypeOf(wireOptional{})

	wireByteListType   = vdl.TypeOf([]byte{})
	wireStringListType = vdl.TypeOf([]string{})
)

func init() {
	bootstrapWireTypes = make(map[*vdl.Type]struct{})

	// The basic wire types for type definition.
	for _, tt := range []*vdl.Type{
		typeIDType,
		wireTypeType,
		wireFieldType,
		wireFieldListType,
	} {
		bootstrapWireTypes[tt] = struct{}{}
	}

	// The extra wire types for each kind of type definition. The field indices
	// in wireType should not be changed.
	wtTypes := []*vdl.Type{
		wireNamedType,
		wireEnumType,
		wireArrayType,
		wireListType,
		wireSetType,
		wireMapType,
		wireStructType,
		wireUnionType,
		wireOptionalType,
	}
	if len(wtTypes) != wireTypeType.NumField() {
		panic("vom: wireType definition changed")
	}
	for ix, tt := range wtTypes {
		if tt != wireTypeType.Field(ix).Type {
			panic("vom: wireType definition changed")
		}
		bootstrapWireTypes[tt] = struct{}{}
	}

	bootstrapIdToType = make(map[TypeId]*vdl.Type)
	bootstrapTypeToId = make(map[*vdl.Type]TypeId)
	bootstrapKindToId = make(map[vdl.Kind]TypeId)

	// The basic bootstrap types can be converted between type, id and kind.
	for id, tt := range map[TypeId]*vdl.Type{
		WireIdBool:       vdl.BoolType,
		WireIdByte:       vdl.ByteType,
		WireIdString:     vdl.StringType,
		WireIdUint16:     vdl.Uint16Type,
		WireIdUint32:     vdl.Uint32Type,
		WireIdUint64:     vdl.Uint64Type,
		WireIdInt8:       vdl.Int8Type,
		WireIdInt16:      vdl.Int16Type,
		WireIdInt32:      vdl.Int32Type,
		WireIdInt64:      vdl.Int64Type,
		WireIdFloat32:    vdl.Float32Type,
		WireIdFloat64:    vdl.Float64Type,
		WireIdTypeObject: vdl.TypeObjectType,
		WireIdAny:        vdl.AnyType,
	} {
		bootstrapIdToType[id] = tt
		bootstrapTypeToId[tt] = id
		bootstrapKindToId[tt.Kind()] = id
	}
	// The extra bootstrap types can be converted between type and id.
	for id, tt := range map[TypeId]*vdl.Type{
		WireIdByteList:   wireByteListType,
		WireIdStringList: wireStringListType,
	} {
		bootstrapIdToType[id] = tt
		bootstrapTypeToId[tt] = id
	}
}

// A generic interface for all wireType types.
type wireTypeGeneric interface {
	TypeName() string
}

func (wt wireTypeNamedT) TypeName() string    { return wt.Value.Name }
func (wt wireTypeEnumT) TypeName() string     { return wt.Value.Name }
func (wt wireTypeArrayT) TypeName() string    { return wt.Value.Name }
func (wt wireTypeListT) TypeName() string     { return wt.Value.Name }
func (wt wireTypeSetT) TypeName() string      { return wt.Value.Name }
func (wt wireTypeMapT) TypeName() string      { return wt.Value.Name }
func (wt wireTypeStructT) TypeName() string   { return wt.Value.Name }
func (wt wireTypeUnionT) TypeName() string    { return wt.Value.Name }
func (wt wireTypeOptionalT) TypeName() string { return wt.Value.Name }
