package mstream

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
)

var (
	trueWire  = []byte{0x01}
	falseWire = []byte{0x00}
)

func EncodeFields(w io.Writer, items ...interface{}) error {
	for _, item := range items {
		if err := EncodeField(w, item); err != nil {
			return err
		}
	}

	return nil
}

func EncodeField(w io.Writer, item interface{}) error {
	var err error
	switch it := item.(type) {
	case Encoder:
		err = it.Encode(w)
	case bool:
		val := falseWire
		if it {
			val = trueWire
		}
		_, err = w.Write(val)
	case uint8:
		_, err = w.Write([]byte{it})
	case uint16:
		b := make([]byte, 2, 2)
		binary.BigEndian.PutUint16(b, it)
		_, err = w.Write(b)
	case uint32:
		b := make([]byte, 4, 4)
		binary.BigEndian.PutUint32(b, it)
		_, err = w.Write(b)
	case uint64:
		b := make([]byte, 8, 8)
		binary.BigEndian.PutUint64(b, it)
		_, err = w.Write(b)
	case []byte:
		if len(it) > math.MaxUint32 {
			return errors.New("variable-length field too large to encode")
		}
		lenB := make([]byte, 4, 4)
		binary.BigEndian.PutUint32(lenB, uint32(len(it)))
		if _, err := w.Write(lenB); err != nil {
			return err
		}
		_, err = w.Write(it)
	case string:
		err = EncodeField(w, []byte(item.(string)))
	default:
		err = encodeReflect(w, item)
	}

	return err
}

func encodeReflect(w io.Writer, item interface{}) error {
	t := reflect.TypeOf(item)

	canonicalized := canonicalizeWellKnown(t)
	if wellKnownEncoders[canonicalized] != nil {
		return wellKnownEncoders[canonicalized](w, item)
	}

	if t.Kind() == reflect.Array {
		itemVal := reflect.ValueOf(item)
		if t.Elem().Kind() == reflect.Uint8 {
			itemPtr := reflect.New(t)
			itemPtr.Elem().Set(itemVal)
			_, err := w.Write(itemPtr.Elem().Slice(0, itemVal.Len()).Bytes())
			return err
		}

		for i := 0; i < itemVal.Len(); i++ {
			if err := EncodeField(w, itemVal.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	}

	if t.Kind() == reflect.Slice {
		val := reflect.ValueOf(item)
		if val.Len() > math.MaxUint32 {
			return errors.New("number of array elements too large to encode")
		}

		itemCount := make([]byte, 4, 4)
		binary.BigEndian.PutUint32(itemCount, uint32(val.Len()))
		if _, err := w.Write(itemCount); err != nil {
			return err
		}

		for i := 0; i < val.Len(); i++ {
			if err := EncodeField(w, val.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	}

	return errors.New(fmt.Sprintf("type %s cannot be encoded", t.String()))
}
