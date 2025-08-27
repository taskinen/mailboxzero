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

func FindSimilarEmails(emails []jmap.Email, threshold float64) []jmap.Email {
	if len(emails) == 0 {
		return nil
	}

	groups := groupSimilarEmails(emails, threshold)

	if len(groups) == 0 {
		return nil
	}

	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Emails) > len(groups[j].Emails)
	})

	return groups[0].Emails
}

func FindSimilarToEmail(targetEmail jmap.Email, emails []jmap.Email, threshold float64) []jmap.Email {
	var similarEmails []jmap.Email

	// Always include the target email itself as the first result
	similarEmails = append(similarEmails, targetEmail)

	for _, email := range emails {
		if email.ID == targetEmail.ID {
			continue
		}

		similarity := calculateEmailSimilarity(targetEmail, email)
		if similarity >= threshold {
			similarEmails = append(similarEmails, email)
		}
	}

	return similarEmails
}

func groupSimilarEmails(emails []jmap.Email, threshold float64) []EmailGroup {
	var groups []EmailGroup
	processed := make(map[string]bool)

	for i, email1 := range emails {
		if processed[email1.ID] {
			continue
		}

		var group []jmap.Email
		group = append(group, email1)
		processed[email1.ID] = true

		for j := i + 1; j < len(emails); j++ {
			email2 := emails[j]
			if processed[email2.ID] {
				continue
			}

			similarity := calculateEmailSimilarity(email1, email2)
			if similarity >= threshold {
				group = append(group, email2)
				processed[email2.ID] = true
			}
		}

		if len(group) > 1 {
			avgSimilarity := calculateGroupSimilarity(group)
			groups = append(groups, EmailGroup{
				Emails:     group,
				Similarity: avgSimilarity,
			})
		}
	}

	return groups
}

func calculateEmailSimilarity(email1, email2 jmap.Email) float64 {
	subjectSim := stringSimilarity(email1.Subject, email2.Subject)

	var senderSim float64
	if len(email1.From) > 0 && len(email2.From) > 0 {
		senderSim = stringSimilarity(email1.From[0].Email, email2.From[0].Email)
	}

	var bodySim float64
	body1 := extractEmailBody(email1)
	body2 := extractEmailBody(email2)
	if body1 != "" && body2 != "" {
		bodySim = stringSimilarity(body1, body2)
	}

	weightedSimilarity := (subjectSim*0.4 + senderSim*0.4 + bodySim*0.2)

	return weightedSimilarity
}

func calculateGroupSimilarity(emails []jmap.Email) float64 {
	if len(emails) <= 1 {
		return 0.0
	}

	var totalSimilarity float64
	var count int

	for i := 0; i < len(emails); i++ {
		for j := i + 1; j < len(emails); j++ {
			similarity := calculateEmailSimilarity(emails[i], emails[j])
			totalSimilarity += similarity
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalSimilarity / float64(count)
}

func stringSimilarity(s1, s2 string) float64 {
	s1 = normalizeString(s1)
	s2 = normalizeString(s2)

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
