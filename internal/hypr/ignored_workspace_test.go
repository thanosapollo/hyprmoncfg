package hypr

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoredWorkspaceQueries(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$4" in
 monitors) printf '%s\n' '[{"id":1,"name":"DP-1","description":"physical"},{"id":7,"name":"LAB","description":"virtual"}]';;
 workspaces) printf '%s\n' '[{"name":"1","monitor":"DP-1"},{"name":"lab","monitor":"LAB"}]';;
 workspacerules) printf '%s\n' '[{"workspaceString":"1","monitor":"DP-1"},{"workspaceString":"lab","monitor":"LAB"},{"workspaceString":"2","monitor":"7"},{"workspaceString":"3","monitor":"id:7"},{"workspaceString":"4","monitor":"desc:virtual"},{"workspaceString":"5","monitor":"disconnected"}]';;
 *) exit 1;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "hyprctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "fixture")
	t.Setenv("HYPRMONCFG_IGNORE_OUTPUTS", "LAB")
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	monitors, err := c.Monitors(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(monitors) != 1 || monitors[0].Name != "DP-1" {
		t.Fatalf("monitors %+v", monitors)
	}
	workspaces, err := c.Workspaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Monitor != "DP-1" {
		t.Fatalf("rollback placements %+v", workspaces)
	}
	rules, err := c.WorkspaceRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].Monitor != "DP-1" || rules[1].Monitor != "disconnected" {
		t.Fatalf("capture/rollback rules %+v", rules)
	}
}

func TestIgnoredDescriptionPrefixesAndDispatchSelectors(t *testing.T) {
	c := &Client{ignoredOutputs: []string{"LAB"}}
	rules, err := c.filterWorkspaceRules([]WorkspaceRule{{Monitor: "desc:virtual"}}, []Monitor{{Name: "LAB", Description: "virtual display"}})
	if err != nil || len(rules) != 0 {
		t.Fatalf("ignored prefix survived: %v %v", rules, err)
	}
	_, err = c.filterWorkspaceRules([]WorkspaceRule{{Monitor: "desc:virtual"}}, []Monitor{{Name: "LAB", Description: "virtual display"}, {Name: "DP-1", Description: "virtual physical"}})
	if err == nil {
		t.Fatal("mixed prefix accepted")
	}
	got := c.filterIgnoredOutputs([]Monitor{{Name: "DP-1", Description: "shared", Make: "Dell"}, {Name: "LAB", Description: "shared"}})
	if len(got) != 1 || got[0].MonitorSelector() != "DP-1" || got[0].HardwareKey() != "dell" || got[0].Description != "shared" {
		t.Fatalf("unsafe selector or changed hardware identity: %+v", got)
	}
}

func TestIgnoredAmbiguousRuleFailsClosed(t *testing.T) {
	c := &Client{ignoredOutputs: []string{"LAB"}}
	_, err := c.filterWorkspaceRules([]WorkspaceRule{{Monitor: "desc:shared"}}, []Monitor{{Name: "LAB", Description: "shared"}, {Name: "DP-1", Description: "shared"}})
	if err == nil {
		t.Fatal("mixed ownership selector accepted")
	}
}
