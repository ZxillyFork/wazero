package frontend

import (
	"github.com/ZxillyFork/wazero/notinternal/engine/wazevo/ssa"
	"github.com/ZxillyFork/wazero/notinternal/wasm"
)

func FunctionIndexToFuncRef(idx wasm.Index) ssa.FuncRef {
	return ssa.FuncRef(idx)
}
