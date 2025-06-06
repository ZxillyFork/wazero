package ssa

import (
	"testing"

	"github.com/ZxillyFork/wazero/notinternal/testing/require"
)

func TestInstruction_InvertConditionalBrx(t *testing.T) {
	i := &Instruction{opcode: OpcodeBrnz}
	i.InvertBrx()
	require.Equal(t, OpcodeBrz, i.opcode)
	i.InvertBrx()
	require.Equal(t, OpcodeBrnz, i.opcode)
}
