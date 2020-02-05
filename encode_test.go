package mstream

import (
	"bytes"
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEncodeFields(t *testing.T) {
	cafe := new(cafeEncodeDecoder)

	var threeTwoByte [32]byte
	threeTwoByte[1] = 0xff

	var buf bytes.Buffer
	require.NoError(t, EncodeFields(
		&buf,
		cafe,
		true,
		false,
		uint8(0),
		uint16(0),
		uint32(0),
		uint64(0),
		uint8(math.MaxUint8),
		uint16(math.MaxUint16),
		uint32(math.MaxUint32),
		uint64(math.MaxUint64),
		threeTwoByte,
		[2]string{
			"testing",
			"testing",
		},
		[]byte{
			0x01, 0x02,
		},
		[]string{
			"testing",
			"testing",
		},
		"hello there",
		time.Unix(1, 0),
	))
	require.Equal(
		t,
		"cafe"+
			"01"+
			"00"+
			"00"+
			"0000"+
			"00000000"+
			"0000000000000000"+
			"ff"+
			"ffff"+
			"ffffffff"+
			"ffffffffffffffff"+
			"00ff000000000000000000000000000000000000000000000000000000000000"+
			"0000000774657374696e670000000774657374696e67"+
			"000000020102"+
			"000000020000000774657374696e670000000774657374696e67"+
			"0000000b68656c6c6f207468657265"+
			"0000000000000001",
		hex.EncodeToString(buf.Bytes()),
	)

	buf.Reset()
	err := EncodeFields(&buf, uint8(1), struct{}{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be encoded")
}

func TestEncode_Errors(t *testing.T) {
	rw := new(NopReadWriter)
	err := EncodeField(rw, &struct{}{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be encoded")
}
