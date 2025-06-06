package experimental_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/ZxillyFork/wazero"
	"github.com/ZxillyFork/wazero/api"
	"github.com/ZxillyFork/wazero/experimental"
	"github.com/ZxillyFork/wazero/imports/wasi_snapshot_preview1"
)

var (
	// These wasm files were generated by the following:
	// cd testdata
	// wat2wasm --debug-names inoutdispatcher.wat
	// wat2wasm --debug-names inoutdispatcherclient.wat

	//go:embed testdata/inoutdispatcher.wasm
	inoutdispatcherWasm []byte
	//go:embed testdata/inoutdispatcherclient.wasm
	inoutdispatcherclientWasm []byte
)

func Example_importResolver() {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// The client imports the inoutdispatcher module that reads from stdin and writes to stdout.
	// This means that we need multiple instances of the inoutdispatcher module to have different stdin/stdout.
	// This example demonstrates a way to do that.
	type mod struct {
		in  bytes.Buffer
		out bytes.Buffer

		client api.Module
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	idm, err := r.CompileModule(ctx, inoutdispatcherWasm)
	if err != nil {
		log.Panicln(err)
	}
	idcm, err := r.CompileModule(ctx, inoutdispatcherclientWasm)
	if err != nil {
		log.Panicln(err)
	}

	const numInstances = 3
	mods := make([]*mod, numInstances)
	for i := range mods {
		mods[i] = &mod{}
		m := mods[i]

		const inoutDispatcherModuleName = "inoutdispatcher"

		dispatcherInstance, err := r.InstantiateModule(ctx, idm,
			wazero.NewModuleConfig().
				WithStdin(&m.in).
				WithStdout(&m.out).
				WithName("")) // Makes it an anonymous module.
		if err != nil {
			log.Panicln(err)
		}

		ctx = experimental.WithImportResolver(ctx, func(name string) api.Module {
			if name == inoutDispatcherModuleName {
				return dispatcherInstance
			}
			return nil
		})

		m.client, err = r.InstantiateModule(ctx, idcm, wazero.NewModuleConfig().WithName(fmt.Sprintf("m%d", i)))
		if err != nil {
			log.Panicln(err)
		}

	}

	for i, m := range mods {
		m.in.WriteString(fmt.Sprintf("Module instance #%d", i))
		_, err := m.client.ExportedFunction("dispatch").Call(ctx)
		if err != nil {
			log.Panicln(err)
		}
	}

	for i, m := range mods {
		fmt.Printf("out%d: %s\n", i, m.out.String())
	}

	// Output:
	// out0: Module instance #0
	// out1: Module instance #1
	// out2: Module instance #2
}
