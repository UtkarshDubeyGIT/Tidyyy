package namer

import (
	"regexp"
	"strings"
)

var (
	reWord           = regexp.MustCompile(`[a-zA-Z0-9]+`)
	reNonSlug        = regexp.MustCompile(`[^a-z0-9-]`)
	reTimestampBlock = regexp.MustCompile(`\b(19|20)\d{2}[-_]?\d{2}[-_]?\d{2}\b`)
	reCounterSuffix  = regexp.MustCompile(`-\d+$`)
)

var slugStopWords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "for": {}, "this": {}, "that": {},
	"with": {}, "from": {}, "into": {}, "in": {}, "on": {}, "to": {}, "of": {}, "at": {},
	"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "by": {}, "as": {}, "it": {},
}

func NormalizeSlug(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	lower = strings.ReplaceAll(lower, "_", "-")
	lower = strings.ReplaceAll(lower, " ", "-")
	lower = reNonSlug.ReplaceAllString(lower, "")
	lower = strings.Trim(lower, "-")
	for strings.Contains(lower, "--") {
		lower = strings.ReplaceAll(lower, "--", "-")
	}
	return lower
}

func IsValidSlug(slug string) bool {
	if slug == "" {
		return false
	}
	if strings.Contains(slug, ".") {
		return false
	}
	if reTimestampBlock.MatchString(slug) {
		return false
	}
	if reCounterSuffix.MatchString(slug) {
		return false
	}
	parts := strings.Split(slug, "-")
	words := 0
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.ContainsAny(p, "_ ") {
			return false
		}
		words++
	}
	return words >= 2 && words <= 5
}

func FallbackSlug(text string, maxWords int) string {
	words := reWord.FindAllString(strings.ToLower(text), -1)
	picked := make([]string, 0, len(words))
	for _, w := range words {
		if _, stop := slugStopWords[w]; stop {
			continue
		}
		picked = append(picked, w)
		if len(picked) >= maxWords {
			break
		}
	}
	if len(picked) < 2 {
		picked = append(picked, "document", "file")
	}
	if len(picked) > 5 {
		picked = picked[:5]
	}
	slug := NormalizeSlug(strings.Join(picked, "-"))
	if !IsValidSlug(slug) {
		return "document-file"
	}
	return slug
}
