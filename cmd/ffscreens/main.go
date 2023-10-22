package main

import (
	"flag"
	"fmt"
	"github.com/codercms/ffthumbs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	width         int
	height        int
	scaleBehavior ffthumbs.ScaleBehavior
	dst           string
	input         string
	points        string
	thumbsNo      int
)

func init() {
	flag.StringVar(&input, "i", "", "Set media path to generate thumbnails")

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

	flag.StringVar(&dst, "dst", "screens/%04d.jpg", "Set output destination path")

	flag.StringVar(&points, "points", "", "Set time points delimited by comma")
	flag.IntVar(&thumbsNo, "thumbsNo", 20, "Set thumbnails count")
}

func main() {
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		log.Fatalf("cannot create dest path: %v", err)
	}

	if len(input) == 0 {
		log.Fatalf("provide input file (-i)")
	}

	thumbsGen, err := ffthumbs.NewScreensGenerator(&ffthumbs.ScreensConfig{
		Logger: nil,
	})

	if err != nil {
		log.Fatal(err)
	}

	var timeUnits []ffthumbs.TimeUnit

	if len(points) > 0 {
		parts := strings.Split(points, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			pathWithoutPerc := strings.TrimSuffix(part, "%")

			val, err := strconv.ParseFloat(pathWithoutPerc, 64)
			if err != nil {
				log.Fatalf("Wrong point val %q: %v", part, err)
				return
			}

			timeUnit := ffthumbs.TimeUnit{Value: val}

			if len(part) != len(pathWithoutPerc) {
				timeUnit.Type = ffthumbs.TimeUnitTypePercent
			}

			timeUnits = append(timeUnits, timeUnit)
		}
	}

	req := &ffthumbs.ScreenshotsRequest{
		MediaURL: input,
		Scale: &ffthumbs.ScaleConfig{
			Width:    width,
			Height:   height,
			Behavior: scaleBehavior,
		},
		ThumbsNo:  thumbsNo,
		TimeUnits: timeUnits,
		OutputDst: dst,
	}

	start := time.Now()

	if err := thumbsGen.Generate(req); err != nil {
		log.Fatal(err)
	}

	log.Printf("Done in %s", time.Since(start))
}
