// Package shared holds atomic, stateless helpers reused across provider
// normalizers. NOTHING here may branch on provider identity — if a helper needs
// to know whether it's anthropic or openai, it belongs in that provider's
// package instead (see CONVENTIONS.md).
package shared

import "unicode"

// ApproxTokens returns a rough token estimate for s. It is intentionally
// provider-agnostic and APPROXIMATE — callers must label any UI derived from it
// as an estimate, never an exact count.
//
// Heuristic: BPE tokenizers emit roughly one token per ~4 ASCII chars, but
// closer to one token per CJK character. We split the two so mixed
// English/Chinese prompts (common here) don't get wildly under- or
// over-counted.
func ApproxTokens(s string) int64 {
	var asciiBytes, cjk int64
	for _, r := range s {
		if isCJK(r) {
			cjk++
			continue
		}
		asciiBytes += int64(utf8Len(r))
	}
	tokens := asciiBytes/4 + cjk
	if tokens == 0 && len(s) > 0 {
		return 1
	}
	return tokens
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func utf8Len(r rune) int {
	switch {
	case r < 0x80:
		return 1
	case r < 0x800:
		return 2
	case r < 0x10000:
		return 3
	default:
		return 4
	}
}
