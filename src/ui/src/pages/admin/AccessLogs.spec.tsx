import { describe, expect, it, beforeEach } from "vitest";
import { fireEvent, waitFor, type RenderResult } from "@testing-library/react";
import nock from "nock";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockUser from "~/testing/data/mock_user";
import mockTeams from "~/testing/data/mock_teams";
import { renderWithRouter } from "~/testing/helpers";
import AccessLogs from "./AccessLogs";

describe("~/pages/admin/AccessLogs.tsx", () => {
  let wrapper: RenderResult;

  const teams = mockTeams();

  const logs = [
    {
      id: "1",
      appId: "app-1",
      domainId: "dom-1",
      hostName: "example.org",
      requestTimestamp: "1750000000",
      method: "GET",
      path: "/api/users",
      statusCode: 500,
      clientIp: "10.0.0.1",
      userAgent: "curl",
      referrer: "",
      isBot: true,
      bytesSent: 12,
      durationMs: 340,
    },
  ];

  const mockLogs = (query: string, hasNextPage = false, cursor?: string) =>
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs" + query)
      .reply(200, {
        accessLogs: logs,
        pagination: { hasNextPage, cursor },
      });

  const createWrapper = (initialPath = "/admin/access-logs?appId=1") => {
    wrapper = renderWithRouter({
      path: "/admin/access-logs",
      initialEntries: [initialPath],
      el: () => (
        <AuthContext.Provider value={{ user: mockUser(), teams }}>
          <AccessLogs />
        </AuthContext.Provider>
      ),
    });
  };

  beforeEach(() => {
    nock.cleanAll();
    localStorage.clear();
  });

  it("does not search until an app is picked", async () => {
    const scope = nock(process.env.API_DOMAIN || "")
      .get(/\/admin\/access-logs/)
      .reply(200, { accessLogs: logs, pagination: { hasNextPage: false } });

    createWrapper("/admin/access-logs?method=GET");

    await waitFor(() => {
      expect(
        wrapper.getByText("Pick an app to see its access logs."),
      ).toBeTruthy();
    });

    expect(scope.isDone()).toBe(false);
    expect(wrapper.getByText("Search").closest("button")?.disabled).toBe(true);
  });

  it("searches once an app is picked", async () => {
    createWrapper("/admin/access-logs?method=GET");

    await waitFor(() => {
      expect(
        wrapper.getByText("Pick an app to see its access logs."),
      ).toBeTruthy();
    });

    nock(process.env.API_DOMAIN || "")
      .get("/v1/apps")
      .query(true)
      .reply(200, { apps: [{ id: "1", displayName: "my-app" }] });

    const scope = mockLogs("?appId=1&method=GET");

    fireEvent.focus(
      wrapper.getByLabelText("Filter by app, host, path, status…"),
    );
    fireEvent.click(wrapper.getByRole("menuitem", { name: "App ID" }));

    await waitFor(() => {
      expect(
        wrapper.getByRole("menuitem", { name: "my-app (1)" }),
      ).toBeTruthy();
    });

    fireEvent.click(wrapper.getByRole("menuitem", { name: "my-app (1)" }));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/api/users")).toBeTruthy();
    });
  });

  it("searches when scoped to a domain", async () => {
    const scope = mockLogs("?domainId=5");
    createWrapper("/admin/access-logs?domainId=5");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("renders access logs returned by the API", async () => {
    const scope = mockLogs("?appId=1");
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/api/users")).toBeTruthy();
      expect(wrapper.getByText("340 ms")).toBeTruthy();
    });
  });

  it("sends filters read from the URL as query params", async () => {
    const scope = mockLogs("?appId=1&method=GET&status=500");
    createWrapper("/admin/access-logs?appId=1&method=GET&status=500");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("renders URL filters as tokens", async () => {
    mockLogs("?appId=1&path=%2Fapi&isBot=true");
    createWrapper("/admin/access-logs?appId=1&path=/api&isBot=true");

    await waitFor(() => {
      expect(wrapper.getByTestId("token-path").textContent).toContain("/api");
      expect(wrapper.getByTestId("token-isBot").textContent).toContain("only");
    });
  });

  it("converts datetime bounds to unix seconds for the API", async () => {
    const from = "2026-07-01T10:30";
    const expected = Math.floor(new Date(from).getTime() / 1000);
    const scope = mockLogs(`?appId=1&from=${expected}`);

    createWrapper(
      `/admin/access-logs?appId=1&from=${encodeURIComponent(from)}`,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("ignores query params that are not known filters", async () => {
    const scope = mockLogs("?appId=1&path=%2Fapi");
    createWrapper("/admin/access-logs?appId=1&path=/api&bogus=1");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("persists a newly added filter to the URL and refetches", async () => {
    mockLogs("?appId=1");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("/api/users")).toBeTruthy();
    });

    const scope = mockLogs("?appId=1&method=GET");

    fireEvent.focus(
      wrapper.getByLabelText("Filter by app, host, path, status…"),
    );
    fireEvent.click(wrapper.getByRole("menuitem", { name: "Method" }));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "GET" }));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByTestId("token-method")).toBeTruthy();
    });
  });

  it("appends the next page when loading more", async () => {
    mockLogs("?appId=1", true, "cursor-1");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Load more")).toBeTruthy();
    });

    const scope = nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?appId=1&cursor=cursor-1")
      .reply(200, {
        accessLogs: [{ ...logs[0], id: "2", path: "/api/teams" }],
        pagination: { hasNextPage: false },
      });

    fireEvent.click(wrapper.getByText("Load more"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/api/users")).toBeTruthy();
      expect(wrapper.getByText("/api/teams")).toBeTruthy();
    });
  });

  it("drops the cursor when filters change", async () => {
    mockLogs("?appId=1", true, "cursor-1");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Load more")).toBeTruthy();
    });

    // Adding a filter must restart from the first page, not reuse the cursor.
    const scope = mockLogs("?appId=1&method=GET");

    fireEvent.focus(
      wrapper.getByLabelText("Filter by app, host, path, status…"),
    );
    fireEvent.click(wrapper.getByRole("menuitem", { name: "Method" }));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "GET" }));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("drops the cursor when the search button refreshes", async () => {
    mockLogs("?appId=1", true, "cursor-1");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Load more")).toBeTruthy();
    });

    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?appId=1&cursor=cursor-1")
      .reply(200, {
        accessLogs: [{ ...logs[0], id: "2", path: "/api/teams" }],
        pagination: { hasNextPage: false },
      });

    fireEvent.click(wrapper.getByText("Load more"));

    await waitFor(() => {
      expect(wrapper.getByText("/api/teams")).toBeTruthy();
    });

    // Refreshing must restart from the first page rather than re-append the
    // page the cursor points at.
    const scope = mockLogs("?appId=1");

    fireEvent.click(wrapper.getByText("Search"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.queryByText("/api/teams")).toBe(null);
    });
  });

  it("warns when To is set without From", async () => {
    mockLogs("?appId=1&to=1751362200");
    createWrapper("/admin/access-logs?appId=1&to=2026-07-01T10%3A30");

    await waitFor(() => {
      expect(wrapper.getByText(/falls back to the last 24 hours/)).toBeTruthy();
    });
  });

  it("ignores a datetime bound the API cannot be given", async () => {
    const scope = mockLogs("?appId=1&path=%2Fapi");
    createWrapper("/admin/access-logs?appId=1&path=/api&from=yesterday");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.queryByTestId("token-from")).toBe(null);
    });
  });

  it("shows the API message when the query times out", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?appId=1")
      .reply(400, {
        error: "The query took too long. Narrow the time range or pick an app.",
      });

    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText(/The query took too long/)).toBeTruthy();
    });
  });

  it("shows the generic error when a 400 has no JSON body", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?appId=1")
      .reply(400, "Bad Request");

    createWrapper();

    await waitFor(() => {
      expect(
        wrapper.getByText("Something went wrong while fetching access logs."),
      ).toBeTruthy();
    });

    expect(wrapper.queryByText("No access logs match your filters.")).toBe(
      null,
    );
  });

  it("does not apply a stale timeout message after filters change", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?appId=1")
      .delayBody(300)
      .reply(400, {
        error: "The query took too long. Narrow the time range or pick an app.",
      });

    createWrapper();

    // Let the 400 arrive so the body is being read when the filters change.
    await new Promise(resolve => setTimeout(resolve, 100));

    const scope = mockLogs("?appId=1&method=GET");

    fireEvent.focus(
      wrapper.getByLabelText("Filter by app, host, path, status…"),
    );
    fireEvent.click(wrapper.getByRole("menuitem", { name: "Method" }));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "GET" }));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/api/users")).toBeTruthy();
    });

    await new Promise(resolve => setTimeout(resolve, 400));

    expect(wrapper.queryByText(/The query took too long/)).toBe(null);
  });

  it("keeps searching disabled for a non-numeric app id", async () => {
    const scope = nock(process.env.API_DOMAIN || "")
      .get(/\/admin\/access-logs/)
      .reply(200, { accessLogs: logs, pagination: { hasNextPage: false } });

    createWrapper("/admin/access-logs?appId=my-app");

    await waitFor(() => {
      expect(wrapper.getByText(/must be numeric/)).toBeTruthy();
    });

    expect(scope.isDone()).toBe(false);
    expect(wrapper.getByText("Search").closest("button")?.disabled).toBe(true);
  });

  it("shows an error when the request fails", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?appId=1")
      .reply(500);

    createWrapper();

    await waitFor(() => {
      expect(
        wrapper.getByText("Something went wrong while fetching access logs."),
      ).toBeTruthy();
    });
  });
});
