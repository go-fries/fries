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

// ReplaceFirst replaces the first occurrence of from in s with to.
//
// If from is empty or not found, ReplaceFirst returns s unchanged.
//
// Example:
//
//	ReplaceFirst("the quick brown fox", "the", "a") // "a quick brown fox"
func ReplaceFirst(s, from, to string) string {
	if from == "" {
		return s
	}
	return strings.Replace(s, from, to, 1)
}

// ReplaceLast replaces the last occurrence of from in s with to.
//
// If from is empty or not found, ReplaceLast returns s unchanged.
//
// Example:
//
//	ReplaceLast("the quick brown fox jumps over the lazy dog", "the", "a") // "the quick brown fox jumps over a lazy dog"
func ReplaceLast(s, from, to string) string {
	if from == "" {
		return s
	}

	index := strings.LastIndex(s, from)
	if index == -1 {
		return s
	}

	return s[:index] + to + s[index+len(from):]
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

// EnsurePrefix returns s with prefix prepended exactly once.
//
// If prefix is empty or s already starts with prefix, EnsurePrefix returns s
// unchanged.
//
// Example:
//
//	EnsurePrefix("api.example.com", "https://") // "https://api.example.com"
func EnsurePrefix(s, prefix string) string {
	if prefix == "" || strings.HasPrefix(s, prefix) {
		return s
	}
	return prefix + s
}

// EnsureSuffix returns s with suffix appended exactly once.
//
// If suffix is empty or s already ends with suffix, EnsureSuffix returns s
// unchanged.
//
// Example:
//
//	EnsureSuffix("api.example.com", "/") // "api.example.com/"
func EnsureSuffix(s, suffix string) string {
	if suffix == "" || strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
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

// AfterLast returns the remainder of subject after the last occurrence of search.
//
// If search is empty or not found, AfterLast returns subject unchanged.
//
// Example:
//
//	AfterLast("path/to/file.go", "/") // "file.go"
func AfterLast(subject, search string) string {
	if search == "" {
		return subject
	}

	index := strings.LastIndex(subject, search)
	if index == -1 {
		return subject
	}

	return subject[index+len(search):]
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

// BeforeLast returns the portion of subject before the last occurrence of search.
//
// If search is empty or not found, BeforeLast returns subject unchanged.
//
// Example:
//
//	BeforeLast("path/to/file.go", "/") // "path/to"
func BeforeLast(subject, search string) string {
	if search == "" {
		return subject
	}

	index := strings.LastIndex(subject, search)
	if index == -1 {
		return subject
	}

	return subject[:index]
}

// Between returns the portion of subject between the first occurrence of from
// and the last following occurrence of to.
//
// If from or to is empty, or either boundary is not found, Between returns
// subject unchanged.
//
// Example:
//
//	Between("[a] [b]", "[", "]") // "a] [b"
func Between(subject, from, to string) string {
	return between(subject, from, to, true)
}

// BetweenFirst returns the portion of subject between the first occurrence of
// from and the first following occurrence of to.
//
// If from or to is empty, or either boundary is not found, BetweenFirst returns
// subject unchanged.
//
// Example:
//
//	BetweenFirst("[a] [b]", "[", "]") // "a"
func BetweenFirst(subject, from, to string) string {
	return between(subject, from, to, false)
}

func between(subject, from, to string, last bool) string {
	if from == "" || to == "" {
		return subject
	}

	start := strings.Index(subject, from)
	if start == -1 {
		return subject
	}

	offset := start + len(from)
	after := subject[offset:]

	end := strings.Index(after, to)
	if last {
		end = strings.LastIndex(after, to)
	}
	if end == -1 {
		return subject
	}

	return after[:end]
}

// NormalizeSpace returns s with leading and trailing whitespace removed and
// each internal run of whitespace replaced by a single space.
//
// Example:
//
//	NormalizeSpace("  hello \n world  ") // "hello world"
func NormalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
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
