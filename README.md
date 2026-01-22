
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