package ffthumbs

import (
	"context"
	"log/slog"
	"os/exec"
	"path"
	"strings"
	"time"
)

type launchParams struct {
	ctx context.Context

	path string
	args []string

	needStdout bool

	logger *slog.Logger

	// LogArgs is an additional log launchParams that will be appended to logs
	LogArgs []slog.Attr
}

var logCtx = context.Background()

func launchCommand(params launchParams) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	if params.ctx != nil {
		cmd = exec.CommandContext(params.ctx, params.path, params.args...)
	} else {
		cmd = exec.Command(params.path, params.args...)
	}

	var stderr strings.Builder // or bytes.Buffer
	cmd.Stderr = &stderr

	cmdName := path.Base(params.path)

	if params.needStdout {
		var stdout strings.Builder
		cmd.Stdout = &stdout
	}

	if params.logger != nil {
		args := params.LogArgs
		args = append(args,
			slog.String("cmd", cmd.String()),
		)

		params.logger.LogAttrs(logCtx, slog.LevelDebug, "Launching "+cmdName, args...)
	}

	start := time.Now()

	if err := cmd.Start(); err != nil {
		if params.logger != nil {
			args := params.LogArgs
			args = append(args,
				slog.String("err", err.Error()),
			)

			params.logger.LogAttrs(logCtx, slog.LevelError, cmdName+" start failed", args...)
		}

		return cmd, err
	}

	if err := cmd.Wait(); err != nil {
		if params.logger != nil {
			args := params.LogArgs
			args = append(args,
				slog.String("stderr", stderr.String()),
				slog.String("err", err.Error()),
			)

			params.logger.LogAttrs(logCtx, slog.LevelError, cmdName+" run failed", args...)
		}

		return cmd, err
	}

	if params.logger != nil {
		args := params.LogArgs
		args = append(args, slog.Duration("duration", time.Since(start)))
		params.logger.LogAttrs(logCtx, slog.LevelInfo, cmdName+" command finished", args...)
	}

	return cmd, nil
}
