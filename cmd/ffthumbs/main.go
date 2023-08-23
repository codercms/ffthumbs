package main

import (
	"flag"
	"fmt"
	"github.com/codercms/ffthumbs"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	snapshotInterval time.Duration
	width            int
	height           int
	scaleBehavior    ffthumbs.ScaleBehavior
	outputType       ffthumbs.OutputType
	spriteRows       int
	spriteCols       int
	dst              string
	input            string
)

func init() {
	flag.StringVar(&input, "i", "", "Set media path to generate thumbnails")

	flag.DurationVar(&snapshotInterval, "interval", time.Second*7, "Set snapshot interval for thumbnails")

	flag.IntVar(&width, "width", 320, "Set desired thumbnails width")
	flag.IntVar(&height, "height", 180, "Set desired thumbnails height")

	vals := fmt.Sprintf("None - %d, FillToKeepAspectRatio - %d, CropToFit - %d",
		ffthumbs.ScaleBehaviorNone,
		ffthumbs.ScaleBehaviorFillToKeepAspectRatio,
		ffthumbs.ScaleBehaviorCropToFit,
	)

	flag.IntVar((*int)(&scaleBehavior), "behavior", int(ffthumbs.ScaleBehaviorNone), "Set scale scaleBehavior:\n"+vals)

	vals = fmt.Sprintf("Thumbs - %d, Sprites - %d",
		ffthumbs.OutputTypeThumbs,
		ffthumbs.OutputTypeSprites,
	)

	flag.IntVar((*int)(&outputType), "type", int(ffthumbs.OutputTypeThumbs), "Set output type:\n"+vals)
	flag.IntVar(&spriteRows, "rows", 8, "Set sprites output rows num")
	flag.IntVar(&spriteCols, "cols", 8, "Set sprites output cols num")

	flag.StringVar(&dst, "dst", "thumbs/%04d.jpg", "Set output destination path")
}

func main() {
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		log.Fatalf("cannot create dest path: %v", err)
	}

	if len(input) == 0 {
		log.Fatalf("provide input file (-i)")
	}

	thumbsGen, err := ffthumbs.NewGenerator(&ffthumbs.Config{
		Outputs: []*ffthumbs.OutputConfig{
			{
				Type:             outputType,
				DstPath:          dst,
				SnapshotInterval: snapshotInterval,
				Scale: ffthumbs.ScaleConfig{
					Width:    width,
					Height:   height,
					Behavior: scaleBehavior,
				},
				Sprites: ffthumbs.SpritesConfig{Dimensions: ffthumbs.SpriteDimensions{
					Columns: spriteCols,
					Rows:    spriteRows,
				}},
			},
		},
		Logger:              nil,
		DisableProgressLogs: false,
	})

	if err != nil {
		log.Fatal(err)
	}

	req := &ffthumbs.GenerateRequest{
		MediaURL: input,
	}

	start := time.Now()

	if err := thumbsGen.Generate(req); err != nil {
		log.Fatal(err)
	}

	log.Printf("Done in %s", time.Since(start))
}
