package wazevoapi

import (
	"testing"

	"github.com/ZxillyFork/wazero/notinternal/testing/require"
)

func TestExitCode_withinByte(t *testing.T) {
	require.True(t, exitCodeMax < ExitCodeMask) //nolint
}
