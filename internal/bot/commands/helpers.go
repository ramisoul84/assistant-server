package commands

import "strings"

// escapeMarkdown escapes special characters for Telegram MarkdownV2.
var mdReplacer = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
	"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
	">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
	".", "\\.", "!", "\\!",
)

func escapeMarkdown(s string) string {
	return mdReplacer.Replace(s)
}
