package model

import "testing"

func TestDisplayRootPathShowsOnlyProjectFolder(t *testing.T) {
	if got := DisplayRootPath("/Users/logan/Documents/codebase-go"); got != ".../codebase-go" {
		t.Fatalf("expected compact project folder path, got %q", got)
	}
	if got := DisplayRootPath("/Users/logan/Documents/codebase-go/"); got != ".../codebase-go" {
		t.Fatalf("expected trailing slash to be ignored, got %q", got)
	}
	if got := DisplayRootPath(""); got != "..." {
		t.Fatalf("expected empty path fallback, got %q", got)
	}
}
