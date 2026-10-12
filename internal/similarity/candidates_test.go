package similarity

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/taskinen/mailboxzero/internal/jmap"
)

// testThresholds covers every regime of the candidate index (all pairs up
// to 30%, body keys up to 50%, sender keys only above) and the values right
// at its limits.
var testThresholds = []float64{0, 0.1, 0.2, 0.25, 0.3, 0.30000001, 0.31, 0.35, 0.4, 0.45, 0.5, 0.50000001, 0.55, 0.6, 0.7, 0.8, 0.85, 0.9, 1}

// randomInbox returns n emails built to hit the cases the candidate index
// and the early exits must get right: shared and different domains,
// subdomains, shared bulk-mail providers, generic and short display names,
// missing senders, empty and identical subjects and bodies, and a small
// vocabulary so that many pairs land near the threshold.
func randomInbox(r *rand.Rand, n int) []jmap.Email {
	senders := []jmap.EmailAddress{
		{Name: "GitHub", Email: "notifications@github.com"},
		{Name: "GitHub", Email: "noreply@github.com"},
		{Name: "Google", Email: "no-reply@accounts.google.com"},
		{Name: "Google Alerts", Email: "alerts@mail.google.com"},
		{Name: "Shop", Email: "news@shop.example.co.uk"},
		{Name: "Shop", Email: "offers@deals.example.co.uk"},
		{Name: "notifications", Email: "bounce@sendgrid.net"},
		{Name: "Acme Weekly", Email: "acme@sendgrid.net"},
		{Name: "Alpha", Email: "alpha@bounce.sendgrid.net"}, // own subdomain of a provider
		{Name: "Beta", Email: "beta@bounce.sendgrid.net"},
		{Name: "Bank", Email: "statements@bank.example"},
		{Name: "team", Email: "team@startup.io"},
		{Name: "Bob", Email: "bob@friends.example"},
		{Name: "", Email: "someone@mailgun.org"},
		{Name: "Newsletter Digest", Email: "digest@news.example.com"},
		{Name: "Newsletter Digest", Email: "digest@other.example.org"},
		{}, // no sender
	}
	words := []string{"order", "shipped", "invoice", "weekly", "digest", "your", "account", "security", "alert", "report", "meeting", "summary", "the", "new", "sale", "receipt", "payment", "reminder", "update", "news"}
	phrase := func(minWords, maxWords int) string {
		parts := make([]string, minWords+r.Intn(maxWords-minWords+1))
		for i := range parts {
			parts[i] = words[r.Intn(len(words))]
		}
		return strings.Join(parts, " ")
	}
	prefixes := []string{"", "", "", "Re: ", "Fwd: ", "[list] "}

	emails := make([]jmap.Email, n)
	for i := range emails {
		email := jmap.Email{ID: fmt.Sprintf("e%d", i)}
		if sender := senders[r.Intn(len(senders))]; sender.Email != "" {
			email.From = []jmap.EmailAddress{sender}
		}
		switch r.Intn(10) {
		case 0: // no subject
		case 1:
			email.Subject = "Your weekly digest"
		default:
			email.Subject = prefixes[r.Intn(len(prefixes))] + phrase(1, 6) + fmt.Sprintf(" #%d", r.Intn(5))
		}
		if r.Intn(8) != 0 {
			email.Preview = phrase(0, 25)
		}
		emails[i] = email
	}
	return emails
}

// originalSimilarity is similarityWithThreshold as it was before the
// candidate index and the early exits, kept to check they change no result.
func originalSimilarity(a, b features, threshold float64) float64 {
	score := senderSimilarity(a, b) * 0.5
	if score+0.5 < threshold {
		return 0
	}

	score += originalSubjectSimilarity(a, b) * 0.3
	if score+0.2 < threshold {
		return 0
	}

	var bodySim float64
	if len(a.bodyTokens) > 0 && len(b.bodyTokens) > 0 {
		bodySim = jaccardSimilarity(a.bodyTokens, b.bodyTokens)
	}
	return score + bodySim*0.2
}

// originalSubjectSimilarity is subjectSimilarity before the early exits.
func originalSubjectSimilarity(a, b features) float64 {
	if a.subjectNorm == "" && b.subjectNorm == "" {
		return 1.0
	}
	if a.subjectNorm == "" || b.subjectNorm == "" {
		return 0.0
	}
	score := normalizedStringSimilarity(a.subjectNorm, b.subjectNorm)
	if len(a.subjectTokens) >= 2 && len(b.subjectTokens) >= 2 {
		if jac := jaccardSimilarity(a.subjectTokens, b.subjectTokens); jac > score {
			score = jac
		}
	}
	return score
}

// bruteForceGroups is the grouping before candidateIndex: repeated passes
// that compare every email not yet in a cluster with every member, scored
// by originalSimilarity. Each group is returned as its sorted email IDs.
func bruteForceGroups(feats []features, threshold float64) [][]string {
	var groups [][]string
	processed := make([]bool, len(feats))
	for i := range feats {
		if processed[i] {
			continue
		}
		processed[i] = true
		cluster := []int{i}
		for added := true; added; {
			added = false
			for j := range feats {
				if processed[j] {
					continue
				}
				for _, member := range cluster {
					if originalSimilarity(feats[member], feats[j], threshold) >= threshold {
						cluster = append(cluster, j)
						processed[j] = true
						added = true
						break
					}
				}
			}
		}
		if len(cluster) > 1 {
			ids := make([]string, len(cluster))
			for k, index := range cluster {
				ids[k] = feats[index].email.ID
			}
			slices.Sort(ids)
			groups = append(groups, ids)
		}
	}
	return groups
}

func groupIDs(groups []EmailGroup) [][]string {
	out := make([][]string, len(groups))
	for i, group := range groups {
		for _, email := range group.Emails {
			out[i] = append(out[i], email.ID)
		}
		slices.Sort(out[i])
	}
	return out
}

// Skipping pairs through the candidate index and stopping early must not
// change any group: both find the same groups, in the same order.
func TestGroupSimilarFeatures_SameGroupsAsComparingAllPairs(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		feats := precomputeAll(randomInbox(rand.New(rand.NewSource(seed)), 100))
		for _, threshold := range testThresholds {
			want := bruteForceGroups(feats, threshold)
			got := groupIDs(groupSimilarFeatures(feats, threshold))
			if !slices.EqualFunc(got, want, slices.Equal[[]string]) {
				t.Errorf("seed %d, threshold %v: groups = %v, want %v", seed, threshold, got, want)
			}
		}
	}
}

// The early exits may only skip work: a pair reaches the threshold exactly
// when it did before, with exactly the same score.
func TestSimilarityWithThreshold_SameResultsAsOriginal(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	feats := precomputeAll(randomInbox(r, 200))
	for i := 0; i < 5000; i++ {
		a, b := feats[r.Intn(len(feats))], feats[r.Intn(len(feats))]

		if got, want := similarityWithThreshold(a, b, 0), originalSimilarity(a, b, 0); got != want {
			t.Fatalf("%q vs %q: full score = %v, want %v", a.email.Subject, b.email.Subject, got, want)
		}
		if got, want := subjectSimilarity(a, b), originalSubjectSimilarity(a, b); got != want {
			t.Fatalf("subjectSimilarity(%q, %q) = %v, want %v", a.email.Subject, b.email.Subject, got, want)
		}

		for _, threshold := range testThresholds {
			got := similarityWithThreshold(a, b, threshold)
			want := originalSimilarity(a, b, threshold)
			if (got >= threshold) != (want >= threshold) || (want >= threshold && got != want) {
				t.Fatalf("%q vs %q at %v: score = %v, want %v", a.email.Subject, b.email.Subject, threshold, got, want)
			}
		}
	}
}

func TestLevenshteinRunes_Limit(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	randomRunes := func() []rune {
		out := make([]rune, r.Intn(12))
		for i := range out {
			out[i] = rune('a' + r.Intn(4))
		}
		return out
	}
	for i := 0; i < 2000; i++ {
		a, b := randomRunes(), randomRunes()
		distance := levenshteinRunes(a, b, len(a)+len(b))
		limit := r.Intn(12)
		got := levenshteinRunes(a, b, limit)
		if distance <= limit && got != distance {
			t.Fatalf("levenshteinRunes(%q, %q, %d) = %d, want the distance %d", string(a), string(b), limit, got, distance)
		}
		if distance > limit && got <= limit {
			t.Fatalf("levenshteinRunes(%q, %q, %d) = %d, want more than the limit (distance %d)", string(a), string(b), limit, got, distance)
		}
	}
}

func TestCandidateIndex(t *testing.T) {
	// Unrelated senders, identical subject and body: 0.3 + 0.2 = 0.5.
	feats := precomputeAll([]jmap.Email{
		{ID: "a", Subject: "Quarterly report", Preview: "numbers for the quarter are in", From: []jmap.EmailAddress{{Email: "a@one.example"}}},
		{ID: "b", Subject: "Quarterly report", Preview: "numbers for the quarter are in", From: []jmap.EmailAddress{{Email: "b@two.example"}}},
		{ID: "c", Subject: "Something else", Preview: "nothing in common", From: []jmap.EmailAddress{{Email: "c@one.example"}}},
	})
	candidatesOf := func(threshold float64, i int) []int {
		var got []int
		newCandidateIndex(feats, threshold).visit(i, func(j int) { got = append(got, j) })
		slices.Sort(got)
		return got
	}

	tests := []struct {
		threshold float64
		want      []int
	}{
		{threshold: 0.3, want: []int{1, 2}}, // every email
		{threshold: 0.5, want: []int{1, 2}}, // same body tokens, same domain
		{threshold: 0.6, want: []int{2}},    // same domain only
	}
	for _, tt := range tests {
		if got := candidatesOf(tt.threshold, 0); !slices.Equal(got, tt.want) {
			t.Errorf("candidates of a at %v = %v, want %v", tt.threshold, got, tt.want)
		}
	}
}

// newsletterInbox is a larger inbox where emails cluster by sender: each of
// 400 senders uses its own words plus some common ones.
func newsletterInbox(n int) []jmap.Email {
	r := rand.New(rand.NewSource(1))
	common := []string{"the", "your", "and", "for", "this", "you", "with", "account", "new", "more"}
	phrase := func(sender, words int) string {
		var b strings.Builder
		for i := 0; i < words; i++ {
			if r.Intn(3) == 0 {
				b.WriteString(common[r.Intn(len(common))])
			} else {
				fmt.Fprintf(&b, "w%dx%d", sender, r.Intn(40))
			}
			b.WriteByte(' ')
		}
		return b.String()
	}
	emails := make([]jmap.Email, n)
	for i := range emails {
		sender := r.Intn(400)
		emails[i] = jmap.Email{
			ID:      fmt.Sprint(i),
			Subject: phrase(sender, 5),
			Preview: phrase(sender, 30),
			From:    []jmap.EmailAddress{{Name: fmt.Sprintf("Shop%d", sender), Email: fmt.Sprintf("news@s%d.d%d.example", sender, sender%150)}},
		}
	}
	return emails
}

func BenchmarkFindSimilarEmails_Thresholds(b *testing.B) {
	emails := newsletterInbox(2000)
	for _, threshold := range []float64{0.6, 0.4, 0.3} {
		b.Run(fmt.Sprintf("%.0f%%", threshold*100), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FindSimilarEmails(emails, threshold)
			}
		})
	}
}
