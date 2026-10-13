package modules

import "fmt"

// ValidatePluginJSON validates a per-module `plugin.json` (the HCJ package
// manifest).
func ValidatePluginJSON(r *Report, folderName string, plugin map[string]any) {
	where := "modules/" + folderName + "/plugin.json"

	for _, req := range []string{"id", "name"} {
		raw, present := plugin[req]
		s, isStr := raw.(string)
		if !present || !isStr || strings_TrimSpaceEmpty(s) {
			r.Error(where, fmt.Sprintf("`%s` is required and must be a non-empty string", req))
		}
	}

	version, _ := plugin["version"].(string)
	if !SemverRE.MatchString(version) {
		r.Error(where, fmt.Sprintf("`version` must be a semver string, got %v", plugin["version"]))
	}

	if matches, present := plugin["matches"]; present && matches != nil {
		if _, ok := matches.([]any); !ok {
			r.Error(where, "`matches` must be a list of match-pattern strings")
		} else {
			for _, m := range matches.([]any) {
				if _, ok := m.(string); !ok {
					r.Error(where, "`matches` must be a list of match-pattern strings")
					break
				}
			}
		}
	}

	if runAt, present := plugin["runAt"]; present && runAt != nil {
		s, _ := runAt.(string)
		if _, ok := PluginJSONRunAt[s]; !ok {
			r.Error(where, fmt.Sprintf("`runAt` must be document_start/document_end/document_idle, got %v", runAt))
		}
	}

	if perms, present := plugin["permissions"]; present && perms != nil {
		arr, ok := perms.([]any)
		if !ok {
			r.Error(where, "`permissions` must be a list of strings")
		} else {
			for _, raw := range arr {
				s, ok := raw.(string)
				if !ok {
					r.Error(where, "`permissions` must be a list of strings")
					break
				}
				if _, ok := PluginJSONPermissions[s]; !ok {
					r.Warning(where, fmt.Sprintf("unknown permission %q", s))
				}
			}
		}
	}
}
