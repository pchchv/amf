package goAMF3

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
