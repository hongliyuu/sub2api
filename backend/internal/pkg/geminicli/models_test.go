package geminicli

import "testing"

func TestDefaultModels_ContainsImageModels(t *testing.T) {
	t.Parallel()

	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	required := []string{
		"gemini-2.5-flash-image",
		"gemini-3.1-flash-image",
	}

	for _, id := range required {
		if _, ok := byID[id]; !ok {
			t.Fatalf("expected curated Gemini model %q to exist", id)
		}
	}
}

func TestVertexDefaultModels_TextFirstAndIncludesImageModels(t *testing.T) {
	t.Parallel()

	if len(VertexDefaultModels) == 0 {
		t.Fatal("expected Vertex curated model list to be non-empty")
	}
	if VertexDefaultModels[0].ID != VertexDefaultTestModel {
		t.Fatalf("expected first Vertex test model to be %q, got %q", VertexDefaultTestModel, VertexDefaultModels[0].ID)
	}

	byID := make(map[string]Model, len(VertexDefaultModels))
	for _, model := range VertexDefaultModels {
		byID[model.ID] = model
	}

	required := []string{
		"gemini-3-flash-preview",
		"gemini-3-pro-preview",
		"gemini-3.1-pro-preview",
		"gemini-3.1-flash-image-preview",
		"gemini-3.1-pro-image-preview",
		"gemini-3.1-flash-lite-preview",
	}

	for _, id := range required {
		if _, ok := byID[id]; !ok {
			t.Fatalf("expected curated Vertex model %q to exist", id)
		}
	}
}
