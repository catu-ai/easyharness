package plan

import "strings"

const (
	PlaceholderPendingUntilArchive       = "PENDING_UNTIL_ARCHIVE"
	PlaceholderUpdateRequiredAfterReopen = "UPDATE_REQUIRED_AFTER_REOPEN"
)

func containsArchivePlaceholderToken(content string) bool {
	return strings.Contains(content, PlaceholderPendingUntilArchive) ||
		strings.Contains(content, PlaceholderUpdateRequiredAfterReopen)
}
