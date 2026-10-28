import { vi } from "vitest";

interface MockResponse {
  status?: number;
  body?: unknown;
}

/**
 * mockFetch answers API calls by "METHOD /path" and records every call.
 * Unknown routes answer 404.
 */
export function mockFetch(routes: Record<string, MockResponse>) {
  const fn = vi.fn(async (url: string, init?: RequestInit) => {
    const path = url.replace(/^http:\/\/api\.test/, "");
    const route = routes[`${init?.method || "GET"} ${path}`];

    if (!route) {
      return new Response("", { status: 404 });
    }

    return new Response(
      route.body === undefined ? "" : JSON.stringify(route.body),
      { status: route.status ?? 200 },
    );
  });

  vi.stubGlobal("fetch", fn);

  return fn;
}

export const mockAccount = {
  id: "1",
  displayName: "Jane Doe",
  avatarUrl: "https://avatars.test/jane",
  emails: ["jane@acme.test"],
  createdAt: 1700000000,
};
