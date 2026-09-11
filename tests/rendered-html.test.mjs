import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const root = new URL("../", import.meta.url);

async function render() {
  const workerUrl = new URL("../dist/server/index.js", import.meta.url);
  workerUrl.searchParams.set("test", `${process.pid}-${Date.now()}`);
  const { default: worker } = await import(workerUrl.href);
  return worker.fetch(new Request("http://localhost/", { headers: { accept: "text/html" } }), {
    ASSETS: { fetch: async () => new Response("Not found", { status: 404 }) },
  }, { waitUntil() {}, passThroughOnException() {} });
}

test("renders the Veil browser prototype", async () => {
  const response = await render();
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.match(html, /<title>Veil — приватный браузер<\/title>/i);
  assert.match(html, /Запускаем защищённую сессию/);
});

test("implements ephemeral session storage and honest iframe fallback", async () => {
  const [page, css] = await Promise.all([
    readFile(new URL("app/page.tsx", root), "utf8"),
    readFile(new URL("app/globals.css", root), "utf8"),
  ]);
  assert.match(page, /sessionStorage\.setItem/);
  assert.match(page, /sessionStorage\.removeItem/);
  assert.match(page, /без фокусов с iframe/);
  assert.match(page, /декорацию за защиту не выдаём/);
  assert.match(page, /depth-0/);
  assert.match(page, /depth-5/);
  assert.match(css, /prefers-reduced-motion:\s*reduce/);
});
