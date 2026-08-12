import assert from "node:assert/strict";
import test from "node:test";

import { createAdapterClient } from "../../src/fava/adapter-client.ts";

function guidePayload(overrides = {}) {
  return {
    data: {
      code: "E-PARSE-DATE",
      topic: "diagnostics/E-PARSE-DATE",
      phase: "fix-first-syntax",
      short_action: "Correct the date",
      what: "A date is invalid.",
      why: "The ledger cannot be evaluated.",
      inspect: ["Inspect the date."],
      safe_steps: ["Correct the reviewed date."],
      example: { before: "2026-02-30", after: "2026-02-28", note: "Use the source record." },
      revalidate: "Run check again.",
      ...overrides,
    },
  };
}

test("adapter validates repair guidance and context payloads", async () => {
  const calls = [];
  const client = createAdapterClient(async (url) => {
    calls.push(String(url));
    if (String(url).includes("diagnostics/context")) {
      return { ok: true, json: async () => ({ available: true, path: "main.bean", focus_line: 2, lines: [{ line: 1, content: "before" }, { line: 2, content: "focus" }, { line: 3, content: "after" }] }) };
    }
    return { ok: true, json: async () => guidePayload() };
  });
  const guide = await client.guide("E-PARSE-DATE", "zh-CN");
  assert.equal(guide.topic, "diagnostics/E-PARSE-DATE");
  const context = await client.diagnosticContext("main.bean", 2);
  assert.equal(context.lines.length, 3);
  assert.match(calls[0], /locale=zh-CN/);
});

test("adapter rejects malformed repair guidance and context", async () => {
  const malformedGuide = createAdapterClient(async () => ({ ok: true, json: async () => guidePayload({ example: { before: "only" } }) }));
  await assert.rejects(() => malformedGuide.guide("E-PARSE-DATE"), /Invalid repair guidance/);

  const malformedContext = createAdapterClient(async () => ({ ok: true, json: async () => ({ available: true, lines: [{ line: 1 }] }) }));
  await assert.rejects(() => malformedContext.diagnosticContext("main.bean", 1), /Invalid diagnostic context/);
});
