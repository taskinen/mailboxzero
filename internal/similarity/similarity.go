package similarity

import (
	"mailboxzero/internal/jmap"
	"sort"
	"strings"
	"unicode"
)

type EmailGroup struct {
	Emails     []jmap.Email
	Similarity float64
}

// features holds the per-email values used during similarity comparison.
// Computing these once per email turns the O(N²) pairwise work from
// "re-normalize + re-tokenize + Levenshtein on bodies" into cheap lookups.
type features struct {
	email       jmap.Email
	subjectNorm string
	senderNorm  string
	bodyTokens  map[string]struct{}
}

func FindSimilarEmails(emails []jmap.Email, threshold float64) []jmap.Email {
	if len(emails) == 0 {
		return nil
	}

	feats := precomputeAll(emails)
	groups := groupSimilarFeatures(feats, threshold)

	if len(groups) == 0 {
		return nil
	}

	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Emails) > len(groups[j].Emails)
	})

	return groups[0].Emails
}

func FindSimilarToEmail(targetEmail jmap.Email, emails []jmap.Email, threshold float64) []jmap.Email {
	similarEmails := []jmap.Email{targetEmail}

	target := precomputeOne(targetEmail)
	for _, email := range emails {
		if email.ID == targetEmail.ID {
			continue
		}

		f := precomputeOne(email)
		if similarityWithThreshold(target, f, threshold) >= threshold {
			similarEmails = append(similarEmails, email)
		}
	}

	return similarEmails
}

func groupSimilarEmails(emails []jmap.Email, threshold float64) []EmailGroup {
	if len(emails) == 0 {
		return nil
	}
	return groupSimilarFeatures(precomputeAll(emails), threshold)
}

func groupSimilarFeatures(feats []features, threshold float64) []EmailGroup {
	var groups []EmailGroup
	processed := make([]bool, len(feats))

	for i := range feats {
		if processed[i] {
			continue
		}

		groupEmails := []jmap.Email{feats[i].email}
		processed[i] = true

		var simSum float64
		var simCount int

		for j := i + 1; j < len(feats); j++ {
			if processed[j] {
				continue
			}
			sim := similarityWithThreshold(feats[i], feats[j], threshold)
			if sim >= threshold {
				groupEmails = append(groupEmails, feats[j].email)
				processed[j] = true
				simSum += sim
				simCount++
			}
		}

		if len(groupEmails) > 1 {
			avg := 0.0
			if simCount > 0 {
				avg = simSum / float64(simCount)
			}
			groups = append(groups, EmailGroup{
				Emails:     groupEmails,
				Similarity: avg,
			})
		}
	}

	return groups
}

func calculateEmailSimilarity(email1, email2 jmap.Email) float64 {
	f1 := precomputeOne(email1)
	f2 := precomputeOne(email2)
	// threshold=0 disables the short-circuit so the full score is always computed.
	return similarityWithThreshold(f1, f2, 0)
}

func calculateGroupSimilarity(emails []jmap.Email) float64 {
	if len(emails) <= 1 {
		return 0.0
	}

	feats := precomputeAll(emails)
	var totalSimilarity float64
	var count int

	for i := 0; i < len(feats); i++ {
		for j := i + 1; j < len(feats); j++ {
			totalSimilarity += similarityWithThreshold(feats[i], feats[j], 0)
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalSimilarity / float64(count)
}

func precomputeAll(emails []jmap.Email) []features {
	out := make([]features, len(emails))
	for i, e := range emails {
		out[i] = precomputeOne(e)
	}
	return out
}

func precomputeOne(email jmap.Email) features {
	var sender string
	if len(email.From) > 0 {
		sender = normalizeString(email.From[0].Email)
	}
	return features{
		email:       email,
		subjectNorm: normalizeString(email.Subject),
		senderNorm:  sender,
		bodyTokens:  tokenizeBody(extractEmailBody(email)),
	}
}

// similarityWithThreshold computes the weighted similarity between two
// precomputed feature sets. It short-circuits as soon as the partial score
// plus the maximum remaining contribution falls below threshold.
func similarityWithThreshold(a, b features, threshold float64) float64 {
	subjectSim := normalizedStringSimilarity(a.subjectNorm, b.subjectNorm)
	score := subjectSim * 0.4
	if score+0.6 < threshold {
		return 0
	}

	var senderSim float64
	if a.senderNorm != "" && b.senderNorm != "" {
		senderSim = normalizedStringSimilarity(a.senderNorm, b.senderNorm)
	}
	score += senderSim * 0.4
	if score+0.2 < threshold {
		return 0
	}

	var bodySim float64
	if len(a.bodyTokens) > 0 && len(b.bodyTokens) > 0 {
		bodySim = jaccardSimilarity(a.bodyTokens, b.bodyTokens)
	}
	score += bodySim * 0.2

	return score
}

func stringSimilarity(s1, s2 string) float64 {
	return normalizedStringSimilarity(normalizeString(s1), normalizeString(s2))
}

// normalizedStringSimilarity is stringSimilarity for inputs that have already
// been normalized. Callers in the hot path use this to avoid re-running
// normalizeString on the same value for every pair.
func normalizedStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if s1 == "" || s2 == "" {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0
	}

	similarity := 1.0 - (float64(distance) / float64(maxLen))

	if containsCommonWords(s1, s2) {
		similarity += 0.1
	}

	if similarity > 1.0 {
		similarity = 1.0
	}

	return similarity
}

func normalizeString(s string) string {
	s = strings.ToLower(s)

	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		} else {
			result.WriteRune(' ')
		}
	}

	return strings.TrimSpace(result.String())
}

func containsCommonWords(s1, s2 string) bool {
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	commonWords := 0
	for _, word1 := range words1 {
		if len(word1) < 3 {
			continue
		}
		for _, word2 := range words2 {
			if word1 == word2 {
				commonWords++
				break
			}
		}
	}

	return commonWords >= 2
}

func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	column := make([]int, len(r1)+1)

	for y := 1; y <= len(r1); y++ {
		column[y] = y
	}

	for x := 1; x <= len(r2); x++ {
		column[0] = x
		lastkey := x - 1
		for y := 1; y <= len(r1); y++ {
			oldkey := column[y]
			var incr int
			if r1[y-1] != r2[x-1] {
				incr = 1
			}

			column[y] = min(column[y]+1, column[y-1]+1, lastkey+incr)
			lastkey = oldkey
		}
	}

	return column[len(r1)]
}

func extractEmailBody(email jmap.Email) string {
	if email.Preview != "" {
		return email.Preview
	}

	for _, bodyValue := range email.BodyValues {
		if bodyValue.Value != "" {
			return normalizeString(bodyValue.Value)
		}
	}

	return ""
}

// tokenizeBody returns the set of normalized word-tokens of length ≥ 3 from
// the email body. Jaccard on these is O(L) per pair, replacing the old
// O(L₁·L₂) Levenshtein on body strings that could be tens of thousands of
// characters.
func tokenizeBody(body string) map[string]struct{} {
	if body == "" {
		return nil
	}
	normalized := normalizeString(body)
	if normalized == "" {
		return nil
	}
	tokens := make(map[string]struct{})
	for _, w := range strings.Fields(normalized) {
		if len(w) >= 3 {
			tokens[w] = struct{}{}
		}
	}
	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}

	intersect := 0
	for token := range small {
		if _, ok := large[token]; ok {
			intersect++
		}
	}

	union := len(a) + len(b) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
