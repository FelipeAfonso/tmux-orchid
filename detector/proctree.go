package detector

// ProcessInfo holds the metadata for a single OS process, as read from the
// platform-specific process table.
type ProcessInfo struct {
	PID     int
	PPID    int
	Name    string
	Cmdline string
	CWD     string
}

// maxDepth limits how deep we walk the process tree to avoid runaway loops.
const maxDepth = 32

// walkTree walks the process tree starting from rootPID, looking for a
// process whose name or cmdline matches a known agent. The walk is
// breadth-first through direct children at each level.
func walkTree(rootPID int) (*ProcessInfo, AgentType) {
	visited := make(map[int]bool)
	queue := []int{rootPID}

	for depth := 0; depth < maxDepth && len(queue) > 0; depth++ {
		var nextQueue []int
		for _, pid := range queue {
			if visited[pid] {
				continue
			}
			visited[pid] = true

			info, err := processInfo(pid)
			if err != nil {
				continue
			}

			agent := ClassifyProcess(info.Name, info.Cmdline)
			if agent != AgentUnknown {
				return info, agent
			}

			children, err := childrenOf(pid)
			if err != nil {
				continue
			}
			nextQueue = append(nextQueue, children...)
		}
		queue = nextQueue
	}

	return nil, AgentUnknown
}
