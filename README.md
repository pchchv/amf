# amf [![Godoc Reference](https://pkg.go.dev/badge/github.com/pchchv/amf)](https://pkg.go.dev/github.com/pchchv/amf) [![Go Report Card](https://goreportcard.com/badge/github.com/pchchv/amf)](https://goreportcard.com/report/github.com/pchchv/amf)

Golang implementation of [AMF3](https://en.wikipedia.org/wiki/Action_Message_Format).

Encodes and decodes all the primitive, as well as dynamic Objects, such as:
```
{
    abc: "abc"
    ,dtl: new Date
    ,num: Number(123.456789)
    ,arr: [0,1,2,"lol",3]
    ,int: 1337
}
```