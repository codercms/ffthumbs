package ffthumbs

import (
	"fmt"
	"time"
)

type ValidationErrType int

const (
	ValidationErrTypeNoOutputs ValidationErrType = iota
	ValidationErrTypeQuality
	ValidationErrTypeSnapshotInterval
	ValidationErrTypeOutputType
	ValidationErrTypeScale
	ValidationErrTypeSpiteDims
	ValidationErrTypeScaleBehavior
)

type ValidationError struct {
	Type ValidationErrType
	Msg  string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

func validateOutputs(outputs []*OutputConfig) error {
	if len(outputs) == 0 {
		return &ValidationError{
			Type: ValidationErrTypeNoOutputs,
			Msg:  "at least one output should be provided",
		}
	}

	for idx, output := range outputs {
		if output.Quality != 0 && (output.Quality < 1 || output.Quality > 31) {
			return &ValidationError{
				Type: ValidationErrTypeQuality,
				Msg:  fmt.Sprintf("output %d has wrong quality, valid values are 1-31, got %d", idx, output.Quality),
			}
		}

		if output.SnapshotInterval < time.Millisecond {
			return &ValidationError{
				Type: ValidationErrTypeSnapshotInterval,
				Msg:  fmt.Sprintf("output %d snapshot interval is less than one millesecond", idx),
			}
		}

		if output.Scale.Width < 0 && output.Scale.Height < 0 {
			return &ValidationError{
				Type: ValidationErrTypeScale,
				Msg:  fmt.Sprintf("output %d scale has both negative width and height", idx),
			}
		}

		if output.Scale.Width == 0 {
			return &ValidationError{
				Type: ValidationErrTypeScale,
				Msg:  fmt.Sprintf("output %d scale width cannot be zero", idx),
			}
		}

		if output.Scale.Height == 0 {
			return &ValidationError{
				Type: ValidationErrTypeScale,
				Msg:  fmt.Sprintf("output %d scale height cannot be zero", idx),
			}
		}

		switch output.Type {
		case OutputTypeThumbs:
		case OutputTypeSprites:
			if output.Sprites.Dimensions.Rows < 1 {
				return &ValidationError{
					Type: ValidationErrTypeSpiteDims,
					Msg:  fmt.Sprintf("output %d sprite rows dimension is less than 1", idx),
				}
			}
			if output.Sprites.Dimensions.Columns < 1 {
				return &ValidationError{
					Type: ValidationErrTypeSpiteDims,
					Msg:  fmt.Sprintf("output %d sprite columns dimension is less than 1", idx),
				}
			}
		default:
			return &ValidationError{
				Type: ValidationErrTypeOutputType,
				Msg:  fmt.Sprintf("output %d has unkwown type: %d", idx, output.Type),
			}
		}

		switch output.Scale.Behavior {
		case ScaleBehaviorNone, ScaleBehaviorFillToKeepAspectRatio, ScaleBehaviorCropToFit:
		default:
			return &ValidationError{
				Type: ValidationErrTypeScaleBehavior,
				Msg:  fmt.Sprintf("output %d has unknown scale behavior: %d", idx, output.Scale.Behavior),
			}
		}
	}

	return nil
}
