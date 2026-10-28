import { describe, expect, it } from "vitest";
import { render, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { AccountProvider } from "./Account.context";
import { mockAccount, mockFetch } from "./testing/fetch";
import App from "./App";

const renderApp = (path = "/") =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <AccountProvider>
        <App />
      </AccountProvider>
    </MemoryRouter>,
  );

describe("App", () => {
  it("sends signed-out visitors to the sign-in page", async () => {
    mockFetch({
      "GET /cloud/auth/providers": { body: { providers: ["github"] } },
    });

    const wrapper = renderApp();

    await waitFor(() => {
      expect(wrapper.getByText("Sign in to Stormkit Cloud")).toBeTruthy();
    });
  });

  it("shows the instances of a signed-in account", async () => {
    localStorage.setItem("skc_session", "skc_token");
    const fetch = mockFetch({
      "GET /cloud/account": { body: { account: mockAccount } },
    });

    const wrapper = renderApp();

    await waitFor(() => {
      expect(wrapper.getByText("Your instances")).toBeTruthy();
      expect(wrapper.getByText("Jane Doe")).toBeTruthy();
    });

    const init = fetch.mock.calls[0][1] as RequestInit;
    expect((init.headers as Record<string, string>).Authorization).toBe(
      "Bearer skc_token",
    );
  });

  it("forgets a session the API no longer accepts", async () => {
    localStorage.setItem("skc_session", "skc_expired");
    mockFetch({
      "GET /cloud/account": { status: 401 },
      "GET /cloud/auth/providers": { body: { providers: [] } },
    });

    const wrapper = renderApp();

    await waitFor(() => {
      expect(wrapper.getByText("Sign in to Stormkit Cloud")).toBeTruthy();
    });

    expect(localStorage.getItem("skc_session")).toBeNull();
  });

  // A browser holding an expired session completes a new sign-in. The old
  // session's 401 arrives last and must not undo the new sign-in.
  it("keeps a new sign-in when the old session's 401 arrives late", async () => {
    localStorage.setItem("skc_session", "skc_expired");
    mockFetch({
      "GET /cloud/account": { status: 401, delay: 50 },
      "POST /cloud/auth/session": {
        body: { token: "skc_new", account: mockAccount },
      },
    });

    const wrapper = renderApp("/auth/callback?code=my-code");

    await waitFor(() => {
      expect(wrapper.getByText("Your instances")).toBeTruthy();
    });

    await new Promise((resolve) => setTimeout(resolve, 100));

    expect(wrapper.getByText("Your instances")).toBeTruthy();
    expect(localStorage.getItem("skc_session")).toBe("skc_new");
  });

  // Reopening a used sign-in link must not sign out a working session.
  it("keeps the session when a used sign-in link is reopened", async () => {
    localStorage.setItem("skc_session", "skc_token");
    const fetch = mockFetch({
      "GET /cloud/account": { body: { account: mockAccount } },
      "POST /cloud/auth/session": { status: 401 },
    });

    const wrapper = renderApp("/auth/callback?code=used-code");

    await waitFor(() => {
      expect(wrapper.getByText(/expired or was used already/)).toBeTruthy();
    });

    const post = fetch.mock.calls.find(([, init]) => init?.method === "POST");
    const headers = post?.[1]?.headers as Record<string, string>;

    expect(headers.Authorization).toBeUndefined();
    expect(localStorage.getItem("skc_session")).toBe("skc_token");
  });

  it("signs out", async () => {
    localStorage.setItem("skc_session", "skc_token");
    const fetch = mockFetch({
      "GET /cloud/account": { body: { account: mockAccount } },
      "DELETE /cloud/auth/session": { body: {} },
      "GET /cloud/auth/providers": { body: { providers: ["github"] } },
    });

    const wrapper = renderApp();

    await waitFor(() => {
      expect(wrapper.getByText("Sign out")).toBeTruthy();
    });

    await userEvent.click(wrapper.getByText("Sign out"));

    await waitFor(() => {
      expect(wrapper.getByText("Sign in to Stormkit Cloud")).toBeTruthy();
    });

    expect(fetch.mock.calls.some(([, init]) => init?.method === "DELETE")).toBe(
      true,
    );
    expect(localStorage.getItem("skc_session")).toBeNull();
  });
});
