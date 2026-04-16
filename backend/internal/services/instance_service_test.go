package services

import (
	"testing"

	"clawreef/internal/models"
)

type stubLLMModelRepository struct {
	active []models.LLMModel
}

func (r *stubLLMModelRepository) List() ([]models.LLMModel, error) {
	items := make([]models.LLMModel, len(r.active))
	copy(items, r.active)
	return items, nil
}

func (r *stubLLMModelRepository) ListActive() ([]models.LLMModel, error) {
	items := make([]models.LLMModel, len(r.active))
	copy(items, r.active)
	return items, nil
}

func (r *stubLLMModelRepository) GetByID(id int) (*models.LLMModel, error) {
	return nil, nil
}

func (r *stubLLMModelRepository) GetByDisplayName(displayName string) (*models.LLMModel, error) {
	return nil, nil
}

func (r *stubLLMModelRepository) Save(model *models.LLMModel) error {
	return nil
}

func (r *stubLLMModelRepository) Delete(id int) error {
	return nil
}

func TestBuildGatewayEnvInjectsGatewayModelCatalog(t *testing.T) {
	t.Setenv("CLAWMANAGER_LLM_GATEWAY_BASE_URL", "http://gateway.example/api/v1/gateway/llm")

	token := "igt_test_token"
	service := &instanceService{
		llmModelRepo: &stubLLMModelRepository{
			active: []models.LLMModel{
				{DisplayName: "GPT-4.1"},
				{DisplayName: "Claude 3.7 Sonnet"},
				{DisplayName: "auto"},
				{ProviderModelName: "deepseek-r1"},
			},
		},
	}

	env, err := service.buildGatewayEnv(&models.Instance{
		Type:        "openclaw",
		AccessToken: &token,
	})
	if err != nil {
		t.Fatalf("buildGatewayEnv returned error: %v", err)
	}

	if env["CLAWMANAGER_LLM_BASE_URL"] != "http://gateway.example/api/v1/gateway/llm" {
		t.Fatalf("expected CLAWMANAGER_LLM_BASE_URL to use gateway base URL, got %q", env["CLAWMANAGER_LLM_BASE_URL"])
	}
	if env["CLAWMANAGER_LLM_MODEL"] != `["auto","GPT-4.1","Claude 3.7 Sonnet","deepseek-r1"]` {
		t.Fatalf("expected CLAWMANAGER_LLM_MODEL to contain injected model catalog JSON, got %q", env["CLAWMANAGER_LLM_MODEL"])
	}
	if env["OPENAI_MODEL"] != "auto" {
		t.Fatalf("expected OPENAI_MODEL to remain the default gateway alias, got %q", env["OPENAI_MODEL"])
	}
	if env["CLAWMANAGER_LLM_API_KEY"] != token || env["OPENAI_API_KEY"] != token {
		t.Fatalf("expected gateway token aliases to be preserved")
	}
}

func TestResolveGatewayModelInjectionRequiresActiveModels(t *testing.T) {
	service := &instanceService{
		llmModelRepo: &stubLLMModelRepository{},
	}

	injection, err := service.resolveGatewayModelInjection()
	if err == nil {
		t.Fatalf("expected resolveGatewayModelInjection to fail when no active models exist, got %#v", injection)
	}
}
