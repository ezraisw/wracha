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

func (l stdLogger) Info(args ...interface{}) {
	fmt.Fprintln(os.Stdout, args...)
}

func (l stdLogger) Debug(args ...interface{}) {
	fmt.Fprintln(os.Stdout, args...)
}

func (l stdLogger) Error(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}
