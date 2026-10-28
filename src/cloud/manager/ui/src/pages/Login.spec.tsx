import { describe, expect, it } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { mockFetch } from "../testing/fetch";
import Login from "./Login";

describe("Login", () => {
  it("links each configured provider to its sign-in endpoint", async () => {
    mockFetch({
      "GET /cloud/auth/providers": {
        body: { providers: ["github", "google"] },
      },
    });

    const wrapper = render(<Login />);

    await waitFor(() => {
      expect(
        wrapper
          .getByText("Continue with GitHub")
          .closest("a")
          ?.getAttribute("href"),
      ).toBe("http://api.test/cloud/auth/github");
    });

    expect(
      wrapper
        .getByText("Continue with Google")
        .closest("a")
        ?.getAttribute("href"),
    ).toBe("http://api.test/cloud/auth/google");

    expect(wrapper.queryByText("Continue with GitLab")).toBeNull();
  });

  it("explains when the providers cannot be loaded", async () => {
    mockFetch({ "GET /cloud/auth/providers": { status: 503 } });

    const wrapper = render(<Login />);

    await waitFor(() => {
      expect(
        wrapper.getByText(/Sign-in is unavailable right now/),
      ).toBeTruthy();
    });
  });
});
