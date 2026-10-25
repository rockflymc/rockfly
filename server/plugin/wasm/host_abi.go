package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

const (
	HostModuleName = "rockfly_env"

	LogLevelInfo  = 0
	LogLevelWarn  = 1
	LogLevelError = 2
)

// registerHostABI registers all Rockfly host functions into the Wazero runtime before plugin instantiation.
func (p *WASMPlugin) registerHostABI(ctx context.Context) error {
	_, err := p.runtime.NewHostModuleBuilder(HostModuleName).
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, ptr uint32, count uint32) uint32 {
			if count == 0 {
				return 0
			}

			mem := mod.Memory()
			if mem == nil {
				return 0
			}

			byteLen := uint64(count) * BlockMutationSize
			raw, ok := mem.Read(ptr, uint32(byteLen))
			if !ok {
				fmt.Printf("[%s] [WASM ERROR] out of bounds in rockfly_commit_block_batch(ptr=%d, count=%d)\n",
					p.manifest.Name, ptr, count)
				return 0
			}

			mutations, err := DecodeBlockBatch(raw, 0, count)
			if err != nil {
				fmt.Printf("[%s] [WASM ERROR] decode failed: %v\n", p.manifest.Name, err)
				return 0
			}

			p.mu.RLock()
			handler := p.batchHandler
			p.mu.RUnlock()

			if handler != nil {
				if err := handler(mutations); err != nil {
					fmt.Printf("[%s] [WASM ERROR] batch commit handler failed: %v\n", p.manifest.Name, err)
					return 0
				}
			}

			return count
		}).
		Export("rockfly_commit_block_batch").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, x int32, y int32, z int32) uint32 {
			p.mu.RLock()
			getter := p.blockGetter
			p.mu.RUnlock()

			if getter != nil {
				return getter(x, y, z)
			}
			return 0
		}).
		Export("rockfly_get_block").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, level uint32, ptr uint32, length uint32) {
			mem := mod.Memory()
			if mem == nil {
				return
			}
			raw, ok := mem.Read(ptr, length)
			if !ok {
				return
			}
			msg := string(raw)

			switch level {
			case LogLevelWarn:
				fmt.Printf("[%s] [WARN] %s\n", p.manifest.Name, msg)
			case LogLevelError:
				fmt.Printf("[%s] [ERROR] %s\n", p.manifest.Name, msg)
			default:
				fmt.Printf("[%s] %s\n", p.manifest.Name, msg)
			}
		}).
		Export("rockfly_log").
		Instantiate(ctx)

	return err
}
