package similarity

import (
	"mailboxzero/internal/jmap"
	"regexp"
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
	email         jmap.Email
	subjectNorm   string
	subjectTokens map[string]struct{}
	senderAddr    string
	senderDomain  string
	senderRoot    string
	senderName    string
	bodyTokens    map[string]struct{}
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

// groupSimilarFeatures forms clusters using single-link expansion: a candidate
// joins a cluster if it is similar enough to ANY existing member, not just
// the seed. This catches sibling emails that diverge from the seed but match
// other members (e.g. four newsletters from one sender where one has a much
// shorter subject than the rest).
func groupSimilarFeatures(feats []features, threshold float64) []EmailGroup {
	var groups []EmailGroup
	processed := make([]bool, len(feats))

	for i := range feats {
		if processed[i] {
			continue
		}

		clusterIdx := []int{i}
		clusterEmails := []jmap.Email{feats[i].email}
		processed[i] = true

		var simSum float64
		var simCount int

		// Repeat expansion passes until no new members join.
		for {
			added := false
			for j := 0; j < len(feats); j++ {
				if processed[j] {
					continue
				}
				var bestSim float64
				for _, ci := range clusterIdx {
					sim := similarityWithThreshold(feats[ci], feats[j], threshold)
					if sim > bestSim {
						bestSim = sim
					}
					if bestSim >= threshold {
						break
					}
				}
				if bestSim >= threshold {
					clusterIdx = append(clusterIdx, j)
					clusterEmails = append(clusterEmails, feats[j].email)
					processed[j] = true
					simSum += bestSim
					simCount++
					added = true
				}
			}
			if !added {
				break
			}
		}

		if len(clusterEmails) > 1 {
			avg := 0.0
			if simCount > 0 {
				avg = simSum / float64(simCount)
			}
			groups = append(groups, EmailGroup{
				Emails:     clusterEmails,
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
	var addr, name string
	if len(email.From) > 0 {
		addr = email.From[0].Email
		name = email.From[0].Name
	}
	senderAddr, senderDomain, senderRoot, senderName := parseSender(addr, name)

	subjectNorm := normalizeSubject(email.Subject)
	return features{
		email:         email,
		subjectNorm:   subjectNorm,
		subjectTokens: tokenizeSubject(subjectNorm),
		senderAddr:    senderAddr,
		senderDomain:  senderDomain,
		senderRoot:    senderRoot,
		senderName:    senderName,
		bodyTokens:    tokenizeBody(extractEmailBody(email)),
	}
}

// similarityWithThreshold composes weighted similarity from sender (0.5),
// subject (0.3), and body (0.2). Sender is computed first because it is the
// most discriminating signal for the newsletter/notification clustering that
// drives the inbox-cleanup workflow, and because the structured comparison
// is cheaper than the subject Levenshtein. Short-circuits when the partial
// score plus the maximum remaining contribution falls below threshold.
func similarityWithThreshold(a, b features, threshold float64) float64 {
	senderSim := senderSimilarity(a, b)
	score := senderSim * 0.5
	if score+0.5 < threshold {
		return 0
	}

	subjectSim := subjectSimilarity(a, b)
	score += subjectSim * 0.3
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

// senderSimilarity uses a structured ladder rather than Levenshtein on the
// normalized address. Levenshtein-on-strings inflated similarity between
// unrelated senders that happened to share generic local parts like
// "noreply" plus the "com" TLD, while penalizing same-organization addresses
// with different local parts. The ladder reflects how senders actually relate.
func senderSimilarity(a, b features) float64 {
	if a.senderAddr != "" && a.senderAddr == b.senderAddr {
		return 1.0
	}

	best := 0.0

	// Same full domain: 0.8. ESP guard: if the domain is a shared
	// bulk-mail provider (sendgrid.net etc.), require the local part to
	// match too — otherwise two unrelated senders relayed through the
	// same provider would collapse together.
	if a.senderDomain != "" && a.senderDomain == b.senderDomain {
		if !sharedESPDomains[a.senderDomain] {
			best = 0.8
		}
	}

	// Same registrable root (handles subdomains: mail.google.com vs
	// accounts.google.com both reduce to google.com).
	if best < 0.7 && a.senderRoot != "" && a.senderRoot == b.senderRoot {
		if !sharedESPDomains[a.senderRoot] {
			best = 0.7
		}
	}

	// Same display name, when it is specific enough to be meaningful.
	// Generic labels ("notifications", "support", "team") are excluded
	// to avoid cross-org false positives.
	if best < 0.6 && a.senderName != "" && a.senderName == b.senderName {
		if len(a.senderName) >= 4 && !genericDisplayNames[a.senderName] {
			best = 0.6
		}
	}

	return best
}

// subjectSimilarity returns max(tokenJaccard, levenshtein). Levenshtein
// alone is fine for near-identical subjects ("X #1" vs "X #2") but bad at
// recognizing that two subjects share important content words when they
// also differ in filler. Token Jaccard handles that case. Both views are
// computed on the prefix-stripped normalized form.
func subjectSimilarity(a, b features) float64 {
	aEmpty := a.subjectNorm == ""
	bEmpty := b.subjectNorm == ""
	if aEmpty && bEmpty {
		return 1.0
	}
	if aEmpty || bEmpty {
		return 0.0
	}

	score := normalizedStringSimilarity(a.subjectNorm, b.subjectNorm)

	// Only credit token Jaccard when both sides have enough surviving
	// tokens to be meaningful. Otherwise a single shared word like
	// "welcome" would trivially clear the threshold.
	if len(a.subjectTokens) >= 2 && len(b.subjectTokens) >= 2 {
		if jac := jaccardSimilarity(a.subjectTokens, b.subjectTokens); jac > score {
			score = jac
		}
	}

	return score
}

func stringSimilarity(s1, s2 string) float64 {
	return normalizedStringSimilarity(normalizeString(s1), normalizeString(s2))
}

// normalizedStringSimilarity returns Levenshtein-based similarity in [0, 1]
// for inputs that have already been normalized.
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

	return 1.0 - (float64(distance) / float64(maxLen))
}

// normalizeString lowercases, drops punctuation (replacing with space), and
// collapses runs of whitespace into a single space. Collapsing matters for
// Levenshtein-based comparisons: without it, "Hello, World!" normalizes to
// "hello  world" (double space) and scores below 1.0 against "hello world".
func normalizeString(s string) string {
	s = strings.ToLower(s)

	var result strings.Builder
	prevSpace := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			result.WriteByte(' ')
			prevSpace = true
		}
	}

	out := result.String()
	if strings.HasSuffix(out, " ") {
		out = out[:len(out)-1]
	}
	return out
}

var (
	subjectPrefixRe = regexp.MustCompile(`(?i)^\s*(re|fwd?|aw|sv|fw)\s*:\s*`)
	subjectTagRe    = regexp.MustCompile(`^\s*\[[^\]]+\]\s*`)
)

// normalizeSubject strips repeated reply/forward prefixes ("Re:", "Fwd:",
// "AW:") and leading mailing-list tags ("[golang-nuts]") before running the
// standard normalization. Without this, "Re: Re: Newsletter" and
// "Newsletter" would score poorly on Levenshtein despite being the same
// conversation.
func normalizeSubject(s string) string {
	for {
		before := s
		s = subjectPrefixRe.ReplaceAllString(s, "")
		s = subjectTagRe.ReplaceAllString(s, "")
		if s == before {
			break
		}
	}
	return normalizeString(s)
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

// tokenizeSubject is like tokenizeBody but additionally drops a small set of
// stop words that show up in nearly every newsletter/notification subject and
// would otherwise inflate Jaccard between unrelated emails.
func tokenizeSubject(normalizedSubject string) map[string]struct{} {
	if normalizedSubject == "" {
		return nil
	}
	tokens := make(map[string]struct{})
	for _, w := range strings.Fields(normalizedSubject) {
		if len(w) < 3 {
			continue
		}
		if subjectStopWords[w] {
			continue
		}
		tokens[w] = struct{}{}
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

// parseSender returns the lowercase full address, the domain, the registrable
// root domain, and the lowercase display name. The registrable root is the
// last two labels of the domain, except when the last two labels form a
// known multi-label public suffix (".co.uk", ".github.io", etc.), in which
// case the last three labels are used.
func parseSender(addr, name string) (senderAddr, senderDomain, senderRoot, senderName string) {
	addr = strings.ToLower(strings.TrimSpace(addr))
	name = strings.ToLower(strings.TrimSpace(name))

	if at := strings.LastIndex(addr, "@"); at >= 0 && at < len(addr)-1 {
		senderAddr = addr
		senderDomain = addr[at+1:]
		senderRoot = registrableRoot(senderDomain)
	}
	senderName = name
	return
}

func registrableRoot(domain string) string {
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return domain
	}

	if len(labels) >= 3 {
		twoLabel := labels[len(labels)-2] + "." + labels[len(labels)-1]
		if multiLabelSuffixes[twoLabel] {
			return labels[len(labels)-3] + "." + twoLabel
		}
	}

	return labels[len(labels)-2] + "." + labels[len(labels)-1]
}

// multiLabelSuffixes is a small subset of the public suffix list covering
// the most common multi-label public suffixes we expect to see in real
// inboxes. The full PSL is unnecessarily heavy for this use case.
var multiLabelSuffixes = map[string]bool{
	"co.uk":       true,
	"co.jp":       true,
	"co.nz":       true,
	"co.kr":       true,
	"com.au":      true,
	"com.br":      true,
	"ac.uk":       true,
	"ac.jp":       true,
	"github.io":   true,
	"vercel.app":  true,
	"netlify.app": true,
}

// sharedESPDomains lists domains operated by bulk email service providers.
// When a sender's domain matches one of these, two emails sharing only the
// domain (different local parts) almost certainly come from different
// organizations relayed through the same provider, so the same-domain rung
// of the sender ladder is suppressed.
var sharedESPDomains = map[string]bool{
	"amazonses.com":     true,
	"sendgrid.net":      true,
	"sendgrid.com":      true,
	"mailgun.org":       true,
	"mailgun.com":       true,
	"mandrillapp.com":   true,
	"mailchimp.com":     true,
	"mcsv.net":          true,
	"postmarkapp.com":   true,
	"sparkpostmail.com": true,
}

// genericDisplayNames are display-name values too common across organizations
// to credit as a sender match.
var genericDisplayNames = map[string]bool{
	"notifications": true,
	"notification":  true,
	"support":       true,
	"team":          true,
	"info":          true,
	"noreply":       true,
	"no-reply":      true,
	"admin":         true,
	"newsletter":    true,
	"alerts":        true,
	"alert":         true,
	"service":       true,
	"system":        true,
}

// subjectStopWords are common newsletter/notification words filtered before
// subject token Jaccard so that two unrelated "Weekly update" subjects do
// not register as similar purely on shared boilerplate.
var subjectStopWords = map[string]bool{
	"the":           true,
	"and":           true,
	"for":           true,
	"your":          true,
	"you":           true,
	"this":          true,
	"that":          true,
	"with":          true,
	"from":          true,
	"new":           true,
	"update":        true,
	"updates":       true,
	"weekly":        true,
	"daily":         true,
	"monthly":       true,
	"newsletter":    true,
	"notification":  true,
	"notifications": true,
	"are":           true,
	"was":           true,
	"has":           true,
	"have":          true,
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
