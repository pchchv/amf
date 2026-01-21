package goAMF3

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	AMF3Null           byte = 0x01
	AMF3True           byte = 0x03
	AMF3Date           byte = 0x08
	AMF3False          byte = 0x02
	AMF3Array          byte = 0x09
	AMF3Double         byte = 0x05
	AMF3String         byte = 0x06
	AMF3Object         byte = 0x0a
	AMF3Dynamic        byte = 0x0b
	AMF3Integer        byte = 0x04
	AMF3Undefined      byte = 0x00
	AMF3ByteArray      byte = 0x0c
	AMF3VectorInt      byte = 0x0d
	AMF3VectorUint     byte = 0x0d
	AMF3Dictionary     byte = 0x11
	AMF3VectorDouble   byte = 0x0d
	AMF3VectorObject   byte = 0x0d
	AMF3Externalizable byte = 0x07
)

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
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

func (cxt *Decoder) RegisterType(flexName string, instance interface{}) {
	cxt.typeMap[flexName] = reflect.TypeOf(instance)
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

// Read a 29-bit compact encoded integer (as defined in AVM3).
func (cxt *Decoder) ReadUint29() (result uint32) {
	for i := 0; i < 4; i++ {
		b := cxt.ReadByte()
		if cxt.errored() {
			return 0
		}

		if i == 3 {
			// Last byte does not use the special 0x80 bit.
			result = (result << 8) + uint32(b)
		} else {
			result = (result << 7) + (uint32(b) & 0x7f)
		}

		if b&0x80 == 0 {
			break
		}
	}
	return
}

func (cxt *Decoder) ReadUint32() (value uint32) {
	err := binary.Read(cxt.stream, binary.BigEndian, &value)
	cxt.saveError(err)
	return value
}

func (cxt *Decoder) ReadFloat64() float64 {
	var value float64
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

func (cxt *Decoder) ReadByte() uint8 {
	buf := make([]byte, 1)
	_, err := cxt.stream.Read(buf)
	cxt.saveError(err)
	return buf[0]
}

func (cxt *Decoder) ReadValueAmf3() interface{} {
	// read type marker
	typeMarker := cxt.ReadByte()
	if cxt.errored() {
		return nil
	}

	switch typeMarker {
	case AMF3Null, AMF3Undefined:
		return nil
	case AMF3False:
		return false
	case AMF3True:
		return true
	case AMF3Integer:
		return cxt.ReadUint29()
	case AMF3Double:
		return cxt.ReadFloat64()
	case AMF3String:
		return cxt.readStringAmf3()
	case AMF3Externalizable:
		// TODO
	case AMF3Date:
		return cxt.readDateAmf3()
	case AMF3Object:
		return cxt.readObjectAmf3()
	case AMF3ByteArray:
		return cxt.readByteArrayAmf3()
	case AMF3Array:
		return cxt.readArrayAmf3()
	}

	cxt.saveError(fmt.Errorf("AMF3 type marker was not supported"))
	return nil
}

func (cxt *Decoder) ReadValue() interface{} {
	return cxt.ReadValueAmf3()
}

func (cxt *Decoder) storeObjectInTable(obj interface{}) {
	cxt.objectTable = append(cxt.objectTable, obj)
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

func (cxt *Decoder) readByteArrayAmf3() []byte {
	// decode the length as a U29 integer. This includes a flag in the lowest bit
	ref := cxt.ReadUint29()
	// the lowest bit is a flag; shift right to get the actual length
	length := int(ref >> 1)
	// allocate the byte array with the obtained length.
	byteArray := make([]byte, length)
	// read the byte array contents
	if n, err := cxt.stream.Read(byteArray); err != nil {
		return nil
	} else if n < length {
		// if we read fewer bytes than expected, it's an error
		return nil
	}

	return byteArray
}

func (cxt *Decoder) readStringAmf3() string {
	ref := cxt.ReadUint29()
	if cxt.errored() {
		return ""
	}

	// check the low bit to see if this is a reference
	if (ref & 1) == 0 {
		if index := int(ref >> 1); index >= len(cxt.stringTable) {
			cxt.saveError(fmt.Errorf("invalid string index: %d", index))
			return ""
		} else {
			return cxt.stringTable[index]
		}
	}

	length := int(ref >> 1)
	if length == 0 {
		return ""
	}

	str := cxt.ReadStringKnownLength(length)
	cxt.stringTable = append(cxt.stringTable, str)
	return str
}

func (cxt *Decoder) readDateAmf3() interface{} {
	// read the first U29 which includes the reference bit
	ref := cxt.ReadUint29()
	// check for error after reading U29
	if cxt.errored() {
		return time.Time{}
	}

	// сheck the low bit; for Date, we do not use object references,
	// so if the low bit is 0, it's an invalid format for a Date
	if (ref & 1) == 0 {
		cxt.saveError(fmt.Errorf("invalid date format"))
		return time.Time{}
	}

	// кead the date value in milliseconds since the Unix epoch,
	// encoded as a 64-bit floating point
	millis := cxt.ReadFloat64()

	// сonvert milliseconds to a time.Time object and return
	// Unix() method in time.Time accepts seconds and nanoseconds,
	// so convert milliseconds to nanoseconds for the second argument
	dtime := time.Unix(0, int64(millis)*int64(time.Millisecond))
	return dtime
}

func (cxt *Decoder) readArrayAmf3() interface{} {
	ref := cxt.ReadUint29()
	if cxt.errored() {
		return nil
	}

	// check the low bit to see if this is a reference
	if (ref & 1) == 0 {
		index := int(ref >> 1)
		if index >= len(cxt.objectTable) {
			cxt.saveError(fmt.Errorf("invalid array reference: %d", index))
			return nil
		}

		return cxt.objectTable[index]
	}

	elementCount := int(ref >> 1)
	// read name-value pairs, if any.
	key := cxt.readStringAmf3()
	// no name-value pairs, return a flat Go array.
	if key == "" {
		result := make([]interface{}, elementCount)
		for i := 0; i < elementCount; i++ {
			result[i] = cxt.ReadValueAmf3()
		}
		return result
	}

	result := &AvmArray{}
	result.fields = make(map[string]interface{})
	// store the object in the table before doing any decoding.
	cxt.storeObjectInTable(result)
	for key != "" {
		result.fields[key] = cxt.ReadValueAmf3()
		key = cxt.readStringAmf3()
	}

	// read dense elements
	result.elements = make([]interface{}, elementCount)
	for i := 0; i < elementCount; i++ {
		result.elements[i] = cxt.ReadValueAmf3()
	}

	return result
}

func (cxt *Decoder) readClassDefinitionAmf3(ref uint32) *AvmClass {
	// сheck for a reference to an existing class definition
	if (ref & 2) == 0 {
		return cxt.classTable[int(ref>>2)]
	}

	// parse a class definition
	className := cxt.readStringAmf3()
	externalizable := ref&4 != 0
	dynamic := ref&8 != 0
	propertyCount := ref >> 4
	class := AvmClass{className, externalizable, dynamic, make([]string, propertyCount)}
	// property names
	for i := uint32(0); i < propertyCount; i++ {
		class.properties[i] = cxt.readStringAmf3()
	}

	// save the new class in the loopup table
	cxt.classTable = append(cxt.classTable, &class)
	fmt.Printf("read class name = %s\n", class.name)
	return &class
}

func (cxt *Decoder) readObjectAmf3() interface{} {
	ref := cxt.ReadUint29()
	if cxt.errored() {
		return nil
	}

	// check the low bit to see if this is a reference
	if ref&1 == 0 {
		index := int(ref >> 1)
		if index >= len(cxt.objectTable) {
			cxt.saveError(fmt.Errorf("invalid object index: %d", index))
			return nil
		}
		return cxt.objectTable[index]
	}

	class := cxt.readClassDefinitionAmf3(ref)
	object := AvmObject{}
	object.class = class
	// for an anonymous class, just return a map[string] interface{}
	if object.class.name == "" {
		result := make(map[string]interface{})
		for prop := range class.properties {
			value := cxt.ReadValueAmf3()
			object.staticFields[prop] = value
		}

		if class.dynamic {
			for {
				if name := cxt.readStringAmf3(); name == "" {
					break
				} else {
					value := cxt.ReadValueAmf3()
					result[name] = value
				}
			}
		}
		return result
	}

	object.dynamicFields = make(map[string]interface{})
	fmt.Printf("AvmObject class name: %s\n", class.name)
	// store the object in the table before doing any decoding
	cxt.storeObjectInTable(&object)
	// read static fields
	object.staticFields = make([]interface{}, len(class.properties))
	for i := range class.properties {
		value := cxt.ReadValueAmf3()
		object.staticFields[i] = value
	}

	fmt.Printf("static fields = %v\n", object.staticFields)
	fmt.Printf("static fields = %v\n", class.properties)
	if class.dynamic {
		// parse dynamic fields
		for {
			if name := cxt.readStringAmf3(); name == "" {
				break
			} else {
				value := cxt.ReadValueAmf3()
				object.dynamicFields[name] = value
			}
		}
	}

	// If this type is registered, then unpack this result into an instance of the type
	// TODO: This could be faster if we didn't create an intermediate AvmObject.
	goType, foundGoType := cxt.typeMap[class.name]
	if foundGoType {
		result := reflect.Indirect(reflect.New(goType))
		for i := 0; i < len(class.properties); i++ {
			value := reflect.ValueOf(object.staticFields[i])
			fieldName := class.properties[i]
			fieldName = strings.ToUpper(fieldName[:1]) + fieldName[1:]
			field := result.FieldByName(fieldName)
			fmt.Printf("Attempting to write %v to field %v\n", object.staticFields[i],
				class.properties[i])
			field.Set(value)
		}
		return result.Interface()
	}

	return object
}
