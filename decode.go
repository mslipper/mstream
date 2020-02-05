package mstream

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/pkg/errors"
)

func DecodeFields(r io.Reader, items ...interface{}) error {
	for _, item := range items {
		if err := DecodeField(r, item); err != nil {
			return err
		}
	}

	return nil
}

func DecodeField(r io.Reader, item interface{}) error {
	var err error
	switch it := item.(type) {
	case Decoder:
		err = it.Decode(r)
	case *bool:
		b := make([]byte, 1, 1)
		if _, err := io.ReadFull(r, b); err != nil {
			return err
		}
		if b[0] == 0x00 {
			*it = false
		} else if b[0] == 0x01 {
			*it = true
		} else {
			return errors.Errorf("invalid boolean value: %x", b[0])
		}
	case *uint8:
		b := make([]byte, 1, 1)
		if _, err := io.ReadFull(r, b); err != nil {
			return err
		}
		*it = b[0]
	case *uint16:
		b := make([]byte, 2, 2)
		if _, err := io.ReadFull(r, b); err != nil {
			return err
		}
		*it = binary.BigEndian.Uint16(b)
	case *uint32:
		b := make([]byte, 4, 4)
		if _, err := io.ReadFull(r, b); err != nil {
			return err
		}
		*it = binary.BigEndian.Uint32(b)
	case *uint64:
		b := make([]byte, 8, 8)
		if _, err := io.ReadFull(r, b); err != nil {
			return err
		}
		*it = binary.BigEndian.Uint64(b)
	case *[]byte:
		lenB := make([]byte, 4, 4)
		if _, err := io.ReadFull(r, lenB); err != nil {
			return err
		}
		l := binary.BigEndian.Uint32(lenB)
		buf := make([]byte, l, l)
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		*it = buf
	case *string:
		var buf []byte
		if err := DecodeField(r, &buf); err != nil {
			return err
		}
		*it = string(buf)
	default:
		err = decodeReflect(r, item)
	}

	return err
}

func decodeReflect(r io.Reader, item interface{}) error {
	itemT := reflect.TypeOf(item)
	if itemT.Kind() != reflect.Ptr {
		return errors.New("can only decode into pointer types")
	}

	canonicalized := canonicalizeWellKnown(itemT.Elem())
	if wellKnownDecoders[canonicalized] != nil {
		return wellKnownDecoders[canonicalized](r, item)
	}

	itemVal := reflect.ValueOf(item)
	indirectVal := reflect.Indirect(itemVal)
	indirectT := indirectVal.Type()

	if indirectVal.Kind() == reflect.Array {
		l := indirectT.Len()
		tmp := reflect.Zero(reflect.ArrayOf(l, indirectT.Elem()))
		tmpPtr := reflect.New(indirectT)
		tmpPtr.Elem().Set(tmp)

		if indirectT.Elem().Kind() == reflect.Uint8 {
			buf := make([]byte, l, l)
			if _, err := io.ReadFull(r, buf); err != nil {
				return err
			}
			reflect.Copy(tmpPtr.Elem().Slice(0, l), reflect.ValueOf(buf))
		} else {
			for i := 0; i < indirectVal.Len(); i++ {
				if err := DecodeField(r, tmpPtr.Elem().Index(i).Addr().Interface()); err != nil {
					return err
				}
			}
		}

		itemVal.Elem().Set(tmpPtr.Elem())
		return nil
	}

	if indirectVal.Kind() == reflect.Slice {
		tmp := reflect.Zero(reflect.SliceOf(indirectT.Elem()))
		tmpPtr := reflect.New(indirectT)
		tmpPtr.Elem().Set(tmp)

		lenB := make([]byte, 4, 4)
		if _, err := io.ReadFull(r, lenB); err != nil {
			return err
		}
		l := binary.BigEndian.Uint32(lenB)

		for i := 0; i < int(l); i++ {
			sliceItem := reflect.Zero(indirectT.Elem())
			sliceItemPtr := reflect.New(indirectT.Elem())
			sliceItemPtr.Elem().Set(sliceItem)
			if err := DecodeField(r, sliceItemPtr.Interface()); err != nil {
				return err
			}
			tmpPtr.Elem().Set(reflect.Append(tmpPtr.Elem(), sliceItemPtr.Elem()))
		}

		itemVal.Elem().Set(tmpPtr.Elem())
		return nil
	}

	return errors.New(fmt.Sprintf("type %s cannot be decoded", itemT.String()))
}
