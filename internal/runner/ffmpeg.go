package runner

import (
	"fmt"
	"os"
	"os/exec"
)

type FFmpegArgs struct {
	Input   string
	Headers map[string]string
}

func FFmpeg(args FFmpegArgs, outputPath string) error {
	cmdArgs := []string{}

	for k, v := range args.Headers {
		cmdArgs = append(cmdArgs, "-headers", k+": "+v)
	}

	cmdArgs = append(cmdArgs,
		"-i", args.Input,
		"-loglevel", "warning",
		"-c", "copy",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("ffmpeg exited with code %d", ee.ExitCode())
		}
		return err
	}
	return nil
}
