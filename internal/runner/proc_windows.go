//go:build windows

package runner

import "os/exec"

func detachProcess(cmd *exec.Cmd) {}
