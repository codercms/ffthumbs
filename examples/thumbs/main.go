package main

import (
	"github.com/codercms/ffthumbs/examples"
	"log"
	"os"
	"time"

	"github.com/codercms/ffthumbs"
)

func main() {
	thumbsGen, err := ffthumbs.NewGenerator(&ffthumbs.Config{
		Outputs: []*ffthumbs.OutputConfig{
			{
				DstPath: "thumbs/%04d.jpg",
				Scale: ffthumbs.ScaleConfig{
					Width:    320,
					Height:   180,
					Behavior: ffthumbs.ScaleBehaviorFillToKeepAspectRatio,
				},
				SnapshotInterval: time.Millisecond * 6500,
				Type:             ffthumbs.OutputTypeThumbs,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("thumbs", 0750); err != nil {
		log.Fatalf("Unable to make thumbs dir")
	}

	req := ffthumbs.GenerateRequest{
		MediaURL: examples.StreamURL,
	}

	if err := thumbsGen.Generate(&req); err != nil {
		log.Fatalf("Unable to generate thumbnails: %v", err)
	}

}
