package cli

import (
	"os"
	"strings"
	"testing"
)

func TestDataDir(t *testing.T) {
	tests := []struct {
		name string
		xdg  string
		want string
	}{
		{
			name: "xdg set",
			xdg:  "/custom/data",
			want: "/custom/data/zburn",
		},
		{
			name: "xdg empty falls back to home",
			xdg:  "",
			want: "/.local/share/zburn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("XDG_DATA_HOME", tt.xdg)
			defer os.Unsetenv("XDG_DATA_HOME")

			got := DataDir()
			if tt.xdg != "" {
				if got != tt.want {
					t.Errorf("DataDir() = %s, want %s", got, tt.want)
				}
			} else {
				if !strings.HasSuffix(got, tt.want) {
					t.Errorf("DataDir() = %s, want suffix %s", got, tt.want)
				}
			}
		})
	}
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{"present", []string{"--json", "--save"}, "--json", true},
		{"absent", []string{"--save"}, "--json", false},
		{"empty", nil, "--json", false},
		{"case insensitive", []string{"--JSON"}, "--json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFlag(tt.args, tt.flag)
			if got != tt.want {
				t.Errorf("hasFlag(%v, %s) = %v, want %v", tt.args, tt.flag, got, tt.want)
			}
		})
	}
}

func TestIsFirstRun(t *testing.T) {
	dir := t.TempDir()
	if !IsFirstRun(dir) {
		t.Error("expected first run for empty dir")
	}

	os.WriteFile(dir+"/salt", []byte("test"), 0o600)
	if IsFirstRun(dir) {
		t.Error("expected not first run after salt exists")
	}
}
