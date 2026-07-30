package services

import (
	"encoding/json"
	"strings"
	"testing"

	"clawreef/internal/models"
)

func TestValidateOpenClawCronConfigAcceptsValidVariants(t *testing.T) {
	t.Parallel()

	cases := []string{
		`{"name":"daily","schedule":{"kind":"cron","expr":"0 9 * * *","tz":"Asia/Shanghai"},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"brief me"},"delivery":{"mode":"announce","channel":"last","bestEffort":true}}`,
		`{"name":"once","schedule":{"kind":"at","at":"2026-07-23T10:00:00Z"},"sessionTarget":"main","wakeMode":"next-heartbeat","payload":{"kind":"systemEvent","text":"ping"},"delivery":{"mode":"none"}}`,
		`{"name":"every","enabled":true,"schedule":{"kind":"every","everyMs":60000},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"tick"},"delivery":{"mode":"webhook","to":"https://example.com/hook"}}`,
	}
	for _, raw := range cases {
		if err := ValidateOpenClawCronConfig(json.RawMessage(raw)); err != nil {
			t.Fatalf("ValidateOpenClawCronConfig(%s) unexpected error: %v", raw, err)
		}
	}
}

func TestValidateOpenClawCronConfigRejectsInvalidVariants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "missing name",
			raw:     `{"name":"","schedule":{"kind":"cron","expr":"0 9 * * *"},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"x"}}`,
			wantErr: "name is required",
		},
		{
			name:    "main requires systemEvent",
			raw:     `{"name":"bad","schedule":{"kind":"cron","expr":"0 9 * * *"},"sessionTarget":"main","wakeMode":"now","payload":{"kind":"agentTurn","message":"x"}}`,
			wantErr: "requires sessionTarget=isolated",
		},
		{
			name:    "isolated requires agentTurn",
			raw:     `{"name":"bad","schedule":{"kind":"cron","expr":"0 9 * * *"},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"systemEvent","text":"x"}}`,
			wantErr: "requires sessionTarget=main",
		},
		{
			name:    "webhook missing to",
			raw:     `{"name":"hook","schedule":{"kind":"cron","expr":"0 9 * * *"},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"x"},"delivery":{"mode":"webhook"}}`,
			wantErr: "delivery.to is required",
		},
		{
			name:    "everyMs must be positive",
			raw:     `{"name":"every","schedule":{"kind":"every","everyMs":0},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"x"}}`,
			wantErr: "everyMs must be > 0",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateOpenClawCronConfig(json.RawMessage(tc.raw))
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateResourceRejectsWrongScheduledTaskFormat(t *testing.T) {
	t.Parallel()

	svc := &openClawConfigService{}
	err := svc.ValidateResource(UpsertOpenClawConfigResourceRequest{
		ResourceType: OpenClawConfigResourceTypeScheduledTask,
		ResourceKey:  "daily-brief",
		Name:         "Daily Brief",
		Enabled:      true,
		Content: json.RawMessage(`{
			"schemaVersion":1,
			"kind":"scheduled_task",
			"format":"task/default@v1",
			"dependsOn":[],
			"config":{
				"name":"daily",
				"schedule":{"kind":"cron","expr":"0 9 * * *"},
				"sessionTarget":"isolated",
				"wakeMode":"now",
				"payload":{"kind":"agentTurn","message":"brief"}
			}
		}`),
	})
	if err == nil || !strings.Contains(err.Error(), OpenClawScheduledTaskFormat) {
		t.Fatalf("expected format error, got %v", err)
	}
}

func TestValidateResourceAcceptsScheduledTaskOpenClawCronFormat(t *testing.T) {
	t.Parallel()

	svc := &openClawConfigService{}
	err := svc.ValidateResource(UpsertOpenClawConfigResourceRequest{
		ResourceType: OpenClawConfigResourceTypeScheduledTask,
		ResourceKey:  "daily-brief",
		Name:         "Daily Brief",
		Enabled:      true,
		Content: json.RawMessage(`{
			"schemaVersion":1,
			"kind":"scheduled_task",
			"format":"task/openclaw-cron@v1",
			"dependsOn":[],
			"config":{
				"name":"daily",
				"schedule":{"kind":"cron","expr":"0 9 * * *","tz":"Asia/Shanghai"},
				"sessionTarget":"isolated",
				"wakeMode":"now",
				"payload":{"kind":"agentTurn","message":"brief"},
				"delivery":{"mode":"announce","channel":"last"}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("ValidateResource() unexpected error: %v", err)
	}
}

func TestRenderCompiledOpenClawPayloadIncludesScheduledTasks(t *testing.T) {
	t.Parallel()

	content := `{"schemaVersion":1,"kind":"scheduled_task","format":"task/openclaw-cron@v1","dependsOn":[],"config":{"name":"daily","schedule":{"kind":"cron","expr":"0 9 * * *"},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"brief"},"delivery":{"mode":"webhook","to":"https://example.com/h"}}}`
	resources := []compiledOpenClawResource{
		{
			model: models.OpenClawConfigResource{
				ID:           42,
				ResourceType: OpenClawConfigResourceTypeScheduledTask,
				ResourceKey:  "daily-brief",
				Name:         "Daily Brief",
				Version:      1,
				ContentJSON:  content,
			},
			tags: []string{"ops"},
			envelope: OpenClawConfigEnvelope{
				SchemaVersion: 1,
				Kind:          "scheduled_task",
				Format:        OpenClawScheduledTaskFormat,
				Config:        json.RawMessage(`{"name":"daily","schedule":{"kind":"cron","expr":"0 9 * * *"},"sessionTarget":"isolated","wakeMode":"now","payload":{"kind":"agentTurn","message":"brief"},"delivery":{"mode":"webhook","to":"https://example.com/h"}}`),
			},
		},
	}

	env, _, _, _, err := renderCompiledOpenClawPayload(OpenClawConfigPlan{Mode: OpenClawConfigPlanModeManual}, nil, resources)
	if err != nil {
		t.Fatalf("renderCompiledOpenClawPayload() error: %v", err)
	}
	raw, ok := env[OpenClawScheduledTasksEnv]
	if !ok || strings.TrimSpace(raw) == "" {
		t.Fatalf("expected %s in rendered env", OpenClawScheduledTasksEnv)
	}
	if !strings.Contains(raw, `"key":"daily-brief"`) {
		t.Fatalf("scheduled tasks payload missing resource key: %s", raw)
	}
	if !strings.Contains(raw, `"mode":"webhook"`) {
		t.Fatalf("scheduled tasks payload missing delivery: %s", raw)
	}

	aliased := runtimeBootstrapEnvValues("hermes", env)
	if _, ok := aliased[HermesScheduledTasksEnv]; !ok {
		t.Fatalf("expected hermes alias %s", HermesScheduledTasksEnv)
	}
	if _, ok := aliased[RuntimeScheduledTasksEnv]; !ok {
		t.Fatalf("expected runtime alias %s", RuntimeScheduledTasksEnv)
	}
}
