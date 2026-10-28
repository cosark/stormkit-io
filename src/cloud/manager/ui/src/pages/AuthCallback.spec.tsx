import { describe, expect, it } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { AccountProvider } from "../Account.context";
import { mockAccount, mockFetch } from "../testing/fetch";
import AuthCallback from "./AuthCallback";

function CurrentURL() {
  const { pathname, search } = useLocation();
  return <div data-testid="url">{`${pathname}${search}`}</div>;
}

const renderCallback = (search: string) =>
  render(
    <MemoryRouter initialEntries={[`/auth/callback${search}`]}>
      <AccountProvider>
        <Routes>
          <Route path="/auth/callback" element={<AuthCallback />} />
          <Route path="/" element={<div>Home</div>} />
          <Route path="/login" element={<div>Login</div>} />
        </Routes>
        <CurrentURL />
      </AccountProvider>
    </MemoryRouter>,
  );

describe("AuthCallback", () => {
  it("trades the code for a session and opens the console", async () => {
    const fetch = mockFetch({
      "POST /cloud/auth/session": {
        body: { token: "skc_token", account: mockAccount },
      },
    });

    const wrapper = renderCallback("?code=my-code");

    await waitFor(() => {
      expect(wrapper.getByText("Home")).toBeTruthy();
    });

    expect(localStorage.getItem("skc_session")).toBe("skc_token");
    expect(JSON.parse(fetch.mock.calls[0][1]?.body as string)).toEqual({
      code: "my-code",
    });
  });

  it("takes the code out of the address bar", async () => {
    mockFetch({ "POST /cloud/auth/session": { status: 401 } });

    const wrapper = renderCallback("?code=my-code");

    await waitFor(() => {
      expect(wrapper.getByTestId("url").textContent).toBe("/auth/callback");
    });
  });

  it("explains an expired or used code", async () => {
    mockFetch({ "POST /cloud/auth/session": { status: 401 } });

    const wrapper = renderCallback("?code=my-code");

    await waitFor(() => {
      expect(wrapper.getByText(/expired or was used already/)).toBeTruthy();
    });

    expect(localStorage.getItem("skc_session")).toBeNull();
  });

  it.each([
    ["email", /no verified email address/],
    ["ambiguous", /more than one Stormkit Cloud account/],
    ["state", /started in another browser/],
    ["unknown", /could not sign you in/],
  ])(
    "explains the %s error without calling the API",
    async (reason, message) => {
      const fetch = mockFetch({});

      const wrapper = renderCallback(`?error=${reason}`);

      expect(wrapper.getByText(message)).toBeTruthy();
      expect(fetch).not.toHaveBeenCalled();
    },
  );
});
