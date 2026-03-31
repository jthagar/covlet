package templatevars

import (
	"strings"

	"github.com/jthagar/covlet/backend/pkg/config"
	"github.com/jthagar/covlet/pkg/tplparse"
)

// ParseTopLevelVars extracts top-level variable names referenced as {{ .Name }} etc.
func ParseTopLevelVars(s string) []string {
	return tplparse.ParseTopLevelVars(s)
}

// ResumeTemplateData builds the data map passed to text/template.Execute. Overrides
// (including keys not on Resume) are merged on top so {{ .Custom }} can be supplied
// from the TUI.
func ResumeTemplateData(r config.Resume, overrides map[string]string) map[string]interface{} {
	out := map[string]interface{}{
		"Name":             r.Name,
		"Email":            r.Email,
		"Phone":            r.Phone,
		"Address":          r.Address,
		"Website":          r.Website,
		"Github":           r.Github,
		"Education":        r.Education,
		"Experience":       r.Experience,
		"Skills":           r.Skills,
		"Projects":         r.Projects,
		"CompanyToApplyTo": r.CompanyToApplyTo,
		"RoleToApplyTo":    r.RoleToApplyTo,
	}
	for k, v := range overrides {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// ApplyOverrides returns a copy of Resume with string fields overridden by the map.
func ApplyOverrides(in config.Resume, overrides map[string]string) config.Resume {
	if len(overrides) == 0 {
		return in
	}
	out := in
	if v, ok := overrides["Name"]; ok {
		out.Name = v
	}
	if v, ok := overrides["Email"]; ok {
		out.Email = v
	}
	if v, ok := overrides["Phone"]; ok {
		out.Phone = v
	}
	if v, ok := overrides["Address"]; ok {
		out.Address = v
	}
	if v, ok := overrides["Website"]; ok {
		out.Website = v
	}
	if v, ok := overrides["Github"]; ok {
		out.Github = v
	}
	if v, ok := overrides["CompanyToApplyTo"]; ok {
		out.CompanyToApplyTo = v
	}
	if v, ok := overrides["RoleToApplyTo"]; ok {
		out.RoleToApplyTo = v
	}
	return out
}

// SanitizeFileName filters characters that are unsafe in filenames.
func SanitizeFileName(s string) string {
	r := make([]rune, 0, len(s))
	for _, ch := range s {
		switch ch {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			r = append(r, '_')
		default:
			r = append(r, ch)
		}
	}
	return string(r)
}
