package telegram

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"
)

// tokenFooterSuffix matches the EP-015 trailing token line (plain text, end-anchored).
var tokenFooterSuffix = regexp.MustCompile(`\nTokens \d+ \(in: \d+ / out: \d+\)\z`)

// SplitTokenFooterSuffix splits a combined handler reply into Markdown body and optional token footer line (without leading newline).
func SplitTokenFooterSuffix(s string) (body, footerLine string) {
	s = strings.TrimRight(s, "\r\n")
	if s == "" {
		return "", ""
	}
	loc := tokenFooterSuffix.FindStringIndex(s)
	if loc == nil {
		return s, ""
	}
	body = strings.TrimRight(s[:loc[0]], "\r\n")
	footerLine = strings.TrimPrefix(s[loc[0]:loc[1]], "\n")
	return body, footerLine
}

// telegramBotAPIMaxMessageRunes is the maximum text length per sendMessage (Telegram Bot API).
const telegramBotAPIMaxMessageRunes = 4096

func fitsTelegramOutboundHTML(source string) bool {
	html := MarkdownToTelegramHTML(source)
	return utf8.RuneCountInString(html) <= telegramBotAPIMaxMessageRunes
}

// splitTelegramOutboundSource splits Markdown-style assistant text so each piece is within
// Telegram's message length limit after MarkdownToTelegramHTML conversion.
func splitTelegramOutboundSource(source string) []string {
	source = strings.TrimRight(source, "\r\n")
	if source == "" {
		return nil
	}
	if fitsTelegramOutboundHTML(source) {
		return []string{source}
	}
	paras := strings.Split(source, "\n\n")
	var chunks []string
	var cur strings.Builder
	flushCur := func() {
		if cur.Len() == 0 {
			return
		}
		chunks = append(chunks, cur.String())
		cur.Reset()
	}
	for _, p := range paras {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if cur.Len() > 0 {
			combined := cur.String() + "\n\n" + p
			if fitsTelegramOutboundHTML(combined) {
				cur.Reset()
				cur.WriteString(combined)
				continue
			}
			flushCur()
		}
		if fitsTelegramOutboundHTML(p) {
			if cur.Len() > 0 {
				cur.WriteString("\n\n")
			}
			cur.WriteString(p)
		} else {
			chunks = append(chunks, splitTelegramOutboundOversized(p)...)
		}
	}
	flushCur()
	return chunks
}

func splitTelegramOutboundOversized(p string) []string {
	if fitsTelegramOutboundHTML(p) {
		return []string{p}
	}
	r := []rune(p)
	if len(r) <= 1 {
		return []string{p}
	}
	mid := len(r) / 2
	left := strings.TrimRight(string(r[:mid]), "\r\n")
	right := strings.TrimLeft(string(r[mid:]), "\r\n")
	var out []string
	if left != "" {
		out = append(out, splitTelegramOutboundOversized(left)...)
	}
	if right != "" {
		out = append(out, splitTelegramOutboundOversized(right)...)
	}
	if len(out) == 0 {
		return []string{p}
	}
	return out
}

// sendLongOutboundText sends source as one or more Telegram messages, each within the API length limit.
// When source ends with an EP-015 token footer line, the footer is applied only to the last outbound chunk.
func sendLongOutboundText(ctx context.Context, tg telegramOutbound, chatID int64, source string) error {
	body, foot := SplitTokenFooterSuffix(source)
	chunks := splitTelegramOutboundSource(body)
	if len(chunks) == 0 {
		return nil
	}
	if foot != "" {
		last := len(chunks) - 1
		combined := chunks[last] + "\n" + foot
		if fitsTelegramOutboundHTML(combined) {
			chunks[last] = combined
		} else {
			chunks = append(chunks, foot)
		}
	}
	for _, chunk := range chunks {
		if err := sendOutboundText(ctx, tg, chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}
