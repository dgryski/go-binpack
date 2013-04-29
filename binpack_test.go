package binpack

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"reflect"
	"testing"
)

func TestStruct(t *testing.T) {

	type B struct {
		Int8          byte
		Int16         int16
		EmbeddedSlice []byte `binpack:"lenprefix=int16"`
	}

	type A struct {
		Slice      []B       `binpack:"lenprefix=int16"`
		EmptySlice []int16   `binpack:"lenprefix=int16"`
		F32        []float32 `binpack:"lenprefix=uint8"`
	}

	a := A{
		[]B{{0, 1, []byte("hello")}, {2, 3, []byte("world")}},
		[]int16{},
		[]float32{123.45, 543.21},
	}

	b := &bytes.Buffer{}

	err := Write(b, binary.LittleEndian, a)

	if err != nil {
		t.Errorf("error packing: %s", err)
	}

	if false {
		log.Printf("\n%s", hex.Dump(b.Bytes()))
	}

	var a2 A

	err = Read(b, binary.LittleEndian, &a2)

	if err != nil {
		t.Errorf("error unpacking: %s", err)
	}

	if !reflect.DeepEqual(a, a2) {
		t.Errorf("unpacking failed: expected: %v got: %v\n", a, a2)
	}
}
