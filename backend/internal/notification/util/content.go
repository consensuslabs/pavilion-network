package util

// FormatVideoContent formats content for video notifications
func FormatVideoContent(title string, action string) string {
	return action + ": " + title
}

// FormatCommentContent formats content for comment notifications
func FormatCommentContent(content string, action string, maxLength int) string {
	truncated := TruncateContent(content, maxLength)
	return action + ": " + truncated
}

// FormatUserContent formats content for user notifications
func FormatUserContent(username string, action string) string {
	return username + " " + action
} 