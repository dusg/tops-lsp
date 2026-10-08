package logging

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
)

func New(writer io.Writer, level slog.Level) *slog.Logger {
	if writer == nil {
		writer = io.Discard
	}
	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				attr.Key = "timestamp"
			}
			return attr
		},
	}))
}

func DocumentID(uri string) string {
	digest := sha256.Sum256([]byte(uri))
	return hex.EncodeToString(digest[:8])
}
