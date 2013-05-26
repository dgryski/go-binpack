/*
Copyright (c) 2013, Damian Gryski <damian@gryski.com>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

   * Redistributions of source code must retain the above copyright notice,
this list of conditions and the following disclaimer.

   * Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/


/*

Package binpack implements translation between numbers and byte
sequences, similar to encoding/binary.  This package allows
variable-length slices contained in structs to be packed and unpacked
by adding a length to the byte stream.  The type of integer to use for
the length is specified in the struct tag:

For example,

   Field []int32 `binpack:"lenprefix=int8"`

indicates that during serialization, the value of Field will be
preceeded by an int8 indicating how many elements follow.

Valid prefix types are: int8, uint8, int16, uint16, int32, uint32,
int64, uint64.

The pack and unpack routines skip fields with blank (_) field names
and those for which the struct tag is "-"

Note that slices may only be contained in a struct, not in other
slices. This means that a declaration such as:

   Field [][]byte `binpack:"lenprefix=uint8"`

will fail to serialize properly.  However, since structs are encoded
with zero padding, an equivalent declaration is:

   Field []struct { InnerField []byte `binpack:"lenprefix="uint16"` } `binpack:"lenprefix="uint8"`

The encoding of Field will be a uint8 indicating how many inner slices
there are, and the inner []byte slices will be length-prefixed with a
uint16

*/
package binpack

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
)

var ErrMissingLenPrefix = errors.New("struct with embedded slice missing value for lenprefix tag")
var ErrUnknownLenPrefix = errors.New("unknown lenprefix pack type")
var ErrSliceTooSmall = errors.New("not enough space in slice")
var ErrSliceTooLarge = errors.New("slice too large for lenprefix type")

var maxSliceLen = map[string]uint64{
	"uint8":  math.MaxUint8,
	"uint16": math.MaxUint16,
	"uint32": math.MaxUint32,
	"uint64": math.MaxUint64,
	"int8":   math.MaxInt8,
	"int16":  math.MaxInt16,
	"int32":  math.MaxInt32,
	"int64":  math.MaxInt64,
}

// Write writes the binary representation of data to w.
func Write(w io.Writer, byteorder binary.ByteOrder, data interface{}) error {

	switch data.(type) {
	case uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
		return binary.Write(w, byteorder, data)
	}

	v := reflect.ValueOf(data)

	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			_, err := w.Write(v.Bytes())
			return err
		}

		l := v.Len()
		for i := 0; i < l; i++ {
			err := Write(w, byteorder, v.Index(i).Interface())
			if err != nil {
				return err
			}
		}

	case reflect.Struct:
		// write all public fields in order
		typ := v.Type()
		l := typ.NumField()
		for i := 0; i < l; i++ {
			f := typ.Field(i)
			if f.PkgPath != "" {
				continue
			}

			fval := v.Field(i)

			// if we have a slice embedded in a struct, get the struct tag that tells us how to write the (unknown) length before the contents
			if f.Type.Kind() == reflect.Slice {
				slen := uint64(fval.Len())
				opts := parseTag(f.Tag)

				if opts.lenprefix == "" {
					return ErrMissingLenPrefix
				}

				maxlen, ok := maxSliceLen[opts.lenprefix]
				if !ok {
					return ErrUnknownLenPrefix
				}

				if slen > maxlen {
					return ErrSliceTooLarge
				}

				var err error
				switch opts.lenprefix {
				case "uint8":
					err = binary.Write(w, byteorder, uint8(slen))

				case "uint16":
					err = binary.Write(w, byteorder, uint16(slen))

				case "uint32":
					err = binary.Write(w, byteorder, uint32(slen))

				case "uint64":
					err = binary.Write(w, byteorder, uint64(slen))

				case "int8":
					err = binary.Write(w, byteorder, int8(slen))

				case "int16":
					err = binary.Write(w, byteorder, int16(slen))

				case "int32":
					err = binary.Write(w, byteorder, int32(slen))

				case "int64":
					err = binary.Write(w, byteorder, int64(slen))
				}

				if err != nil {
					return err
				}
			}

			err := Write(w, byteorder, v.Field(i).Interface())
			if err != nil {
				return err
			}
		}
	}

	return nil

}

// Read reads structured binary data from r into data.
func Read(r io.Reader, byteorder binary.ByteOrder, data interface{}) error {

	switch data.(type) {
	case *uint8, *uint16, *uint32, *uint64, *int8, *int16, *int32, *int64, *float32, *float64:
		return binary.Read(r, byteorder, data)
	}

	v := reflect.ValueOf(data)

	return unpack(r, byteorder, v.Elem())

}

type packopts struct {
	skip      bool
	lenprefix string
}

func parseTag(tag reflect.StructTag) packopts {
	var opts packopts

	bpTag := tag.Get("binpack")

	for _, t := range strings.Split(string(bpTag), ",") {
		if t == "-" {
			opts.skip = true
		}
		if strings.HasPrefix(t, "lenprefix=") {
			opts.lenprefix = strings.TrimPrefix(t, "lenprefix=")
		}
	}

	return opts
}

func unpack(r io.Reader, byteorder binary.ByteOrder, v reflect.Value) error {

	var err error

	switch v.Kind() {
	case reflect.Uint8:
		var n uint8
		err = binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Uint16:
		var n uint16
		err = binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Uint32:
		var n uint32
		err = binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Uint64:
		var n uint64
		err = binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Int8:
		var n int8
		err = binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Int16:
		var n int16
		err = binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Int32:
		var n int32
		err = binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Int64:
		var n int64
		err = binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Float32:
		var n uint32
		err = binary.Read(r, byteorder, &n)
		v.SetFloat(float64(math.Float32frombits(n)))

	case reflect.Float64:
		var n uint64
		err = binary.Read(r, byteorder, &n)
		v.SetFloat(math.Float64frombits(n))

	case reflect.Array, reflect.Slice:
		l := v.Len()
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			b := make([]byte, l, l)
			_, err := r.Read(b)
			if err == nil {
				v.SetBytes(b)
			}
			return err
		}
		for i := 0; i < l; i++ {
			err := unpack(r, byteorder, v.Index(i))
			if err != nil {
				return err
			}
		}

	case reflect.Struct:
		// write all public fields in order
		typ := v.Type()
		l := typ.NumField()
		for i := 0; i < l; i++ {
			f := typ.Field(i)

			opts := parseTag(f.Tag)

			if opts.skip || f.Name == "_" || f.PkgPath != "" {
				continue
			}

			fval := v.Field(i)

			// if we have a slice embedded in a struct, get the struct tag that tells us how to write the (unknown) length before the contents
			if f.Type.Kind() == reflect.Slice {

				var slen int
				switch opts.lenprefix {
				case "":
					return ErrMissingLenPrefix

				case "uint8":
					var n uint8
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "uint16":
					var n uint16
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "uint32":
					var n uint32
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "uint64":
					var n uint64
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "int8":
					var n int8
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "int16":
					var n int16
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "int32":
					var n int32
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				case "int64":
					var n int64
					err = binary.Read(r, byteorder, &n)
					slen = int(n)

				default:
					err = ErrUnknownLenPrefix
				}

				if err != nil {
					return err
				}

				if fval.IsNil() {
					slice := reflect.MakeSlice(f.Type, slen, slen)
					fval.Set(slice)
				}

				if fval.Cap() < slen {
					return ErrSliceTooSmall
				}

				fval.SetLen(slen) // handle case where they passed in a non-nil slice
			}

			err = unpack(r, byteorder, fval)
			if err != nil {
				return err
			}
		}

	default:
		return errors.New("unknown type: " + v.Type().Kind().String())
	}

	return err
}
