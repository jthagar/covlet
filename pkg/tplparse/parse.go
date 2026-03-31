package tplparse

import "strings"

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
