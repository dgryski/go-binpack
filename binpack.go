package binpack

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"strings"
)

var ErrMissingLenPrefix = errors.New("struct with embedded slice missing value for lenprefix tag")

func Write(w io.Writer, byteorder binary.ByteOrder, data interface{}) error {

	switch data.(type) {
	case uint8, uint16, uint32, uint64, int8, int16, int32, int64:
		return binary.Write(w, byteorder, data)
	}

	v := reflect.ValueOf(data)

	switch v.Kind() {
	case reflect.Array, reflect.Slice:
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
				slen := fval.Len()
				opts := parseTag(f.Tag)
				switch opts.lenprefix {
				case "":
					return ErrMissingLenPrefix

				case "varint":
					panic("not implemented")
				case "uint8":
					Write(w, byteorder, uint8(slen))

				case "uint16":
					Write(w, byteorder, uint16(slen))

				case "uint32":
					Write(w, byteorder, uint32(slen))

				case "uint64":
					Write(w, byteorder, uint64(slen))
				case "int8":
					Write(w, byteorder, int8(slen))

				case "int16":
					Write(w, byteorder, int16(slen))

				case "int32":
					Write(w, byteorder, int32(slen))

				case "int64":
					Write(w, byteorder, int64(slen))
				default:
					return errors.New("unknown value for lenprefix: " + opts.lenprefix)
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

func Read(r io.Reader, byteorder binary.ByteOrder, data interface{}) error {

	switch data.(type) {
	case *uint8, *uint16, *uint32, *uint64, *int8, *int16, *int32, *int64:
		return binary.Read(r, byteorder, data)
	}

	v := reflect.ValueOf(data)

	return unpack(r, byteorder, v.Elem())

}

type packopts struct {
	lenprefix string
}

func parseTag(tag reflect.StructTag) packopts {
	var opts packopts

	bpTag := tag.Get("binpack")

	for _, t := range strings.Split(string(bpTag), ",") {
		if strings.HasPrefix(t, "lenprefix=") {
			opts.lenprefix = strings.TrimPrefix(t, "lenprefix=")
		}
	}

	return opts
}

func unpack(r io.Reader, byteorder binary.ByteOrder, v reflect.Value) error {

	switch v.Kind() {
	case reflect.Uint8:
		var n uint8
		binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Uint16:
		var n uint16
		binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Uint32:
		var n uint32
		binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Uint64:
		var n uint64
		binary.Read(r, byteorder, &n)
		v.SetUint(uint64(n))

	case reflect.Int8:
		var n int8
		binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Int16:
		var n int16
		binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Int32:
		var n int32
		binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Int64:
		var n int64
		binary.Read(r, byteorder, &n)
		v.SetInt(int64(n))

	case reflect.Array, reflect.Slice:
		l := v.Len()
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
			if f.PkgPath != "" {
				continue
			}

			fval := v.Field(i)

			// if we have a slice embedded in a struct, get the struct tag that tells us how to write the (unknown) length before the contents
			if f.Type.Kind() == reflect.Slice {

				var slen int
				opts := parseTag(f.Tag)
				switch opts.lenprefix {
				case "":
					return ErrMissingLenPrefix

				case "uint8":
					var n uint8
					Read(r, byteorder, &n)
					slen = int(n)

				case "uint16":
					var n uint16
					Read(r, byteorder, &n)
					slen = int(n)

				case "uint32":
					var n uint32
					Read(r, byteorder, &n)
					slen = int(n)

				case "uint64":
					var n uint64
					Read(r, byteorder, &n)
					slen = int(n)

				case "int8":
					var n int8
					Read(r, byteorder, &n)
					slen = int(n)

				case "int16":
					var n int16
					Read(r, byteorder, &n)
					slen = int(n)

				case "int32":
					var n int32
					Read(r, byteorder, &n)
					slen = int(n)

				case "int64":
					var n int64
					Read(r, byteorder, &n)
					slen = int(n)
				default:
					return errors.New("unknown value for lenprefix: " + opts.lenprefix)
				}

				if fval.IsNil() {
					slice := reflect.MakeSlice(f.Type, slen, slen)
					fval.Set(slice)
				}

				if fval.Cap() < slen {
					return errors.New("not enough space in slice")
				}
				fval.SetLen(slen) // handle case where they passed in a non-nil slice
			}

			err := unpack(r, byteorder, fval)
			if err != nil {
				return err
			}
		}

	default:
		return errors.New("unknown type: " + v.Type().Kind().String())
	}

	return nil
}
