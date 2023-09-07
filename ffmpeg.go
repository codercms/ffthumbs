package ffthumbs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
)

var (
	progressPattern = regexp.MustCompile(`progress=([\w.]+)`)
	outTimePattern  = regexp.MustCompile(`out_time=([^ ]+)`)
	speedPattern    = regexp.MustCompile(`speed=([^ ]+)`)

	versionPattern = regexp.MustCompile(`ffmpeg version ([0-9.]+)`)
)

type (
	Generator struct {
		ffmpegPath string
		cmdArgs    []string

		cfg *Config

		logger *slog.Logger

		pool *ants.PoolWithFunc

		wg sync.WaitGroup

		lastReqId atomic.Uint64
	}

	GenerateRequest struct {
		// id internal request id used for async processing
		id uint64

		// MediaURL path to media file (can be either a network path or a local fs path)
		MediaURL string

		// OutputDst allows to override OutputConfig.DstPath
		// map format is an output index => dest path
		OutputDst map[int]string

		// Context is used to cancel command
		Context context.Context

		// DoneChan channel to receive request processing result
		DoneChan chan *GenerateResult

		// LogArgs is an additional log args that will be appended to logs
		LogArgs []slog.Attr
	}

	GenerateResult struct {
		// Req is a processed request
		Req *GenerateRequest
		// Err is an error during Req processing
		Err error
		// Duration measures how much time was spent to process Req
		Duration time.Duration
	}
)

// GetId returns request id for better async processing, i.e. user could identify what request was processed
func (r *GenerateRequest) GetId() uint64 {
	return r.id
}

// NewGenerator constructs new Generator based on provided config
func NewGenerator(cfg *Config) (*Generator, error) {
	if cfg == nil {
		return nil, errors.New("nil cfg passed")
	}

	ffmpegPath, err := getVerifiedFfmpegPath(cfg.FfmpegPath)
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

	for idx, output := range cfg.Outputs {
		output.idx = idx
		if len(output.DstPath) == 0 {
			output.DstPath = DefaultFilename
		}
	}

	filtersStr, err := BuildComplexFilters(cfg.Outputs)
	if err != nil {
		return nil, err
	}

	cfg.filtersStr = filtersStr

	gen := &Generator{
		ffmpegPath: ffmpegPath,
		cmdArgs:    cmdArgs,
		cfg:        cfg,
		logger:     logger,
	}

	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 2
	}

	pool, err := ants.NewPoolWithFunc(concurrency, gen.handleRequest)
	if err != nil {
		return nil, fmt.Errorf("cannot create worker pool: %w", err)
	}

	gen.pool = pool

	return gen, nil
}

// GetConcurrency returns current concurrency setting
func (g *Generator) GetConcurrency() int {
	return g.pool.Cap()
}

// SetConcurrency sets current concurrency number, new concurrency should be a positive int,
// otherwise value 2 will be used
func (g *Generator) SetConcurrency(newConcurrency int) {
	if newConcurrency < 0 {
		newConcurrency = 2
	}

	g.pool.Tune(newConcurrency)
}

// Wait waits until all requests processed
func (g *Generator) Wait() {
	g.wg.Wait()
}

func (g *Generator) handleRequest(reqRaw any) {
	req := reqRaw.(*GenerateRequest)

	var timeStart time.Time
	if req.DoneChan != nil {
		timeStart = time.Now()
	}

	err := g.Generate(req)

	if req.DoneChan != nil {
		res := GenerateResult{
			Req:      req,
			Err:      err,
			Duration: time.Since(timeStart),
		}

		req.DoneChan <- &res
	}
}

// GenerateAsync is an asynchronous thumbnails generation using the underlying goroutine pool.
// Concurrency is limited by Config.Concurrency.
// When all goroutines in pool are busy this method would block until a free goroutine is available.
//
// Each request passed to this method will get unique identifier, you can get it by calling GenerateRequest.GetId().
func (g *Generator) GenerateAsync(req *GenerateRequest) error {
	req.id = g.lastReqId.Add(1)

	return g.pool.Invoke(req)
}

// Generate is a blocking thumbnails generation, if you want to go async see GenerateAsync
func (g *Generator) Generate(req *GenerateRequest) error {
	g.wg.Add(1)
	defer g.wg.Done()

	logCtx := context.Background()
	slogArgs := req.LogArgs

	if req.id > 0 {
		slogArgs = append(slogArgs, slog.Uint64("req", req.id))
	}

	cmdArgs := g.cmdArgs
	cmdArgs = append(cmdArgs, "-i", req.MediaURL)
	cmdArgs = append(cmdArgs, "-filter_complex", g.cfg.filtersStr)
	cmdArgs = append(cmdArgs, "-vsync", "0")

	for _, output := range g.cfg.Outputs {
		cmdArgs = append(cmdArgs, "-map", fmt.Sprintf("[%s]", output.outName))

		if output.Quality > 0 {
			cmdArgs = append(cmdArgs, "-q:v", strconv.Itoa(output.Quality))
		}

		outputDst := output.DstPath
		if dstFolder, ok := req.OutputDst[output.idx]; ok {
			outputDst = dstFolder
		}

		cmdArgs = append(cmdArgs, outputDst)
	}

	if !g.cfg.DisableProgressLogs {
		cmdArgs = append(cmdArgs, "-progress", "pipe:1")
	}

	var cmd *exec.Cmd

	if req.Context != nil {
		cmd = exec.CommandContext(req.Context, g.ffmpegPath, cmdArgs...)
	} else {
		cmd = exec.Command(g.ffmpegPath, cmdArgs...)
	}

	{
		args := slogArgs
		args = append(args,
			slog.String("cmd", cmd.String()),
		)

		g.logger.LogAttrs(logCtx, slog.LevelDebug, "Launching ffmpeg", args...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	start := time.Now()

	if err := cmd.Start(); err != nil {
		args := slogArgs
		args = append(args,
			slog.String("err", err.Error()),
		)

		g.logger.LogAttrs(logCtx, slog.LevelError, "ffmpeg start failed", args...)

		return err
	}

	// Read stderr (error) log
	var stdErrLog strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)

		for scanner.Scan() {
			stdErrLog.Write(scanner.Bytes())
			stdErrLog.WriteString("\n")
		}
	}()

	if !g.cfg.DisableProgressLogs {
		g.listenForProgressLogs(stdout, slogArgs)
	}

	if err := cmd.Wait(); err != nil {
		args := slogArgs
		args = append(args,
			slog.String("stderr", stdErrLog.String()),
			slog.String("err", err.Error()),
		)

		g.logger.LogAttrs(logCtx, slog.LevelError, "ffmpeg run failed", args...)

		return err
	}

	{
		args := slogArgs
		args = append(args, slog.Duration("duration", time.Since(start)))
		g.logger.LogAttrs(logCtx, slog.LevelInfo, "ffmpeg command finished", args...)
	}

	return nil
}

func (g *Generator) listenForProgressLogs(stdout io.Reader, slogArgs []slog.Attr) {
	scanner := bufio.NewScanner(stdout)

	var progress, currTime, speed string
	var progressChanged bool

	for scanner.Scan() {
		line := scanner.Text()

		if match := progressPattern.FindStringSubmatch(line); len(match) > 1 {
			progress = match[1]
			progressChanged = true
		}

		if match := outTimePattern.FindStringSubmatch(line); len(match) > 1 {
			currTime = match[1]
		}

		if match := speedPattern.FindStringSubmatch(line); len(match) > 1 {
			speed = match[1]
		}

		if progressChanged {
			progressChanged = false

			args := slogArgs
			args = append(args,
				slog.String("progress", progress),
				slog.String("time", currTime),
				slog.String("speed", speed),
			)

			g.logger.LogAttrs(context.Background(), slog.LevelInfo, "Progress update", args...)
		}
	}
}
