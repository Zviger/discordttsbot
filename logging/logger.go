package logging

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type ColorFormatter struct {
	Colors bool
}

func (f *ColorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// Level color mapping
	levelColor := func(level logrus.Level) string {
		if !f.Colors {
			return ""
		}
		switch level {
		case logrus.DebugLevel:
			return "\033[36m" // Cyan
		case logrus.InfoLevel:
			return "\033[32m" // Green
		case logrus.WarnLevel:
			return "\033[33m" // Yellow
		case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
			return "\033[31m" // Red
		default:
			return "\033[0m" // Default
		}
	}
	resetColor := "\033[0m"

	// Main message line
	levelStr := strings.ToUpper(entry.Level.String())
	if f.Colors {
		b.WriteString(fmt.Sprintf("%s [%s%s%s] %s\n",
			timestamp, levelColor(entry.Level), levelStr, resetColor, entry.Message))
	} else {
		b.WriteString(fmt.Sprintf("%s [%s] %s\n", timestamp, levelStr, entry.Message))
	}

	// Fields line (indented)
	if len(entry.Data) > 0 {
		b.WriteString("  └─ ")
		fields := make([]string, 0, len(entry.Data))
		for k, v := range entry.Data {
			if k != "time" && k != "level" && k != "msg" {
				fields = append(fields, fmt.Sprintf("%s=%v", k, v))
			}
		}
		b.WriteString(strings.Join(fields, " | "))
		b.WriteString("\n")
	}

	return b.Bytes(), nil
}
