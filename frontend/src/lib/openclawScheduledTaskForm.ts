export type ScheduledTaskScheduleKind = "at" | "every" | "cron";
export type ScheduledTaskSessionTarget = "main" | "isolated";
export type ScheduledTaskWakeMode = "next-heartbeat" | "now";
export type ScheduledTaskPayloadKind = "systemEvent" | "agentTurn";
export type ScheduledTaskDeliveryMode = "none" | "announce" | "webhook";
export type ScheduledTaskSchedulePreset =
  | "daily9"
  | "hourly"
  | "every5m"
  | "custom";

export interface ScheduledTaskFormState {
  name: string;
  description: string;
  enabled: boolean;
  deleteAfterRun: boolean;
  scheduleKind: ScheduledTaskScheduleKind;
  scheduleAt: string;
  scheduleEveryMs: string;
  scheduleExpr: string;
  scheduleTz: string;
  sessionTarget: ScheduledTaskSessionTarget;
  wakeMode: ScheduledTaskWakeMode;
  payloadKind: ScheduledTaskPayloadKind;
  payloadText: string;
  payloadMessage: string;
  payloadModel: string;
  deliveryMode: ScheduledTaskDeliveryMode;
  deliveryChannel: string;
  deliveryTo: string;
  deliveryBestEffort: boolean;
}

export const SCHEDULE_PRESETS: Record<
  Exclude<ScheduledTaskSchedulePreset, "custom">,
  { expr: string; tz: string }
> = {
  daily9: { expr: "0 9 * * *", tz: "Asia/Shanghai" },
  hourly: { expr: "0 * * * *", tz: "Asia/Shanghai" },
  every5m: { expr: "*/5 * * * *", tz: "Asia/Shanghai" },
};

const defaultFormState = (): ScheduledTaskFormState => ({
  name: "daily-brief",
  description: "",
  enabled: true,
  deleteAfterRun: false,
  scheduleKind: "cron",
  scheduleAt: "",
  scheduleEveryMs: "60000",
  scheduleExpr: SCHEDULE_PRESETS.daily9.expr,
  scheduleTz: SCHEDULE_PRESETS.daily9.tz,
  sessionTarget: "isolated",
  wakeMode: "now",
  payloadKind: "agentTurn",
  payloadText: "",
  payloadMessage: "Summarize overnight updates and send a short brief.",
  payloadModel: "",
  deliveryMode: "announce",
  deliveryChannel: "last",
  deliveryTo: "",
  deliveryBestEffort: true,
});

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

export const parseScheduledTaskContent = (
  contentText: string,
): Record<string, unknown> | null => {
  try {
    const parsed = JSON.parse(contentText) as unknown;
    return isRecord(parsed) ? parsed : null;
  } catch {
    return null;
  }
};

export const readScheduledTaskFormState = (
  contentText: string,
): ScheduledTaskFormState | null => {
  const parsed = parseScheduledTaskContent(contentText);
  if (!parsed) {
    return null;
  }
  const config = isRecord(parsed.config) ? parsed.config : null;
  if (!config) {
    return null;
  }
  const schedule = isRecord(config.schedule) ? config.schedule : {};
  const payload = isRecord(config.payload) ? config.payload : {};
  const delivery = isRecord(config.delivery) ? config.delivery : {};
  const scheduleKind =
    schedule.kind === "at" || schedule.kind === "every" || schedule.kind === "cron"
      ? schedule.kind
      : "cron";
  const sessionTarget =
    config.sessionTarget === "main" || config.sessionTarget === "isolated"
      ? config.sessionTarget
      : "isolated";
  const wakeMode =
    config.wakeMode === "next-heartbeat" || config.wakeMode === "now"
      ? config.wakeMode
      : "now";
  const payloadKind =
    payload.kind === "systemEvent" || payload.kind === "agentTurn"
      ? payload.kind
      : sessionTarget === "main"
        ? "systemEvent"
        : "agentTurn";
  const deliveryMode =
    delivery.mode === "none" ||
    delivery.mode === "announce" ||
    delivery.mode === "webhook"
      ? delivery.mode
      : "announce";

  return {
    name: typeof config.name === "string" ? config.name : "",
    description: typeof config.description === "string" ? config.description : "",
    enabled: config.enabled !== false,
    deleteAfterRun: config.deleteAfterRun === true,
    scheduleKind,
    scheduleAt: typeof schedule.at === "string" ? schedule.at : "",
    scheduleEveryMs:
      typeof schedule.everyMs === "number"
        ? String(schedule.everyMs)
        : defaultFormState().scheduleEveryMs,
    scheduleExpr: typeof schedule.expr === "string" ? schedule.expr : "0 9 * * *",
    scheduleTz: typeof schedule.tz === "string" ? schedule.tz : "Asia/Shanghai",
    sessionTarget,
    wakeMode,
    payloadKind,
    payloadText: typeof payload.text === "string" ? payload.text : "",
    payloadMessage: typeof payload.message === "string" ? payload.message : "",
    payloadModel: typeof payload.model === "string" ? payload.model : "",
    deliveryMode,
    deliveryChannel: typeof delivery.channel === "string" ? delivery.channel : "last",
    deliveryTo: typeof delivery.to === "string" ? delivery.to : "",
    deliveryBestEffort: delivery.bestEffort !== false,
  };
};

export const resolveSchedulePreset = (
  form: ScheduledTaskFormState,
): ScheduledTaskSchedulePreset => {
  if (form.scheduleKind !== "cron") {
    return "custom";
  }
  const expr = form.scheduleExpr.trim();
  const tz = form.scheduleTz.trim() || "Asia/Shanghai";
  for (const [id, preset] of Object.entries(SCHEDULE_PRESETS) as Array<
    [Exclude<ScheduledTaskSchedulePreset, "custom">, { expr: string; tz: string }]
  >) {
    if (preset.expr === expr && preset.tz === tz) {
      return id;
    }
  }
  return "custom";
};

export const patchFromSchedulePreset = (
  preset: ScheduledTaskSchedulePreset,
): Partial<ScheduledTaskFormState> => {
  if (preset === "custom") {
    return { scheduleKind: "cron" };
  }
  const resolved = SCHEDULE_PRESETS[preset];
  return {
    scheduleKind: "cron",
    scheduleExpr: resolved.expr,
    scheduleTz: resolved.tz,
  };
};

export const buildScheduledTaskContent = (
  contentText: string,
  patch: Partial<ScheduledTaskFormState>,
): string => {
  const parsed = parseScheduledTaskContent(contentText) || {
    schemaVersion: 1,
    kind: "scheduled_task",
    format: "task/openclaw-cron@v1",
    dependsOn: [],
    config: {},
  };
  const current = readScheduledTaskFormState(contentText) || defaultFormState();
  const next: ScheduledTaskFormState = { ...current, ...patch };

  // Keep OpenClaw CRITICAL CONSTRAINTS consistent when switching sessionTarget.
  if (patch.sessionTarget === "main") {
    next.payloadKind = "systemEvent";
  } else if (patch.sessionTarget === "isolated") {
    next.payloadKind = "agentTurn";
  } else if (patch.payloadKind === "systemEvent") {
    next.sessionTarget = "main";
  } else if (patch.payloadKind === "agentTurn") {
    next.sessionTarget = "isolated";
  }

  const schedule: Record<string, unknown> = { kind: next.scheduleKind };
  if (next.scheduleKind === "at") {
    schedule.at = next.scheduleAt;
  } else if (next.scheduleKind === "every") {
    const everyMs = Number(next.scheduleEveryMs);
    schedule.everyMs = Number.isFinite(everyMs) ? everyMs : 60000;
  } else {
    schedule.expr = next.scheduleExpr;
    if (next.scheduleTz.trim()) {
      schedule.tz = next.scheduleTz.trim();
    }
  }

  const payload: Record<string, unknown> = { kind: next.payloadKind };
  if (next.payloadKind === "systemEvent") {
    payload.text = next.payloadText;
  } else {
    payload.message = next.payloadMessage;
    if (next.payloadModel.trim()) {
      payload.model = next.payloadModel.trim();
    }
  }

  const delivery: Record<string, unknown> = {
    mode: next.deliveryMode,
    bestEffort: next.deliveryBestEffort,
  };
  if (next.deliveryMode === "announce" && next.deliveryChannel.trim()) {
    delivery.channel = next.deliveryChannel.trim();
  }
  if (next.deliveryMode === "webhook" && next.deliveryTo.trim()) {
    delivery.to = next.deliveryTo.trim();
  }

  parsed.schemaVersion = 1;
  parsed.kind = "scheduled_task";
  parsed.format = "task/openclaw-cron@v1";
  if (!Array.isArray(parsed.dependsOn)) {
    parsed.dependsOn = [];
  }
  parsed.config = {
    name: next.name,
    description: next.description,
    enabled: next.enabled,
    deleteAfterRun: next.deleteAfterRun,
    schedule,
    sessionTarget: next.sessionTarget,
    wakeMode: next.wakeMode,
    payload,
    delivery,
  };

  return JSON.stringify(parsed, null, 2);
};

export const validateScheduledTaskContent = (contentText: string): string | null => {
  const form = readScheduledTaskFormState(contentText);
  if (!form) {
    return "invalid_json";
  }
  if (!form.name.trim()) {
    return "name_required";
  }
  if (form.scheduleKind === "at" && !form.scheduleAt.trim()) {
    return "schedule_at_required";
  }
  if (form.scheduleKind === "every") {
    const everyMs = Number(form.scheduleEveryMs);
    if (!Number.isFinite(everyMs) || everyMs <= 0) {
      return "schedule_every_required";
    }
  }
  if (form.scheduleKind === "cron" && !form.scheduleExpr.trim()) {
    return "schedule_expr_required";
  }
  if (form.sessionTarget === "main" && form.payloadKind !== "systemEvent") {
    return "main_requires_system_event";
  }
  if (form.sessionTarget === "isolated" && form.payloadKind !== "agentTurn") {
    return "isolated_requires_agent_turn";
  }
  if (form.payloadKind === "systemEvent" && !form.payloadText.trim()) {
    return "payload_text_required";
  }
  if (form.payloadKind === "agentTurn" && !form.payloadMessage.trim()) {
    return "payload_message_required";
  }
  if (form.deliveryMode === "webhook") {
    const to = form.deliveryTo.trim();
    if (!to || !/^https?:\/\//i.test(to)) {
      return "delivery_webhook_url_required";
    }
  }
  return null;
};
