package hypr

import "testing"

func TestIgnoredOutputsFilterQueriesAndEvents(t *testing.T) {
	patterns, err := parseIgnoredOutputs(" LAB , HEADLESS-* , WAYLAND-* ")
	if err != nil {
		t.Fatal(err)
	}
	c := &Client{ignoredOutputs: patterns}
	monitors := []Monitor{{Name: "DP-1"}, {Name: "LAB"}, {Name: "HEADLESS-1"}, {Name: "WAYLAND-1"}, {Name: "HDMI-A-1", Disabled: true}}
	got := c.filterIgnoredOutputs(monitors)
	if len(got) != 2 || got[0].Name != "DP-1" || got[1].Name != "HDMI-A-1" {
		t.Fatalf("physical outputs not preserved: %+v", got)
	}
	for _, line := range []string{"monitoradded>>LAB", "monitorremoved>>LAB", "monitoraddedv2>>7,LAB,", "monitorremovedv2>>7,LAB,", "monitoradded>>HEADLESS-2", "monitoradded>>WAYLAND-3"} {
		if _, ok := c.monitorEvent(line); ok {
			t.Fatalf("ignored event passed: %s", line)
		}
	}
	for _, line := range []string{"monitoradded>>DP-1", "monitoraddedv2>>1,HDMI-A-1,Dell", "monitorremoved>>DP-1"} {
		if _, ok := c.monitorEvent(line); !ok {
			t.Fatalf("physical event lost: %s", line)
		}
	}
}

func TestIgnoredOutputPatternsRejectInvalidSyntax(t *testing.T) {
	for _, value := range []string{"[", "LAB,[", "LAB,,HEADLESS-*"} {
		if _, err := parseIgnoredOutputs(value); err == nil {
			t.Fatalf("accepted malformed patterns %q", value)
		}
	}
	if patterns, err := parseIgnoredOutputs(""); err != nil || len(patterns) != 0 {
		t.Fatalf("empty default: %v %v", patterns, err)
	}
	c := &Client{}
	if got := c.filterIgnoredOutputs([]Monitor{{Name: "LAB"}, {Name: "DP-1"}}); len(got) != 2 {
		t.Fatal("unset option changed behavior")
	}
}
