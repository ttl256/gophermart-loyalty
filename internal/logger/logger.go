package logger

import (
	"log/slog"
	"os"
	"sync"
)

var once sync.Once //nolint: gochecknoglobals //fine

func Initialize(level slog.Level) {
	once.Do(func() {
		lVar := slog.LevelVar{}
		lVar.Set(level)
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     &lVar,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if len(groups) == 0 && a.Key == slog.TimeKey {
					a.Value = slog.TimeValue(a.Value.Time().UTC())
				}
				return a
			},
		}))
		slog.SetDefault(logger)
	})
}
