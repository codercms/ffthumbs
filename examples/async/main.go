package main

import (
	"github.com/codercms/ffthumbs/examples"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codercms/ffthumbs"
)

// On successful run you will see something like this on command output:
// 2023/08/23 16:58:34 Request 2 processed, time spent: 49.4448109s
// 2023/08/23 16:58:35 Request 1 processed, time spent: 49.9029628s
// 2023/08/23 16:58:35 Background goroutine exit
// 2023/08/23 16:58:35 Done in 49.9029628s
func main() {
	thumbsGen, err := ffthumbs.NewGenerator(&ffthumbs.Config{
		// Concurrency default value is 2, feel free to set concurrency for your needs
		Concurrency: 2,
		Outputs: []*ffthumbs.OutputConfig{
			{
				Scale: ffthumbs.ScaleConfig{
					Width:    320,
					Height:   180,
					Behavior: ffthumbs.ScaleBehaviorFillToKeepAspectRatio,
				},
				SnapshotInterval: time.Millisecond * 6500,
				Type:             ffthumbs.OutputTypeThumbs,
			},
		},

		// Feel free do disable ffmpeg progress logs
		//DisableProgressLogs: true,
	})

	if err != nil {
		log.Fatal(err)
	}

	doneChan := make(chan *ffthumbs.GenerateResult, 1)

	reqs := []ffthumbs.GenerateRequest{
		{
			MediaURL: examples.StreamURL,
			// Here we override DstPath for output 0 (index of OutputConfig in an outputs array)
			OutputDst: map[int]string{
				0: "tears_of_steel_1/%04d.jpg",
			},
			DoneChan: doneChan,
		},
		{
			MediaURL: examples.StreamURL,
			OutputDst: map[int]string{
				0: "tears_of_steel_2/%04d.jpg",
			},
			DoneChan: doneChan,
		},
	}

	start := time.Now()

	var wg sync.WaitGroup

	for _, req := range reqs {
		for _, dst := range req.OutputDst {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				log.Fatalf("cannot create req dst dir: %v", err)
			}
		}

		// See https://github.com/golang/go/wiki/CommonMistakes
		reqCopy := req
		if err := thumbsGen.GenerateAsync(&reqCopy); err != nil {
			log.Fatalf("Unable to send generate thumbnails request: %v", err)
		}

		wg.Add(1)
	}

	bgDoneChan := make(chan struct{})
	go func() {
		for reqDone := range doneChan {
			handleProcessedRequest(&wg, reqDone)
		}

		log.Println("Background goroutine exit")

		bgDoneChan <- struct{}{}
	}()

	wg.Wait()
	close(doneChan)

	<-bgDoneChan

	log.Printf("Done in %s", time.Since(start))
}

func handleProcessedRequest(wg *sync.WaitGroup, res *ffthumbs.GenerateResult) {
	defer wg.Done()

	if res.Err != nil {
		log.Printf("Request %d failed, time spent: %s, err: %v", res.Req.GetId(), res.Duration, res.Err)
	} else {
		log.Printf("Request %d processed, time spent: %s", res.Req.GetId(), res.Duration)
	}
}
