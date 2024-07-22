// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package wasm

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/ultravioletrs/cocos/agent/algorithm"
	"github.com/ultravioletrs/cocos/agent/events"
)

const wasmtimeBinary = "wasmtime"

var _ algorithm.Algorithm = (*wasm)(nil)

type wasm struct {
	algoFile string
	datasets []string
	logger   *slog.Logger
	stderr   io.Writer
	stdout   io.Writer
}

func NewAlgorithm(logger *slog.Logger, eventsSvc events.Service, algoFile string, datasets ...string) algorithm.Algorithm {
	return &wasm{
		algoFile: algoFile,
		datasets: datasets,
		logger:   logger,
		stderr:   &algorithm.Stderr{Logger: logger, EventSvc: eventsSvc},
		stdout:   &algorithm.Stdout{Logger: logger},
	}
}

func (b *wasm) Run() ([]byte, error) {
	defer func() {
		os.Remove(b.algoFile)
		for _, file := range b.datasets {
			os.Remove(file)
		}
	}()

	args := []string{b.algoFile}
	args = append(args, b.datasets...)
	cmd := exec.Command(wasmtimeBinary, args...)

	cmd.Stderr = b.stderr
	var outBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(&outBuf, b.stdout)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting algorithm: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("algorithm execution error: %v", err)
	}

	return outBuf.Bytes(), nil
}
