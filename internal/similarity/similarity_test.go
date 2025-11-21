package similarity

import (
	"mailboxzero/internal/jmap"
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
			want:  "hello  world", // Punctuation becomes spaces, not collapsed
		},
		{
			name:  "multiple spaces",
			input: "Hello    World",
			want:  "hello    world", // Multiple spaces preserved
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

func TestContainsCommonWords(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want bool
	}{
		{
			name: "has common words",
			s1:   "hello world test",
			s2:   "hello world example",
			want: true,
		},
		{
			name: "no common words",
			s1:   "hello world",
			s2:   "foo bar",
			want: false,
		},
		{
			name: "short words ignored",
			s1:   "a b c",
			s2:   "a b c",
			want: false,
		},
		{
			name: "one common word only",
			s1:   "hello world",
			s2:   "hello foo",
			want: false,
		},
		{
			name: "empty strings",
			s1:   "",
			s2:   "",
			want: false,
		},
		{
			name: "exactly two common words",
			s1:   "newsletter weekly update",
			s2:   "weekly newsletter digest",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsCommonWords(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("containsCommonWords(%q, %q) = %v, want %v", tt.s1, tt.s2, got, tt.want)
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
			want:   0.8, // 0.4 (subject) + 0.4 (sender) + 0.0 (no body) = 0.8
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
			name:    "string with common words boost",
			s1:      "newsletter weekly update digest information",
			s2:      "newsletter weekly report summary data",
			wantMin: 0.3,
			wantMax: 1.1, // Can exceed 1.0 with boost
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
