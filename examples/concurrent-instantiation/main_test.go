package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ZxillyFork/wazero/notinternal/testing/maintester"
	"github.com/ZxillyFork/wazero/notinternal/testing/require"
)

// Test_main ensures the following will work:
//
//	go run main.go
func Test_main(t *testing.T) {
	stdout, _ := maintester.TestMain(t, main, "main")

	for i := 0; i < 50; i++ {
		require.True(t, strings.Contains(stdout, strconv.Itoa(i*2)))
	}
}
