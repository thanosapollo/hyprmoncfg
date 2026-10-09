package hypr

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// parseIgnoredOutputs validates explicit connector-name globs. No hardware
// metadata heuristic is used: real outputs can legitimately lack EDID data.
func parseIgnoredOutputs(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	patterns := strings.Split(value, ",")
	for i, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return nil, fmt.Errorf("HYPRMONCFG_IGNORE_OUTPUTS contains an empty pattern")
		}
		if _, err := filepath.Match(pattern, ""); err != nil {
			return nil, fmt.Errorf("invalid HYPRMONCFG_IGNORE_OUTPUTS pattern %q: %w", pattern, err)
		}
		patterns[i] = pattern
	}
	return patterns, nil
}

func (c *Client) ignoredOutput(name string) bool {
	for _, pattern := range c.ignoredOutputs {
		if match, _ := filepath.Match(pattern, name); match {
			return true
		}
	}
	return false
}

func (c *Client) filterIgnoredOutputs(monitors []Monitor) []Monitor {
	filtered := monitors[:0]
	for _, monitor := range monitors {
		if !c.ignoredOutput(monitor.Name) {
			// Description selectors match prefixes and may also address an
			// excluded head. Keep hardware identity, but dispatch by connector.
			monitor.ConnectorOnly = len(c.ignoredOutputs) > 0
			filtered = append(filtered, monitor)
		}
	}
	return filtered
}

func (c *Client) filterWorkspaceRules(rules []WorkspaceRule, monitors []Monitor) ([]WorkspaceRule, error) {
	filtered := rules[:0]
	for _, rule := range rules {
		selector := strings.TrimSpace(rule.Monitor)
		if c.ignoredOutput(selector) {
			continue
		}
		ignored, managed := false, false
		for _, monitor := range monitors {
			matches := selector == monitor.Name || selector == strconv.Itoa(monitor.ID) || selector == "id:"+strconv.Itoa(monitor.ID)
			if strings.HasPrefix(selector, "desc:") {
				matches = strings.HasPrefix(strings.TrimSpace(monitor.Description), strings.TrimSpace(strings.TrimPrefix(selector, "desc:")))
			}
			if matches {
				if c.ignoredOutput(monitor.Name) {
					ignored = true
				} else {
					managed = true
				}
			}
		}
		if ignored && managed {
			return nil, fmt.Errorf("workspace monitor selector %q matches both ignored and managed outputs", selector)
		}
		if !ignored {
			filtered = append(filtered, rule)
		}
	}
	return filtered, nil
}

func (c *Client) monitorEvent(line string) (Event, bool) {
	event, ok := parseEvent(line)
	if !ok {
		return event, false
	}
	name := event.Value
	eventType := strings.SplitN(line, ">>", 2)[0]
	if eventType == "monitoraddedv2" || eventType == "monitorremovedv2" {
		fields := strings.SplitN(event.Value, ",", 3)
		if len(fields) > 1 {
			name = fields[1]
		}
	}
	return event, !c.ignoredOutput(name)
}
