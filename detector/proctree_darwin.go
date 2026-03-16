//go:build darwin

package detector

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// processInfo reads process metadata using ps(1) on macOS.
func processInfo(pid int) (*ProcessInfo, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps ppid for pid %d: %w", pid, err)
	}
	ppid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return nil, fmt.Errorf("parsing ppid for pid %d: %w", pid, err)
	}

	nameOut, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps comm for pid %d: %w", pid, err)
	}
	name := strings.TrimSpace(string(nameOut))

	argsOut, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		argsOut = nil
	}
	cmdline := strings.TrimSpace(string(argsOut))

	cwd := readCWDDarwin(pid)

	return &ProcessInfo{
		PID:     pid,
		PPID:    ppid,
		Name:    name,
		Cmdline: cmdline,
		CWD:     cwd,
	}, nil
}

// childrenOf returns the PIDs of all direct children of the given PID.
func childrenOf(pid int) ([]int, error) {
	out, err := exec.Command("ps", "-eo", "pid=,ppid=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps listing for children of %d: %w", pid, err)
	}

	var children []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		childPID, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		parentPID, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		if parentPID == pid {
			children = append(children, childPID)
		}
	}

	return children, nil
}

// readCWDDarwin attempts to read the cwd of a process on macOS using lsof.
func readCWDDarwin(pid int) string {
	out, err := exec.Command("lsof", "-p", strconv.Itoa(pid), "-Fn", "-a", "-d", "cwd").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n")
		}
	}
	return ""
}
