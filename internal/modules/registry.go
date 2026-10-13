package modules

import "fmt"

// ValidateRegistryEntry checks one entry in registry.json::modules[i].
func ValidateRegistryEntry(r *Report, entry map[string]any, index int) {
	where := fmt.Sprintf("registry.json::modules[%d]", index)

	for _, req := range []string{"id", "path", "name"} {
		raw, present := entry[req]
		s, isStr := raw.(string)
		if !present || !isStr || strings_TrimSpaceEmpty(s) {
			r.Error(where, fmt.Sprintf("`%s` is required and must be a non-empty string", req))
		}
	}

	if path, ok := entry["path"].(string); ok && path != "" {
		if !KebabCaseRE.MatchString(path) {
			r.Error(where, fmt.Sprintf("`path` must be kebab-case, got %q", path))
		}
	}

	version, _ := entry["version"].(string)
	if !SemverRE.MatchString(version) {
		r.Error(where, fmt.Sprintf("`version` must be a semver string, got %q", entry["version"]))
	}

	category := "OTHER"
	if raw, ok := entry["category"].(string); ok {
		category = raw
	}
	if _, ok := AllowedCategories[category]; !ok {
		r.Error(where, fmt.Sprintf("`category` must be one of the allowed values, got %q", category))
	}

	runAt := "DOCUMENT_END"
	if raw, ok := entry["runAt"].(string); ok {
		runAt = raw
	}
	if _, ok := AllowedRunAt[runAt]; !ok {
		r.Error(where, fmt.Sprintf("`runAt` must be one of the allowed values, got %q", runAt))
	}

	for _, perm := range ExpectListOfStr(r, where, entry["permissions"], "permissions") {
		if _, ok := AllowedPermissions[perm]; !ok {
			r.Error(where, fmt.Sprintf("unknown permission %q", perm))
		}
	}

	ValidateURLMatches(r, where, entry["urlMatches"])
	ValidateAuthor(r, where, entry["author"])

	if v, present := entry["minAppVersion"]; present {
		if _, isInt := v.(float64); !isInt {
			r.Error(where, "`minAppVersion` must be an integer")
		}
	}

	if v, present := entry["hasCss"]; present {
		if _, isBool := v.(bool); !isBool {
			r.Error(where, "`hasCss` must be a boolean")
		}
	}

	if v, present := entry["iconUrl"]; present && v != nil {
		if _, isStr := v.(string); !isStr {
			r.Error(where, "`iconUrl` must be a string when present")
		}
	}

	sourceType := "CUSTOM"
	if raw, ok := entry["sourceType"].(string); ok {
		sourceType = raw
	}
	if _, ok := AllowedSourceTypes[sourceType]; !ok {
		r.Error(where, fmt.Sprintf("`sourceType` must be one of {CHROME_EXTENSION, CUSTOM}, got %q", sourceType))
	}

	if sourceType == "CHROME_EXTENSION" {
		storeID, _ := entry["storeId"].(string)
		if !StoreIDRE.MatchString(storeID) {
			r.Error(where, "`storeId` is required for CHROME_EXTENSION and must be a 32-char Chrome Web Store ID")
		}
	}
}

// ValidateRegistry checks the top-level registry.json structure and
// per-entry requirements. Returns the list of valid (non-nil) entries.
func ValidateRegistry(r *Report, registry map[string]any) []map[string]any {
	where := "registry.json"

	if schema, ok := registry["schema"].(float64); !ok || schema != 1 {
		r.Error(where, fmt.Sprintf("`schema` must be 1 (got %v); future versions need a parser update", registry["schema"]))
	}
	if v, present := registry["updatedAt"]; present {
		if _, isStr := v.(string); !isStr {
			r.Error(where, "`updatedAt` must be a string")
		}
	}

	arr, ok := registry["modules"].([]any)
	if !ok {
		r.Error(where, "`modules` must be a list")
		return nil
	}

	seenIDs := map[string]struct{}{}
	seenPaths := map[string]struct{}{}
	out := make([]map[string]any, 0, len(arr))

	for i, item := range arr {
		entry, ok := item.(map[string]any)
		if !ok {
			r.Error(fmt.Sprintf("%s::modules[%d]", where, i), "must be an object")
			continue
		}
		ValidateRegistryEntry(r, entry, i)

		if id, ok := entry["id"].(string); ok && id != "" {
			if _, dup := seenIDs[id]; dup {
				r.Error(fmt.Sprintf("%s::modules[%d]", where, i), fmt.Sprintf("duplicate id %q", id))
			}
			if !KebabCaseRE.MatchString(id) {
				r.Error(fmt.Sprintf("%s::modules[%d]", where, i), fmt.Sprintf("id %q must be kebab-case", id))
			}
			seenIDs[id] = struct{}{}
		}

		if path, ok := entry["path"].(string); ok && path != "" {
			if _, dup := seenPaths[path]; dup {
				r.Error(fmt.Sprintf("%s::modules[%d]", where, i), fmt.Sprintf("duplicate path %q", path))
			}
			seenPaths[path] = struct{}{}
		}

		out = append(out, entry)
	}
	return out
}
