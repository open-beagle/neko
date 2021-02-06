package capture

import (
	"fmt"
	"sync"
	"reflect"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"demodesk/neko/internal/types"
	"demodesk/neko/internal/types/codec"
	"demodesk/neko/internal/capture/gst"
)

type StreamManagerCtx struct {
	logger       zerolog.Logger
	mu           sync.Mutex
	codec        codec.RTPCodec
	pipelineStr  string
	pipeline     *gst.Pipeline
	sample       chan types.Sample
	listeners    map[uintptr]func(sample types.Sample)
	emitMu       sync.Mutex
	emitUpdate   chan bool
	emitStop     chan bool
	started      bool
}

func streamNew(codec codec.RTPCodec, pipelineStr string) *StreamManagerCtx {
	manager := &StreamManagerCtx{
		logger:       log.With().Str("module", "capture").Str("submodule", "stream").Logger(),
		codec:        codec,
		pipelineStr:  pipelineStr,
		listeners:    map[uintptr]func(sample types.Sample){},
		emitUpdate:   make(chan bool),
		emitStop:     make(chan bool),
		started:      false,
	}

	go func() {
		manager.logger.Debug().Msg("started emitting samples")

		for {
			select {
			case <-manager.emitStop:
				manager.logger.Debug().Msg("stopped emitting samples")
				return
			case <-manager.emitUpdate:
				manager.logger.Debug().Msg("update emitting samples")
			case sample := <-manager.sample:
				manager.emitMu.Lock()
				for _, emit := range manager.listeners {
					emit(sample)
				}
				manager.emitMu.Unlock()
			}
		}
	}()

	return manager
}

func (manager *StreamManagerCtx) shutdown() {
	manager.logger.Info().Msgf("shutting down")

	manager.destroyPipeline()
	manager.emitStop <- true
}

func (manager *StreamManagerCtx) Codec() codec.RTPCodec {
	return manager.codec
}

func (manager *StreamManagerCtx) AddListener(listener func(sample types.Sample)) {
	manager.emitMu.Lock()
	defer manager.emitMu.Unlock()

	ptr := reflect.ValueOf(listener).Pointer()
	manager.listeners[ptr] = listener
}

func (manager *StreamManagerCtx) RemoveListener(listener func(sample types.Sample)) {
	manager.emitMu.Lock()
	defer manager.emitMu.Unlock()

	ptr := reflect.ValueOf(listener).Pointer()
	delete(manager.listeners, ptr)
}

func (manager *StreamManagerCtx) Start() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	err := manager.createPipeline()
	if err != nil {
		return err
	}

	manager.started = true
	return nil
}

func (manager *StreamManagerCtx) Stop() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.started = false
	manager.destroyPipeline()
}

func (manager *StreamManagerCtx) Started() bool {
	return manager.started
}

func (manager *StreamManagerCtx) createPipeline() error {
	if manager.pipeline != nil {
		return fmt.Errorf("pipeline already exists")
	}

	var err error

	codec := manager.Codec()
	manager.logger.Info().
		Str("codec", codec.Name).
		Str("src", manager.pipelineStr).
		Msgf("creating pipeline")

	manager.pipeline, err = gst.CreatePipeline(manager.pipelineStr)
	if err != nil {
		return err
	}

	manager.pipeline.Start()

	manager.sample = manager.pipeline.Sample
	manager.emitUpdate <-true
	return nil
}

func (manager *StreamManagerCtx) destroyPipeline() {
	if manager.pipeline == nil {
		return
	}

	manager.pipeline.Stop()
	manager.logger.Info().Msgf("destroying pipeline")
	manager.pipeline = nil
}
