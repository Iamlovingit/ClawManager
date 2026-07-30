package services

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"clawreef/internal/models"
)

func TestWriteManagedTeamWorkspaceOverlayPreservesOpenClawDefaultsAndReplacesOldOverlay(t *testing.T) {
	path := filepath.Join(t.TempDir(), teamAgentsFileName)
	if err := os.WriteFile(path, []byte("# Default workspace rules\n\nKeep this content.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedTeamWorkspaceOverlay(path, "# First Team\nmember_id=developer"); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedTeamWorkspaceOverlay(path, "# Updated Team\nmember_id=reviewer"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, expected := range []string{"# Default workspace rules", "Keep this content.", "# Updated Team", "member_id=reviewer"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("overlay result missing %q: %s", expected, got)
		}
	}
	if strings.Contains(got, "# First Team") || strings.Count(got, teamManagedOverlayStart) != 1 {
		t.Fatalf("overlay should replace exactly one prior managed block: %s", got)
	}
}

func TestRepairLitePromptWorkspaceOwnershipUsesInstanceUIDWithoutRecursing(t *testing.T) {
	workspace := t.TempDir()
	promptWorkspace := filepath.Join(workspace, "home", ".openclaw", "workspace")
	nestedDir := filepath.Join(promptWorkspace, "skills")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{teamAgentsFileName, teamSoulFileName, teamConfigFileName, "TOOLS.md"} {
		if err := os.WriteFile(filepath.Join(promptWorkspace, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	unmanagedFile := filepath.Join(promptWorkspace, "USER.md")
	if err := os.WriteFile(unmanagedFile, []byte("preserve user ownership"), 0o644); err != nil {
		t.Fatal(err)
	}
	nestedFile := filepath.Join(nestedDir, "user-skill.md")
	if err := os.WriteFile(nestedFile, []byte("preserve"), 0o644); err != nil {
		t.Fatal(err)
	}

	original := chownLitePromptWorkspacePath
	t.Cleanup(func() { chownLitePromptWorkspacePath = original })
	type call struct {
		path     string
		uid, gid int
	}
	var calls []call
	chownLitePromptWorkspacePath = func(path string, uid, gid int) error {
		calls = append(calls, call{path: filepath.Clean(path), uid: uid, gid: gid})
		return nil
	}

	instance := &models.Instance{ID: 77, Type: "openclaw", InstanceMode: InstanceModeLite, WorkspacePath: &workspace}
	if err := repairLitePromptWorkspaceOwnership(instance); err != nil {
		t.Fatalf("repairLitePromptWorkspaceOwnership() error = %v", err)
	}
	sort.Slice(calls, func(i, j int) bool { return calls[i].path < calls[j].path })
	for _, got := range calls {
		if got.uid != RuntimeLinuxID(77) || got.gid != teamSharedGID {
			t.Fatalf("chown %s used %d:%d, want %d:%d", got.path, got.uid, got.gid, RuntimeLinuxID(77), teamSharedGID)
		}
		if got.path == nestedDir || got.path == nestedFile || got.path == unmanagedFile {
			t.Fatalf("ownership repair must not change user-managed prompt workspace paths")
		}
	}
	if len(calls) != 5 {
		t.Fatalf("ownership repair calls = %#v, want prompt root plus four immediate prompt files", calls)
	}
}

func TestRepairLitePromptWorkspaceOwnershipKeepsReadableRootSquashedFilesUsable(t *testing.T) {
	workspace := t.TempDir()
	promptWorkspace := filepath.Join(workspace, "home", ".openclaw", "workspace")
	if err := os.MkdirAll(promptWorkspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(promptWorkspace, "TOOLS.md"), []byte("tools"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := chownLitePromptWorkspacePath
	t.Cleanup(func() { chownLitePromptWorkspacePath = original })
	chownLitePromptWorkspacePath = func(string, int, int) error {
		return os.ErrPermission
	}
	instance := &models.Instance{ID: 96, Type: "openclaw", InstanceMode: InstanceModeLite, WorkspacePath: &workspace}
	if err := repairLitePromptWorkspaceOwnership(instance); err != nil {
		t.Fatalf("readable root-squashed prompt files must remain compatible: %v", err)
	}
}

func TestWriteLiteOpenClawTeamIdentityFilesUseInjectedWorkspace(t *testing.T) {
	workspace := t.TempDir()
	plans, err := planTeamMembers("team", []CreateTeamMemberRequest{{MemberID: "leader", Role: "leader"}})
	if err != nil {
		t.Fatal(err)
	}
	team := &models.Team{ID: 77, CommunicationMode: teamCommunicationModeLeaderMediated, SharedMountPath: "/team"}
	instance := &models.Instance{Type: "openclaw", InstanceMode: InstanceModeLite, WorkspacePath: &workspace}
	actualAgents := filepath.Join(workspace, "home", ".openclaw", "workspace", teamAgentsFileName)
	if err := os.MkdirAll(filepath.Dir(actualAgents), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(actualAgents, []byte("# OpenClaw default\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := (&teamService{}).writeLiteTeamMemberIdentityFiles(instance, team, plans[0], `{"members":[{"memberId":"leader"}]}`); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{teamAgentsFileName, teamSoulFileName} {
		data, readErr := os.ReadFile(filepath.Join(workspace, "home", ".openclaw", "workspace", name))
		if readErr != nil || !strings.Contains(string(data), teamManagedOverlayStart) {
			t.Fatalf("injected %s invalid: data=%q err=%v", name, string(data), readErr)
		}
		if name == teamSoulFileName && !strings.Contains(string(data), "Member ID: leader") {
			t.Fatalf("injected SOUL.md missing member identity: %s", string(data))
		}
	}
	agents, err := os.ReadFile(actualAgents)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"## Leader Team Context Preflight",
		"./team.json",
		"./team-introduction.md",
		"$CLAWMANAGER_TEAM_SHARED_DIR/team.json",
	} {
		if !strings.Contains(string(agents), expected) {
			t.Fatalf("Leader AGENTS.md missing %q: %s", expected, string(agents))
		}
	}
	roster, err := os.ReadFile(filepath.Join(workspace, "home", ".openclaw", "workspace", teamConfigFileName))
	if err != nil || !strings.Contains(string(roster), `"memberId":"leader"`) {
		t.Fatalf("injected team.json invalid: data=%q err=%v", string(roster), err)
	}
	if _, err := os.Stat(filepath.Join(workspace, teamAgentsFileName)); !os.IsNotExist(err) {
		t.Fatalf("OpenClaw Team AGENTS.md must not be written to the unused workspace root: %v", err)
	}
}
