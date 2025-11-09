package fuzzy

import (
	"sort"
	"strings"
	"unicode"
)

type MatchResult struct {
	Text  string
	Score int
	Index int
}

func Match(pattern, text string) int {
	if pattern == "" || text == "" {
		return 0
	}

	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	if pattern == text {
		return 100
	}

	if len(pattern) > len(text) {
		return 0
	}

	positions := findMatchingPositions(pattern, text)
	if len(positions) == 0 {
		return 0
	}

	score := calculateScore(pattern, text, positions)

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

func MatchMany(pattern string, texts []string, threshold int) []MatchResult {
	results := make([]MatchResult, 0, len(texts))

	for i, text := range texts {
		score := Match(pattern, text)
		if score >= threshold {
			results = append(results, MatchResult{
				Text:  text,
				Score: score,
				Index: i,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func findMatchingPositions(pattern, text string) []int {
	positions := make([]int, 0, len(pattern))
	patternIdx := 0
	textIdx := 0

	patternRunes := []rune(pattern)
	textRunes := []rune(text)

	for patternIdx < len(patternRunes) && textIdx < len(textRunes) {
		if patternRunes[patternIdx] == textRunes[textIdx] {
			positions = append(positions, textIdx)
			patternIdx++
		}
		textIdx++
	}

	if patternIdx < len(patternRunes) {
		return nil
	}

	return positions
}

func calculateScore(pattern, text string, positions []int) int {
	if len(positions) == 0 {
		return 0
	}

	patternLen := len([]rune(pattern))
	textLen := len([]rune(text))

	score := 50.0

	lengthRatio := float64(patternLen) / float64(textLen)
	if lengthRatio == 1.0 {
		score += 30.0
	} else {
		score += lengthRatio * 25.0
	}

	if positions[0] == 0 {
		score += 12.0
	}

	consecutiveCount := countConsecutiveMatches(positions)
	consecutiveRatio := float64(consecutiveCount) / float64(patternLen)
	consecutiveBonus := consecutiveRatio * 20.0
	if patternLen < 3 {
		consecutiveBonus *= 0.6
	} else if patternLen < 5 {
		consecutiveBonus *= 0.8
	}
	score += consecutiveBonus

	scatterCount := patternLen - consecutiveCount
	if scatterCount > 0 {
		scatterPenalty := float64(scatterCount) * 4.0
		score -= scatterPenalty

		if consecutiveCount == 1 {
			score -= 10.0
		}
	}

	avgPosition := calculateAveragePosition(positions)
	positionRatio := avgPosition / float64(textLen)
	earlyBonus := (1.0 - positionRatio) * 10.0
	score += earlyBonus

	if hasCamelCaseMatch(pattern, text, positions) {
		score += 10.0
	}

	if hasWordBoundaryMatch(text, positions) {
		score += 8.0
	}

	if positions[0] == 0 && consecutiveCount == patternLen {
		if patternLen == textLen {
			score += 20.0
		} else if float64(patternLen)/float64(textLen) >= 0.5 {
			score += 10.0
		} else {
			score += 5.0
		}
	}

	if textLen > patternLen {
		extraChars := textLen - patternLen
		penaltyRate := 0.5
		if patternLen < 3 {
			penaltyRate = 1.0
		} else if patternLen < 5 {
			penaltyRate = 0.7
		}
		lengthPenalty := float64(extraChars) * penaltyRate
		score -= lengthPenalty
	}

	return int(score)
}

func countConsecutiveMatches(positions []int) int {
	if len(positions) == 0 {
		return 0
	}

	consecutive := 1
	maxConsecutive := 1

	for i := 1; i < len(positions); i++ {
		if positions[i] == positions[i-1]+1 {
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
		} else {
			consecutive = 1
		}
	}

	return maxConsecutive
}

func calculateAveragePosition(positions []int) float64 {
	if len(positions) == 0 {
		return 0
	}

	sum := 0
	for _, pos := range positions {
		sum += pos
	}

	return float64(sum) / float64(len(positions))
}

func hasCamelCaseMatch(pattern, text string, positions []int) bool {
	textRunes := []rune(text)


	camelCaseCount := 0
	for _, pos := range positions {
		if pos == 0 {
			camelCaseCount++
		} else if pos < len(textRunes) {
			prevChar := textRunes[pos-1]
			if !unicode.IsLetter(prevChar) && !unicode.IsDigit(prevChar) {
				camelCaseCount++
			}
		}
	}

	return float64(camelCaseCount)/float64(len(positions)) > 0.5
}

func hasWordBoundaryMatch(text string, positions []int) bool {
	textRunes := []rune(text)

	boundaryCount := 0
	for _, pos := range positions {
		if pos == 0 {
			boundaryCount++
		} else if pos < len(textRunes) {
			prevChar := textRunes[pos-1]
			if prevChar == ' ' || prevChar == '-' || prevChar == '_' || prevChar == '/' {
				boundaryCount++
			}
		}
	}

	return float64(boundaryCount)/float64(len(positions)) >= 0.3
}
