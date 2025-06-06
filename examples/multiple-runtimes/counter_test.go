package main

import (
	"testing"

	"github.com/ZxillyFork/wazero/notinternal/testing/maintester"
	"github.com/ZxillyFork/wazero/notinternal/testing/require"
)

// Test_main ensures the following will work:
//
//	go run counter.go
func Test_main(t *testing.T) {
	stdout, _ := maintester.TestMain(t, main, "counter")
	require.Equal(t, `m1 counter=0
m2 counter=0
m1 counter=1
m2 counter=1
`, stdout)
}
