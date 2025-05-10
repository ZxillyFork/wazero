package v1

import (
	"context"
	"testing"

	"github.com/ZxillyFork/wazero"
	"github.com/ZxillyFork/wazero/api"
	"github.com/ZxillyFork/wazero/notinternal/integration_test/spectest"
	"github.com/ZxillyFork/wazero/notinternal/platform"
)

func TestCompiler(t *testing.T) {
	if !platform.CompilerSupported() {
		t.Skip()
	}
	spectest.Run(t, Testcases, context.Background(), wazero.NewRuntimeConfigCompiler().WithCoreFeatures(api.CoreFeaturesV1))
}

func TestInterpreter(t *testing.T) {
	spectest.Run(t, Testcases, context.Background(), wazero.NewRuntimeConfigInterpreter().WithCoreFeatures(api.CoreFeaturesV1))
}
