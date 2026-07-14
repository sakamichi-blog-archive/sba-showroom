package runner

import (
	"fmt"
	"os"
	"os/exec"
)

type FFmpegArgs struct {
	Input    string
	Headers  map[string]string
	Detached bool // run in own process group so terminal Ctrl+C doesn't reach ffmpeg
}

// FFmpegProcess is a running ffmpeg process.
type FFmpegProcess struct {
	cmd  *exec.Cmd
	done <-chan error
}

// Wait blocks until ffmpeg exits and returns its error. Call at most once.
func (p *FFmpegProcess) Wait() error {
	return <-p.done
}

// Stop sends SIGINT to ffmpeg to request a graceful stop. If the signal
// cannot be delivered, it falls back to Kill.
func (p *FFmpegProcess) Stop() {
	if err := p.cmd.Process.Signal(os.Interrupt); err != nil {
		_ = p.cmd.Process.Kill()
	}
}

// Kill sends SIGKILL to ffmpeg unconditionally.
func (p *FFmpegProcess) Kill() {
	_ = p.cmd.Process.Kill()
}

// StartFFmpeg starts ffmpeg and returns a handle immediately.
func StartFFmpeg(args FFmpegArgs, outputPath string) (*FFmpegProcess, error) {
	cmd := exec.Command("ffmpeg", buildCmdArgs(args, outputPath)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if args.Detached {
		cmd.Stdin = nil
		detachProcess(cmd)
	} else {
		cmd.Stdin = os.Stdin
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("ffmpeg exited with code %d", ee.ExitCode())
		}
		ch <- err
	}()

	return &FFmpegProcess{cmd: cmd, done: ch}, nil
}

// FFmpeg runs ffmpeg synchronously.
func FFmpeg(args FFmpegArgs, outputPath string) error {
	p, err := StartFFmpeg(args, outputPath)
	if err != nil {
		return err
	}
	return p.Wait()
}

func buildCmdArgs(args FFmpegArgs, outputPath string) []string {
	var cmdArgs []string
	for k, v := range args.Headers {
		cmdArgs = append(cmdArgs, "-headers", k+": "+v)
	}
	return append(cmdArgs,
		"-i", args.Input,
		"-loglevel", "warning",
		"-c", "copy",
		outputPath,
	)
}
