# Resource Management Guide

Resource Management is the reusable asset layer for OpenClaw workspaces in ClawManager. It is centered on channels, skills, bundles, and the snapshots used to compile those assets into instance-ready configuration.

## Main Resource Types

- `Channels` for workspace connectivity and integration templates
- `Skills` for reusable uploaded packages that can be installed into runtime instances
- Config skills for bootstrap configuration delivered through runtime environment payloads
- `Scheduled tasks` (`scheduled_task`) for OpenClaw-compatible cron jobs (`schedule` + `payload` + `delivery`) injected at instance create/start
- `Bundles` for composing repeatable resource sets, including both config resources and uploaded skills
- injection snapshots for tracking the compiled result applied to an instance

## Scheduled Task Bootstrap

- Platform format: `task/openclaw-cron@v1`
- Acceptance benchmark: OpenClaw cron `schedule` / `payload` / `delivery` (including announce and webhook)
- Compiled env: `CLAWMANAGER_OPENCLAW_SCHEDULED_TASKS_JSON` (Hermes also receives `CLAWMANAGER_HERMES_*` / `CLAWMANAGER_RUNTIME_*` aliases)
- OpenClaw Lite/Pro: managed jobs are upserted into `~/.openclaw/cron/jobs.json` with ids `cm-st-{resource_id}`
- Hermes Lite/Pro: the same OpenClaw-benchmark config is **translated** into Hermes native cron jobs under `~/.hermes/cron/jobs.json`, then executed by Hermes gateway cron
- Managed jobs never delete user-created cron entries
- Runtime apply is soft-fail: invalid bootstrap JSON is logged and skipped; instance/gateway startup continues
- Identical payload hash + managed id set skips rewriting the cron store
- Hermes translation ignores OpenClaw-only `wakeMode` / `sessionTarget` (recorded as `ignored_fields`); Hermes always runs fresh agent sessions

## Core Workflows

1. Create or import channels, skills, and scheduled tasks in the OpenClaw Config Center.
2. Organize selected config resources and uploaded skills into reusable bundles.
3. Review scan posture for skills through Security Center.
4. Apply resources or bundles to OpenClaw/Hermes workspaces at instance creation.
5. Inspect runtime state and instance-level resource results after injection.

## How It Connects to the Platform

- Resource Management defines what should be delivered to a workspace.
- Config resources are compiled into bootstrap environment payloads. Uploaded skills in a bundle are installed through the Agent Control Plane skill installation path.
- Agent Control Plane / runtime agents apply and track those changes at runtime.
- Security Center and `skill-scanner` help review the risk posture of reusable skills before broad rollout.

## Related Guides

- [Security / Skill Scanner Guide](./security-skill-scanner.md)
- [Agent Control Plane Guide](./agent-control-plane.md)
- [Admin and User Guide](./admin-user-guide.md)
