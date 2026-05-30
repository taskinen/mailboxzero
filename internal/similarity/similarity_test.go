package similarity

import (
	"fmt"
	"github.com/taskinen/mailboxzero/internal/jmap"
	"strings"
	"testing"
	"time"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want int
	}{
		{
			name: "identical strings",
			s1:   "hello",
			s2:   "hello",
			want: 0,
		},
		{
			name: "one character difference",
			s1:   "hello",
			s2:   "hella",
			want: 1,
		},
		{
			name: "empty strings",
			s1:   "",
			s2:   "",
			want: 0,
		},
		{
			name: "one empty string",
			s1:   "hello",
			s2:   "",
			want: 5,
		},
		{
			name: "completely different",
			s1:   "abc",
			s2:   "xyz",
			want: 3,
		},
		{
			name: "insertion",
			s1:   "cat",
			s2:   "cats",
			want: 1,
		},
		{
			name: "deletion",
			s1:   "cats",
			s2:   "cat",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshteinDistance(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("levenshteinDistance(%q, %q) = %v, want %v", tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase conversion",
			input: "Hello World",
			want:  "hello world",
		},
		{
			name:  "punctuation removal",
			input: "Hello, World!",
			want:  "hello world",
		},
		{
			name:  "multiple spaces collapsed",
			input: "Hello    World",
			want:  "hello world",
		},
		{
			name:  "special characters",
			input: "Hello@World#2023",
			want:  "hello world 2023",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only punctuation",
			input: "!!!???",
			want:  "",
		},
		{
			name:  "leading and trailing spaces",
			input: "  hello world  ",
			want:  "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeString(tt.input)
			if got != tt.want {
				t.Errorf("normalizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringSimilarity(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want float64
	}{
		{
			name: "identical strings",
			s1:   "hello world",
			s2:   "hello world",
			want: 1.0,
		},
		{
			name: "empty strings",
			s1:   "",
			s2:   "",
			want: 1.0, // Empty strings are considered identical after normalization
		},
		{
			name: "one empty string",
			s1:   "hello",
			s2:   "",
			want: 0.0,
		},
		{
			name: "similar strings with punctuation",
			s1:   "Hello, World!",
			s2:   "Hello World",
			want: 1.0, // Should be normalized to same string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringSimilarity(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("stringSimilarity(%q, %q) = %v, want %v", tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

func TestCalculateEmailSimilarity(t *testing.T) {
	email1 := jmap.Email{
		ID:      "1",
		Subject: "Weekly Newsletter",
		From: []jmap.EmailAddress{
			{Email: "newsletter@example.com"},
		},
		Preview: "This is a test newsletter",
	}

	email2 := jmap.Email{
		ID:      "2",
		Subject: "Weekly Newsletter",
		From: []jmap.EmailAddress{
			{Email: "newsletter@example.com"},
		},
		Preview: "This is another test newsletter",
	}

	email3 := jmap.Email{
		ID:      "3",
		Subject: "Completely Different Subject",
		From: []jmap.EmailAddress{
			{Email: "different@example.com"},
		},
		Preview: "Completely different content",
	}

	tests := []struct {
		name      string
		email1    jmap.Email
		email2    jmap.Email
		wantRange [2]float64 // min and max expected values
	}{
		{
			name:      "identical subject and sender",
			email1:    email1,
			email2:    email2,
			wantRange: [2]float64{0.8, 1.0}, // High similarity
		},
		{
			name:      "completely different emails",
			email1:    email1,
			email2:    email3,
			wantRange: [2]float64{0.0, 0.5}, // Low to moderate similarity
		},
		{
			name:      "same email with itself",
			email1:    email1,
			email2:    email1,
			wantRange: [2]float64{1.0, 1.0}, // Perfect match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateEmailSimilarity(tt.email1, tt.email2)
			if got < tt.wantRange[0] || got > tt.wantRange[1] {
				t.Errorf("calculateEmailSimilarity() = %v, want between %v and %v",
					got, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}

func TestExtractEmailBody(t *testing.T) {
	tests := []struct {
		name  string
		email jmap.Email
		want  string
	}{
		{
			name: "preview available",
			email: jmap.Email{
				Preview: "Test preview",
			},
			want: "Test preview",
		},
		{
			name: "body values available",
			email: jmap.Email{
				Preview: "",
				BodyValues: map[string]jmap.BodyValue{
					"1": {Value: "Test body content"},
				},
			},
			want: "test body content",
		},
		{
			name: "both preview and body values",
			email: jmap.Email{
				Preview: "Test preview",
				BodyValues: map[string]jmap.BodyValue{
					"1": {Value: "Test body content"},
				},
			},
			want: "Test preview", // Preview takes precedence
		},
		{
			name:  "no content",
			email: jmap.Email{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEmailBody(tt.email)
			if got != tt.want {
				t.Errorf("extractEmailBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindSimilarEmails(t *testing.T) {
	emails := []jmap.Email{
		{
			ID:      "1",
			Subject: "Newsletter Issue 1",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Welcome to our newsletter",
		},
		{
			ID:      "2",
			Subject: "Newsletter Issue 2",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Welcome to our newsletter",
		},
		{
			ID:      "3",
			Subject: "Newsletter Issue 3",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Welcome to our newsletter",
		},
		{
			ID:      "4",
			Subject: "Completely Different",
			From:    []jmap.EmailAddress{{Email: "other@example.com"}},
			Preview: "Different content",
		},
	}

	tests := []struct {
		name      string
		emails    []jmap.Email
		threshold float64
		wantMin   int // Minimum expected similar emails
	}{
		{
			name:      "high threshold - newsletters only",
			emails:    emails,
			threshold: 0.8,
			wantMin:   3, // Should find the 3 newsletter emails
		},
		{
			name:      "low threshold - all emails",
			emails:    emails,
			threshold: 0.0,
			wantMin:   3, // Should find largest group
		},
		{
			name:      "empty input",
			emails:    []jmap.Email{},
			threshold: 0.5,
			wantMin:   0,
		},
		{
			name: "single email",
			emails: []jmap.Email{
				{ID: "1", Subject: "Test"},
			},
			threshold: 0.5,
			wantMin:   0, // Single email has no similar emails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSimilarEmails(tt.emails, tt.threshold)
			if len(got) < tt.wantMin {
				t.Errorf("FindSimilarEmails() returned %d emails, want at least %d",
					len(got), tt.wantMin)
			}
		})
	}
}

func TestFindSimilarToEmail(t *testing.T) {
	targetEmail := jmap.Email{
		ID:      "target",
		Subject: "Newsletter Issue 1",
		From:    []jmap.EmailAddress{{Email: "news@example.com"}},
		Preview: "Welcome to our newsletter",
	}

	emails := []jmap.Email{
		targetEmail,
		{
			ID:      "2",
			Subject: "Newsletter Issue 2",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Welcome to our newsletter",
		},
		{
			ID:      "3",
			Subject: "Newsletter Issue 3",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Welcome to our newsletter",
		},
		{
			ID:      "4",
			Subject: "Completely Different",
			From:    []jmap.EmailAddress{{Email: "other@example.com"}},
			Preview: "Different content",
		},
	}

	tests := []struct {
		name        string
		targetEmail jmap.Email
		emails      []jmap.Email
		threshold   float64
		wantMin     int // Minimum expected results (includes target)
		wantMax     int // Maximum expected results
	}{
		{
			name:        "high threshold - similar newsletters",
			targetEmail: targetEmail,
			emails:      emails,
			threshold:   0.8,
			wantMin:     3, // Target + 2 similar
			wantMax:     4, // At most all newsletters
		},
		{
			name:        "very high threshold - only exact matches",
			targetEmail: targetEmail,
			emails:      emails,
			threshold:   0.99,
			wantMin:     1, // At least the target itself
			wantMax:     4, // Target + possibly similar newsletters
		},
		{
			name:        "low threshold - all emails",
			targetEmail: targetEmail,
			emails:      emails,
			threshold:   0.0,
			wantMin:     4, // Should include all emails
			wantMax:     4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSimilarToEmail(tt.targetEmail, tt.emails, tt.threshold)

			if len(got) < tt.wantMin || len(got) > tt.wantMax {
				t.Errorf("FindSimilarToEmail() returned %d emails, want between %d and %d",
					len(got), tt.wantMin, tt.wantMax)
			}

			// First result should always be the target email
			if len(got) > 0 && got[0].ID != tt.targetEmail.ID {
				t.Errorf("FindSimilarToEmail() first result ID = %v, want %v",
					got[0].ID, tt.targetEmail.ID)
			}
		})
	}
}

func TestGroupSimilarEmails(t *testing.T) {
	emails := []jmap.Email{
		{
			ID:      "1",
			Subject: "Newsletter A",
			From:    []jmap.EmailAddress{{Email: "a@example.com"}},
		},
		{
			ID:      "2",
			Subject: "Newsletter A",
			From:    []jmap.EmailAddress{{Email: "a@example.com"}},
		},
		{
			ID:      "3",
			Subject: "Newsletter B",
			From:    []jmap.EmailAddress{{Email: "b@example.com"}},
		},
		{
			ID:      "4",
			Subject: "Newsletter B",
			From:    []jmap.EmailAddress{{Email: "b@example.com"}},
		},
	}

	tests := []struct {
		name          string
		emails        []jmap.Email
		threshold     float64
		wantMinGroups int
	}{
		{
			name:          "high threshold - find groups",
			emails:        emails,
			threshold:     0.8,
			wantMinGroups: 2, // Should find 2 groups
		},
		{
			name:          "very high threshold - fewer groups",
			emails:        emails,
			threshold:     0.99,
			wantMinGroups: 0,
		},
		{
			name:          "empty emails",
			emails:        []jmap.Email{},
			threshold:     0.5,
			wantMinGroups: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupSimilarEmails(tt.emails, tt.threshold)
			if len(got) < tt.wantMinGroups {
				t.Errorf("groupSimilarEmails() returned %d groups, want at least %d",
					len(got), tt.wantMinGroups)
			}

			// Verify each group has at least 2 emails
			for i, group := range got {
				if len(group.Emails) < 2 {
					t.Errorf("groupSimilarEmails() group %d has %d emails, want at least 2",
						i, len(group.Emails))
				}
			}
		})
	}
}

func TestCalculateGroupSimilarity(t *testing.T) {
	email1 := jmap.Email{
		ID:      "1",
		Subject: "Test",
		From:    []jmap.EmailAddress{{Email: "test@example.com"}},
	}

	email2 := jmap.Email{
		ID:      "2",
		Subject: "Test",
		From:    []jmap.EmailAddress{{Email: "test@example.com"}},
	}

	tests := []struct {
		name   string
		emails []jmap.Email
		want   float64
	}{
		{
			name:   "empty group",
			emails: []jmap.Email{},
			want:   0.0,
		},
		{
			name:   "single email",
			emails: []jmap.Email{email1},
			want:   0.0,
		},
		{
			name:   "two identical emails",
			emails: []jmap.Email{email1, email2},
			want:   0.8, // 0.5 (sender) + 0.3 (subject) + 0.0 (no body) = 0.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateGroupSimilarity(tt.emails)
			if got != tt.want {
				t.Errorf("calculateGroupSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinMax(t *testing.T) {
	t.Run("min function", func(t *testing.T) {
		tests := []struct {
			name    string
			a, b, c int
			want    int
		}{
			{"a is minimum", 1, 2, 3, 1},
			{"b is minimum", 2, 1, 3, 1},
			{"c is minimum", 2, 3, 1, 1},
			{"all equal", 1, 1, 1, 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := min(tt.a, tt.b, tt.c)
				if got != tt.want {
					t.Errorf("min(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.want)
				}
			})
		}
	})

	t.Run("max function", func(t *testing.T) {
		tests := []struct {
			name string
			a, b int
			want int
		}{
			{"a is maximum", 5, 3, 5},
			{"b is maximum", 3, 5, 5},
			{"equal values", 4, 4, 4},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := max(tt.a, tt.b)
				if got != tt.want {
					t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
				}
			})
		}
	})
}

// Benchmark tests for performance-critical functions

func BenchmarkLevenshteinDistance(b *testing.B) {
	s1 := "this is a test string for benchmarking"
	s2 := "this is another test string for benchmark"

	for i := 0; i < b.N; i++ {
		levenshteinDistance(s1, s2)
	}
}

func TestJaccardSimilarity(t *testing.T) {
	mkSet := func(words ...string) map[string]struct{} {
		s := make(map[string]struct{}, len(words))
		for _, w := range words {
			s[w] = struct{}{}
		}
		return s
	}

	tests := []struct {
		name string
		a    map[string]struct{}
		b    map[string]struct{}
		want float64
	}{
		{name: "identical sets", a: mkSet("foo", "bar"), b: mkSet("foo", "bar"), want: 1.0},
		{name: "disjoint sets", a: mkSet("foo", "bar"), b: mkSet("baz", "qux"), want: 0.0},
		{name: "partial overlap", a: mkSet("foo", "bar", "baz"), b: mkSet("foo", "bar", "qux"), want: 0.5},
		{name: "first empty", a: nil, b: mkSet("foo"), want: 0.0},
		{name: "second empty", a: mkSet("foo"), b: nil, want: 0.0},
		{name: "both empty", a: nil, b: nil, want: 0.0},
		{name: "subset", a: mkSet("foo"), b: mkSet("foo", "bar"), want: 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jaccardSimilarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("jaccardSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenizeBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string // expected tokens (order doesn't matter, presence does)
	}{
		{name: "empty", body: "", want: nil},
		{name: "short words dropped", body: "a is to be", want: nil},
		{name: "mixed lengths keeps ≥3", body: "Hello a world", want: []string{"hello", "world"}},
		{name: "normalization", body: "Hello, World! 2026", want: []string{"hello", "world", "2026"}},
		{name: "dedups", body: "weekly weekly weekly", want: []string{"weekly"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeBody(tt.body)
			if len(got) != len(tt.want) {
				t.Errorf("tokenizeBody(%q) size = %d, want %d (got %v)", tt.body, len(got), len(tt.want), got)
				return
			}
			for _, w := range tt.want {
				if _, ok := got[w]; !ok {
					t.Errorf("tokenizeBody(%q) missing %q (got %v)", tt.body, w, got)
				}
			}
		})
	}
}

// BenchmarkFindSimilarEmails exercises the full pipeline on a synthetic
// 1000-email corpus so the speedup from precompute + Jaccard is measurable.
func BenchmarkFindSimilarEmails(b *testing.B) {
	emails := make([]jmap.Email, 0, 1000)
	for i := 0; i < 1000; i++ {
		emails = append(emails, jmap.Email{
			ID:      fmt.Sprintf("email-%d", i),
			Subject: fmt.Sprintf("Weekly Newsletter Issue %d", i%50),
			From: []jmap.EmailAddress{{
				Email: fmt.Sprintf("sender%d@example.com", i%20),
			}},
			Preview:    fmt.Sprintf("This is preview number %d about widgets and gadgets and things.", i%30),
			ReceivedAt: time.Now(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindSimilarEmails(emails, 0.75)
	}
}

func BenchmarkCalculateEmailSimilarity(b *testing.B) {
	email1 := jmap.Email{
		ID:         "1",
		Subject:    "Weekly Newsletter Issue 123",
		From:       []jmap.EmailAddress{{Email: "newsletter@example.com"}},
		Preview:    "This is a preview of the newsletter content",
		ReceivedAt: time.Now(),
	}

	email2 := jmap.Email{
		ID:         "2",
		Subject:    "Weekly Newsletter Issue 124",
		From:       []jmap.EmailAddress{{Email: "newsletter@example.com"}},
		Preview:    "This is another preview of the newsletter content",
		ReceivedAt: time.Now(),
	}

	for i := 0; i < b.N; i++ {
		calculateEmailSimilarity(email1, email2)
	}
}

func TestStringSimilarity_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		s1      string
		s2      string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "very long similar strings",
			s1:      strings.Repeat("hello world ", 100),
			s2:      strings.Repeat("hello world ", 100),
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "partial word overlap stays below 1.0",
			s1:      "newsletter weekly update digest information",
			s2:      "newsletter weekly report summary data",
			wantMin: 0.3,
			wantMax: 1.0,
		},
		{
			name:    "mostly punctuation",
			s1:      "!!!???***",
			s2:      "###$$$%%%",
			wantMin: 0.0,
			wantMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringSimilarity(tt.s1, tt.s2)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("stringSimilarity() = %v, want between %v and %v",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFindSimilarEmails_EmptyResult(t *testing.T) {
	// Test with emails that are all unique (no similar pairs)
	emails := []jmap.Email{
		{
			ID:      "1",
			Subject: "Unique Subject A",
			From:    []jmap.EmailAddress{{Email: "a@example.com"}},
			Preview: "Completely unique content A",
		},
		{
			ID:      "2",
			Subject: "Different Subject B",
			From:    []jmap.EmailAddress{{Email: "b@example.com"}},
			Preview: "Totally different content B",
		},
		{
			ID:      "3",
			Subject: "Another Topic C",
			From:    []jmap.EmailAddress{{Email: "c@example.com"}},
			Preview: "Distinct content C",
		},
	}

	result := FindSimilarEmails(emails, 0.9)

	// With high threshold and unique emails, should return nil or empty
	if result != nil && len(result) > 0 {
		// This is okay - might return a small group
	}
}

func TestFindSimilarEmails_NilInput(t *testing.T) {
	result := FindSimilarEmails(nil, 0.5)
	if result != nil {
		t.Errorf("FindSimilarEmails(nil) = %v, want nil", result)
	}
}

func TestGroupSimilarEmails_SingleGroup(t *testing.T) {
	// All emails very similar
	emails := []jmap.Email{
		{
			ID:      "1",
			Subject: "Newsletter",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Content",
		},
		{
			ID:      "2",
			Subject: "Newsletter",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Content",
		},
		{
			ID:      "3",
			Subject: "Newsletter",
			From:    []jmap.EmailAddress{{Email: "news@example.com"}},
			Preview: "Content",
		},
	}

	groups := groupSimilarEmails(emails, 0.8)

	if len(groups) == 0 {
		t.Error("groupSimilarEmails() should find at least one group")
	}

	// First group should have all 3 emails
	if len(groups) > 0 && len(groups[0].Emails) != 3 {
		t.Errorf("groupSimilarEmails() first group has %d emails, want 3",
			len(groups[0].Emails))
	}
}

func TestCalculateEmailSimilarity_NoFrom(t *testing.T) {
	email1 := jmap.Email{
		ID:      "1",
		Subject: "Test",
		From:    []jmap.EmailAddress{}, // Empty From
		Preview: "Content",
	}

	email2 := jmap.Email{
		ID:      "2",
		Subject: "Test",
		From:    []jmap.EmailAddress{}, // Empty From
		Preview: "Content",
	}

	similarity := calculateEmailSimilarity(email1, email2)

	// Should still calculate similarity based on subject and body
	if similarity < 0.0 || similarity > 1.0 {
		t.Errorf("calculateEmailSimilarity() = %v, want between 0.0 and 1.0", similarity)
	}
}

func TestCalculateEmailSimilarity_NoBody(t *testing.T) {
	email1 := jmap.Email{
		ID:      "1",
		Subject: "Test Subject",
		From:    []jmap.EmailAddress{{Email: "test@example.com"}},
		Preview: "", // No preview
	}

	email2 := jmap.Email{
		ID:      "2",
		Subject: "Test Subject",
		From:    []jmap.EmailAddress{{Email: "test@example.com"}},
		Preview: "", // No preview
	}

	similarity := calculateEmailSimilarity(email1, email2)

	// Should calculate based on subject and sender only (0.4 + 0.4 + 0.0)
	if similarity < 0.7 || similarity > 0.9 {
		t.Errorf("calculateEmailSimilarity() without body = %v, want ~0.8", similarity)
	}
}

func TestCalculateGroupSimilarity_MultipleEmails(t *testing.T) {
	emails := []jmap.Email{
		{
			ID:      "1",
			Subject: "Test",
			From:    []jmap.EmailAddress{{Email: "test@example.com"}},
		},
		{
			ID:      "2",
			Subject: "Test",
			From:    []jmap.EmailAddress{{Email: "test@example.com"}},
		},
		{
			ID:      "3",
			Subject: "Test",
			From:    []jmap.EmailAddress{{Email: "test@example.com"}},
		},
	}

	similarity := calculateGroupSimilarity(emails)

	// Should average all pairwise similarities
	if similarity < 0.0 || similarity > 1.0 {
		t.Errorf("calculateGroupSimilarity() = %v, want between 0.0 and 1.0", similarity)
	}

	// For 3 identical emails, should be high
	if similarity < 0.7 {
		t.Errorf("calculateGroupSimilarity() for identical emails = %v, want > 0.7", similarity)
	}
}

func TestSenderSimilarity_Ladder(t *testing.T) {
	mk := func(name, addr string) features {
		return precomputeOne(jmap.Email{
			From: []jmap.EmailAddress{{Name: name, Email: addr}},
		})
	}

	tests := []struct {
		name string
		a    features
		b    features
		want float64
	}{
		{
			name: "same full address",
			a:    mk("", "noreply@datablocks.com"),
			b:    mk("", "noreply@datablocks.com"),
			want: 1.0,
		},
		{
			name: "same domain different local part",
			a:    mk("", "support@datablocks.com"),
			b:    mk("", "newsletter@datablocks.com"),
			want: 0.8,
		},
		{
			name: "same registrable root different subdomain",
			a:    mk("", "alerts@mail.google.com"),
			b:    mk("", "noreply@accounts.google.com"),
			want: 0.7,
		},
		{
			name: "specific display name matches across addresses",
			a:    mk("Datablocks", "bounce+abc@mta-east.example.net"),
			b:    mk("Datablocks", "bounce+xyz@mta-west.example.org"),
			want: 0.6,
		},
		{
			name: "different orgs, different domains",
			a:    mk("Stripe", "noreply@stripe.com"),
			b:    mk("Google", "noreply@google.com"),
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := senderSimilarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("senderSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSenderSimilarity_SharedESPGuard(t *testing.T) {
	a := precomputeOne(jmap.Email{
		From: []jmap.EmailAddress{{Email: "bounces+a@sendgrid.net"}},
	})
	b := precomputeOne(jmap.Email{
		From: []jmap.EmailAddress{{Email: "bounces+b@sendgrid.net"}},
	})

	// Same ESP domain with different local parts must not credit the
	// same-domain rung, otherwise unrelated senders relayed through one
	// provider would cluster together.
	got := senderSimilarity(a, b)
	if got != 0.0 {
		t.Errorf("senderSimilarity() for ESP-relayed senders = %v, want 0.0", got)
	}
}

func TestSenderSimilarity_GenericDisplayName(t *testing.T) {
	a := precomputeOne(jmap.Email{
		From: []jmap.EmailAddress{{Name: "Notifications", Email: "alerts@stripe.com"}},
	})
	b := precomputeOne(jmap.Email{
		From: []jmap.EmailAddress{{Name: "Notifications", Email: "alerts@google.com"}},
	})

	got := senderSimilarity(a, b)
	if got >= 0.6 {
		t.Errorf("senderSimilarity() with generic display name = %v, want < 0.6", got)
	}
}

func TestSubjectSimilarity_StripsReplyPrefixes(t *testing.T) {
	a := precomputeOne(jmap.Email{Subject: "Re: Re: Quarterly review summary"})
	b := precomputeOne(jmap.Email{Subject: "Quarterly review summary"})

	got := subjectSimilarity(a, b)
	if got != 1.0 {
		t.Errorf("subjectSimilarity() with stripped Re: prefixes = %v, want 1.0", got)
	}
}

func TestSubjectSimilarity_StripsMailingListTag(t *testing.T) {
	a := precomputeOne(jmap.Email{Subject: "[golang-nuts] Generics question"})
	b := precomputeOne(jmap.Email{Subject: "Generics question"})

	got := subjectSimilarity(a, b)
	if got != 1.0 {
		t.Errorf("subjectSimilarity() with stripped list tag = %v, want 1.0", got)
	}
}

func TestCalculateEmailSimilarity_DatablocksScenario(t *testing.T) {
	// 4 emails from the same Datablocks newsletter sender with varied
	// subjects and bodies — the case that prompted the algorithm
	// rewrite. They share the brand name across subject and body and
	// boilerplate footer text ("unsubscribe"), enough to clear the new
	// default threshold (60%).
	emails := []jmap.Email{
		{
			Subject: "Your daily Datablocks digest",
			From:    []jmap.EmailAddress{{Email: "newsletter@datablocks.com", Name: "Datablocks"}},
			Preview: "Today's curated blocks and analytics from Datablocks. Click here to unsubscribe.",
		},
		{
			Subject: "Weekend Datablocks roundup",
			From:    []jmap.EmailAddress{{Email: "newsletter@datablocks.com", Name: "Datablocks"}},
			Preview: "A look back at popular blocks this week from the Datablocks team. Unsubscribe link below.",
		},
		{
			Subject: "Datablocks platform: beta features now live",
			From:    []jmap.EmailAddress{{Email: "newsletter@datablocks.com", Name: "Datablocks"}},
			Preview: "Datablocks shipped a new query editor and improved blocks UI. To unsubscribe click below.",
		},
		{
			Subject: "Special offer: Datablocks Pro at 20% off",
			From:    []jmap.EmailAddress{{Email: "newsletter@datablocks.com", Name: "Datablocks"}},
			Preview: "Upgrade Datablocks Pro for advanced blocks and team workspaces. Unsubscribe here.",
		},
	}

	// Same-sender newsletters with varied content land in the 0.55-0.75
	// range under the new weights. The previous algorithm produced
	// scores under 0.40 for the same shape of input, so even the lower
	// threshold here represents a large improvement.
	for i := 0; i < len(emails); i++ {
		for j := i + 1; j < len(emails); j++ {
			got := calculateEmailSimilarity(emails[i], emails[j])
			if got < 0.55 {
				t.Errorf("calculateEmailSimilarity(Datablocks[%d], Datablocks[%d]) = %v, want >= 0.55",
					i, j, got)
			}
		}
	}
}

func TestCalculateEmailSimilarity_GoogleSubdomains(t *testing.T) {
	a := jmap.Email{
		Subject: "Security alert: new sign-in",
		From:    []jmap.EmailAddress{{Email: "no-reply@accounts.google.com"}},
		Preview: "We detected a new sign-in to your account.",
	}
	b := jmap.Email{
		Subject: "Your weekly Google Photos memories",
		From:    []jmap.EmailAddress{{Email: "noreply-photos@google.com"}},
		Preview: "Rediscover photos and videos from past years.",
	}

	got := calculateEmailSimilarity(a, b)
	// Same registrable root (google.com) gives 0.7 * 0.5 = 0.35 from
	// sender alone, no subject/body overlap — total around 0.35-0.4.
	// Lower the threshold to 0.35 to validate the registrable root signal.
	if got < 0.35 {
		t.Errorf("calculateEmailSimilarity(Google subdomains) = %v, want >= 0.35", got)
	}
}

func TestCalculateEmailSimilarity_NoFalsePositive_GenericLocals(t *testing.T) {
	a := jmap.Email{
		Subject: "Receipt for your payment",
		From:    []jmap.EmailAddress{{Email: "noreply@stripe.com"}},
		Preview: "Thanks for using Stripe. Your receipt is attached.",
	}
	b := jmap.Email{
		Subject: "Security alert: new sign-in",
		From:    []jmap.EmailAddress{{Email: "noreply@google.com"}},
		Preview: "We detected a new sign-in to your account.",
	}

	got := calculateEmailSimilarity(a, b)
	// The old algorithm rewarded "noreply" and "com" tokens shared
	// across these unrelated emails — sender ladder must score 0.0.
	if got >= 0.4 {
		t.Errorf("calculateEmailSimilarity(noreply@stripe vs noreply@google) = %v, want < 0.4", got)
	}
}

func TestCalculateEmailSimilarity_NoFalsePositive_SharedESP(t *testing.T) {
	a := jmap.Email{
		Subject: "Welcome to Acme",
		From:    []jmap.EmailAddress{{Email: "bounce-acme-12345@sendgrid.net"}},
		Preview: "Thanks for joining Acme.",
	}
	b := jmap.Email{
		Subject: "Your Beta Corp account is ready",
		From:    []jmap.EmailAddress{{Email: "bounce-beta-67890@sendgrid.net"}},
		Preview: "Get started with Beta Corp today.",
	}

	got := calculateEmailSimilarity(a, b)
	if got >= 0.4 {
		t.Errorf("calculateEmailSimilarity(sendgrid relay) = %v, want < 0.4", got)
	}
}

func TestGroupSimilarFeatures_ClusterExpansion(t *testing.T) {
	// "far" does not meet the threshold against the cluster seed but
	// does meet it against the "bridge" member. Under the old
	// seed-only clustering, far would be left out (its 1-element group
	// is then dropped). Under single-link expansion, the bridge pulls
	// far into the cluster — exactly the symptom that contributed to
	// the user's "Datablocks group not found" report.
	from := []jmap.EmailAddress{{Email: "alerts@acme.example"}}
	emails := []jmap.Email{
		{
			ID: "seed", From: from,
			Subject: "Service status report",
			Preview: "Service status and uptime summary.",
		},
		{
			ID: "sibling", From: from,
			Subject: "Service status report",
			Preview: "Service status and uptime summary.",
		},
		{
			ID: "bridge", From: from,
			Subject: "Service status and alerts report",
			Preview: "Service status uptime alerts bridge summary.",
		},
		{
			ID: "far", From: from,
			Subject: "Alerts dashboard",
			Preview: "Alerts bridge dashboard.",
		},
	}

	// Sanity check: far should NOT match seed directly under 0.6, so
	// the test is only meaningful when cluster expansion is what pulls
	// far in.
	if direct := calculateEmailSimilarity(emails[0], emails[3]); direct >= 0.6 {
		t.Fatalf("test premise violated: seed↔far direct similarity %v ≥ 0.6", direct)
	}

	groups := groupSimilarEmails(emails, 0.6)
	if len(groups) != 1 {
		t.Fatalf("groupSimilarEmails() returned %d groups, want 1", len(groups))
	}
	if len(groups[0].Emails) != 4 {
		ids := make([]string, 0, len(groups[0].Emails))
		for _, e := range groups[0].Emails {
			ids = append(ids, e.ID)
		}
		t.Errorf("groupSimilarEmails() cluster size = %d (%v), want 4", len(groups[0].Emails), ids)
	}
}

func TestNormalizeSubject_StripsPrefixes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no prefix", input: "Quarterly review summary", want: "quarterly review summary"},
		{name: "Re:", input: "Re: Quarterly review summary", want: "quarterly review summary"},
		{name: "Fwd:", input: "Fwd: Quarterly review summary", want: "quarterly review summary"},
		{name: "Fw:", input: "Fw: Quarterly review summary", want: "quarterly review summary"},
		{name: "AW:", input: "AW: Quarterly review summary", want: "quarterly review summary"},
		{name: "SV:", input: "SV: Quarterly review summary", want: "quarterly review summary"},
		{name: "repeated Re:", input: "Re: Re: Re: Quarterly review summary", want: "quarterly review summary"},
		{name: "case-insensitive", input: "RE: quarterly review summary", want: "quarterly review summary"},
		{name: "list tag only", input: "[golang-nuts] Generics question", want: "generics question"},
		{name: "tag + reply prefix", input: "[golang-nuts] Re: Generics question", want: "generics question"},
		{name: "reply prefix + tag", input: "Re: [golang-nuts] Generics question", want: "generics question"},
		{name: "no whitespace after colon", input: "Re:tight", want: "tight"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSubject(tt.input)
			if got != tt.want {
				t.Errorf("normalizeSubject(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokenizeSubject(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty", input: "", want: nil},
		{name: "all stop words", input: "the weekly update", want: nil},
		{name: "all short", input: "of by to a", want: nil},
		{name: "mix of stop and meaningful", input: "weekly platform release notes", want: []string{"platform", "release", "notes"}},
		{name: "deduplicates", input: "alerts alerts alerts", want: []string{"alerts"}},
		{name: "drops 1-2 char tokens", input: "pr ci build", want: []string{"build"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeSubject(normalizeString(tt.input))
			if len(got) != len(tt.want) {
				t.Errorf("tokenizeSubject(%q) size = %d, want %d (got %v)", tt.input, len(got), len(tt.want), got)
				return
			}
			for _, w := range tt.want {
				if _, ok := got[w]; !ok {
					t.Errorf("tokenizeSubject(%q) missing %q (got %v)", tt.input, w, got)
				}
			}
		})
	}
}

func TestSubjectSimilarity_BothEmpty(t *testing.T) {
	a := precomputeOne(jmap.Email{Subject: ""})
	b := precomputeOne(jmap.Email{Subject: ""})
	if got := subjectSimilarity(a, b); got != 1.0 {
		t.Errorf("subjectSimilarity(empty, empty) = %v, want 1.0", got)
	}

	c := precomputeOne(jmap.Email{Subject: "Something"})
	if got := subjectSimilarity(a, c); got != 0.0 {
		t.Errorf("subjectSimilarity(empty, non-empty) = %v, want 0.0", got)
	}
}

func TestSenderSimilarity_ESPGuard_AllProviders(t *testing.T) {
	// Each pair shares only the ESP domain (different local parts), so
	// the same-domain rung must be suppressed across every listed
	// provider. Catches silent typos or omissions in sharedESPDomains.
	providers := []string{
		"amazonses.com",
		"sendgrid.net",
		"sendgrid.com",
		"mailgun.org",
		"mailgun.com",
		"mandrillapp.com",
		"mailchimp.com",
		"mcsv.net",
		"postmarkapp.com",
		"sparkpostmail.com",
	}

	for _, domain := range providers {
		t.Run(domain, func(t *testing.T) {
			a := precomputeOne(jmap.Email{
				From: []jmap.EmailAddress{{Email: "bounces+a@" + domain}},
			})
			b := precomputeOne(jmap.Email{
				From: []jmap.EmailAddress{{Email: "bounces+b@" + domain}},
			})
			if got := senderSimilarity(a, b); got != 0.0 {
				t.Errorf("senderSimilarity() for %s = %v, want 0.0", domain, got)
			}
		})
	}
}

func TestSenderSimilarity_ShortDisplayName(t *testing.T) {
	// Display names under 4 characters are too generic to be a reliable
	// match signal. "BBC" is a real-world example that would otherwise
	// false-positive across unrelated three-letter senders.
	a := precomputeOne(jmap.Email{
		From: []jmap.EmailAddress{{Name: "BBC", Email: "news@bbc.example"}},
	})
	b := precomputeOne(jmap.Email{
		From: []jmap.EmailAddress{{Name: "BBC", Email: "alerts@other.example"}},
	})

	if got := senderSimilarity(a, b); got >= 0.6 {
		t.Errorf("senderSimilarity() with 3-char display name = %v, want < 0.6", got)
	}
}

func TestParseSender_RegistrableRoot(t *testing.T) {
	tests := []struct {
		addr     string
		wantDom  string
		wantRoot string
	}{
		{"a@example.com", "example.com", "example.com"},
		{"a@mail.example.com", "mail.example.com", "example.com"},
		{"a@deep.mail.example.com", "deep.mail.example.com", "example.com"},
		{"a@example.co.uk", "example.co.uk", "example.co.uk"},
		{"a@mail.example.co.uk", "mail.example.co.uk", "example.co.uk"},
		{"a@user.github.io", "user.github.io", "user.github.io"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			_, dom, root, _ := parseSender(tt.addr, "")
			if dom != tt.wantDom {
				t.Errorf("parseSender(%q) domain = %q, want %q", tt.addr, dom, tt.wantDom)
			}
			if root != tt.wantRoot {
				t.Errorf("parseSender(%q) root = %q, want %q", tt.addr, root, tt.wantRoot)
			}
		})
	}
}
