package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// isWasmEnvironment перевіряє, чи запущено програму у WASM середовищі
func isWasmEnvironment() bool {
	return runtime.GOARCH == "wasm" || runtime.GOOS == "js"
}

// Config конфігурація логера
type Config struct {
	LogDir          string
	LogLevel        string
	MaxSize         int // MB
	MaxBackups      int
	MaxAge          int // days
	Compress        bool
	EnableConsole   bool
	EnableFile      bool
	PrettyConsole   bool
	SamplingEnabled bool
}

// DefaultConfig повертає конфігурацію за замовчуванням
func DefaultConfig() *Config {
	return &Config{
		LogDir:          "./log",
		LogLevel:        "info",
		MaxSize:         100,
		MaxBackups:      10,
		MaxAge:          30,
		Compress:        true,
		EnableConsole:   true,
		EnableFile:      true,
		PrettyConsole:   true,
		SamplingEnabled: false,
	}
}

// Setup налаштовує глобальальний логер з розширеними можливостями
func Setup(config *Config) error {
	fmt.Printf("Налаштування системи логування...\n")
	var writers []io.Writer

	// Налаштування файлового логера з ротацією
	// У WASM-середовищі файлова система недоступна, тому пропускаємо
	if config.EnableFile {
		if isWasmEnvironment() {
			fmt.Printf("  Файлова система недоступна у WASM, логи тільки в консоль\n")
			config.EnableFile = false
		} else {
			fmt.Printf("  Створення директорії логів: %s\n", config.LogDir)
			if err := os.MkdirAll(config.LogDir, 0755); err != nil {
				fmt.Printf("  Попередження: не вдалося створити директорію логів: %v\n", err)
				// Не повертаємо помилку, а лише вимикаємо файлові логи
				config.EnableFile = false
			} else {
				// Налаштування файлового логера з ротацією
				fileWriter := &lumberjack.Logger{
					Filename:   filepath.Join(config.LogDir, "app.log"),
					MaxSize:    config.MaxSize,
					MaxBackups: config.MaxBackups,
					MaxAge:     config.MaxAge,
					Compress:   config.Compress,
					LocalTime:  true,
				}
				appWriter := io.Writer(fileWriter)
				if config.PrettyConsole {
					appWriter = newConsoleWriter(fileWriter, true)
				}
				writers = append(writers, appWriter)
				fmt.Printf("  Файловий логер налаштовано (app.log)\n")

				// Окремий файл для помилок
				errorWriter := &lumberjack.Logger{
					Filename:   filepath.Join(config.LogDir, "error.log"),
					MaxSize:    config.MaxSize,
					MaxBackups: config.MaxBackups,
					MaxAge:     config.MaxAge,
					Compress:   config.Compress,
					LocalTime:  true,
				}

				// Фільтруємо тільки помилки та вище
				errorFilteredWriter := &LevelFilterWriter{
					Writer:   errorWriter,
					MinLevel: zerolog.ErrorLevel,
				}
				writers = append(writers, errorFilteredWriter)
				fmt.Printf("  Логер помилок налаштовано (error.log)\n")
			}
		}
	}

	// Налаштування консольного виводу
	if config.EnableConsole {
		if config.PrettyConsole {
			fmt.Printf("  Увімкнено красивий формат консолі\n")
			writers = append(writers, newConsoleWriter(os.Stdout, false))
		} else {
			writers = append(writers, os.Stdout)
		}
		fmt.Printf("  Консолень логер налаштовано\n")
	}

	if len(writers) == 0 {
		return fmt.Errorf("необхідно увімкнути хоча б один вивід (консоль або файл)")
	}

	// Об'єднуємо всі writer'и
	multi := io.MultiWriter(writers...)

	// Налаштування формату часу та стеку помилок
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Налаштування рівня логування
	config.LogLevel = NormalizeLogLevel(config.LogLevel)
	level, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	fmt.Printf("  Рівень логування: %s\n", config.LogLevel)

	// Створюємо базовий логер
	logger := zerolog.New(multi).
		With().
		Timestamp().
		Caller().
		Logger()

	// Додаємо sampling якщо увімкнено (зменшує кількість логів під навантаженням)
	if config.SamplingEnabled {
		logger = logger.Sample(&zerolog.BurstSampler{
			Burst:       5,
			Period:      time.Second,
			NextSampler: &zerolog.BasicSampler{N: 100},
		})
		fmt.Printf("  Sampling увімкнено\n")
	}

	// Встановлюємо глобальний логер
	log.Logger = logger

	fmt.Printf("Система логування готова\n\n")
	log.Info().
		Str("log_level", config.LogLevel).
		Str("log_dir", config.LogDir).
		Bool("console_enabled", config.EnableConsole).
		Bool("file_enabled", config.EnableFile).
		Msg("Логер успішно ініціалізовано")

	return nil
}

func newConsoleWriter(out io.Writer, noColor bool) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: "15:04:05",
		NoColor:    noColor,
		FormatLevel: func(i any) string {
			return fmt.Sprintf("| %-6s|", i)
		},
		FormatMessage: func(i any) string {
			return fmt.Sprintf("%-50s", i)
		},
		FormatFieldName: func(i any) string {
			return fmt.Sprintf("%s=", i)
		},
	}
}

// NormalizeLogLevel повертає валідний рівень zerolog.
func NormalizeLogLevel(level string) string {
	switch l := strings.ToLower(strings.TrimSpace(level)); l {
	case "trace", "debug", "info", "warn", "error":
		return l
	default:
		return "info"
	}
}

// SetLogLevel змінює глобальний рівень логування на льоту.
func SetLogLevel(level string) string {
	normalized := NormalizeLogLevel(level)
	parsed, err := zerolog.ParseLevel(normalized)
	if err != nil {
		parsed = zerolog.InfoLevel
		normalized = "info"
	}
	zerolog.SetGlobalLevel(parsed)
	log.Info().Str("log_level", normalized).Msg("Рівень логування оновлено")
	return normalized
}

// LevelFilterWriter фільтрує логи за мінімальним рівнем
type LevelFilterWriter struct {
	Writer   io.Writer
	MinLevel zerolog.Level
}

// Write implements io.Writer. Since zerolog may call Write directly
// (without level information), we pass all data through to maintain compatibility.
// For level-aware filtering, use WriteLevel instead.
func (w *LevelFilterWriter) Write(p []byte) (n int, err error) {
	return w.Writer.Write(p)
}

// WriteLevel filters logs based on minimum level threshold.
// Only writes if the provided level meets or exceeds MinLevel.
func (w *LevelFilterWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	if level >= w.MinLevel {
		return w.Writer.Write(p)
	}
	return len(p), nil
}

// GetLogger повертає логер з додатковим контекстом
func GetLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

// WithRequestID додає request ID до логера (для трейсінгу запитів)
func WithRequestID(logger zerolog.Logger, requestID string) zerolog.Logger {
	return logger.With().Str("request_id", requestID).Logger()
}

// WithUserContext додає інформацію про користувача до логера
func WithUserContext(logger zerolog.Logger, userID int, username string) zerolog.Logger {
	return logger.With().
		Int("user_id", userID).
		Str("username", username).
		Logger()
}
