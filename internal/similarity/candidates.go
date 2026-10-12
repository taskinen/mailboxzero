package similarity

import (
	"math"
	"sort"
)

// candidateIndex lists, for each email, the other emails it could possibly
// reach the threshold with, so grouping never scores a pair that cannot
// match. Below 50% the cheap sender check cannot rule out a pair from
// unrelated senders (subject and body alone can reach the threshold), so
// scoring every pair meant computing a subject Levenshtein distance for
// nearly all of them.
//
// Emails that could match share a key:
//   - Sender keys mirror senderSimilarity: two emails score above zero on
//     sender exactly when they share their address, domain, root domain or
//     display name (with the same exceptions). Above 50% a pair cannot pass
//     without a sender score, so sender keys are enough.
//   - Body keys add the pairs from unrelated senders above 30%. Such a pair
//     needs a body Jaccard similarity of at least (threshold-0.3)/0.2, and
//     token sets that similar always share one of their rarest tokens
//     (prefix filtering), so each email is indexed under its rarest tokens
//     only.
//
// At 30% and below, a pair can pass on subject alone, so every email is a
// candidate of every other.
type candidateIndex struct {
	all      bool    // every email is a candidate of every other
	keys     [][]int // keys[i] lists the keys of email i
	postings [][]int // postings[k] lists the emails with key k

	// seen[j] == visits marks email j as already visited in this call
	// of visit, so emails sharing several keys are visited once.
	seen   []int
	visits int
}

func newCandidateIndex(feats []features, threshold float64) *candidateIndex {
	idx := &candidateIndex{seen: make([]int, len(feats))}

	// similarityWithThreshold rejects a pair without a sender score when
	// 0.5 (the most subject and body can add) is below the threshold.
	senderOnly := 0.5 < threshold
	// The epsilon keeps rounding from excluding a pair right at the limit;
	// a lower minimum only indexes more tokens.
	minBodyJaccard := (threshold-0.3)/0.2 - 1e-9
	if !senderOnly && minBodyJaccard <= 0 {
		idx.all = true
		return idx
	}

	idx.keys = make([][]int, len(feats))
	keyIDs := make(map[string]int)
	add := func(i int, key string) {
		id, ok := keyIDs[key]
		if !ok {
			id = len(idx.postings)
			keyIDs[key] = id
			idx.postings = append(idx.postings, nil)
		}
		idx.keys[i] = append(idx.keys[i], id)
		idx.postings[id] = append(idx.postings[id], i)
	}

	for i, f := range feats {
		for _, key := range senderKeys(f) {
			add(i, key)
		}
	}

	if !senderOnly {
		frequency := make(map[string]int)
		for _, f := range feats {
			for token := range f.bodyTokens {
				frequency[token]++
			}
		}
		for i, f := range feats {
			for _, token := range rarestTokens(f.bodyTokens, frequency, minBodyJaccard) {
				add(i, "body:"+token)
			}
		}
	}

	return idx
}

// senderKeys returns the keys under which two emails share a key exactly
// when senderSimilarity scores them above zero.
func senderKeys(f features) []string {
	var keys []string
	if f.senderAddr != "" {
		keys = append(keys, "addr:"+f.senderAddr)
	}
	if f.senderDomain != "" && !sharedESPDomains[f.senderDomain] {
		keys = append(keys, "domain:"+f.senderDomain)
	}
	if f.senderRoot != "" && !sharedESPDomains[f.senderRoot] {
		keys = append(keys, "root:"+f.senderRoot)
	}
	if f.senderName != "" && len(f.senderName) >= 4 && !genericDisplayNames[f.senderName] {
		keys = append(keys, "name:"+f.senderName)
	}
	return keys
}

// rarestTokens returns the prefix of tokens, sorted rarest first, that any
// token set with a Jaccard similarity of at least minJaccard to them must
// share a token with: the first len - ceil(minJaccard*len) + 1 tokens.
// minJaccard must be in (0, 1].
func rarestTokens(tokens map[string]struct{}, frequency map[string]int, minJaccard float64) []string {
	if len(tokens) == 0 {
		return nil
	}
	sorted := make([]string, 0, len(tokens))
	for token := range tokens {
		sorted = append(sorted, token)
	}
	sort.Slice(sorted, func(a, b int) bool {
		fa, fb := frequency[sorted[a]], frequency[sorted[b]]
		if fa != fb {
			return fa < fb
		}
		return sorted[a] < sorted[b]
	})
	return sorted[:len(sorted)-int(math.Ceil(minJaccard*float64(len(sorted))))+1]
}

// visit calls fn once for every candidate of email i.
func (idx *candidateIndex) visit(i int, fn func(j int)) {
	if idx.all {
		for j := range idx.seen {
			if j != i {
				fn(j)
			}
		}
		return
	}

	idx.visits++
	for _, key := range idx.keys[i] {
		for _, j := range idx.postings[key] {
			if j != i && idx.seen[j] != idx.visits {
				idx.seen[j] = idx.visits
				fn(j)
			}
		}
	}
}
