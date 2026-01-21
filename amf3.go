package goAMF3

import "reflect"

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
