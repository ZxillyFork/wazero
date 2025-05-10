package main

import (
	"testing"

	"github.com/ZxillyFork/wazero/notinternal/testing/maintester"
	"github.com/ZxillyFork/wazero/notinternal/testing/require"
)

// Test_main ensures the following will work:
//
// go run assemblyscript.go 7
func Test_main(t *testing.T) {
	stdout, stderr := maintester.TestMain(t, main, "assemblyscript", "7")
	require.Equal(t, "hello_world returned: 10", stdout)
	require.Equal(t, "sad sad world at index.ts:7:3\n", stderr)
}
