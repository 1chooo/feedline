import assert from "node:assert/strict";
import test from "node:test";
import { loadFeed } from "../src/lib/feed.ts";

test("social posts remain readable when advertising is unavailable", async () => {
  const user = {username:"member", displayName:"Member", bio:"", role:"member"};
  const result = await loadFeed(async () => [{post:{id:"1",username:"member",title:"A community post"},author:user}], async () => { throw new Error("ad service unavailable"); });
  assert.equal(result.error, undefined);
  assert.equal(result.items.length, 1);
  assert.equal(result.items[0].post.title, "A community post");
});

test("a failed social read has a recoverable error, not an empty success", async () => {
  const result = await loadFeed(async () => { throw new Error("social unavailable"); }, async () => []);
  assert.ok(result.error);
  assert.equal(result.items.length, 0);
});
