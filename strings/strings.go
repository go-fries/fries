package strings

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	// randomLetters is the letters used in Random.
	randomLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var uuidRegex = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Is reports whether value matches pattern.
// The pattern can contain the wildcard character *, which matches any sequence
// of characters.
//
// Example:
//
//	Is("*.example.com", "www.example.com") // true
//	Is("*.example.com", "example.com") // false
func Is(pattern, value string) bool {
	if pattern == value {
		return true
	}

	patternIndex, valueIndex := 0, 0
	starIndex, matchIndex := -1, 0

	for valueIndex < len(value) {
		if patternIndex < len(pattern) && pattern[patternIndex] == '*' {
			starIndex = patternIndex
			matchIndex = valueIndex
			patternIndex++
			continue
		}

		if patternIndex < len(pattern) && pattern[patternIndex] == value[valueIndex] {
			patternIndex++
			valueIndex++
			continue
		}

		if starIndex == -1 {
			return false
		}

		patternIndex = starIndex + 1
		matchIndex++
		valueIndex = matchIndex
	}

	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}

	return patternIndex == len(pattern)
}

// InSlice reports whether s exists in slice.
//
// Example:
//
//	InSlice([]string{"1", "2"}, "1") // true
func InSlice(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

// MD5 returns the MD5 hash of s as a lowercase hexadecimal string.
//
// Example:
//
//	MD5("abc") // 900150983cd24fb0d6963f7d28e17f72
func MD5(s string) string {
	sm := md5.Sum([]byte(s))

	return hex.EncodeToString(sm[:])
}

// SHA1 returns the SHA-1 hash of s as a lowercase hexadecimal string.
//
// Example:
//
//	SHA1("abc") // a9993e364706816aba3e25717850c26c9cd0d89d
func SHA1(s string) string {
	sm := sha1.Sum([]byte(s))

	return hex.EncodeToString(sm[:])
}

// Reverse returns s with its Unicode code points in reverse order.
//
// Example:
//
//	Reverse("abc") // "cba"
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Replace replaces all occurrences of from in s with to.
//
// Example:
//
//	Replace("aabbcc", "a", "b") // "bbbbcc"
func Replace(s, from, to string) string {
	return strings.NewReplacer(from, to).Replace(s)
}

// Shuffle returns s with its characters in random order.
//
// Example:
//
//	Shuffle("abc") // "bca"
func Shuffle(s string) string {
	ss := strings.Split(s, "")

	rand.Shuffle(len(ss), func(i, j int) {
		ss[i], ss[j] = ss[j], ss[i]
	})

	return strings.Join(ss, "")
}

// Random returns a random alphabetic string with the specified length.
//
// Random uses math/rand/v2 and is not suitable for security-sensitive values.
//
// Example:
//
//	Random(10) // "qujrlkhyqr"
func Random(length int) string {
	if length <= 0 {
		return ""
	}

	letters := []rune(randomLetters)
	lettersLength := len(letters)

	b := make([]rune, length)

	for i := range b {
		b[i] = letters[rand.IntN(lettersLength)]
	}

	return string(b)
}

// Len returns the number of Unicode code points in s.
//
// Example:
//
//	Len("abc") // 3
//	Len("张三李四") // 4
func Len(s string) int {
	return utf8.RuneCountInString(s)
}

// IsUUID returns true if the string is a valid UUID.
func IsUUID(str string) bool {
	return uuidRegex.MatchString(str)
}

// UUID returns a new UUID string.
func UUID() string {
	return uuid.New().String()
}

// After returns the remainder of subject after the first occurrence of search.
//
// If search is empty or not found, After returns subject unchanged.
//
// Example:
//
//	After("Hello, World!", ",") //  World!
//	After("张三李四", "三") // 李四
func After(subject, search string) string {
	if search == "" {
		return subject
	}
	_, after, ok := strings.Cut(subject, search)
	if !ok {
		return subject
	}
	return after
}

// Before returns the portion of subject before the first occurrence of search.
//
// If search is empty or not found, Before returns subject unchanged.
//
// Example:
//
//	Before("Hello, World!", ",") // Hello
//	Before("张三李四", "李") // 张三
func Before(subject, search string) string {
	if search == "" {
		return subject
	}
	before, _, ok := strings.Cut(subject, search)
	if !ok {
		return subject
	}
	return before
}

// SubstrCount returns the number of non-overlapping instances of needle in
// haystack.
//
// The offset and optional length arguments are byte indexes.
//
// Example:
//
//	SubstrCount("babababbaaba", "a", 0, 10) //  5
//	SubstrCount("121212312", "1", 1, 5) // 2
func SubstrCount(haystack, needle string, offset int, lengths ...int) int {
	if offset < 0 || offset >= len(haystack) {
		return 0
	}

	var end int
	if len(lengths) > 0 {
		if lengths[0] < 0 {
			return 0
		}
		end = min(offset+lengths[0], len(haystack))
	} else {
		end = len(haystack)
	}

	substr := haystack[offset:end]

	return strings.Count(substr, needle)
}
