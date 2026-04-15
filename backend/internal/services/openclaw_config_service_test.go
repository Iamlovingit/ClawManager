package services

import (
	"encoding/json"
	"reflect"
	"testing"

	"clawreef/internal/models"
)

func TestRenderCompiledOpenClawPayloadRendersChannelsAsKeyedConfigMap(t *testing.T) {
	t.Parallel()

	resources := []compiledOpenClawResource{
		{
			model: models.OpenClawConfigResource{
				ID:           1,
				ResourceType: OpenClawConfigResourceTypeChannel,
				ResourceKey:  "dingtalk-connector",
				Name:         "DingTalk",
				Version:      1,
				ContentJSON:  `{"schemaVersion":1,"kind":"channel","format":"channel/dingtalk-connector@v1","dependsOn":[],"config":{"enabled":false,"clientId":"ding-xxxxxxxxxxxxxx","clientSecret":"xxxxxxxxxxxxxxxxxxxxxxx","allowFrom":["*"],"legacyField":"drop-me"}}`,
			},
			envelope: OpenClawConfigEnvelope{
				SchemaVersion: 1,
				Kind:          "channel",
				Format:        "channel/dingtalk-connector@v1",
				Config:        json.RawMessage(`{"enabled":false,"clientId":"ding-xxxxxxxxxxxxxx","clientSecret":"xxxxxxxxxxxxxxxxxxxxxxx","allowFrom":["*"],"legacyField":"drop-me"}`),
			},
		},
		{
			model: models.OpenClawConfigResource{
				ID:           2,
				ResourceType: OpenClawConfigResourceTypeChannel,
				ResourceKey:  "feishu",
				Name:         "Feishu",
				Version:      1,
				ContentJSON:  `{"schemaVersion":1,"kind":"channel","format":"channel/feishu@v1","dependsOn":[],"config":{"enabled":false,"domain":"feishu","appId":"cli_top","appSecret":"top_secret","defaultAccount":"default","accounts":{"default":{"appId":"cli_xxx","appSecret":"xxx","botName":"old-bot","enabled":true}},"requireMention":true}}`,
			},
			envelope: OpenClawConfigEnvelope{
				SchemaVersion: 1,
				Kind:          "channel",
				Format:        "channel/feishu@v1",
				Config:        json.RawMessage(`{"enabled":false,"domain":"feishu","appId":"cli_top","appSecret":"top_secret","defaultAccount":"default","accounts":{"default":{"appId":"cli_xxx","appSecret":"xxx","botName":"old-bot","enabled":true}},"requireMention":true}`),
			},
		},
		{
			model: models.OpenClawConfigResource{
				ID:           3,
				ResourceType: OpenClawConfigResourceTypeChannel,
				ResourceKey:  "slack",
				Name:         "Slack",
				Version:      1,
				ContentJSON:  `{"schemaVersion":1,"kind":"channel","format":"channel/slack@v1","dependsOn":[],"config":{"enabled":false,"botToken":"xoxb-xxx","appToken":"xapp-xxx","groupPolicy":"allowlist","channels":{"#general":{"allow":true}},"capabilities":{"interactiveReplies":true}}}`,
			},
			envelope: OpenClawConfigEnvelope{
				SchemaVersion: 1,
				Kind:          "channel",
				Format:        "channel/slack@v1",
				Config:        json.RawMessage(`{"enabled":false,"botToken":"xoxb-xxx","appToken":"xapp-xxx","groupPolicy":"allowlist","channels":{"#general":{"allow":true}},"capabilities":{"interactiveReplies":true}}`),
			},
		},
		{
			model: models.OpenClawConfigResource{
				ID:           4,
				ResourceType: OpenClawConfigResourceTypeChannel,
				ResourceKey:  "telegram",
				Name:         "Telegram",
				Version:      1,
				ContentJSON:  `{"schemaVersion":1,"kind":"channel","format":"channel/telegram@v1","dependsOn":[],"config":{"enabled":false,"botToken":"123456:xxx","dmPolicy":"open","allowFrom":["*"],"network":{"autoSelectFamily":false}}}`,
			},
			envelope: OpenClawConfigEnvelope{
				SchemaVersion: 1,
				Kind:          "channel",
				Format:        "channel/telegram@v1",
				Config:        json.RawMessage(`{"enabled":false,"botToken":"123456:xxx","dmPolicy":"open","allowFrom":["*"],"network":{"autoSelectFamily":false}}`),
			},
		},
		{
			model: models.OpenClawConfigResource{
				ID:           5,
				ResourceType: OpenClawConfigResourceTypeSkill,
				ResourceKey:  "support-bot",
				Name:         "Support Bot",
				Version:      1,
				ContentJSON:  `{"schemaVersion":1,"kind":"skill","format":"skill/custom@v1","dependsOn":[],"config":{"prompt":"help"}}`,
			},
			tags: []string{"skill"},
			envelope: OpenClawConfigEnvelope{
				SchemaVersion: 1,
				Kind:          "skill",
				Format:        "skill/custom@v1",
				Config:        json.RawMessage(`{"prompt":"help"}`),
			},
		},
	}

	renderedEnv, _, _, _, err := renderCompiledOpenClawPayload(OpenClawConfigPlan{Mode: OpenClawConfigPlanModeManual}, nil, resources)
	if err != nil {
		t.Fatalf("renderCompiledOpenClawPayload returned error: %v", err)
	}

	gotChannels := renderedEnv[OpenClawChannelsEnv]
	wantChannels := `{"dingtalk-connector":{"allowFrom":["*"],"clientId":"ding-xxxxxxxxxxxxxx","clientSecret":"xxxxxxxxxxxxxxxxxxxxxxx","enabled":true},"feishu":{"accounts":{"main":{"appId":"cli_xxx","appSecret":"xxx"}},"enabled":true},"slack":{"appToken":"xapp-xxx","botToken":"xoxb-xxx","capabilities":{"interactiveReplies":true},"channels":{"#general":{"allow":true}},"enabled":true,"groupPolicy":"allowlist"},"telegram":{"allowFrom":["*"],"botToken":"123456:xxx","dmPolicy":"open","enabled":true}}`
	if gotChannels != wantChannels {
		t.Fatalf("unexpected channel payload:\nwant: %s\ngot:  %s", wantChannels, gotChannels)
	}

	gotSkills := renderedEnv[OpenClawSkillsEnv]
	wantSkills := `{"items":[{"content":{"schemaVersion":1,"kind":"skill","format":"skill/custom@v1","dependsOn":[],"config":{"prompt":"help"}},"id":5,"key":"support-bot","name":"Support Bot","tags":["skill"],"type":"skill","version":1}],"schemaVersion":1}`
	if gotSkills != wantSkills {
		t.Fatalf("unexpected skill payload:\nwant: %s\ngot:  %s", wantSkills, gotSkills)
	}
}

func TestResourcePayloadFromModelNormalizesStoredChannelJSON(t *testing.T) {
	t.Parallel()

	item := models.OpenClawConfigResource{
		ID:           10,
		UserID:       1,
		ResourceType: OpenClawConfigResourceTypeChannel,
		ResourceKey:  "slack",
		Name:         "Slack",
		Enabled:      true,
		Version:      1,
		TagsJSON:     `["channel","builtin","slack"]`,
		ContentJSON:  `{"schemaVersion":1,"kind":"channel","format":"channel/slack@v1","dependsOn":[],"config":{"enabled":false,"botToken":"xoxb-xxxxxxxxx","appToken":"xapp-xxxxxxxxxxxxxx","groupPolicy":"allowlist","channels":{"#general":{"allow":true}},"capabilities":{"interactiveReplies":true},"legacyField":"drop-me"}}`,
	}

	payload, err := resourcePayloadFromModel(item)
	if err != nil {
		t.Fatalf("resourcePayloadFromModel returned error: %v", err)
	}

	got := string(payload.Content)
	want := `{"schemaVersion":1,"kind":"channel","format":"channel/slack@v1","dependsOn":[],"config":{"enabled":true,"botToken":"xoxb-xxxxxxxxx","appToken":"xapp-xxxxxxxxxxxxxx","groupPolicy":"allowlist","channels":{"#general":{"allow":true}},"capabilities":{"interactiveReplies":true}}}`

	var gotJSON interface{}
	if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
		t.Fatalf("failed to unmarshal normalized resource content: %v", err)
	}

	var wantJSON interface{}
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatalf("failed to unmarshal expected normalized resource content: %v", err)
	}

	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Fatalf("unexpected normalized resource content:\nwant: %s\ngot:  %s", want, got)
	}
}

func TestResourcePayloadFromModelNormalizesStoredDingTalkChannelJSON(t *testing.T) {
	t.Parallel()

	item := models.OpenClawConfigResource{
		ID:           11,
		UserID:       1,
		ResourceType: OpenClawConfigResourceTypeChannel,
		ResourceKey:  "dingtalk-connector",
		Name:         "DingTalk",
		Enabled:      true,
		Version:      1,
		TagsJSON:     `["channel","builtin","dingtalk-connector"]`,
		ContentJSON:  `{"schemaVersion":1,"kind":"channel","format":"channel/dingtalk-connector@v1","dependsOn":[],"config":{"enabled":false,"clientId":"ding-xxxxxxxxxxxxxx","clientSecret":"xxxxxxxxxxxxxxxxxxxxxxx","allowFrom":[],"legacyField":"drop-me"}}`,
	}

	payload, err := resourcePayloadFromModel(item)
	if err != nil {
		t.Fatalf("resourcePayloadFromModel returned error: %v", err)
	}

	got := string(payload.Content)
	want := `{"schemaVersion":1,"kind":"channel","format":"channel/dingtalk-connector@v1","dependsOn":[],"config":{"enabled":true,"clientId":"ding-xxxxxxxxxxxxxx","clientSecret":"xxxxxxxxxxxxxxxxxxxxxxx","allowFrom":["*"]}}`

	var gotJSON interface{}
	if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
		t.Fatalf("failed to unmarshal normalized resource content: %v", err)
	}

	var wantJSON interface{}
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatalf("failed to unmarshal expected normalized resource content: %v", err)
	}

	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Fatalf("unexpected normalized resource content:\nwant: %s\ngot:  %s", want, got)
	}
}
