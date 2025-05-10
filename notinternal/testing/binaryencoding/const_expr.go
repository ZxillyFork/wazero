package binaryencoding

import (
	"github.com/ZxillyFork/wazero/notinternal/wasm"
)

func encodeConstantExpression(expr wasm.ConstantExpression) (ret []byte) {
	if expr.Opcode == wasm.OpcodeVecV128Const {
		ret = append(ret, wasm.OpcodeVecPrefix)
	}
	ret = append(ret, expr.Opcode)
	ret = append(ret, expr.Data...)
	ret = append(ret, wasm.OpcodeEnd)
	return
}
