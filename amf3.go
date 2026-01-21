package goAMF3

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type Reader interface {
	Read(p []byte) (n int, err error)
}

type AvmClass struct {
	name           string
	dynamic        bool
	externalizable bool
	properties     []string
}

type AvmObject struct {
	class         *AvmClass
	staticFields  []interface{}
	dynamicFields map[string]interface{}
}

// An "Array" in AVM land is actually stored as
// a combination of an array and a dictionary.
type AvmArray struct {
	elements []interface{}
	fields   map[string]interface{}
}

type Decoder struct {
	stream     Reader
	AmfVersion uint16
	// AMF3 messages can include references to previously-unpacked objects.
	// These tables hang on to objects for later use.
	stringTable []string
	classTable  []*AvmClass
	objectTable []interface{}
	decodeError error
	// When unpacking objects, we'll look in this map for the type name.
	// If found, we'll unpack the value into an instance of the associated type.
	typeMap map[string]reflect.Type
}

func NewDecoder(stream Reader, amfVersion uint16) *Decoder {
	decoder := &Decoder{}
	decoder.stream = stream
	decoder.AmfVersion = amfVersion
	decoder.typeMap = make(map[string]reflect.Type)
	return decoder
}

func (cxt *Decoder) ReadUint8() (value uint8) {
	err := binary.Read(cxt.stream, binary.BigEndian, &value)
	cxt.saveError(err)
	return value
}

func (cxt *Decoder) ReadUint16() (value uint16) {
	err := binary.Read(cxt.stream, binary.BigEndian, &value)
	cxt.saveError(err)
	return value
}

func (cxt *Decoder) ReadUint32() (value uint32) {
	err := binary.Read(cxt.stream, binary.BigEndian, &value)
	cxt.saveError(err)
	return value
}

func (cxt *Decoder) ReadStringKnownLength(length int) string {
	data := make([]byte, length)
	n, err := cxt.stream.Read(data)
	if n < length {
		cxt.saveError(fmt.Errorf("not enough bytes in ReadStringKnownLength (expected %d, found %d)", length, n))
		return ""
	}

	cxt.saveError(err)
	return string(data)
}

func (cxt *Decoder) ReadString() string {
	length := int(cxt.ReadUint16())
	if cxt.errored() {
		return ""
	}
	return cxt.ReadStringKnownLength(length)
}

func (cxt *Decoder) errored() bool {
	return cxt.decodeError != nil
}

func (cxt *Decoder) saveError(err error) {
	if err != nil {
		if cxt.decodeError != nil {
			fmt.Println("warning: duplicate errors on Decoder")
		} else {
			cxt.decodeError = err
		}
	}
}
