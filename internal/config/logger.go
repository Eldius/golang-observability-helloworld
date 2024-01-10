package config

import (
	"context"
	"fmt"
	"log/slog"
)

func NewLogger(ctx context.Context) *slog.Logger {
	fmt.Printf("%v\n", ctx)
	return slog.Default()
}
