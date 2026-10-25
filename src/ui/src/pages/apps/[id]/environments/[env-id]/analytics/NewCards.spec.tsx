import type { AnalyticsReport, TimeSpan } from "./index.d";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockDomain from "~/testing/data/mock_domain";
import { mockFetchAnalyticsReport } from "~/testing/nocks/nock_analytics";
import { reportRange } from "./actions";
import Channels from "./Channels";
import EntryPages, { topChannels } from "./EntryPages";
import Campaigns, { campaignName } from "./Campaigns";

const report = (
  metric: string,
  rows: AnalyticsReport["rows"],
): AnalyticsReport => ({
  metric,
  range: { from: "2024-01-08", to: "2024-01-14", days: 7 },
  totals: {},
  rows,
});

describe("analytics report cards", () => {
  const env = mockEnvironments({ app: mockApp() })[0];
  const domain = mockDomain();

  const mock = (
    ts: TimeSpan,
    query: Record<string, string>,
    response: AnalyticsReport,
  ) =>
    mockFetchAnalyticsReport({
      query: {
        envId: env.id!,
        domainId: domain.id,
        ...reportRange(ts),
        ...query,
      },
      response,
    });

  describe("Channels", () => {
    it("lists visits per channel with their change", async () => {
      const scope = mock(
        "7d",
        { metric: "channels", compare: "previous" },
        report("channels", [
          { channel: "search", visits: 600, previous: 500 },
          { channel: "ai", visits: 90, previous: 100 },
          { channel: "internal", visits: 40 },
        ]),
      );

      const wrapper = render(
        <Channels environment={env} domain={domain} ts="7d" />,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("Search")).toBeTruthy();
        expect(wrapper.getByText("+20%")).toBeTruthy();
        expect(wrapper.getByText("AI assistants")).toBeTruthy();
        expect(wrapper.getByText("-10%")).toBeTruthy();
        expect(wrapper.getByText("Within this site")).toBeTruthy();
      });
    });

    it("does not compare the partial 24 hours", async () => {
      const scope = mock(
        "24h",
        { metric: "channels" },
        report("channels", [{ channel: "direct", visits: 5 }]),
      );

      const wrapper = render(
        <Channels environment={env} domain={domain} ts="24h" />,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("Direct")).toBeTruthy();
      });
    });
  });

  describe("Entry pages", () => {
    it("lists landing pages with their top channels", async () => {
      const scope = mock(
        "30d",
        { metric: "entries", limit: "50" },
        report("entries", [
          {
            path: "/blog/a",
            visits: 70,
            channels: { search: 40, direct: 20, ai: 6, email: 4 },
          },
        ]),
      );

      const wrapper = render(
        <EntryPages environment={env} domain={domain} ts="30d" />,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("/blog/a")).toBeTruthy();
        expect(
          wrapper.getByText("Search 40 · Direct 20 · AI assistants 6"),
        ).toBeTruthy();
      });
    });

    it("summarizes the three largest channels", () => {
      expect(topChannels({ social: 1, search: 9, direct: 9 })).toBe(
        "Direct 9 · Search 9 · Social 1",
      );
      expect(topChannels()).toBe("");
    });
  });

  describe("Campaigns", () => {
    it("names campaigns by their parameters", async () => {
      const scope = mock(
        "7d",
        { metric: "campaigns", limit: "50" },
        report("campaigns", [
          {
            utmSource: "newsletter",
            utmMedium: "email",
            utmCampaign: "launch",
            visits: 30,
          },
          { ref: "producthunt", visits: 12 },
        ]),
      );

      const wrapper = render(
        <Campaigns environment={env} domain={domain} ts="7d" />,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("newsletter / email / launch")).toBeTruthy();
        expect(wrapper.getByText("ref=producthunt")).toBeTruthy();
      });
    });

    it("explains how to tag links when there are none", async () => {
      const scope = mock(
        "7d",
        { metric: "campaigns", limit: "50" },
        report("campaigns", []),
      );
      const wrapper = render(
        <Campaigns environment={env} domain={domain} ts="7d" />,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText(/Tag the links you share/)).toBeTruthy();
      });
    });

    it("joins UTM parameters and ref", () => {
      expect(campaignName({ utmSource: "x", ref: "hn" })).toBe("x · ref=hn");
      expect(campaignName({ utmSource: "(other)" })).toBe("(other)");
    });
  });
});
