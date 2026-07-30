import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const detailSource = readFileSync(
  path.resolve(scriptDir, "../src/pages/instances/InstanceDetailPage.tsx"),
  "utf8",
);
const serviceSource = readFileSync(
  path.resolve(scriptDir, "../src/services/instanceService.ts"),
  "utf8",
);
const typeSource = readFileSync(
  path.resolve(scriptDir, "../src/types/instance.ts"),
  "utf8",
);

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

for (const label of ["1 hour", "24 hours", "7 days", "30 days", "Custom", "Permanent"]) {
  assert(detailSource.includes(label), `External access expiration option missing: ${label}`);
}

assert(
  detailSource.includes('useState<ExternalAccessExpirationMode>("preset")') &&
    detailSource.includes('useState<ExternalAccessExpirationPreset>("24h")'),
  "External access UI must default to a 24 hour preset.",
);
assert(
  serviceSource.includes("ExternalAccessRequest") &&
    serviceSource.includes("external-access/share-link") &&
    serviceSource.includes("external-access/password"),
  "Instance service must send expiration payloads to share link and password endpoints.",
);
assert(
  detailSource.includes("Share Link") &&
    detailSource.includes("Password") &&
    !detailSource.includes("Public Link") &&
    !detailSource.includes("API Key"),
  "Instance external access UI must use Share Link and Password labels.",
);
assert(
  !serviceSource.includes("external-access/public-link") &&
    !serviceSource.includes("external-access/api-key"),
  "Instance service must not call legacy public-link or api-key endpoints.",
);
assert(
  !typeSource.includes("public_slug:") && !typeSource.includes("token?:"),
  "Frontend external access types must not expose legacy long-link slug or token fields.",
);
assert(
  detailSource.includes("Shared workspace") &&
    detailSource.includes("Full file access") &&
    detailSource.includes("View and download") &&
    detailSource.includes("Do not share files") &&
    typeSource.includes('workspace_access: "none" | "read" | "write"'),
  "Share Link must expose an explicit single-instance workspace scope.",
);

console.log("Instance external access short-link UI contract is valid.");
