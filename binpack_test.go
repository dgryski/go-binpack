package binpack

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"reflect"
	"testing"
)

func TestStructs(t *testing.T) {

	type B struct {
		Int8          byte
		Int16         int16
		EmbeddedSlice []byte `binpack:"lenprefix=int16"`
	}

	type A struct {
		Slice      []B       `binpack:"lenprefix=int16"`
		EmptySlice []int16   `binpack:"lenprefix=int16"`
		F32        []float32 `binpack:"lenprefix=uint8"`
		A4Int8     [4]int8
	}

	a := A{
		[]B{{0, 1, []byte("hello")}, {2, 3, []byte("world")}},
		[]int16{},
		[]float32{123.45, 543.21},
		[4]int8{11, 12, 13, 14},
	}

	var a2 A

	packUnpack(t, a, &a2, a)

}

func TestSkips(t *testing.T) {

	type A struct {
		Int8  uint8
		_     uint32
		Int16 int16
	}

	type B struct {
		Int8   uint8
		SkipMe uint32 `binpack:"-"`
		Int16  int16
	}

	a := A{}
	a.Int8 = 1
	a.Int16 = 2
	b := B{}
	e := B{1, 0, 2}

	packUnpack(t, a, &b, e)
}

func packUnpack(t *testing.T, s1 interface{}, s2 interface{}, e1 interface{}) {

	b := &bytes.Buffer{}

	err := Write(b, binary.LittleEndian, s1)

	if err != nil {
		t.Errorf("error packing: %s", err)
	}

	if false {
		log.Printf("\n%s", hex.Dump(b.Bytes()))
	}

	err = Read(b, binary.LittleEndian, s2)

	if err != nil {
		t.Errorf("error unpacking: %s", err)
	}

	re1 := reflect.ValueOf(e1)
	rs2 := reflect.ValueOf(s2)

	if !reflect.DeepEqual(re1.Interface(), rs2.Elem().Interface()) {
		t.Errorf("unpacking failed: expected: %v got: %v\n", s1, s2)
	}
}

func TestErrors(t *testing.T) {

	var a struct {
		S []int `binpack:"lenprefix=uint8"`
	}

	a.S = make([]int, 256)

	b := &bytes.Buffer{}
	err := Write(b, binary.LittleEndian, a)

	if err != ErrSliceTooLarge {
		t.Errorf("got wrong err: expected: ErrSliceTooLarge got: %s\n", err)
	}
}

func TestEndian(t *testing.T) {

	var expected []byte

	var bigaIn, bigaOut struct {
		Little []uint16 `binpack:"lenprefix=uint16,endian=little"`
		Big    uint32
	}

	bigaIn.Little = []uint16{0x1122, 0x3344}
	bigaIn.Big = 0x11223344

	b := &bytes.Buffer{}
	err := Write(b, binary.BigEndian, bigaIn)

	if err != nil {
		t.Errorf("error packing: %s", err)
	}

	expected = []byte{0x02, 0x00, 0x22, 0x11, 0x44, 0x33, 0x11, 0x22, 0x33, 0x44}

	if !bytes.Equal(b.Bytes(), expected) {
		t.Errorf("failed endian-tag packing: got `% 02x` expected=`% 02x`", b.Bytes(), expected)
	}

	err = Read(b, binary.BigEndian, &bigaOut)

	if err != nil {
		t.Errorf("error unpacking: %s", err)
	}

	if !reflect.DeepEqual(bigaIn, bigaOut) {
		t.Errorf("failed endian-tag unpacking: got `%v` expected=`%v`", bigaIn, bigaOut)
	}

	var littleaIn, littleaOut struct {
		Little []uint16 `binpack:"lenprefix=uint16"`
		Big    uint32   `binpack:"endian=big"`
	}

	littleaIn.Little = []uint16{0x1122, 0x3344}
	littleaIn.Big = 0x11223344

	b = &bytes.Buffer{}
	err = Write(b, binary.LittleEndian, littleaIn)

	if err != nil {
		t.Errorf("error packing: %s", err)
	}

	if !bytes.Equal(b.Bytes(), expected) {
		t.Errorf("failed endian-tag packing: got `% 02x` expected=`% 02x`", b.Bytes(), expected)
	}

	err = Read(b, binary.LittleEndian, &littleaOut)

	if err != nil {
		t.Errorf("error unpacking: %s", err)
	}

	if !reflect.DeepEqual(littleaIn, littleaOut) {
		t.Errorf("failed endian-tag unpacking: got `%v` expected=`%v`", littleaIn, littleaOut)
	}

	var a struct {
		Little []uint16 `binpack:"lenprefix=uint16,endian=little"`
		Big    uint32   `binpack:"endian=big"`
	}

	a.Little = []uint16{0x1122, 0x3344}
	a.Big = 0x11223344

	b = &bytes.Buffer{}
	err = Write(b, nil, a)

	if err != nil {
		t.Errorf("error packing: %s", err)
	}

	if !bytes.Equal(b.Bytes(), expected) {
		t.Errorf("failed endian-tag packing: got `% 02x` expected=`% 02x`", b.Bytes(), expected)
	}

}
