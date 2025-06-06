package wasi_snapshot_preview1_test

import (
	"bytes"
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/ZxillyFork/wazero"
	"github.com/ZxillyFork/wazero/api"
	"github.com/ZxillyFork/wazero/experimental"
	"github.com/ZxillyFork/wazero/experimental/logging"
	"github.com/ZxillyFork/wazero/imports/wasi_snapshot_preview1"
	"github.com/ZxillyFork/wazero/notinternal/testing/proxy"
	"github.com/ZxillyFork/wazero/notinternal/testing/require"
	"github.com/ZxillyFork/wazero/notinternal/wasip1"
	"github.com/ZxillyFork/wazero/sys"
)

type arbitrary struct{}

// testCtx is an arbitrary, non-default context. Non-nil also prevents linter errors.
var testCtx = context.WithValue(context.Background(), arbitrary{}, "arbitrary")

const testMemoryPageSize = 1

// exitOnStartUnstableWasm was generated by the following:
//
//	cd testdata; wat2wasm --debug-names exit_on_start_unstable.wat
//
//go:embed testdata/exit_on_start_unstable.wasm
var exitOnStartUnstableWasm []byte

func TestNewFunctionExporter(t *testing.T) {
	t.Run("export as wasi_unstable", func(t *testing.T) {
		r := wazero.NewRuntime(testCtx)
		defer r.Close(testCtx)

		// Instantiate the current WASI functions under the wasi_unstable
		// instead of wasi_snapshot_preview1.
		wasiBuilder := r.NewHostModuleBuilder("wasi_unstable")
		wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)
		_, err := wasiBuilder.Instantiate(testCtx)
		require.NoError(t, err)

		// Instantiate our test binary, but using the old import names.
		_, err = r.Instantiate(testCtx, exitOnStartUnstableWasm)

		// Ensure the test binary worked. It should return exit code 2.
		require.Equal(t, uint32(2), err.(*sys.ExitError).ExitCode())
	})

	t.Run("override function", func(t *testing.T) {
		r := wazero.NewRuntime(testCtx)
		defer r.Close(testCtx)

		// Export the default WASI functions
		wasiBuilder := r.NewHostModuleBuilder(wasi_snapshot_preview1.ModuleName)
		wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)

		// Override proc_exit to prove the point that you can add or replace
		// functions like this.
		wasiBuilder.NewFunctionBuilder().
			WithFunc(func(ctx context.Context, mod api.Module, exitCode uint32) {
				require.Equal(t, uint32(2), exitCode)
				// ignore the code instead!
				mod.Close(ctx)
			}).Export("proc_exit")

		_, err := wasiBuilder.Instantiate(testCtx)
		require.NoError(t, err)

		// Instantiate our test binary which will use our modified WASI.
		_, err = r.Instantiate(testCtx, exitOnStartWasm)

		// Ensure the modified function was used!
		require.Nil(t, err)
	})
}

// maskMemory sets the first memory in the store to '?' * size, so tests can see what's written.
func maskMemory(t *testing.T, mod api.Module, size int) {
	for i := uint32(0); i < uint32(size); i++ {
		require.True(t, mod.Memory().WriteByte(i, '?'))
	}
}

func requireProxyModule(t *testing.T, config wazero.ModuleConfig) (api.Module, api.Closer, *bytes.Buffer) {
	return requireProxyModuleWithContext(testCtx, t, config)
}

func requireProxyModuleWithContext(ctx context.Context, t *testing.T, config wazero.ModuleConfig) (api.Module, api.Closer, *bytes.Buffer) {
	var log bytes.Buffer

	// Set context to one that has an experimental listener
	ctx = experimental.WithFunctionListenerFactory(ctx,
		proxy.NewLoggingListenerFactory(&log, logging.LogScopeAll))

	r := wazero.NewRuntime(ctx)

	wasiModuleCompiled, err := wasi_snapshot_preview1.NewBuilder(r).Compile(ctx)
	require.NoError(t, err)

	_, err = r.InstantiateModule(ctx, wasiModuleCompiled, config)
	require.NoError(t, err)

	proxyBin := proxy.NewModuleBinary(wasi_snapshot_preview1.ModuleName, wasiModuleCompiled)

	proxyCompiled, err := r.CompileModule(ctx, proxyBin)
	require.NoError(t, err)

	mod, err := r.InstantiateModule(ctx, proxyCompiled, config)
	require.NoError(t, err)

	return mod, r, &log
}

// requireErrnoNosys ensures a call of the given function returns errno. The log
// message returned can verify the output is wasm `-->` or a host `==>`
// function.
func requireErrnoNosys(t *testing.T, funcName string, params ...uint64) string {
	var log bytes.Buffer

	// Set context to one that has an experimental listener
	ctx := experimental.WithFunctionListenerFactory(testCtx,
		proxy.NewLoggingListenerFactory(&log, logging.LogScopeAll))

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Instantiate the wasi module.
	wasiModuleCompiled, err := wasi_snapshot_preview1.NewBuilder(r).Compile(ctx)
	require.NoError(t, err)

	_, err = r.InstantiateModule(ctx, wasiModuleCompiled, wazero.NewModuleConfig())
	require.NoError(t, err)

	proxyBin := proxy.NewModuleBinary(wasi_snapshot_preview1.ModuleName, wasiModuleCompiled)

	proxyCompiled, err := r.CompileModule(ctx, proxyBin)
	require.NoError(t, err)

	mod, err := r.InstantiateModule(ctx, proxyCompiled, wazero.NewModuleConfig())
	require.NoError(t, err)

	requireErrnoResult(t, wasip1.ErrnoNosys, mod, funcName, params...)
	return "\n" + log.String()
}

func requireErrnoResult(t *testing.T, expectedErrno wasip1.Errno, mod api.Closer, funcName string, params ...uint64) {
	results, err := mod.(api.Module).ExportedFunction(funcName).Call(testCtx, params...)
	require.NoError(t, err)
	errno := wasip1.Errno(results[0])
	require.Equal(t, expectedErrno, errno, "want %s but have %s", wasip1.ErrnoName(expectedErrno), wasip1.ErrnoName(errno))
}

func newBlockingReader(t *testing.T) blockingReader {
	timeout, cancelFunc := context.WithTimeout(testCtx, 5*time.Second)
	t.Cleanup(cancelFunc)
	return blockingReader{ctx: timeout}
}

// blockingReader is an io.Reader that never terminates its read
// unless the embedded context is Done()
type blockingReader struct {
	ctx context.Context
}

// Read implements io.Reader
func (b blockingReader) Read(buf []byte) (n int, err error) {
	<-b.ctx.Done()
	return 0, nil
}
