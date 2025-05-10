package experimental_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ZxillyFork/wazero"
	"github.com/ZxillyFork/wazero/api"
	"github.com/ZxillyFork/wazero/experimental"
	"github.com/ZxillyFork/wazero/notinternal/testing/binaryencoding"
	"github.com/ZxillyFork/wazero/notinternal/testing/require"
	"github.com/ZxillyFork/wazero/notinternal/wasm"
)

func TestImportResolver(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	for i := 0; i < 5; i++ {
		var callCount int
		start := func(ctx context.Context) {
			callCount++
		}
		modImport, err := r.NewHostModuleBuilder(fmt.Sprintf("env%d", i)).
			NewFunctionBuilder().WithFunc(start).Export("start").
			Compile(ctx)
		require.NoError(t, err)
		// Anonymous module, it will be resolved by the import resolver.
		instanceImport, err := r.InstantiateModule(ctx, modImport, wazero.NewModuleConfig().WithName(""))
		require.NoError(t, err)

		resolveImport := func(name string) api.Module {
			if name == "env" {
				return instanceImport
			}
			return nil
		}

		// Set the import resolver in the context.
		ctx = experimental.WithImportResolver(context.Background(), resolveImport)

		one := uint32(1)
		binary := binaryencoding.EncodeModule(&wasm.Module{
			TypeSection:     []wasm.FunctionType{{}},
			ImportSection:   []wasm.Import{{Module: "env", Name: "start", Type: wasm.ExternTypeFunc, DescFunc: 0}},
			FunctionSection: []wasm.Index{0},
			CodeSection: []wasm.Code{
				{Body: []byte{wasm.OpcodeCall, 0, wasm.OpcodeEnd}}, // Call the imported env.start.
			},
			StartSection: &one,
		})

		modMain, err := r.CompileModule(ctx, binary)
		require.NoError(t, err)

		_, err = r.InstantiateModule(ctx, modMain, wazero.NewModuleConfig())
		require.NoError(t, err)
		require.Equal(t, 1, callCount)
	}
}
