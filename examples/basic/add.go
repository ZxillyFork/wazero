package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/ZxillyFork/wazero"
	"github.com/ZxillyFork/wazero/imports/wasi_snapshot_preview1"
)

// addWasm was generated by the following:
//
//	cd testdata; tinygo build -o add.wasm -target=wasi add.go
//
//go:embed testdata/add.wasm
var addWasm []byte

// main is an example of how to extend a Go application with an addition
// function defined in WebAssembly.
//
// Since addWasm was compiled with TinyGo's `wasi` target, we need to configure
// WASI host imports.
func main() {
	// Parse positional arguments.
	flag.Parse()

	// Choose the context to use for function calls.
	ctx := context.Background()

	// Create a new WebAssembly Runtime.
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx) // This closes everything this Runtime created.

	// Instantiate WASI, which implements host functions needed for TinyGo to
	// implement `panic`.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Instantiate the guest Wasm into the same runtime. It exports the `add`
	// function, implemented in WebAssembly.
	mod, err := r.InstantiateWithConfig(ctx, addWasm, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
	if err != nil {
		log.Panicf("failed to instantiate module: %v", err)
	}

	// Read two args to add.
	x, y, err := readTwoArgs(flag.Arg(0), flag.Arg(1))
	if err != nil {
		log.Panicf("failed to read arguments: %v", err)
	}

	// Call the `add` function and print the results to the console.
	add := mod.ExportedFunction("add")
	results, err := add.Call(ctx, x, y)
	if err != nil {
		log.Panicf("failed to call add: %v", err)
	}

	fmt.Printf("%d + %d = %d\n", x, y, results[0])
}

func readTwoArgs(xs, ys string) (uint64, uint64, error) {
	if xs == "" || ys == "" {
		return 0, 0, errors.New("must specify two command line arguments")
	}

	x, err := strconv.ParseUint(xs, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("argument X: %v", err)
	}

	y, err := strconv.ParseUint(ys, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("argument Y: %v", err)
	}

	return x, y, nil
}
