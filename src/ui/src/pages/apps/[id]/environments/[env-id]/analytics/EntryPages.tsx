import type { TimeSpan } from "./index.d";
import Typography from "@mui/material/Typography";
import CardRow from "~/components/CardRow";
import { useFetchAnalyticsReport } from "./actions";
import { channelLabels, spanText, truncate } from "./helpers";
import ReportCard from "./ReportCard";

const formatter = new Intl.NumberFormat();

// topChannels summarizes an entry page's visits by its three largest channels.
export function topChannels(channels: Record<string, number> = {}): string {
  return Object.entries(channels)
    .sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
    .slice(0, 3)
    .map(
      ([channel, visits]) =>
        `${channelLabels[channel] || channel} ${formatter.format(visits)}`,
    )
    .join(" · ");
}

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

// EntryPages lists the pages visitors landed on from outside the site.
export default function EntryPages({ environment, domain, ts }: Props) {
  const { report, error, loading } = useFetchAnalyticsReport({
    envId: environment.id!,
    domainId: domain?.id,
    query: { metric: "entries", ts, limit: 50 },
  });

  return (
    <ReportCard
      title="Entry pages"
      subtitle={`Pages visitors landed on from outside the site ${spanText(ts)}.`}
      footer="Navigation within the site is not counted."
      empty={`No visits from outside the site ${spanText(ts)}.`}
      report={report}
      loading={loading}
      error={error}
      renderRow={row => (
        <CardRow
          key={row.path}
          chipLabel={
            <Typography component="span">
              {formatter.format(row.visits || 0)}
            </Typography>
          }
        >
          <Typography component="span">{truncate(row.path!)}</Typography>
          <Typography
            component="span"
            sx={{ display: "block", fontSize: 12, opacity: 0.6 }}
          >
            {topChannels(row.channels)}
          </Typography>
        </CardRow>
      )}
    />
  );
}
