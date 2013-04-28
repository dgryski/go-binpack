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

	a := struct {
		Slice []int8 `lenprefix:"int16"`
	}{
		[]int8{0, 1, 2, 3, 4},
	}

	b := &bytes.Buffer{}

	err := Write(b, binary.LittleEndian, a)

	if err != nil {
		t.Errorf("error packing: %s", err)
	}

	if false {
		log.Printf("\n%s", hex.Dump(b.Bytes()))
	}

	var a2 struct {
		Slice []int8 `lenprefix:"int16"`
	}

	err = Read(b, binary.LittleEndian, &a2)

	if err != nil {
		t.Errorf("error unpacking: %s", err)
	}

	if !reflect.DeepEqual(a, a2) {
		t.Errorf("unpacking failed: expected: %v got: %v\n", a, a2)
	}
}
