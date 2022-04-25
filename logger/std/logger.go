package std

import (
	"fmt"
	"os"

	"github.com/pwnedgod/wracha/logger"
)

type stdLogger struct {
}

func NewLogger() logger.Logger {
	return &stdLogger{}
}

func (l stdLogger) Info(args ...any) {
	fmt.Fprintln(os.Stdout, args...)
}

func (l stdLogger) Debug(args ...any) {
	fmt.Fprintln(os.Stdout, args...)
}

func (l stdLogger) Error(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
}
