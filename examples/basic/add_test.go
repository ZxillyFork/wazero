package main

import (
	"testing"

	"github.com/ZxillyFork/wazero/notinternal/testing/maintester"
	"github.com/ZxillyFork/wazero/notinternal/testing/require"
)

// Test_main ensures the following will work:
//
//	go run add.go 7 9
func Test_main(t *testing.T) {
	stdout, _ := maintester.TestMain(t, main, "add", "7", "9")
	require.Equal(t, `7 + 9 = 16
`, stdout)
}
