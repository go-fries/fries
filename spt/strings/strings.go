package strings

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const (
	// randomLetters is the letters used in Random.
	randomLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// uuidPattern is the pattern used in IsUuid.
	uuidPattern = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
)

// Is returns true if the value matches the pattern.
// The pattern can contain the wildcard character *.
//
// Example:
//
//	Is("*.example.com", "www.example.com") // true
//	Is("*.example.com", "example.com") // false
func Is(pattern, value string) bool {
	if pattern == value {
		return true
	}

	pattern = strings.ReplaceAll(pattern, "*", ".*")

	match, err := regexp.Match(pattern, []byte(value))
	if err != nil {
		return false
	}

	return match
}

// InSlice checks if a string is in a string slice.
//
// Example:
//
//	InSlice([]string{"1", "2"}, "1") // true
func InSlice(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}

	return false
}

// MD5 returns the md5 hash of a string.
//
// Example:
//
//	MD5("abc") // 900150983cd24fb0d6963f7d28e17f72
func MD5(s string) string {
	sm := md5.Sum([]byte(s))

	return fmt.Sprintf("%x", sm)
}

// SHA1 returns the sha1 hash of a string.
//
// Example:
//
//	SHA1("abc") // a9993e364706816aba3e25717850c26c9cd0d89d
func SHA1(s string) string {
	sm := sha1.Sum([]byte(s))

	return fmt.Sprintf("%x", sm)
}

// Reverse returns a string with its characters in reverse order.
//
// Example:
//
//	Reverse("abc") // "cba"
func Reverse(s string) string {
	var reversed string

	for _, v := range s {
		reversed = string(v) + reversed
	}

	return reversed
}

// Replace replaces all occurrences of one substring with another.
//
// Example:
//
//	Replace("aabbcc", "a", "b") // "bbbbcc"
func Replace(s, from, to string) string {
	return strings.NewReplacer(from, to).Replace(s)
}

// Shuffle returns a string with its characters in random order.
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

// Random returns a random string with the specified length.
//
// Example:
//
//	Random(10) // "qujrlkhyqr"
func Random(length int) string {
	letters := []rune(randomLetters)
	lettersLength := len(letters)

	b := make([]rune, length)

	for i := range b {
		b[i] = letters[rand.Intn(lettersLength)]
	}

	return string(b)
}

// Len returns the length of a string.
// Support chinese characters.
//
// Example:
//
//	Len("abc") // 3
//	Len("张三李四") // 4
func Len(s string) int {
	return len([]rune(s))
}

// IsUUID returns true if the string is a valid UUID.
func IsUUID(str string) bool {
	match, err := regexp.MatchString(uuidPattern, str)

	return err == nil && match
}

// UUID generate uuid string
func UUID() string {
	return uuid.New().String()
}

// After Return the remainder of a string after a given value.
// Support chinese characters.
//
// Example:
//
//	After("Hello, World!", ",") //  World!
//	After("张三李四", "三") // 李四
func After(subject, search string) string {
	if search == "" {
		return subject
	}
	index := strings.Index(subject, search)
	if index == -1 {
		return subject
	}
	return subject[index+len(search):]
}

// Before Get the portion of a string before the first occurrence of a given value.
// Support chinese characters.
//
// Example:
//
//	After("Hello, World!", ",") //  Hello
//	After("张三李四", "李") // 张三
func Before(subject, search string) string {
	if search == "" {
		return subject
	}
	index := strings.Index(subject, search)
	if index == -1 {
		return subject
	}
	return subject[:index]
}

// SubstrCount Returns the number of substring occurrences.
//
// Example:
//
//	SubstrCount("babababbaaba", "a", 0, 10) //  5
//	SubstrCount("121212312", "1", 1, 5) // 2
func SubstrCount(haystack, needle string, offset int, length ...int) int {
	if offset < 0 || offset >= len(haystack) {
		return 0
	}

	var end int
	if len(length) > 0 {
		end = offset + length[0]
		if end > len(haystack) {
			end = len(haystack)
		}
	} else {
		end = len(haystack)
	}

	substr := haystack[offset:end]

	return strings.Count(substr, needle)
}
