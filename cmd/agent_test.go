package cmd

import (
	"testing"
)

func TestBuildProfile(t *testing.T) {
	profile := buildProfile("test-agent", "standard", "*", "prompt", true)
	if profile.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", profile.Name, "test-agent")
	}
	if profile.ApprovalMode != "prompt" {
		t.Errorf("ApprovalMode = %q, want %q", profile.ApprovalMode, "prompt")
	}
	if !profile.RequireApproval {
		t.Error("RequireApproval should be true")
	}
}

func TestBuildProfileDefaults(t *testing.T) {
	profile := buildProfile("readonly-agent", "read-only", "bank/*", "deny", false)
	if profile.Name != "readonly-agent" {
		t.Errorf("Name = %q", profile.Name)
	}
	if profile.ApprovalMode != "deny" {
		t.Errorf("ApprovalMode = %q, want %q", profile.ApprovalMode, "deny")
	}
	if profile.RequireApproval {
		t.Error("RequireApproval should be false")
	}
	if len(profile.AllowedPaths) != 1 || profile.AllowedPaths[0] != "bank/*" {
		t.Errorf("AllowedPaths = %v, want [bank/*]", profile.AllowedPaths)
	}
}
