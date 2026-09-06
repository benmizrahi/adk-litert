package litert

import (
	"context"
	"strings"

	"google.golang.org/adk/v2/model"
)

// Register registers the LiteRT model loader with ADK's model registry.
// Pattern defaults to "^(?i)litert:.*" if empty.
func Register(pattern string, defaultCfg Config) {
	if pattern == "" {
		pattern = "^(?i)litert:.*"
	}
	model.Register(pattern, func(ctx context.Context, name string) (model.LLM, error) {
		cfg := defaultCfg
		cfg.ModelName = name
		if strings.HasPrefix(strings.ToLower(name), "litert:") {
			path := strings.TrimPrefix(name, "litert:")
			path = strings.TrimPrefix(path, "LITERT:")
			if path != "" {
				cfg.ModelPath = path
			}
		}
		return NewModel(ctx, cfg)
	})
}
