import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, afterEach, it, vi } from "vitest";
import { waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import nock from "nock";
import { mockTrustedLogin } from "~/testing/nocks/nock_user";
import { renderWithRouter } from "~/testing/helpers";
import api from "~/utils/api/Api";
import { AuthContext } from "./Auth.context";
import TrustedLogin from "./TrustedLogin";

describe("~/pages/auth/TrustedLogin.tsx", () => {
  const originalLocation = window.location;

  let wrapper: RenderResult;
  let replaceSpy: ReturnType<typeof vi.fn>;

  const createWrapper = (code: string | null) => {
    wrapper = renderWithRouter({
      initialEntries: ["/auth/callback"],
      el: () => (
        <AuthContext.Provider value={{ trustedLoginCode: code }}>
          <TrustedLogin />
        </AuthContext.Provider>
      ),
    });
  };

  beforeEach(() => {
    vi.clearAllMocks();
    replaceSpy = vi.fn();

    Object.defineProperty(window, "location", {
      value: { ...originalLocation, replace: replaceSpy },
      writable: true,
    });
  });

  afterEach(() => {
    api.removeAuthToken();

    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
    });
  });

  it("redeems the code, stores the session and opens the dashboard", async () => {
    const scope = mockTrustedLogin({
      code: "my-code",
      response: { sessionToken: "my-session-token" },
    });

    createWrapper("my-code");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(api.getAuthToken()).toBe("my-session-token");
      expect(replaceSpy).toHaveBeenCalledWith("/");
    });
  });

  it("explains that the link was rejected without offering a retry", async () => {
    const scope = mockTrustedLogin({ code: "my-code", status: 401 });

    createWrapper("my-code");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText(/could not sign you in with this link/),
      ).toBeTruthy();
    });

    expect(wrapper.queryByRole("button", { name: "Try again" })).toBeNull();
    expect(api.getAuthToken()).toBeUndefined();
    expect(replaceSpy).not.toHaveBeenCalled();
  });

  it.each([429, 503])(
    "retries the same code after a %s answer",
    async status => {
      const failed = mockTrustedLogin({ code: "my-code", status });

      createWrapper("my-code");

      await waitFor(() => {
        expect(failed.isDone()).toBe(true);
        expect(wrapper.getByText(/Try again in a minute/)).toBeTruthy();
      });

      const retried = mockTrustedLogin({
        code: "my-code",
        response: { sessionToken: "my-session-token" },
      });

      await userEvent.click(wrapper.getByRole("button", { name: "Try again" }));

      await waitFor(() => {
        expect(retried.isDone()).toBe(true);
        expect(replaceSpy).toHaveBeenCalledWith("/");
      });
    },
  );

  it("does not offer a retry when the code was spent without a session", async () => {
    const scope = mockTrustedLogin({ code: "my-code", response: {} });

    createWrapper("my-code");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(/Something went wrong/)).toBeTruthy();
    });

    expect(wrapper.queryByRole("button", { name: "Try again" })).toBeNull();
    expect(replaceSpy).not.toHaveBeenCalled();
  });

  it("explains when trusted login is disabled on the instance", async () => {
    const scope = mockTrustedLogin({ code: "my-code", status: 404 });

    createWrapper("my-code");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(/not enabled on this instance/)).toBeTruthy();
    });
  });

  it("points to the owner when trusted login is misconfigured", async () => {
    const scope = mockTrustedLogin({ code: "my-code", status: 500 });

    createWrapper("my-code");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(/misconfigured on this instance/)).toBeTruthy();
    });

    expect(wrapper.queryByRole("button", { name: "Try again" })).toBeNull();
  });

  it("does not call the api without a code", async () => {
    const scope = nock(process.env.API_DOMAIN || "")
      .post("/auth/trusted")
      .reply(200, { sessionToken: "my-session-token" });

    createWrapper(null);

    await waitFor(() => {
      expect(
        wrapper.getByText(/could not sign you in with this link/),
      ).toBeTruthy();
    });

    expect(scope.isDone()).toBe(false);
    expect(replaceSpy).not.toHaveBeenCalled();

    nock.cleanAll();
  });
});
