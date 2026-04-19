package openai

import "testing"

func TestDefaultModelIDs_GPT54ProFollowsGPT54(t *testing.T) {
	ids := DefaultModelIDs()

	indexOf := func(target string) int {
		for i, id := range ids {
			if id == target {
				return i
			}
		}
		return -1
	}

	idx54 := indexOf("gpt-5.4")
	idx54Pro := indexOf("gpt-5.4-pro")
	idx54Mini := indexOf("gpt-5.4-mini")

	if idx54 == -1 || idx54Pro == -1 || idx54Mini == -1 {
		t.Fatalf("expected gpt-5.4 family IDs in defaults, got: %v", ids)
	}

	if idx54 >= idx54Pro || idx54Pro >= idx54Mini {
		t.Fatalf("expected gpt-5.4 < gpt-5.4-pro < gpt-5.4-mini, got indexes: gpt-5.4=%d gpt-5.4-pro=%d gpt-5.4-mini=%d", idx54, idx54Pro, idx54Mini)
	}
}

func TestDefaultModels_GPT54ProDisplayName(t *testing.T) {
	for _, model := range DefaultModels {
		if model.ID != "gpt-5.4-pro" {
			continue
		}

		if model.DisplayName != "GPT-5.4 Pro" {
			t.Fatalf("DisplayName = %q, want %q", model.DisplayName, "GPT-5.4 Pro")
		}

		return
	}

	t.Fatal("gpt-5.4-pro not found in DefaultModels")
}
