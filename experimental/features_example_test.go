package experimental_test

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ZxillyFork/wazero"
	"github.com/ZxillyFork/wazero/api"
	"github.com/ZxillyFork/wazero/experimental"
	"github.com/ZxillyFork/wazero/imports/wasi_snapshot_preview1"
)

// pthreadWasm was generated by the following:
//
//	docker run -it --rm -v `pwd`/testdata:/workspace ghcr.io/webassembly/wasi-sdk:wasi-sdk-20 sh -c '$CC -o /workspace/pthread.wasm /workspace/pthread.c --target=wasm32-wasi-threads --sysroot=/wasi-sysroot -pthread -mexec-model=reactor -Wl,--export=run -Wl,--export=get'
//
// TODO: Use zig cc instead of wasi-sdk to compile when it supports wasm32-wasi-threads
// https://github.com/ziglang/zig/issues/15484
//
//go:embed testdata/pthread.wasm
var pthreadWasm []byte

//go:embed testdata/memory.wasm
var memoryWasm []byte

// This shows how to use a WebAssembly module compiled with the threads feature with wasi sdk.
func ExampleCoreFeaturesThreads() {
	// Use a default context
	ctx := context.Background()

	// Threads support must be enabled explicitly in addition to standard V2 features.
	cfg := wazero.NewRuntimeConfig().WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesThreads)

	r := wazero.NewRuntimeWithConfig(ctx, cfg)
	defer r.Close(ctx)

	wasmCompiled, err := r.CompileModule(ctx, pthreadWasm)
	if err != nil {
		log.Panicln(err)
	}

	// Because we are using wasi-sdk to compile the guest, we must initialize WASI.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	if _, err := r.InstantiateWithConfig(ctx, memoryWasm, wazero.NewModuleConfig().WithName("env")); err != nil {
		log.Panicln(err)
	}

	mod, err := r.InstantiateModule(ctx, wasmCompiled, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
	if err != nil {
		log.Panicln(err)
	}

	// Channel to synchronize start of goroutines before running.
	startCh := make(chan struct{})
	// Channel to synchronize end of goroutines.
	endCh := make(chan struct{})

	// We start up 8 goroutines and run for 100000 iterations each. The count should reach
	// 800000, at the end, but it would not if threads weren't working!
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { endCh <- struct{}{} }()
			<-startCh

			// We must instantiate a child per simultaneous thread. This should normally be pooled
			// among arbitrary goroutine invocations.
			child := createChildModule(r, mod, wasmCompiled)
			fn := child.mod.ExportedFunction("run")
			for i := 0; i < 100000; i++ {
				_, err := fn.Call(ctx)
				if err != nil {
					log.Panicln(err)
				}
			}
			runtime.KeepAlive(child)
		}()
	}
	for i := 0; i < 8; i++ {
		startCh <- struct{}{}
	}
	for i := 0; i < 8; i++ {
		<-endCh
	}

	res, err := mod.ExportedFunction("get").Call(ctx)
	if err != nil {
		log.Panicln(err)
	}
	fmt.Println(res[0])
	// Output: 800000
}

type childModule struct {
	mod        api.Module
	tlsBasePtr uint32
}

var prevTID uint32

var childModuleMu sync.Mutex

// wasi sdk maintains a stack per thread within memory, so we must allocate one separately per child
// module, corresponding to a host thread, or the stack accesses would collide. wasi sdk does not
// currently plan to implement this so we must implement it ourselves. We allocate memory for a stack,
// initialize a pthread struct at the beginning of the stack, and set globals to reference it.
// https://github.com/WebAssembly/wasi-threads/issues/45
func createChildModule(rt wazero.Runtime, root api.Module, wasmCompiled wazero.CompiledModule) *childModule {
	childModuleMu.Lock()
	defer childModuleMu.Unlock()

	ctx := context.Background()

	// Not executing function so the current stack pointer is end of stack
	stackPointer := root.ExportedGlobal("__stack_pointer").Get()
	tlsBase := root.ExportedGlobal("__tls_base").Get()

	// Thread-local-storage for the main thread is from __tls_base to __stack_pointer
	size := stackPointer - tlsBase

	malloc := root.ExportedFunction("malloc")

	// Allocate memory for the child thread stack. We go ahead and use the same size
	// as the root stack.
	res, err := malloc.Call(ctx, size)
	if err != nil {
		panic(err)
	}
	ptr := uint32(res[0])

	child, err := rt.InstantiateModule(ctx, wasmCompiled, wazero.NewModuleConfig().
		// Don't need to execute start functions again in child, it crashes anyways because
		// LLVM only allows calling them once.
		WithStartFunctions())
	if err != nil {
		panic(err)
	}

	// Call LLVM's TLS initialiation.
	initTLS := child.ExportedFunction("__wasm_init_tls")
	if _, err := initTLS.Call(ctx, uint64(ptr)); err != nil {
		panic(err)
	}

	// Populate the stack pointer and thread ID into the pthread struct at the beginning of stack.
	// This is relying on libc implementation details. The structure has been stable for a long time
	// though it is possible it could change if compiling with a different version of wasi sdk.
	tid := atomic.AddUint32(&prevTID, 1)
	child.Memory().WriteUint32Le(ptr, ptr)
	child.Memory().WriteUint32Le(ptr+20, tid)
	child.ExportedGlobal("__stack_pointer").(api.MutableGlobal).Set(uint64(ptr) + size)

	ret := &childModule{
		mod:        child,
		tlsBasePtr: ptr,
	}

	// Set a finalizer to clear the allocated stack if the module gets collected.
	runtime.SetFinalizer(ret, func(obj interface{}) {
		cm := obj.(*childModule)
		free := cm.mod.ExportedFunction("free")
		// Ignore errors since runtime may have been closed before this is called.
		_, _ = free.Call(ctx, uint64(cm.tlsBasePtr))
		_ = cm.mod.Close(context.Background())
	})
	return ret
}
