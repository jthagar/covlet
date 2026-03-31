package tplparse

import (
	"strings"
	"testing"
)

func TestParseTopLevelVars_Basic(t *testing.T) {
	src := `Hello {{ .Name }}\nEmail: {{.Email}} and {{ .Phone }}\n{{ if .CompanyToApplyTo }}Apply to {{ .CompanyToApplyTo }} as {{ .RoleToApplyTo }}{{ end }}\n{{ with (index .Experience 0) }}Worked at {{ .Company }}{{ end }}`
	vars := ParseTopLevelVars(src)
	joined := strings.Join(vars, ",")

	mustContain := []string{"Name", "Email", "Phone", "CompanyToApplyTo", "RoleToApplyTo", "Experience"}
	for _, v := range mustContain {
		if !strings.Contains(joined, v) {
			t.Fatalf("expected vars to contain %q, got %v", v, vars)
		}
	}

	seen := map[string]bool{}
	for _, v := range vars {
		if seen[v] {
			t.Fatalf("duplicate var reported: %s", v)
		}
		seen[v] = true
	}
}
