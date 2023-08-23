package ffthumbs

import (
	"log/slog"
	"time"
)

// ScaleBehavior configures how scaling will be performed
// See: https://superuser.com/questions/547296/resizing-videos-with-ffmpeg-avconv-to-fit-into-static-sized-player/1136305#1136305
type ScaleBehavior int

const (
	// ScaleBehaviorNone do not attempt to change scaling behavior
	ScaleBehaviorNone ScaleBehavior = iota
	// ScaleBehaviorFillToKeepAspectRatio is useful when you want to resize to fixed resolution
	// (width and height must be set to fixed size), but preserve original aspect ratio.
	// Letterboxing will occur instead of pillarboxing if the input aspect ratio
	// is wider than the output aspect ratio.
	// For example, an input with a 2.35:1 aspect ratio fit into a 16:9 output will result in letterboxing.
	ScaleBehaviorFillToKeepAspectRatio
	// ScaleBehaviorCropToFit is useful when you want to resize to fixed resolution
	// (width and height must be set to fixed size), but preserve fixed size by
	// cropping frame to fit into target resolution.
	ScaleBehaviorCropToFit
)

// OutputType configures output type, e.g. spites or thumbs
type OutputType int

const (
	// OutputTypeThumbs output thumbnail for each OutputConfig.SnapshotInterval
	OutputTypeThumbs OutputType = iota
	// OutputTypeSprites output sprite for each OutputConfig.SnapshotInterval respecting OutputConfig.Sprites
	OutputTypeSprites
)

const (
	// DefaultFilename is an output default filename
	DefaultFilename = "%04d.jpg"
)

type (
	Config struct {
		// FfmpegPath path to ffmpeg binary, default: search binary in OS $PATH variable
		FfmpegPath string
		// Concurrency limit amount of concurrent thumbnails generation, default: 2
		Concurrency int
		// Headers configures which headers should pass ffmpeg if requested file is a network url
		Headers map[string]string
		// Outputs configure outputs of snapshots (thumbs)
		Outputs []*OutputConfig
		// Logger set pre-configured logger if you have one, default: json logger to stdout with debug log level
		Logger *slog.Logger
		// DisableProgressLogs ffmpeg's progress logs
		DisableProgressLogs bool

		filtersStr string
	}

	// ScaleConfig is an output files resolution config
	ScaleConfig struct {
		// Width is an outgoing width resolution, could be -1 to resize by Height respecting aspect ratio
		Width int
		// Height is an outgoing height resolution, could be -1 to resize by Width respecting aspect ratio
		Height int
		// Behavior is configuring how scaling will be performed
		Behavior ScaleBehavior
	}

	OutputConfig struct {
		idx     int
		inName  string
		outName string

		// DstPath sets thumbs output path, default: app work dir + DefaultFilename
		// can be overridden in GenerateRequest.OutputDst
		DstPath string

		// Scale configure scaling behavior
		Scale ScaleConfig

		// SnapshotInterval indicates how often to make screenshots from video
		SnapshotInterval time.Duration

		// Type configures output type, e.g. sprites or thumbs
		Type OutputType

		// Sprites configures output sprites behavior when Type is set to OutputTypeSprites
		Sprites SpritesConfig

		// Quality configures quality level (0 = default, valid values are 1-31, lower is better)
		// See: https://ffmpeg.org/ffmpeg-codecs.html#Options-21 (q:v option)
		Quality int
	}

	// SpritesConfig is a sprites output configuration
	SpritesConfig struct {
		// Dimensions is an output grid size,
		// configure how many tiles and how tiles will be placed in an output file
		Dimensions SpriteDimensions
	}

	// SpriteDimensions configure how many tiles and how tiles will be placed in an output file
	SpriteDimensions struct {
		Columns int
		Rows    int
	}
)

// Eq is ScaleConfig equal to another scale config
func (c *ScaleConfig) Eq(cfg *ScaleConfig) bool {
	return IsSameScaleConfig(c, cfg)
}

// IsSameScaleConfig check is two scale configurations equal
func IsSameScaleConfig(c1, c2 *ScaleConfig) bool {
	if c1 == nil && c2 == nil {
		return true
	}

	if c1 != nil || c2 != nil {
		return false
	}

	return c1.Width == c2.Width &&
		c1.Height == c2.Height &&
		c1.Behavior == c2.Behavior
}

func (c *ScaleConfig) IsFixedResolution() bool {
	return c.Width > 0 && c.Height > 0
}

func (c *OutputConfig) EqFilters(cfg *OutputConfig) bool {
	return IsSameOutputConfigFilters(c, cfg)
}

func IsSameOutputConfigFilters(c1, c2 *OutputConfig) bool {
	if c1 == nil && c2 == nil {
		return true
	}

	if c1 != nil || c2 != nil {
		return false
	}

	return c1.Scale.Eq(&c2.Scale) &&
		c1.SnapshotInterval == c2.SnapshotInterval
}
