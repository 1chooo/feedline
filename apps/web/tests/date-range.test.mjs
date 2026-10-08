import assert from "node:assert/strict";
import test from "node:test";
import { reportingRange } from "../src/lib/date-range.ts";

test("reporting ranges are inclusive, UTC, and match the backend limit", () => {
  assert.deepEqual(reportingRange(undefined, undefined, new Date("2026-10-08T01:00:00-07:00")), {from:"2026-09-09", to:"2026-10-08"});
  assert.equal(reportingRange("2026-01-01", "2026-03-31").error, undefined);
  assert.ok(reportingRange("2026-01-01", "2026-04-01").error);
  assert.ok(reportingRange("2026-10-08", "2026-10-01").error);
  assert.ok(reportingRange("2026-02-30", "2026-03-01").error);
  assert.equal(reportingRange("2026-10-08", "2026-10-08").error, undefined);
});
