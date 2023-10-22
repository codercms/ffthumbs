package ffthumbs

import (
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-version"
)

var (
	minVersionRequired = version.Must(version.NewVersion("5.0.0"))
)

// GetFfmpegVersion returns ffmpeg version number, e.g. 6.0 or 5.3.1
func GetFfmpegVersion(ffmpegPath string) (*version.Version, error) {
	output, err := exec.Command(ffmpegPath, "-version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cannot check ffmpeg version: %w", err)
	}

	if match := versionPattern.FindStringSubmatch(string(output)); len(match) > 1 {
		ver, err := version.NewVersion(match[1])
		if err != nil {
			return nil, fmt.Errorf("wrong ffmpeg version reported: %s :%w", match[1], err)
		}

		return ver, nil
	}

	return nil, fmt.Errorf("cannot find ffmpeg version")
}

// VerifyFfmpegVersion verifies that the provided ffmpeg binary meets the minimal version requirement
func VerifyFfmpegVersion(ffmpegPath string) error {
	ver, err := GetFfmpegVersion(ffmpegPath)
	if err != nil {
		return err
	}

	if ver.LessThan(minVersionRequired) {
		return fmt.Errorf("ffmpeg is too old: required %s, current %s", minVersionRequired, ver)
	}

	return nil
}

// FindFfmpeg finds path to ffmpeg in OS $PATH path variable
func FindFfmpeg() (string, error) {
	// Find full path to the "ffmpeg" executable
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("cannot find ffmpeg binary in OS $PATH variable: %w", err)
	}

	return ffmpegPath, nil
}

// FindProbe finds path to ffprobe in OS $PATH path variable
func FindProbe() (string, error) {
	// Find full path to the "ffprobe" executable
	ffmpegPath, err := exec.LookPath("ffprobe")
	if err != nil {
		return "", fmt.Errorf("cannot find ffmpeg binary in OS $PATH variable: %w", err)
	}

	return ffmpegPath, nil
}

func getVerifiedFfmpegPath(ffmpegPath string) (string, error) {
	if len(ffmpegPath) == 0 {
		realPath, err := FindFfmpeg()
		if err != nil {
			return "", err
		}

		ffmpegPath = realPath
	}

	if err := VerifyFfmpegVersion(ffmpegPath); err != nil {
		return "", err
	}

	return ffmpegPath, nil
}

func getVerifiedFfprobePath(ffmpegPath string) (string, error) {
	if len(ffmpegPath) == 0 {
		realPath, err := FindProbe()
		if err != nil {
			return "", err
		}

		ffmpegPath = realPath
	}

	//if err := VerifyFfmpegVersion(ffmpegPath); err != nil {
	//	return "", err
	//}

	return ffmpegPath, nil
}
