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
