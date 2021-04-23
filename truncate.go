package main

// Much lower than Discord limit
const (
	limit = 300
	slack = 100
)

// Returns the string unchanged if the length is within limit+slack, otherwise returns it
// truncated to limit chars.
func truncateMessage(s string) string {
	if len(s) <= limit+slack {
		return s
	}
	// Cutting off everything beyond limit doesn't care about markdown formatting and can leave
	// unclosed markup.
	// TODO: Maybe use a markdown parser to properly handle these.
	return s[:limit] + "…"
}
