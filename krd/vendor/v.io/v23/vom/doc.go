// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vom implements the Vanadium Object Marshaling serialization format.
//
//   Concept: https://vanadium.github.io/concepts/rpc.html#vom
//   Specification: https://vanadium.github.io/designdocs/vom-spec.html
//
// VOM supports serialization of all types representable by v.io/v23/vdl, and is
// a self-describing format that retains full type information.  It is the
// underlying serialization format used by v.io/v23/rpc.
//
// The API is almost identical to encoding/gob.  To marshal objects create an
// Encoder and present it with a series of values.  To unmarshal objects create
// a Decoder and retrieve values.  The implementation creates a stream of
// messages between the Encoder and Decoder.
package vom

/*
TODO: Describe user-defined coders (VomEncode?)
TODO: Describe wire format, something like this:

Wire protocol. Version 0x80

The protocol consists of a stream of messages, where each message describes
either a type or a value.  All values are typed.  Here's the protocol grammar:
  VOM:
    (TypeMsg | ValueMsg)*
  TypeMsg:
    -typeID len(WireType) WireType
  ValueMsg:
    +typeID primitive // typeobject primitives are represented by their type id
  | +typeID len(ValueMsg) CompositeV
  Value:
    primitive |  CompositeV
  CompositeV:
    ArrayV | ListV | SetV | MapV | StructV | UnionV | OptionalV | AnyV
  ArrayV:
    len Value*len
    // len is always 0 for array since we know the exact size of the array. This
    // prefix is to ensure the decoder can distinguish NIL from the array value.
  ListV:
    len Value*len
  SetV:
    len Value*len
  MapV:
    len (Value Value)*len
  StructV:
    (index Value)* EOF  // index is the 0-based field index and
                        // zero value fields can be skipped.
  UnionV:
    index Value         // index is the 0-based field index.
  OptionalV:
    NIL
  | Value
  AnyV:
    NIL
  | +typeID Value

Wire protocol. Version 0x81

The protocol consists of a stream of messages, where each message describes
either a type or a value.  All values are typed.  Here's the protocol grammar:
  VOM:
    (TypeMsg | ValueMsg)*
  TypeMsg:
    incompleteFlag? -typeID len(WireType) WireType
  ValueMsg:
    +typeID primitive // non-typeobject primitive
  | +typeID len(RefTypes) typeID* refTypesIndex // typeobject primitive
  | +typeID len(ValueMsg) CompositeV
  | +typeID len(RefTypes) typeID* len(ValueMsg) CompositeV // message with typeobject but no any
  | +typeID len(RefTypes) typeID* len(AnyMsgLens) len(anyMsg)* len(ValueMsg) CompositeV // message with any
  Value:
    primitive |  CompositeV
  CompositeV:
    ArrayV | ListV | SetV | MapV | StructV | UnionV | OptionalV | AnyV
  ArrayV:
    len Value*len
    // len is always 0 for array since we know the exact size of the array. This
    // prefix is to ensure the decoder can distinguish NIL from the array value.
  ListV:
    len Value*len
  SetV:
    len Value*len
  MapV:
    len (Value Value)*len
  StructV:
    (index Value)* EOF  // index is the 0-based field index and
                        // zero value fields can be skipped.
  UnionV:
    index Value         // index is the 0-based field index.
  OptionalV:
    NIL
  | Value
  AnyV:
    NIL
  | refTypesIndex Value

Wire protocol. Version 0x82

The protocol consists of a stream of interleaved messages, broken
into into atomic chunks. Each message describes either a typed
value or a type definition. The grammar for chunked type and value
messages takes the following form:

  VOM:
    TypeMessage | ValueMessage
  ValueMessage:
  	TypeId MessageData |
  	WireCtrlValueFirstChunk TypeId MessageData
  	  (WireCtrlValueChunk MessageData)*
  	  WireCtrlValueLastChunk MessageData |
  	WireCtrlValueFirstChunk TypeId ReferencedTypeIds MessageData TypeMessage*
  	  (WireCtrlValueChunk ReferencedTypeIds MessageData TypeMessage*)*
  	  WireCtrlValueLastChunk ReferencedTypeIds MessageData
  TypeMessage:
    -TypeId MessageData |
    WireCtrlTypeFirstChunk -TypeId MessageData
    	(WireCtrlTypeChunk MessageData)*
    	WireCtrlValueLastChunk MessageData
  ReferencedTypeIds:
    TypeId*

The MessageData from each TypeMessage or ValueMessage is concatenated
together to form the corresponding TypeMessageBody or ValueMessageBody.
In addition, any ReferencedTypeIds that are sent in a value message are
concatenated to form the ReferencedTypeLookupTable for that message.
Here is the grammar for the contents:
  ValueMessageBody:
  	primitive | len CompositeV
  TypeMessageBody:
  	WireType (handled as a Value)
  Value:
    primitive |  CompositeV
  CompositeV:
    ArrayV | ListV | SetV | MapV | StructV | UnionV | OptionalV | AnyV
  ArrayV:
    len Value*len
    // len is always 0 for array since we know the exact size of the array. This
    // prefix is to ensure the decoder can distinguish NIL from the array value.
  ListV:
    len Value*len
  SetV:
    len Value*len
  MapV:
    len (Value Value)*len
  StructV:
    (index Value)* EOF  // index is the 0-based field index and
                        // zero value fields can be skipped.
  UnionV:
    index Value         // index is the 0-based field index.
  OptionalV:
    NIL
  | Value
  AnyV:
    NIL
  | Index into ReferencedTypeLookupTable.

TODO(toddw): We need the message lengths for fast binary->binary transcoding.

The basis for the encoding is a variable-length unsigned integer (var128), with
a max size of 128 bits (16 bytes).  This is a byte-based encoding.  The first
byte encodes values 0x00...0x7F verbatim.  Otherwise it encodes the length of
the value, and the value is encoded in the subsequent bytes in big-endian order.
In addition we have space for 112 control entries.

The var128 encoding tries to strike a balance between the coding size and
performance; we try to not be overtly wasteful of space, but still keep the
format simple to encode and decode.

  First byte of var128:
  |7|6|5|4|3|2|1|0|
  |---------------|
  |0| Single value| 0x00...0x7F Single-byte value (0...127)
  -----------------
  |1|0|x|x|x|x|x|x| 0x80...0xBF Control1 (64 entries)
  |1|1|0|x|x|x|x|x| 0xC0...0xDF Control2 (32 entries)
  |1|1|1|0|x|x|x|x| 0xE0...0xEF Control3 (16 entries)
  |1|1|1|1| Len   | 0xF0...0xFF Multi-byte length (FF=1 byte, FE=2 bytes, ...)
  -----------------             (i.e. the length is -Len)

The encoding of the value and control entries are all disjoint from each other;
each var128 can hold either a single 128 bit value, or 4 to 6 control bits. The
encoding favors small values; values less than 0x7F and control entries are all
encoded in one byte.

The primitives are all encoded using var128:
  o Unsigned: Verbatim.
  o Signed :  Low bit 0 for positive and 1 for negative, and indicates whether
              to complement the other bits to recover the signed value.
  o Float:    Byte-reversed ieee754 64-bit float.
  o Complex:  Two floats, real and imaginary.
  o String:   Byte count followed by uninterpreted bytes.

Controls are used to represent special properties and values:
  0xE0  // NIL       - represents any(nil), a non-existent value.
  0xEF  // EOF       - end of fields, e.g. used for structs
  ...
TODO(toddw): Add a flag indicating there is a local TypeID table for Any types.

The first byte of each message takes advantage of the var128 flags and reserved
entries, to make common encodings smaller, but still easy to decode.  The
assumption is that values will be encoded more frequently than types; we expect
values of the same type to be encoded repeatedly.  Conceptually the first byte
needs to distinguish TypeMsg from ValueMsg, and also tell us the TypeID.

First byte of each message:
  |7|6|5|4|3|2|1|0|
  |---------------|
  |0|0|0|0|0|0|0|0| Reserved (1 entry   0x00)
  |0|1|x|x|x|0|0|0| Reserved (8 entries 0x40, 48, 50, 58, 60, 68, 70, 78)
  |0|0|1|x|x|0|0|0| Reserved (4 entries 0x20, 28, 30, 38)
  |0|0|0|1|0|0|0|0| TypeMsg  (0x10, TypeID encoded next, then WireType)
  |0|0|0|0|1|0|0|0| ValueMsg bool false (0x08)
  |0|0|0|1|1|0|0|0| ValueMsg bool true  (0x18)
  |0| StrLen|0|1|0| ValueMsg small string len (0...15)
  |0| Uint  |1|0|0| ValueMsg small uint (0...15)
  |0| Int   |1|1|0| ValueMsg small int (-8...7)
  |0| TypeID    |1| ValueMsg (6-bit built-in TypeID)
  -----------------
  |1|0|   TypeID  | ValueMsg (6-bit user TypeID)
  |1|1|0|  Resv   | Reserved (32 entries 0xC0...0xDF)
  |1|1|1|0| Flag  | Flag     (16 entries 0xE0...0xEF)
  |1|1|1|1| Len   | Multi-byte length (FF=1 byte, FE=2 bytes, ..., F8=8 bytes)
  -----------------                   (i.e. the length is -Len)

If the first byte is 0x10, this is a TypeMsg, and we encode the TypeID next,
followed by the WireType.  The WireType is simply encoded as a regular value,
using the protocol described in the grammar above.

Otherwise this is a ValueMsg.  We encode small bool, uint and int values that
fit into 4 bits directly into the first byte, along with their TypeID.  For
small strings with len <= 15, we encode the length into the first byte, followed
by the bytes of the string value; empty strings are a single byte 0x02.

The first byte of the ValueMsg also contains TypeIDs [0...127], where the
built-in TypeIDs occupy [0...63], and user-defined TypeIDs start at 64.
User-defined TypeIDs larger than 127 are encoded as regular multi-byte var128.

TODO(toddw): For small value encoding to be useful, we'll want to use it for all
values that can fit, but we'll be dropping the sizes of int and uint, and the
type names.  Now that Union is labeled, the only issue is Any.  And now that we
have Signature with type information, maybe we can drop the type names
regularly, and only send them when the Signature says "Any".  This also impacts
where we perform value conversions - does it happen on the server or the client?

*/
