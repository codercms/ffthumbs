package ffthumbs

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func BuildHeadersStr(headers map[string]string) string {
	var builder strings.Builder

	for key, val := range headers {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(val)
		builder.WriteString("\r\n")
	}

	return builder.String()
}

// BuildComplexFilters builds ffmpeg -filter_complex arg based on provided outputs config,
// on fail it returns ValidationError
func BuildComplexFilters(outputs []*OutputConfig) (string, error) {
	if err := validateOutputs(outputs); err != nil {
		return "", err
	}

	grpOutputs := groupOutputs(outputs)

	var builder strings.Builder

	var mainIdx int
	for _, subgrp := range grpOutputs {
		// Outputs with same scale settings can be optimized
		if len(subgrp) == 1 {
			var outputs []*OutputConfig

			for _, subgrpOutputs := range subgrp {
				outputs = subgrpOutputs
				break
			}

			builder.WriteString(buildSelectFramesArg(outputs[0]))
			builder.WriteString(buildScaleArg(&outputs[0].Scale))
			builder.WriteString(buildSplitArg(outputs))

			mainIdx++
			// More outputs are coming
			if mainIdx < len(grpOutputs) {
				builder.WriteString(";")
			}

			continue
		}

		// Handle outputs with different scale settings
		for _, outputs := range subgrp {
			for idx, output := range outputs {
				builder.WriteString(buildSelectFramesArg(output))
				builder.WriteString(buildScaleArg(&output.Scale))
				builder.WriteString(buildSplitArg([]*OutputConfig{output}))

				if idx+1 < len(outputs) {
					builder.WriteString(";")
				}
			}
		}

		mainIdx++
		// More outputs are coming
		if mainIdx < len(grpOutputs) {
			builder.WriteString(";")
		}
	}

	return builder.String(), nil
}

// groupOutputs groups outputs by snapshot interval and then scale settings
func groupOutputs(outputs []*OutputConfig) map[time.Duration]map[string][]*OutputConfig {
	res := map[time.Duration]map[string][]*OutputConfig{}

	for _, output := range outputs {
		var ok bool
		var snapshotMap map[string][]*OutputConfig

		if snapshotMap, ok = res[output.SnapshotInterval]; !ok {
			snapshotMap = map[string][]*OutputConfig{}
			res[output.SnapshotInterval] = snapshotMap
		}

		tmpBytes := make([]byte, 0, 24)
		binary.LittleEndian.AppendUint64(tmpBytes, uint64(output.Scale.Behavior))
		binary.LittleEndian.AppendUint64(tmpBytes, uint64(output.Scale.Width))
		binary.LittleEndian.AppendUint64(tmpBytes, uint64(output.Scale.Height))

		hashSum := sha256.Sum256(tmpBytes)
		hashStr := hex.EncodeToString(hashSum[:])

		if _, ok = snapshotMap[hashStr]; !ok {
			snapshotMap[hashStr] = []*OutputConfig{output}
		} else {
			snapshotMap[hashStr] = append(snapshotMap[hashStr], output)
		}
	}

	return res
}

func buildSelectFramesArg(output *OutputConfig) string {
	var builder strings.Builder

	builder.WriteString(`[0:v]select=bitor(gte(t-prev_selected_t\,`)
	builder.WriteString(fmt.Sprintf("%g", output.SnapshotInterval.Truncate(time.Microsecond).Seconds()))
	builder.WriteString(`)\,isnan(prev_selected_t)),`)

	return builder.String()
}

func buildScaleArg(scale *ScaleConfig) string {
	var builder strings.Builder

	builder.WriteString(`scale=`)
	builder.WriteString(strconv.Itoa(scale.Width))
	builder.WriteString(`:`)
	builder.WriteString(strconv.Itoa(scale.Height))

	switch scale.Behavior {
	case ScaleBehaviorFillToKeepAspectRatio:
		builder.WriteString(`:force_original_aspect_ratio=decrease,pad=`)
		builder.WriteString(strconv.Itoa(scale.Width))
		builder.WriteString(`:`)
		builder.WriteString(strconv.Itoa(scale.Height))
		builder.WriteString(`:-1:-1:color=black`)
	case ScaleBehaviorCropToFit:
		builder.WriteString(`:force_original_aspect_ratio=increase,crop=`)
		builder.WriteString(strconv.Itoa(scale.Width))
		builder.WriteString(`:`)
		builder.WriteString(strconv.Itoa(scale.Height))
	}

	return builder.String()
}

func buildSplitArg(outputs []*OutputConfig) string {
	var builder strings.Builder

	for _, output := range outputs {
		switch output.Type {
		case OutputTypeThumbs:
			output.outName = buildSplitArgThumbOutName(output)
		case OutputTypeSprites:
			in, out := buildSplitArgSpiteInOutNames(output)
			output.inName = in
			output.outName = out
		}
	}

	if len(outputs) == 1 {
		output := outputs[0]

		switch output.Type {
		case OutputTypeThumbs:
			writeFilterOutputName(&builder, output.outName)
		case OutputTypeSprites:
			builder.WriteString(",")
			builder.WriteString(buildSplitSpriteTileArg(output))
			writeFilterOutputName(&builder, output.outName)
		}

		return builder.String()
	}

	builder.WriteString(",split=")
	builder.WriteString(strconv.Itoa(len(outputs)))

	var needExtendedProcessing []*OutputConfig

	for _, output := range outputs {
		builder.WriteString("[")

		switch output.Type {
		case OutputTypeThumbs:
			builder.WriteString(output.outName)
		case OutputTypeSprites:
			needExtendedProcessing = append(needExtendedProcessing, output)
			builder.WriteString(output.inName)
		}

		builder.WriteString("]")
	}

	if len(needExtendedProcessing) == 0 {
		return builder.String()
	}

	builder.WriteString(";")

	var idx = 0
	for _, output := range needExtendedProcessing {
		switch output.Type {
		case OutputTypeSprites:
			builder.WriteString(buildSplitSpriteArg(output))
		}

		idx++
		if idx < len(outputs) {
			builder.WriteString(";")
		}
	}

	return builder.String()
}

func buildSplitArgSpiteInOutNames(output *OutputConfig) (in, out string) {
	var nameBuilder strings.Builder

	nameBuilder.WriteString("sprites-")
	nameBuilder.WriteString(strconv.Itoa(output.idx))

	in = nameBuilder.String()

	nameBuilder.WriteString("-out")

	out = nameBuilder.String()

	return
}

func buildSplitArgThumbOutName(output *OutputConfig) string {
	var nameBuilder strings.Builder

	nameBuilder.WriteString("thumbs-")
	nameBuilder.WriteString(strconv.Itoa(output.idx))
	nameBuilder.WriteString("-out")

	return nameBuilder.String()
}

func buildSplitSpriteArg(output *OutputConfig) string {
	var builder strings.Builder

	writeFilterOutputName(&builder, output.inName)

	builder.WriteString(buildSplitSpriteTileArg(output))

	writeFilterOutputName(&builder, output.outName)

	return builder.String()
}

func buildSplitSpriteTileArg(output *OutputConfig) string {
	var builder strings.Builder

	builder.WriteString("tile=")
	builder.WriteString(strconv.Itoa(output.Sprites.Dimensions.Columns))
	builder.WriteString(`x`)
	builder.WriteString(strconv.Itoa(output.Sprites.Dimensions.Rows))

	return builder.String()
}

func writeFilterOutputName(builder *strings.Builder, name string) {
	builder.WriteString("[")
	builder.WriteString(name)
	builder.WriteString("]")
}
