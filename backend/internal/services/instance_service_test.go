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
	for _, instanceType := range []string{"openclaw", "hermes"} {
		t.Run(instanceType, func(t *testing.T) {
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
				Type:        instanceType,
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
		})
	}
}

func TestBuildGatewayEnvSkipsUnmanagedRuntime(t *testing.T) {
	token := "igt_test_token"
	service := &instanceService{}

	env, err := service.buildGatewayEnv(&models.Instance{
		Type:        "ubuntu",
		AccessToken: &token,
	})
	if err != nil {
		t.Fatalf("buildGatewayEnv returned error: %v", err)
	}
	if len(env) != 0 {
		t.Fatalf("expected unmanaged runtime to receive no gateway env, got %#v", env)
	}
}

func TestBuildAgentEnvInjectsHermesAgentConfig(t *testing.T) {
	t.Setenv("CLAWMANAGER_AGENT_CONTROL_BASE_URL", "http://agent-control.example")

	token := "agt_boot_test_token"
	service := &instanceService{}

	env, err := service.buildAgentEnv(&models.Instance{
		ID:                  24,
		Type:                "hermes",
		DiskGB:              20,
		AgentBootstrapToken: &token,
	})
	if err != nil {
		t.Fatalf("buildAgentEnv returned error: %v", err)
	}

	if env["CLAWMANAGER_AGENT_ENABLED"] != "true" {
		t.Fatalf("expected Hermes agent to be enabled")
	}
	if env["CLAWMANAGER_AGENT_BASE_URL"] != "http://agent-control.example" {
		t.Fatalf("expected Hermes agent base URL to be injected, got %q", env["CLAWMANAGER_AGENT_BASE_URL"])
	}
	if env["CLAWMANAGER_AGENT_BOOTSTRAP_TOKEN"] != token {
		t.Fatalf("expected Hermes agent bootstrap token to be injected")
	}
	if env["CLAWMANAGER_AGENT_INSTANCE_ID"] != "24" {
		t.Fatalf("expected Hermes instance id to be injected, got %q", env["CLAWMANAGER_AGENT_INSTANCE_ID"])
	}
	if env["CLAWMANAGER_AGENT_PERSISTENT_DIR"] != "/config/.hermes" {
		t.Fatalf("expected Hermes persistent dir /config/.hermes, got %q", env["CLAWMANAGER_AGENT_PERSISTENT_DIR"])
	}
	if env["CLAWMANAGER_AGENT_DISK_LIMIT_BYTES"] != "21474836480" {
		t.Fatalf("expected Hermes disk limit bytes to be injected, got %q", env["CLAWMANAGER_AGENT_DISK_LIMIT_BYTES"])
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
