package workspace

import "testing"

func TestLayoutPaths(t *testing.T) {
	layout := New("/repo")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "config",
			got:  layout.ConfigPath(),
			want: "/repo/skillctl.yaml",
		},
		{
			name: "lock",
			got:  layout.LockPath(),
			want: "/repo/skillctl.lock",
		},
		{
			name: "source",
			got:  layout.SourceDir("superpowers"),
			want: "/repo/sources/superpowers",
		},
		{
			name: "patch",
			got:  layout.PatchFile("brainstorming"),
			want: "/repo/patches/brainstorming/local.patch",
		},
		{
			name: "dist",
			got:  layout.DistSkillDir("brainstorming"),
			want: "/repo/dist/brainstorming",
		},
		{
			name: "backup",
			got:  layout.BackupDir("trae-workspace"),
			want: "/repo/backups/trae-workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("path = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
