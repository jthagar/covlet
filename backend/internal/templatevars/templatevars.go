package templatevars

import (
	"strings"

	"github.com/jthagar/covlet/backend/pkg/config"
)

// ParseTopLevelVars extracts top-level variable names referenced as {{ .Name }} etc.
func ParseTopLevelVars(s string) []string {
	type void struct{}
	seen := map[string]void{}
	order := []string{}
	i := 0
	for i < len(s) {
		start := strings.Index(s[i:], "{{")
		if start < 0 {
			break
		}
		start += i + 2
		endRel := strings.Index(s[start:], "}}")
		if endRel < 0 {
			break
		}
		end := start + endRel
		expr := s[start:end]
		j := 0
		for j < len(expr) && (expr[j] == ' ' || expr[j] == '\n' || expr[j] == '\t') {
			j++
		}
		for j < len(expr) {
			if expr[j] == '.' {
				k := j + 1
				if k < len(expr) && isIdentStart(expr[k]) {
					startName := k
					k++
					for k < len(expr) && isIdentPart(expr[k]) {
						k++
					}
					name := expr[startName:k]
					if name != "" {
						if _, ok := seen[name]; !ok {
							seen[name] = void{}
							order = append(order, name)
						}
					}
				}
			}
			j++
		}

		i = end + 2
	}
	return order
}

func isIdentStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}

func isIdentPart(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

// ApplyOverrides returns a copy of Resume with string fields overridden by the map.
func ApplyOverrides(in config.Resume, overrides map[string]string) config.Resume {
	if overrides == nil || len(overrides) == 0 {
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
