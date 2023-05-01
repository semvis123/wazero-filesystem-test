package main

import (
	"context"
	"log"
	"os"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/logging"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed filesystem_test.wasm
var binary []byte

var debug = true

func main() {
	ctx := context.Background()
	if debug {
		ctx = context.WithValue(ctx, experimental.FunctionListenerFactoryKey{}, logging.NewLoggingListenerFactory(os.Stdout))
	}
	r := wazero.NewRuntime(ctx)
	compiled, err := r.CompileModule(ctx, binary)
	if err != nil {
		log.Panicf("failed to compile module: %v", err)
	}
	builder := r.NewHostModuleBuilder("env")
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	emscripten.NewFunctionExporter().ExportFunctions(builder)
	builder.NewFunctionBuilder().WithFunc(func(_ int32) int32 { return 0 }).Export("system")
	_, err = builder.Instantiate(ctx)
	if err != nil {
		log.Panicf("failed to instantiate module: %v", err)
	}

	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithStartFunctions("_initialize").
		WithFSConfig(wazero.NewFSConfig().WithDirMount("/", "/")).
		WithStderr(os.Stderr).
		WithStdout(os.Stdout))
	if err != nil {
		log.Panicf("failed to instantiate module: %v", err)
	}

	file := "/this-is-not-a-file"
	strPtrR, err := mod.ExportedFunction("malloc").Call(ctx, uint64(len(file)+1))
	if err != nil {
		log.Panicf("failed to call malloc")
	}
	strPtr := strPtrR[0]

	defer mod.ExportedFunction("free").Call(ctx, strPtr)
	mod.Memory().WriteString(uint32(strPtr), file)

	results, err := mod.ExportedFunction("FileExists").Call(ctx, strPtr)
	if err != nil {
		log.Fatalf("could not call file exists: %s", err)
	}
	if results[0] == 1 {
		log.Println("file exists")
	} else {
		log.Println("file does not exist")
	}
}
