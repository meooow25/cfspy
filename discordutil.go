package main

import "time"

// Duration for which to persist an error message.
const timedErrorMsgTTL = 30 * time.Second

// Checks whether the given substring is surrounded by <>. Used to check if a link embed is
// suppressed.
func checkEmbedsSuppressed(s string, start, end int) bool {
	return start > 0 && s[start-1] == '<' && end < len(s)-1 && s[end+1] == '>'
}
