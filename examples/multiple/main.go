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
			{
				DstPath: "sprites/%04d.jpg",
				Scale: ffthumbs.ScaleConfig{
					Width:    320,
					Height:   180,
					Behavior: ffthumbs.ScaleBehaviorFillToKeepAspectRatio,
				},
				SnapshotInterval: time.Millisecond * 6500,
				Type:             ffthumbs.OutputTypeSprites,
				Sprites: ffthumbs.SpritesConfig{
					Dimensions: ffthumbs.SpriteDimensions{
						Columns: 1,
						Rows:    64,
					},
				},
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("thumbs", 0750); err != nil {
		log.Fatalf("Unable to make thumbs dir")
	}
	if err := os.MkdirAll("sprites", 0750); err != nil {
		log.Fatalf("Unable to make sprites dir")
	}

	req := ffthumbs.GenerateRequest{
		MediaURL: examples.StreamURL,
	}

	if err := thumbsGen.Generate(&req); err != nil {
		log.Fatalf("Unable to generate thumbnails: %v", err)
	}

}
