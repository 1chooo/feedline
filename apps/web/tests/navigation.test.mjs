import assert from "node:assert/strict";
import test from "node:test";
import { authDestination, authHref } from "../src/lib/navigation.ts";

test("advertising intent survives switching authentication forms", () => {
  const destination = "/advertiser?package=2";
  for (const kind of ["signup", "login"]) {
    const url = new URL(authHref(kind, destination), "https://stream.invalid");
    assert.equal(authDestination(url.searchParams.get("next")), destination);
  }
  assert.equal(authDestination(undefined), "/");
});

test("authentication cannot redirect outside the product", () => {
  for (const destination of ["https://example.com", "//example.com", "/\\example.com", "/unknown", "/advertiser?package=unsafe", ["/admin"]]) {
    assert.equal(authDestination(destination), destination === "/advertiser?package=unsafe" ? "/advertiser" : "/");
  }
  assert.equal(authDestination("/advertiser?package=2&redirect=https://example.com"), "/advertiser?package=2");
});
