package goAMF3

type Reader interface {
	Read(p []byte) (n int, err error)
}
