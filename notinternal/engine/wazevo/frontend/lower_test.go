package frontend

import (
	"testing"
	"unsafe"

	"github.com/ZxillyFork/wazero/notinternal/testing/require"
	"github.com/ZxillyFork/wazero/notinternal/wasm"
)

func Test_Offsets(t *testing.T) {
	var memInstance wasm.MemoryInstance
	require.Equal(t, int(unsafe.Offsetof(memInstance.Buffer)), memoryInstanceBufOffset)
	var tableInstance wasm.TableInstance
	require.Equal(t, int(unsafe.Offsetof(tableInstance.References)), tableInstanceBaseAddressOffset)

	var dataInstance wasm.DataInstance
	var elementInstance wasm.ElementInstance

	require.Equal(t, int(unsafe.Sizeof(dataInstance)), elementOrDataInstanceSize)
	require.Equal(t, int(unsafe.Sizeof(elementInstance)), elementOrDataInstanceSize)
}
