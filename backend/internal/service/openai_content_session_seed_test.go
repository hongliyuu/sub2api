package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveOpenAIContentSessionSeed_EmptyInputs(t *testing.T) {
	require.Empty(t, deriveOpenAIContentSessionSeed(nil))
	require.Empty(t, deriveOpenAIContentSessionSeed([]byte{}))
	require.Empty(t, deriveOpenAIContentSessionSeed([]byte(`{}`)))
}

func TestDeriveOpenAIContentSessionSeed_StableAcrossLaterTurns(t *testing.T) {
	turn1 := []byte(`{"model":"gpt-5.4","messages":[{"role":"system","content":"You are helpful."},{"role":"user","content":"Hello"}]}`)
	turn2 := []byte(`{"model":"gpt-5.4","messages":[{"role":"system","content":"You are helpful."},{"role":"user","content":"Hello"},{"role":"assistant","content":"Hi"},{"role":"user","content":"How are you?"}]}`)
	require.Equal(t, deriveOpenAIContentSessionSeed(turn1), deriveOpenAIContentSessionSeed(turn2))
}

func TestDeriveOpenAIContentSessionSeed_UsesToolsAndFunctions(t *testing.T) {
	withTools := []byte(`{"model":"gpt-5.4","tools":[{"type":"function","function":{"name":"get_weather"}}],"messages":[{"role":"user","content":"Hello"}]}`)
	withoutTools := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"Hello"}]}`)
	require.NotEqual(t, deriveOpenAIContentSessionSeed(withTools), deriveOpenAIContentSessionSeed(withoutTools))

	withFunctions := []byte(`{"model":"gpt-5.4","functions":[{"name":"get_weather","parameters":{}}],"messages":[{"role":"user","content":"Hello"}]}`)
	require.Contains(t, deriveOpenAIContentSessionSeed(withFunctions), "|functions=")
}

func TestDeriveOpenAIContentSessionSeed_ResponsesInputAndInstructions(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","instructions":"You are a coding assistant.","input":"Write hello world"}`)
	seed := deriveOpenAIContentSessionSeed(body)
	require.Contains(t, seed, "|instructions=You are a coding assistant.")
	require.Contains(t, seed, "|input=Write hello world")
}

func TestDeriveOpenAIContentSessionSeed_JSONCanonicalisation(t *testing.T) {
	compact := []byte(`{"model":"gpt-5.4","tools":[{"type":"function","function":{"name":"get_weather","description":"Get weather"}}],"messages":[{"role":"user","content":"Hi"}]}`)
	spaced := []byte(`{"model":"gpt-5.4","tools":[{"type":"function","function":{"description":"Get weather","name":"get_weather"}}],"messages":[{"role":"user","content":"Hi"}]}`)
	require.Equal(t, deriveOpenAIContentSessionSeed(compact), deriveOpenAIContentSessionSeed(spaced))
}
