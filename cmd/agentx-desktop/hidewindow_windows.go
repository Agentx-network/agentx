package main

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow sets the process attributes to prevent a console window
// from appearing when launching a subprocess on Windows.
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
