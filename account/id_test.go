package account

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUUIDToID(t *testing.T) {
	uuid := "2418d087-648d-4990-86e8-19dca1d006d3"
	expectedByteSlice := []byte{0x24, 0x18, 0xd0, 0x87, 0x64, 0x8d, 0x49, 0x90, 0x86, 0xe8, 0x19, 0xdc, 0xa1, 0xd0, 0x06, 0xd3}
	actualByteSlice, _ := UUIDToID(uuid)
	if cmp.Equal(actualByteSlice, expectedByteSlice) == false {
		t.Errorf("Err in Func UUIDToID, uuid is %s, want %v but get %v.", uuid, expectedByteSlice, actualByteSlice)
	}
}
