# ffthumbs
Golang FFmpeg thumbnails generator

# Requirements
* FFmpeg >= 5.0.0
* Go >= 1.21.0

## Supported thumbnails format
* Simple thumbnails (OutputTypeThumbs)
* Sprites (each sprite contains multiple thumbs - tiles) (OutputTypeSprites)

## Supported scale operations
* Scale to fixed resolution (set width and height to fixed numbers)
  * Fill to fit into fixed resolution aspect ratio (ScaleBehaviorFillToKeepAspectRatio)
  * Crop to fit into fixed resolution (ScaleBehaviorCropToFit)
* Automatically scale to preserve original media aspect ratio (set width or height to -1)

For more options see [config.go](config.go)

This package is goroutine-safe (could be used with an unlimited number of concurrent calls).

## Examples
* [Thumbnails generation](examples/thumbs/main.go)
* [Sprites generation](examples/sprites/main.go)
* [Thumbnails & Sprites generation](examples/multiple/main.go)
* [Asynchronous processing](examples/async/main.go)
