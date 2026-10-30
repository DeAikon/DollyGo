package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Signing 是解析出的签名配置（仅用于展示与 keystore 存在性校验，不做重签）。
type Signing struct {
	Found          bool
	StoreFile      string
	KeyAlias       string
	StorePassword  string
	KeyPassword    string
	KeystoreExists bool
	Warn           string
}

var (
	reStoreFile     = regexp.MustCompile(`storeFile\s+file\(\s*["']([^"']+)["']\s*\)`)
	reKeyAlias      = regexp.MustCompile(`keyAlias\s+["']([^"']+)["']`)
	reStorePassword = regexp.MustCompile(`storePassword\s+["']([^"']+)["']`)
	reKeyPassword   = regexp.MustCompile(`keyPassword\s+["']([^"']+)["']`)
)

// ParseSigning 从 <root>/app/build.gradle[.kts] 的 signingConfigs.release 解析签名配置。
// 解析失败不阻断主流程（仅返回 Warn），因为渠道打包本身不依赖签名信息。
func ParseSigning(root string) Signing {
	src, path, err := readAppGradle(root)
	if err != nil {
		return Signing{Warn: "read " + path + ": " + err.Error()}
	}

	block := extractBlock(src, "signingConfigs")
	if block == "" {
		return Signing{Warn: "signingConfigs block not found in " + path}
	}
	release := extractBlock(block, "release")
	if release == "" {
		return Signing{Warn: "release signing block not found in " + path}
	}

	s := Signing{
		Found:         true,
		StoreFile:     firstGroup(reStoreFile, release),
		KeyAlias:      firstGroup(reKeyAlias, release),
		StorePassword: firstGroup(reStorePassword, release),
		KeyPassword:   firstGroup(reKeyPassword, release),
	}

	if s.StoreFile == "" && s.KeyAlias == "" {
		return Signing{Warn: "signing fields not parsed from " + path}
	}

	if s.StoreFile != "" {
		p := s.StoreFile
		if !filepath.IsAbs(p) {
			// storeFile 相对 app/ 目录解析（如 file("../jialeSign.jks")）。
			p = filepath.Clean(filepath.Join(root, "app", p))
		}
		s.StoreFile = p
		if _, statErr := os.Stat(p); statErr == nil {
			s.KeystoreExists = true
		}
	}
	return s
}

func readAppGradle(root string) (string, string, error) {
	for _, name := range []string{"build.gradle", "build.gradle.kts"} {
		p := filepath.Join(root, "app", name)
		b, err := os.ReadFile(p)
		if err == nil {
			return string(b), p, nil
		}
	}
	return "", filepath.Join(root, "app", "build.gradle"), os.ErrNotExist
}

// extractBlock 返回 src 中「marker 之后紧跟（可含空白）花括号块」的完整内容（含花括号）。
//
// 仅认 `signingConfigs {` 这类块声明；跳过 `signingConfigs.release` 这类「引用用法」，
// 避免误取其后无关的块（例如 buildTypes 先于 signingConfigs 出现时的场景）。
func extractBlock(src, marker string) string {
	from := 0
	for {
		i := strings.Index(src[from:], marker)
		if i < 0 {
			return ""
		}
		i += from
		k := i + len(marker)
		for k < len(src) && isSpaceByte(src[k]) {
			k++
		}
		if k < len(src) && src[k] == '{' {
			return braceBlock(src, k)
		}
		from = i + len(marker)
	}
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// braceBlock 返回从 open（指向 '{'）开始、括号配对的完整块（含花括号）。
func braceBlock(src string, open int) string {
	depth := 0
	for k := open; k < len(src); k++ {
		switch src[k] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[open : k+1]
			}
		}
	}
	return ""
}

func firstGroup(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}
