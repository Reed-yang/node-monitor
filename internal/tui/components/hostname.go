package components

import "strings"

// ComputeDisplayNames computes short display names for a list of hostnames.
// It strips the longest common prefix (at a natural boundary) and returns
// a map from full hostname to display name.
func ComputeDisplayNames(hosts []string) map[string]string {
	result := make(map[string]string, len(hosts))

	if len(hosts) <= 1 {
		for _, h := range hosts {
			result[h] = h
		}
		return result
	}

	// Find longest common prefix
	prefix := hosts[0]
	for _, h := range hosts[1:] {
		for !strings.HasPrefix(h, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				break
			}
		}
		if prefix == "" {
			break
		}
	}

	// Trim prefix to natural boundary (last '.', '-', or '_')
	trimmed := prefix
	for len(trimmed) > 0 {
		last := trimmed[len(trimmed)-1]
		if last == '.' || last == '-' || last == '_' {
			break
		}
		trimmed = trimmed[:len(trimmed)-1]
	}

	// Validate: stripping must leave non-empty, distinct suffixes
	if trimmed == "" {
		for _, h := range hosts {
			result[h] = h
		}
		return result
	}

	allNonEmpty := true
	for _, h := range hosts {
		suffix := strings.TrimPrefix(h, trimmed)
		if suffix == "" {
			allNonEmpty = false
			break
		}
	}

	if !allNonEmpty {
		for _, h := range hosts {
			result[h] = h
		}
		return result
	}

	for _, h := range hosts {
		result[h] = strings.TrimPrefix(h, trimmed)
	}
	return result
}

// TruncateHostname truncates a display name to maxLen, adding ".." prefix if needed.
func TruncateHostname(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	if maxLen <= 2 {
		return name[:maxLen]
	}
	return ".." + name[len(name)-(maxLen-2):]
}
