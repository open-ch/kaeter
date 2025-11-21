package actions

import (
	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// applyTags applies tags to a version metadata based on the provided tags pointer.
// - nil pointer: don't change existing tags
// - empty slice or single empty string: clear tags
// - non-empty slice: set tags (filtering out empty strings)
func applyTags(versionMeta *modules.VersionMetadata, tags *[]string) {
	if tags == nil {
		// Tags not provided, don't change existing tags
		return
	}

	tagsValue := *tags
	if len(tagsValue) == 0 || (len(tagsValue) == 1 && tagsValue[0] == "") {
		// Empty or single empty string: clear tags
		versionMeta.Tags = nil
		log.Debug("Cleared tags from version")
		return
	}

	// Filter out any empty strings and set tags
	filteredTags := make([]string, 0, len(tagsValue))
	for _, tag := range tagsValue {
		if tag != "" {
			filteredTags = append(filteredTags, tag)
		}
	}

	if len(filteredTags) > 0 {
		versionMeta.Tags = filteredTags
		log.Debug("Applied tags to version", "tags", filteredTags)
	} else {
		versionMeta.Tags = nil
		log.Debug("Cleared tags from version")
	}
}
