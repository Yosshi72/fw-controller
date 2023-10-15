package util

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Result struct {
	StatusUpdated bool
	SpecUpdated   bool
	Requeue       bool
	RequeueAfter  time.Duration
}

func NewResult() Result {
	return Result{
		StatusUpdated: false,
		SpecUpdated:   false,
		Requeue:       false,
		RequeueAfter:  5 * time.Second,
	}
}

func CallerFUNC() string {
	pt, _, _, ok := runtime.Caller(2)
	if !ok {
		return fmt.Sprintf("ERR")
	}
	fn := runtime.FuncForPC(pt).Name()
	fna := strings.Split(fn, "/")
	sn := fna[len(fna)-1]
	return fmt.Sprintf("%s", sn)
}

func CallerLINE() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return fmt.Sprintf("ERR")
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func LINE() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Sprintf("ERR")
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// 順序を気にせず，スライスの要素の比較をする
func MatchElements(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	// スライスの要素をソートする
	sort.Strings(slice1)
	sort.Strings(slice2)

	for i := 0; i < len(slice1); i++ {
		if slice1[i] != slice2[i] {
			return false
		}
	}
	return true
}

// compare s1 with s2. return value is (items in s1 but not in s2, items in s2 but not in s1)
func CmpElements(s1, s2 []string) ([]string, []string) {
	m1 := make(map[string]bool)
	m2 := make(map[string]bool)

	for _, item := range s1 {
		m1[item] = true
	}
	for _, item := range s2 {
		m2[item] = true
	}

	var s1Diff []string
	var s2Diff []string

	// Find items in s1 that are not in s2
	for item := range m1 {
		if !m2[item] {
			s1Diff = append(s1Diff, item)
		}
	}

	// Find items in s2 that are not in s1
	for item := range m2 {
		if !m1[item] {
			s2Diff = append(s2Diff, item)
		}
	}

	return s1Diff, s2Diff
}
