package main

import (
	"strings"
	"unicode/utf8"
)

type Rank struct {
	// Source is used as the source for matching.
	Source string

	// Target is the word matched against.
	Target string

	// Distance is the Levenshtein distance between Source and Target.
	Distance int

	// Location of Target in original list
	OriginalIndex int
}

type Ranks []Rank

func (r Ranks) Len() int {
	return len(r)
}

func (r Ranks) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Ranks) Less(i, j int) bool {
	return r[i].Distance < r[j].Distance
}

func RankFind(source string, targets []string) Ranks {
	return rankFind(source, targets)
}

func rankFind(source string, targets []string) Ranks {

	var r Ranks

	for index, target := range targets {
		if matchTransformed(source, target) {
			distance := LevenshteinDistance(source, target)
			r = append(r, Rank{source, target, distance, index})
		}
	}
	return r
}

func matchTransformed(source, target string) bool {
	source = strings.ToLower(source)
	target = strings.ToLower(target)
	lenDiff := len(target) - len(source)

	if lenDiff < 0 {
		return false
	}

	if lenDiff == 0 && source == target {
		return true
	}

Outer:
	for _, r1 := range source {
		for i, r2 := range target {
			if r1 == r2 {
				target = target[i+utf8.RuneLen(r2):]
				continue Outer
			}
		}
		return false
	}

	return true
}

func LevenshteinDistance(s, t string) int {
	r1, r2 := []rune(s), []rune(t)
	column := make([]int, 1, 64)

	for y := 1; y <= len(r1); y++ {
		column = append(column, y)
	}

	for x := 1; x <= len(r2); x++ {
		column[0] = x

		for y, lastDiag := 1, x-1; y <= len(r1); y++ {
			oldDiag := column[y]
			cost := 0
			if r1[y-1] != r2[x-1] {
				cost = 1
			}
			column[y] = min(column[y]+1, column[y-1]+1, lastDiag+cost)
			lastDiag = oldDiag
		}
	}

	return column[len(r1)]
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min(a, b, c int) int {
	return min2(min2(a, b), c)
}
