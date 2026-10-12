package modules

import "fmt"

// ValidatePluginRegistryConsistency checks that plugin.json and registry.json
// agree on the shared fields (id, name, version).
func ValidatePluginRegistryConsistency(
	r *Report,
	entry map[string]any,
	plugin map[string]any,
	folderName string,
) {
	where := "modules/" + folderName

	if entry["id"] != plugin["id"] {
		r.Error(where, fmt.Sprintf(
			"`id` mismatch: registry says %v, plugin.json says %v",
			entry["id"], plugin["id"],
		))
	}
	if entry["name"] != plugin["name"] {
		r.Error(where, fmt.Sprintf(
			"`name` mismatch: registry says %v, plugin.json says %v",
			entry["name"], plugin["name"],
		))
	}
	if regVersion, ok := entry["version"].(string); ok && regVersion != "" {
		if plugin["version"] != regVersion {
			r.Error(where, fmt.Sprintf(
				"`version` mismatch: registry=%q, plugin.json=%v",
				regVersion, plugin["version"],
			))
		}
	}
}
