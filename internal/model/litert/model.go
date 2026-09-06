package litert

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR} -I${SRCDIR}/csrc -I${SRCDIR}/../../../include -I/usr/local/include
#cgo LDFLAGS: -L${SRCDIR}/../../../lib -Wl,-rpath,${SRCDIR}/../../../lib -L/usr/local/lib -Wl,-rpath,/usr/local/lib -llitert-lm -lstdc++
#include "litert_lm_c_api.h"
#include <stdlib.h>

extern bool goTokenCallbackGatewayLitertInternal(char* token, bool is_done, void* user_data);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// Model represents an in-memory LiteRT-LM instance implementing model.LLM.
type Model struct {
	name   string
	config Config
	engine *C.LiteRTLMEngine
	mu     sync.Mutex
}

var _ model.LLM = (*Model)(nil)

// NewModel loads a .litertlm model into memory and returns an ADK model.LLM instance.
func NewModel(ctx context.Context, config Config) (*Model, error) {
	if config.ModelPath == "" {
		return nil, errors.New("litert: ModelPath is required")
	}

	cPath := C.CString(config.ModelPath)
	defer C.free(unsafe.Pointer(cPath))

	cConfig := C.LiteRTLMConfig{
		model_path:  cPath,
		backend:     C.int(config.Backend),
		max_seq_len: C.int(config.MaxSeqLen),
		temperature: C.float(config.Temperature),
		top_p:       C.float(config.TopP),
		top_k:       C.int(config.TopK),
	}

	var cErrMsg *C.char
	engine := C.litert_lm_create_engine(&cConfig, &cErrMsg)
	if engine == nil {
		defer C.free(unsafe.Pointer(cErrMsg))
		msg := "unknown error"
		if cErrMsg != nil {
			msg = C.GoString(cErrMsg)
		}
		return nil, fmt.Errorf("litert: failed to load model: %s", msg)
	}

	name := config.ModelName
	if name == "" {
		name = "litert-inmemory"
	}

	m := &Model{
		name:   name,
		config: config,
		engine: engine,
	}

	runtime.SetFinalizer(m, (*Model).Close)
	return m, nil
}

// Close explicitly unloads the model from memory and releases all C++ runtime handles.
func (m *Model) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine != nil {
		C.litert_lm_free_engine(m.engine)
		m.engine = nil
	}
}

// Name returns the name identifier of this model.
func (m *Model) Name() string {
	return m.name
}

// GenerateContent satisfies the ADK model.LLM interface.
func (m *Model) GenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(nil, err)
			return
		}

		prompt := formatADKContentsToPrompt(req.Contents)

		cPrompt := C.CString(prompt)
		defer C.free(unsafe.Pointer(cPrompt))

		m.mu.Lock()
		engine := m.engine
		m.mu.Unlock()

		if engine == nil {
			yield(nil, errors.New("litert: model engine is not loaded or has been closed"))
			return
		}

		type tokenEvent struct {
			token  string
			isDone bool
		}

		tokenChan := make(chan tokenEvent, 32)
		stopChan := make(chan struct{})

		tokenHandler := func(token string, isDone bool) bool {
			select {
			case <-ctx.Done():
				return false
			case <-stopChan:
				return false
			case tokenChan <- tokenEvent{token: token, isDone: isDone}:
				return true
			}
		}

		callbackID := registerCallback(tokenHandler)
		defer unregisterCallback(callbackID)

		doneChan := make(chan struct{})
		var genErr error
		go func() {
			defer close(doneChan)
			defer close(tokenChan)
			var cErrMsg *C.char
			ok := C.litert_lm_generate(
				engine,
				cPrompt,
				(C.LiteRTLMTokenCallback)(C.goTokenCallbackGatewayLitertInternal),
				unsafe.Pointer(&callbackID),
				&cErrMsg,
			)
			if !bool(ok) && cErrMsg != nil {
				defer C.free(unsafe.Pointer(cErrMsg))
				genErr = fmt.Errorf("litert generation failed: %s", C.GoString(cErrMsg))
			}
		}()

		var fullResponse strings.Builder
		for ev := range tokenChan {
			fullResponse.WriteString(ev.token)

			if stream {
				resp := &model.LLMResponse{
					Content: &genai.Content{
						Role: "model",
						Parts: []*genai.Part{
							{Text: ev.token},
						},
					},
					Partial:      !ev.isDone,
					TurnComplete: ev.isDone,
				}
				if !yield(resp, nil) {
					close(stopChan)
					<-doneChan
					return
				}
			}
		}

		<-doneChan

		if genErr != nil {
			yield(nil, genErr)
			return
		}

		if !stream {
			resp := &model.LLMResponse{
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{Text: fullResponse.String()},
					},
				},
				TurnComplete: true,
			}
			yield(resp, nil)
		}
	}
}

// formatADKContentsToPrompt converts ADK Contents to standard chat turn tags.
func formatADKContentsToPrompt(contents []*genai.Content) string {
	var b strings.Builder
	b.WriteString("<bos>")

	var currentRole string
	var currentParts []string

	flushTurn := func() {
		if len(currentParts) == 0 {
			return
		}
		role := currentRole
		if role == "" || role == "system" {
			role = "user"
		}
		b.WriteString("<start_of_turn>")
		b.WriteString(role)
		b.WriteString("\n")
		b.WriteString(strings.Join(currentParts, "\n\n"))
		b.WriteString("<end_of_turn>\n")
		currentParts = nil
	}

	for _, c := range contents {
		normRole := c.Role
		if normRole == "" || normRole == "system" {
			normRole = "user"
		} else if normRole != "model" {
			normRole = "user"
		}

		if currentRole != "" && currentRole != normRole {
			flushTurn()
		}
		currentRole = normRole

		for _, p := range c.Parts {
			if p.Text != "" {
				currentParts = append(currentParts, p.Text)
			}
		}
	}
	flushTurn()

	b.WriteString("<start_of_turn>model\n")
	return b.String()
}
