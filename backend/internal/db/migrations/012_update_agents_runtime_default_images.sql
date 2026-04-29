UPDATE system_image_settings
SET image = 'ghcr.io/Yuan-lab-LLM/AgentsRuntime/openclaw:latest'
WHERE instance_type = 'openclaw'
  AND image IN (
    'ericpearlee/openclaw:v2026.3.24',
    'ghcr.io/yuan-lab-llm/clawmanager-openclaw-image/openclaw:latest'
  );

UPDATE system_image_settings
SET image = 'ghcr.io/Yuan-lab-LLM/AgentsRuntime/hermes:latest'
WHERE instance_type = 'hermes'
  AND image IN (
    'registry.example.com/hermes-webtop:latest',
    'lscr.io/linuxserver/webtop:ubuntu-xfce'
  );
