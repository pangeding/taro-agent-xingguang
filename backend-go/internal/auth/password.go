package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost 是口令哈希强度。12 在本地单实例场景下约 250ms/次，
// 配合登录限速可接受，且明显高于默认的 10。
const BcryptCost = 12

// DefaultDevUsername / DefaultDevPassword 是 APP_ENV=dev 且 users 表为空时的引导账号。
const (
	DefaultDevUsername = "admin"
	DefaultDevPassword = "123456"
)

// MinPasswordLength 是生产环境下引导口令与新口令的最小长度。
const MinPasswordLength = 12

// weakPasswords 是禁止使用的弱口令表。
// 用途有二：
//  1. 生产环境下校验新设置的口令；
//  2. 启动时逐个用户做 bcrypt 比对，命中即拒绝在生产环境启动——
//     这条是防止「把 dev 库直接搬到 prod」的保险，见技术文档 §3.3 L2。
var weakPasswords = []string{
	"123456", "1234567", "12345678", "123456789", "1234567890",
	"password", "passw0rd", "admin", "admin123", "root",
	"qwerty", "qwerty123", "111111", "000000", "123123",
	"abc123", "iloveyou", "test", "test123", "letmein",
	"changeme", "secret", "tarot", "tarot123",
}

// reservedUsernames 是生产环境禁止用作引导管理员用户名的保留字。
var reservedUsernames = []string{
	"admin", "root", "administrator", "test", "guest", "user",
	"superuser", "sysadmin", "operator",
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9_.-]{3,50}$`)

// ErrWeakPassword 表示口令不满足强度要求。
var ErrWeakPassword = errors.New("口令强度不足")

// NormalizeUsername 规范化用户名。入库与查询前都必须调用，
// 否则 "Admin" 与 "admin" 会同时存在，绕过保留字与唯一性检查。
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// ValidateUsername 校验规范化后的用户名格式。
func ValidateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("用户名只能包含小写字母、数字、下划线、点和连字符，长度 3-50")
	}
	return nil
}

// IsReservedUsername 判断是否为保留用户名（生产环境禁止用于引导管理员）。
func IsReservedUsername(username string) bool {
	u := NormalizeUsername(username)
	for _, r := range reservedUsernames {
		if u == r {
			return true
		}
	}
	return false
}

// IsWeakPassword 判断口令是否命中弱口令表，或长度不足。
func IsWeakPassword(password string) bool {
	return PasswordStrengthError(password, true) != nil
}

// PasswordStrengthError 校验口令强度。
//
// requireLength 为 true 时要求长度 >= MinPasswordLength（用于生产引导口令与新口令）；
// 为 false 时只做「非空 + 非弱口令 + 非常见结构」检查（用于 dev 引导口令，允许 "123456"）。
func PasswordStrengthError(password string, requireLength bool) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("%w：口令不能为空", ErrWeakPassword)
	}
	if requireLength && len([]rune(password)) < MinPasswordLength {
		return fmt.Errorf("%w：口令长度至少 %d 位", ErrWeakPassword, MinPasswordLength)
	}
	if isAllSameRune(password) {
		return fmt.Errorf("%w：口令不能是单一重复字符", ErrWeakPassword)
	}
	for _, w := range weakPasswords {
		if strings.EqualFold(password, w) {
			return fmt.Errorf("%w：口令过于常见", ErrWeakPassword)
		}
	}
	return nil
}

// ValidateBootstrapCredential 校验引导账号的口令（用于生产环境）。
// username 会被用来排除「口令等于用户名」这种情形。
func ValidateBootstrapCredential(username, password string) error {
	if err := PasswordStrengthError(password, true); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(password), NormalizeUsername(username)) {
		return fmt.Errorf("%w：口令不能与用户名相同", ErrWeakPassword)
	}
	return nil
}

// MatchesKnownWeakPassword 判断某个已存的哈希是否对应弱口令表中的口令。
// 用于启动时的存量扫描。bcrypt 比对是常数时间的，无法反查，只能逐条试。
func MatchesKnownWeakPassword(hash string) bool {
	for _, w := range weakPasswords {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(w)) == nil {
			return true
		}
	}
	return false
}

// HashPassword 生成 bcrypt 哈希。
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验口令。哈希非法或口令不符均返回 error。
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// dummyHash 是一个固定的合法 bcrypt 哈希（cost 12，对应一个随机口令）。
// 用户不存在时用它做一次比对，抹平「用户不存在」与「口令错误」的响应时间差，
// 避免把登录接口变成用户名枚举预言机。
//
// 必须是结构合法的 bcrypt 哈希：非法哈希会立即返回解析错误（实测 ~120ns，
// 而合法哈希 ~213ms），反而把「用户是否存在」暴露得更彻底。
const dummyHash = "$2a$12$yysSkZJXOy82qXBtX8c26OWImG2X6O3EtuZ9J0pr53byo7e760jA."

// VerifyDummy 在用户不存在时调用，消耗与真实校验相当的时间。
func VerifyDummy(password string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
}

// isAllSameRune 判断字符串是否由单一重复字符组成（如 "aaaaaa"、"111111"）。
func isAllSameRune(s string) bool {
	runes := []rune(s)
	if len(runes) < 2 {
		return false
	}
	first := runes[0]
	for _, r := range runes[1:] {
		if r != first {
			return false
		}
	}
	return unicode.IsLetter(first) || unicode.IsDigit(first)
}
