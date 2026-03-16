//go:build linux

package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// processInfo reads process metadata from /proc/<pid> on Linux.
func processInfo(pid int) (*ProcessInfo, error) {
	procDir := fmt.Sprintf("/proc/%d", pid)

	name, err := readProcFile(filepath.Join(procDir, "comm"))
	if err != nil {
		return nil, fmt.Errorf("reading comm for pid %d: %w", pid, err)
	}

	cmdlineRaw, err := readProcFile(filepath.Join(procDir, "cmdline"))
	if err != nil {
		cmdlineRaw = ""
	}
	cmdline := strings.ReplaceAll(cmdlineRaw, "\x00", " ")
	cmdline = strings.TrimSpace(cmdline)

	cwd, err := os.Readlink(filepath.Join(procDir, "cwd"))
	if err != nil {
		cwd = ""
	}

	return &ProcessInfo{
		PID:     pid,
		PPID:    readPPID(pid),
		Name:    strings.TrimSpace(name),
		Cmdline: cmdline,
		CWD:     cwd,
	}, nil
}

// childrenOf returns the PIDs of all direct children of the given PID.
func childrenOf(pid int) ([]int, error) {
	childrenPath := fmt.Sprintf("/proc/%d/task/%d/children", pid, pid)
	data, err := os.ReadFile(childrenPath)
	if err == nil {
		return parseSpaceSeparatedPIDs(strings.TrimSpace(string(data))), nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	var children []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPID, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if readPPID(childPID) == pid {
			children = append(children, childPID)
		}
	}
	return children, nil
}

// readPPID reads the parent PID from /proc/<pid>/stat. Returns 0 on error.
func readPPID(pid int) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	return parsePPIDFromStat(string(data))
}

// parsePPIDFromStat extracts the PPID (field 4) from a /proc/pid/stat line.
func parsePPIDFromStat(stat string) int {
	idx := strings.LastIndex(stat, ")")
	if idx < 0 || idx+2 >= len(stat) {
		return 0
	}
	rest := strings.TrimSpace(stat[idx+1:])
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return 0
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return ppid
}

// readProcFile reads the contents of a /proc file.
func readProcFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// parseSpaceSeparatedPIDs parses a space-separated list of PID values.
func parseSpaceSeparatedPIDs(s string) []int {
	if s == "" {
		return nil
	}
	fields := strings.Fields(s)
	pids := make([]int, 0, len(fields))
	for _, f := range fields {
		pid, err := strconv.Atoi(f)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}
