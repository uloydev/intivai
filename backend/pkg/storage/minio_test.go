package storage

import "testing"

func TestValidateOrgObjectPath(t *testing.T) {
	orgID := "a1b2c3d4-0000-4000-8000-000000000000"
	foreign := "f9f8f7f6-0000-4000-8000-000000000000"
	tests := []struct {
		name string
		path string
		org  string
		want bool
	}{
		{"cv path valid", "cvs/" + orgID + "/candidate-uuid.pdf", orgID, true},
		{"context path valid", "contexts/" + orgID + "/abc123.pdf", orgID, true},
		{"deep context valid", "contexts/" + orgID + "/a/b/file.txt", orgID, true},
		{"foreign org cv rejected", "cvs/" + foreign + "/candidate-uuid.pdf", orgID, false},
		{"foreign org context rejected", "contexts/" + foreign + "/abc.pdf", orgID, false},
		{"traversal rejected", "contexts/" + orgID + "/../../etc/passwd", orgID, false},
		{"leading traversal rejected", "../cvs/" + orgID + "/x.pdf", orgID, false},
		{"absolute rejected", "/cvs/" + orgID + "/x.pdf", orgID, false},
		{"dot segment rejected", "contexts/" + orgID + "/./x.pdf", orgID, false},
		{"empty segment rejected", "contexts//x.pdf", orgID, false},
		{"org swapped to namespace rejected", orgID + "/contexts/x.pdf", orgID, false},
		{"missing org segment rejected", "contexts/x.pdf", orgID, false},
		{"empty path rejected", "", orgID, false},
		{"empty org rejected", "cvs//x.pdf", "", false},
		{"org id not uuid still prefix-checked", "cvs/demo-org/abc.pdf", "demo-org", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateOrgObjectPath(tt.path, tt.org); got != tt.want {
				t.Fatalf("ValidateOrgObjectPath(%q, %q) = %v, want %v", tt.path, tt.org, got, tt.want)
			}
		})
	}
}
