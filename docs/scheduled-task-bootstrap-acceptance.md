# Scheduled Task Bootstrap Acceptance

Manual checklist after rebuilding ClawManager + OpenClaw/Hermes runtime images.

## Shared setup

1. In Resource Management, create a `scheduled_task` with format `task/openclaw-cron@v1`.
2. Configure one announce job (`delivery.mode=announce`) and one webhook job (`delivery.mode=webhook` with an http(s) URL).
3. Create instances with those resources selected (manual or bundle).

## Matrix

For each of: OpenClaw Lite, OpenClaw Pro, Hermes Lite, Hermes Pro:

1. Instance reaches running and agent/gateway is healthy.
2. Managed jobs appear with ids `cm-st-{resource_id}`:
   - OpenClaw: `~/.openclaw/cron/jobs.json`
   - Hermes: `~/.hermes/cron/jobs.json`
   - **Owner must be the instance Linux UID/GID (`200000 + instance_id`), not `root`.** A `root:root` `jobs.json` causes Control UI cron pages to fail with `EACCES`.
3. Announce job fires and delivers to the configured channel/origin.
4. Webhook job fires and POSTs (OpenClaw native delivery, or Hermes prompt+local outbox/`cron/webhooks/*.url` path).
5. Restart instance / recreate Lite gateway:
   - identical bootstrap payload skips rewrite (or remains idempotent)
   - user-created non-`cm-st-*` cron jobs remain
6. Invalid scheduled-tasks env (if injected manually) logs an error and does **not** prevent startup.

## Notes

- Hermes ignores OpenClaw `wakeMode` / `sessionTarget`; expect `ignored_fields` in bootstrap state.
- Platform save validation remains strict; only runtime apply is soft-fail.
