package pake

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// NeedsASCIIPackageName reports whether name is unsafe for WiX / artifact filenames.
func NeedsASCIIPackageName(name string) bool {
	for _, r := range strings.TrimSpace(name) {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

// ASCIIPackageName turns a display name into an ASCII package / file name.
// Chinese runs become Title-cased pinyin; other letters/digits are kept.
func ASCIIPackageName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "App"
	}
	if !NeedsASCIIPackageName(name) {
		return sanitizeASCIIName(name)
	}

	args := pinyin.NewArgs()
	args.Style = pinyin.Normal

	var b strings.Builder
	for _, r := range name {
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-' || r == '.' || r == '·':
			b.WriteByte('-')
		case unicode.Is(unicode.Han, r):
			for _, syl := range pinyin.SinglePinyin(r, args) {
				b.WriteString(titleASCII(syl))
			}
		}
	}

	out := sanitizeASCIIName(b.String())
	if out == "" {
		return "App"
	}
	return out
}

// NormalizePackageIdentity rewrites Name to an ASCII package name when needed.
// The original display name is preserved in Title if Title was empty.
// Returns a short user-facing note when a rewrite happened (empty otherwise).
func NormalizePackageIdentity(o *Options) string {
	if o == nil {
		return ""
	}
	original := strings.TrimSpace(o.Name)
	if original == "" || !NeedsASCIIPackageName(original) {
		return ""
	}
	pkg := ASCIIPackageName(original)
	if pkg == "" || pkg == original {
		return ""
	}
	if strings.TrimSpace(o.Title) == "" {
		o.Title = original
	}
	o.Name = pkg
	return "应用名含非 ASCII，打包名使用 " + pkg + "（窗口标题：" + strings.TrimSpace(o.Title) + "）"
}

func titleASCII(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func sanitizeASCIIName(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == '.' || r == ' ':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "App"
	}
	return out
}
