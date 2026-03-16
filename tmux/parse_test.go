package tmux

import (
	"testing"
)

func TestParsePanes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Pane
		wantErr bool
	}{
		{
			name:  "single pane",
			input: "main\t0\tbash\t0\t%0\t200\t50\t1\t12345\tbash\t/home/user\n",
			want: []Pane{
				{
					SessionName:    "main",
					WindowIndex:    0,
					WindowName:     "bash",
					PaneIndex:      0,
					PaneID:         "%0",
					PaneWidth:      200,
					PaneHeight:     50,
					PaneActive:     true,
					PanePID:        12345,
					CurrentCommand: "bash",
					CurrentPath:    "/home/user",
				},
			},
		},
		{
			name: "multiple panes across sessions",
			input: "dev\t0\teditor\t0\t%0\t120\t40\t1\t1001\tvim\t/home/user/project\n" +
				"dev\t0\teditor\t1\t%1\t120\t40\t0\t1002\tbash\t/home/user/project\n" +
				"dev\t1\tserver\t0\t%2\t240\t80\t1\t1003\tnode\t/home/user/project/api\n" +
				"ops\t0\tlogs\t0\t%3\t160\t60\t1\t2001\ttail\t/var/log\n",
			want: []Pane{
				{
					SessionName:    "dev",
					WindowIndex:    0,
					WindowName:     "editor",
					PaneIndex:      0,
					PaneID:         "%0",
					PaneWidth:      120,
					PaneHeight:     40,
					PaneActive:     true,
					PanePID:        1001,
					CurrentCommand: "vim",
					CurrentPath:    "/home/user/project",
				},
				{
					SessionName:    "dev",
					WindowIndex:    0,
					WindowName:     "editor",
					PaneIndex:      1,
					PaneID:         "%1",
					PaneWidth:      120,
					PaneHeight:     40,
					PaneActive:     false,
					PanePID:        1002,
					CurrentCommand: "bash",
					CurrentPath:    "/home/user/project",
				},
				{
					SessionName:    "dev",
					WindowIndex:    1,
					WindowName:     "server",
					PaneIndex:      0,
					PaneID:         "%2",
					PaneWidth:      240,
					PaneHeight:     80,
					PaneActive:     true,
					PanePID:        1003,
					CurrentCommand: "node",
					CurrentPath:    "/home/user/project/api",
				},
				{
					SessionName:    "ops",
					WindowIndex:    0,
					WindowName:     "logs",
					PaneIndex:      0,
					PaneID:         "%3",
					PaneWidth:      160,
					PaneHeight:     60,
					PaneActive:     true,
					PanePID:        2001,
					CurrentCommand: "tail",
					CurrentPath:    "/var/log",
				},
			},
		},
		{
			name:  "pane with spaces in path",
			input: "work\t0\tcode\t0\t%5\t180\t45\t1\t9999\tzsh\t/home/user/my project\n",
			want: []Pane{
				{
					SessionName:    "work",
					WindowIndex:    0,
					WindowName:     "code",
					PaneIndex:      0,
					PaneID:         "%5",
					PaneWidth:      180,
					PaneHeight:     45,
					PaneActive:     true,
					PanePID:        9999,
					CurrentCommand: "zsh",
					CurrentPath:    "/home/user/my project",
				},
			},
		},
		{
			name:  "inactive pane",
			input: "main\t0\tbash\t0\t%0\t200\t50\t0\t12345\tbash\t/home/user\n",
			want: []Pane{
				{
					SessionName:    "main",
					WindowIndex:    0,
					WindowName:     "bash",
					PaneIndex:      0,
					PaneID:         "%0",
					PaneWidth:      200,
					PaneHeight:     50,
					PaneActive:     false,
					PanePID:        12345,
					CurrentCommand: "bash",
					CurrentPath:    "/home/user",
				},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  []Pane{},
		},
		{
			name:  "only newlines",
			input: "\n\n\n",
			want:  []Pane{},
		},
		{
			name:    "too few fields",
			input:   "main\t0\tbash\t0\t%0\n",
			wantErr: true,
		},
		{
			name:    "too many fields",
			input:   "main\t0\tbash\t0\t%0\t200\t50\t1\t12345\tbash\t/home\textra\n",
			wantErr: true,
		},
		{
			name:    "invalid window index",
			input:   "main\tabc\tbash\t0\t%0\t200\t50\t1\t12345\tbash\t/home\n",
			wantErr: true,
		},
		{
			name:    "invalid pane index",
			input:   "main\t0\tbash\txyz\t%0\t200\t50\t1\t12345\tbash\t/home\n",
			wantErr: true,
		},
		{
			name:    "invalid pane width",
			input:   "main\t0\tbash\t0\t%0\tbad\t50\t1\t12345\tbash\t/home\n",
			wantErr: true,
		},
		{
			name:    "invalid pane height",
			input:   "main\t0\tbash\t0\t%0\t200\tbad\t1\t12345\tbash\t/home\n",
			wantErr: true,
		},
		{
			name:    "invalid pane pid",
			input:   "main\t0\tbash\t0\t%0\t200\t50\t1\tnope\tbash\t/home\n",
			wantErr: true,
		},
		{
			name:  "trailing carriage return",
			input: "main\t0\tbash\t0\t%0\t200\t50\t1\t12345\tbash\t/home/user\r\n",
			want: []Pane{
				{
					SessionName:    "main",
					WindowIndex:    0,
					WindowName:     "bash",
					PaneIndex:      0,
					PaneID:         "%0",
					PaneWidth:      200,
					PaneHeight:     50,
					PaneActive:     true,
					PanePID:        12345,
					CurrentCommand: "bash",
					CurrentPath:    "/home/user",
				},
			},
		},
		{
			name: "real tmux output - typical session",
			input: "orchid\t0\tzsh\t0\t%0\t238\t58\t1\t48291\tzsh\t/home/dev/tmux-orchid\n" +
				"orchid\t1\tserver\t0\t%1\t119\t58\t1\t48350\tgo\t/home/dev/tmux-orchid\n" +
				"orchid\t1\tserver\t1\t%2\t118\t58\t0\t48395\ttail\t/home/dev/tmux-orchid/logs\n" +
				"agents\t0\tclaude\t0\t%3\t238\t58\t1\t50012\tclaude\t/home/dev/project-alpha\n" +
				"agents\t0\tclaude\t1\t%4\t238\t29\t0\t50100\tbash\t/home/dev/project-alpha\n",
			want: []Pane{
				{
					SessionName:    "orchid",
					WindowIndex:    0,
					WindowName:     "zsh",
					PaneIndex:      0,
					PaneID:         "%0",
					PaneWidth:      238,
					PaneHeight:     58,
					PaneActive:     true,
					PanePID:        48291,
					CurrentCommand: "zsh",
					CurrentPath:    "/home/dev/tmux-orchid",
				},
				{
					SessionName:    "orchid",
					WindowIndex:    1,
					WindowName:     "server",
					PaneIndex:      0,
					PaneID:         "%1",
					PaneWidth:      119,
					PaneHeight:     58,
					PaneActive:     true,
					PanePID:        48350,
					CurrentCommand: "go",
					CurrentPath:    "/home/dev/tmux-orchid",
				},
				{
					SessionName:    "orchid",
					WindowIndex:    1,
					WindowName:     "server",
					PaneIndex:      1,
					PaneID:         "%2",
					PaneWidth:      118,
					PaneHeight:     58,
					PaneActive:     false,
					PanePID:        48395,
					CurrentCommand: "tail",
					CurrentPath:    "/home/dev/tmux-orchid/logs",
				},
				{
					SessionName:    "agents",
					WindowIndex:    0,
					WindowName:     "claude",
					PaneIndex:      0,
					PaneID:         "%3",
					PaneWidth:      238,
					PaneHeight:     58,
					PaneActive:     true,
					PanePID:        50012,
					CurrentCommand: "claude",
					CurrentPath:    "/home/dev/project-alpha",
				},
				{
					SessionName:    "agents",
					WindowIndex:    0,
					WindowName:     "claude",
					PaneIndex:      1,
					PaneID:         "%4",
					PaneWidth:      238,
					PaneHeight:     29,
					PaneActive:     false,
					PanePID:        50100,
					CurrentCommand: "bash",
					CurrentPath:    "/home/dev/project-alpha",
				},
			},
		},
		{
			name:  "high pane ids and large dimensions",
			input: "prod\t5\tmonitor\t3\t%127\t400\t100\t0\t99999\thtop\t/root\n",
			want: []Pane{
				{
					SessionName:    "prod",
					WindowIndex:    5,
					WindowName:     "monitor",
					PaneIndex:      3,
					PaneID:         "%127",
					PaneWidth:      400,
					PaneHeight:     100,
					PaneActive:     false,
					PanePID:        99999,
					CurrentCommand: "htop",
					CurrentPath:    "/root",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePanes(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d panes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("pane %d:\n  got  %+v\n  want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
