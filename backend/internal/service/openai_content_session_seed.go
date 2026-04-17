package service

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

// contentSessionSeedPrefix prevents collisions between content-derived seeds
// and explicit session IDs like sess_* or compat_cc_*.
const contentSessionSeedPrefix = "compat_cs_"

// deriveOpenAIContentSessionSeed builds a stable fallback seed from fields that
// should stay constant across turns for non-Codex clients.
func deriveOpenAIContentSessionSeed(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var b strings.Builder

	if model := gjson.GetBytes(body, "model").String(); model != "" {
		b.WriteString("model=")
		b.WriteString(model)
	}

	if tools := gjson.GetBytes(body, "tools"); tools.Exists() && tools.IsArray() && tools.Raw != "[]" {
		b.WriteString("|tools=")
		b.WriteString(normalizeCompatSeedJSON(json.RawMessage(tools.Raw)))
	}

	if funcs := gjson.GetBytes(body, "functions"); funcs.Exists() && funcs.IsArray() && funcs.Raw != "[]" {
		b.WriteString("|functions=")
		b.WriteString(normalizeCompatSeedJSON(json.RawMessage(funcs.Raw)))
	}

	if instructions := gjson.GetBytes(body, "instructions").String(); instructions != "" {
		b.WriteString("|instructions=")
		b.WriteString(instructions)
	}

	firstUserCaptured := false
	msgs := gjson.GetBytes(body, "messages")
	if msgs.Exists() && msgs.IsArray() {
		msgs.ForEach(func(_, msg gjson.Result) bool {
			switch msg.Get("role").String() {
			case "system", "developer":
				b.WriteString("|system=")
				if c := msg.Get("content"); c.Exists() {
					b.WriteString(normalizeCompatSeedJSON(json.RawMessage(c.Raw)))
				}
			case "user":
				if !firstUserCaptured {
					b.WriteString("|first_user=")
					if c := msg.Get("content"); c.Exists() {
						b.WriteString(normalizeCompatSeedJSON(json.RawMessage(c.Raw)))
					}
					firstUserCaptured = true
				}
			}
			return true
		})
	} else if input := gjson.GetBytes(body, "input"); input.Exists() {
		if input.Type == gjson.String {
			b.WriteString("|input=")
			b.WriteString(input.String())
		} else if input.IsArray() {
			input.ForEach(func(_, item gjson.Result) bool {
				switch item.Get("role").String() {
				case "system", "developer":
					b.WriteString("|system=")
					if c := item.Get("content"); c.Exists() {
						b.WriteString(normalizeCompatSeedJSON(json.RawMessage(c.Raw)))
					}
				case "user":
					if !firstUserCaptured {
						b.WriteString("|first_user=")
						if c := item.Get("content"); c.Exists() {
							b.WriteString(normalizeCompatSeedJSON(json.RawMessage(c.Raw)))
						}
						firstUserCaptured = true
					}
				}
				if !firstUserCaptured && item.Get("type").String() == "input_text" {
					b.WriteString("|first_user=")
					if text := item.Get("text").String(); text != "" {
						b.WriteString(text)
					}
					firstUserCaptured = true
				}
				return true
			})
		}
	}

	if b.Len() == 0 {
		return ""
	}
	return contentSessionSeedPrefix + b.String()
}
