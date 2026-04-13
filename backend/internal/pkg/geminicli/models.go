package geminicli

// Model represents a selectable Gemini model for UI/testing purposes.
// Keep JSON fields consistent with existing frontend expectations.
type Model struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func newModel(id, displayName string) Model {
	return Model{
		ID:          id,
		Type:        "model",
		DisplayName: displayName,
		CreatedAt:   "",
	}
}

// DefaultModels is the curated Gemini model list used by the admin UI "test account" flow.
var DefaultModels = []Model{
	newModel("gemini-2.0-flash", "Gemini 2.0 Flash"),
	newModel("gemini-2.5-flash", "Gemini 2.5 Flash"),
	newModel("gemini-2.5-flash-image", "Gemini 2.5 Flash Image"),
	newModel("gemini-2.5-pro", "Gemini 2.5 Pro"),
	newModel("gemini-3-flash-preview", "Gemini 3 Flash Preview"),
	newModel("gemini-3-pro-preview", "Gemini 3 Pro Preview"),
	newModel("gemini-3.1-pro-preview", "Gemini 3.1 Pro Preview"),
	newModel("gemini-3.1-flash-image", "Gemini 3.1 Flash Image"),
}

// VertexDefaultModels keeps Vertex's test-model list separate from AI Studio/OAuth Gemini models.
// The order is text-first so connection tests default to lower-friction text generation while
// still exposing image models when the account supports them.
var VertexDefaultModels = []Model{
	newModel("gemini-3-flash-preview", "Gemini 3 Flash Preview"),
	newModel("gemini-3-pro-preview", "Gemini 3 Pro Preview"),
	newModel("gemini-3.1-pro-preview", "Gemini 3.1 Pro Preview"),
	newModel("gemini-3.1-flash-image-preview", "Gemini 3.1 Flash Image Preview"),
	newModel("gemini-3.1-pro-image-preview", "Gemini 3.1 Pro Image Preview"),
	newModel("gemini-3.1-flash-lite-preview", "Gemini 3.1 Flash Lite Preview"),
}

// DefaultTestModel is the default model to preselect in test flows.
const DefaultTestModel = "gemini-3-flash-preview"

// VertexDefaultTestModel is the default model to preselect for Vertex test flows.
const VertexDefaultTestModel = "gemini-3-flash-preview"
