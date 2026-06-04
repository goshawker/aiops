package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// dangerousPatterns contains shell commands/patterns that are blocked.
var dangerousPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs",
	"dd if=",
	"> /dev/sd",
	"chmod 777 /",
	"chown root",
	":(){ :|:& };:", // fork bomb
	"wget | sh",
	"curl | sh",
	"curl | bash",
	"nc -e",
	"ncat -e",
	"/etc/shadow",
	"/etc/passwd",
	"eval ",
	"$(",
	"`",
}

// urlPattern validates HTTP/HTTPS URLs.
var urlPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+$`)

// ValidateShellCommand checks for dangerous shell command patterns.
func ValidateShellCommand(content string) error {
	contentLower := strings.ToLower(strings.TrimSpace(content))

	if contentLower == "" {
		return fmt.Errorf("命令内容不能为空")
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return fmt.Errorf("命令包含危险模式: %s", pattern)
		}
	}

	// Block commands that try to write to system directories
	if strings.Contains(contentLower, "> /etc/") || strings.Contains(contentLower, "> /usr/") || strings.Contains(contentLower, "> /var/") {
		return fmt.Errorf("命令尝试写入系统目录")
	}

	// Block command chaining
	if strings.Contains(content, ";") || strings.Contains(content, "&&") || strings.Contains(content, "||") {
		return fmt.Errorf("命令包含链式操作符")
	}

	return nil
}

// ValidateHTTPURL checks if the URL is a valid HTTP/HTTPS URL.
func ValidateHTTPURL(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("URL 不能为空")
	}
	if !urlPattern.MatchString(content) {
		return fmt.Errorf("无效的 HTTP/HTTPS URL")
	}
	return nil
}
