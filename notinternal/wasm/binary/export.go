package binary

import (
	"bytes"
	"fmt"

	"github.com/ZxillyFork/wazero/notinternal/leb128"
	"github.com/ZxillyFork/wazero/notinternal/wasm"
)

func decodeExport(r *bytes.Reader, ret *wasm.Export) (err error) {
	if ret.Name, _, err = decodeUTF8(r, "export name"); err != nil {
		return
	}

	b, err := r.ReadByte()
	if err != nil {
		err = fmt.Errorf("error decoding export kind: %w", err)
		return
	}

	ret.Type = b
	switch ret.Type {
	case wasm.ExternTypeFunc, wasm.ExternTypeTable, wasm.ExternTypeMemory, wasm.ExternTypeGlobal:
		if ret.Index, _, err = leb128.DecodeUint32(r); err != nil {
			err = fmt.Errorf("error decoding export index: %w", err)
		}
	default:
		err = fmt.Errorf("%w: invalid byte for exportdesc: %#x", ErrInvalidByte, b)
	}
	return
}
