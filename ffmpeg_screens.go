package ffthumbs

import (
	"context"
	"errors"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type TimeUnitType int

const (
	TimeUnitTypePoint TimeUnitType = iota
	TimeUnitTypePercent
)

type (
	ScreensConfig struct {
		// FfmpegPath path to ffmpeg binary, default: search binary in OS $PATH variable
		FfmpegPath  string
		FfprobePath string

		// Headers configures which headers should pass ffmpeg if requested file is a network url
		Headers map[string]string
		// Logger set pre-configured logger if you have one, default: json logger to stdout with debug log level
		Logger *slog.Logger

		filtersStr string
	}

	ScreenGenerator struct {
		ffmpegPath  string
		ffprobePath string
		cmdArgs     []string

		cfg *ScreensConfig

		logger *slog.Logger

		pool *ants.PoolWithFunc

		wg sync.WaitGroup

		lastReqId atomic.Uint64
	}

	TimeUnit struct {
		Type TimeUnitType
		// Value time point in seconds or percent of media duration
		Value float64
	}

	ScreenshotsRequest struct {
		// MediaURL is a path to media file
		MediaURL string

		Scale *ScaleConfig

		// ThumbsNo is total count of screenshots
		ThumbsNo int

		// TimeUnits make screenshots by provided time uints instead of ThumbsNo
		TimeUnits []TimeUnit

		OutputDst string

		// Context is used to cancel command
		Context context.Context

		// LogArgs is an additional log launchParams that will be appended to logs
		LogArgs []slog.Attr
	}
)

// NewScreensGenerator constructs new ScreenGenerator based on provided config
func NewScreensGenerator(cfg *ScreensConfig) (*ScreenGenerator, error) {
	if cfg == nil {
		return nil, errors.New("nil cfg passed")
	}

	ffmpegPath, err := getVerifiedFfmpegPath(cfg.FfmpegPath)
	if err != nil {
		return nil, err
	}

	ffprobePath, err := getVerifiedFfprobePath(cfg.FfprobePath)
	if err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	cmdArgs := []string{"-loglevel", "error"}

	if len(cfg.Headers) > 0 {
		headersStr := BuildHeadersStr(cfg.Headers)
		cmdArgs = append(cmdArgs, "-headers", headersStr)
	}

	//filtersStr, err := BuildComplexFilters(cfg.Outputs)
	if err != nil {
		return nil, err
	}

	gen := &ScreenGenerator{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		cmdArgs:     cmdArgs,
		cfg:         cfg,
		logger:      logger,
	}

	return gen, nil
}

func (g *ScreenGenerator) getDuration(req *ScreenshotsRequest) (float64, error) {
	cmd, err := launchCommand(launchParams{
		ctx:  req.Context,
		path: g.ffprobePath,
		args: []string{
			"-v", "error",
			"-show_entries",
			"format=duration",
			"-of",
			"default=noprint_wrappers=1:nokey=1",
			req.MediaURL,
		},
		needStdout: true,
		logger:     g.logger,
		LogArgs:    req.LogArgs,
	})

	if err != nil {
		return 0, err
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(cmd.Stdout.(*strings.Builder).String()), 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse duration: %w", err)
	}

	return duration, err
}

func (g *ScreenGenerator) Generate(req *ScreenshotsRequest) error {
	duration, err := g.getDuration(req)
	if err != nil {
		return err
	}

	filters := []string{
		"thumbnail=200",
	}

	if req.Scale != nil {
		filters = append(filters, buildScaleArg(req.Scale))
	}

	filtersStr := strings.Join(filters, ",")

	logCtx := context.Background()
	slogArgs := req.LogArgs

	outputDst := "image_%d.jpg"
	if len(req.OutputDst) > 0 {
		outputDst = req.OutputDst
	}

	if len(req.TimeUnits) > 0 {
		var timePoint float64

		for i, timeUnit := range req.TimeUnits {
			if timeUnit.Type == TimeUnitTypePercent {
				timePoint = (duration / 100) * timeUnit.Value
			} else {
				timePoint = timeUnit.Value
			}

			outputFilename := fmt.Sprintf(outputDst, i)

			{
				args := slogArgs
				args = append(args,
					slog.Float64("time", timePoint),
					slog.String("dst", outputFilename),
				)
				g.logger.LogAttrs(logCtx, slog.LevelDebug, "Generating thumb", args...)
			}

			cmdArgs := []string{
				"-loglevel", "error",
				"-ss", fmt.Sprintf("%f", timePoint),
				"-i", req.MediaURL,
				"-vf", filtersStr,
				"-vframes", "1",
				outputFilename,
			}

			_, err := launchCommand(launchParams{
				ctx:        req.Context,
				path:       g.ffmpegPath,
				args:       cmdArgs,
				needStdout: false,
				logger:     g.logger,
				LogArgs:    req.LogArgs,
			})
			if err != nil {
				return err
			}
		}

		return nil
	}

	for i := 1; i <= req.ThumbsNo; i++ {
		timePoint := float64(i) / (float64(req.ThumbsNo) + 1) * duration

		outputFilename := fmt.Sprintf(outputDst, i)

		{
			args := slogArgs
			args = append(args,
				slog.Float64("time", timePoint),
				slog.String("dst", outputFilename),
			)
			g.logger.LogAttrs(logCtx, slog.LevelDebug, "Generating thumb", args...)
		}

		cmdArgs := []string{
			"-loglevel", "error",
			"-ss", fmt.Sprintf("%f", timePoint),
			"-i", req.MediaURL,
			"-vf", filtersStr,
			"-vframes", "1",
			outputFilename,
		}

		_, err := launchCommand(launchParams{
			ctx:        req.Context,
			path:       g.ffmpegPath,
			args:       cmdArgs,
			needStdout: false,
			logger:     g.logger,
			LogArgs:    req.LogArgs,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
