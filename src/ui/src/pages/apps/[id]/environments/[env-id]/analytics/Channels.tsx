import type { TimeSpan } from "./index.d";
import Typography from "@mui/material/Typography";
import CardRow from "~/components/CardRow";
import { useFetchAnalyticsReport } from "./actions";
import { channelLabels, spanText } from "./helpers";
import { Change } from "./Visitors";
import ReportCard from "./ReportCard";

const formatter = new Intl.NumberFormat();

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

// Channels shows where visits came from: search, AI assistants, social, email,
// other sites, direct and in-site navigation.
export default function Channels({ environment, domain, ts }: Props) {
  const { report, error, loading } = useFetchAnalyticsReport({
    envId: environment.id!,
    domainId: domain?.id,
    query: { metric: "channels", ts, compare: ts !== "24h" },
  });

  return (
    <ReportCard
      title="Channels"
      subtitle={`Where visits came from ${spanText(ts)}.`}
      footer="Direct means no referrer, including apps and links that hide it."
      empty={`No visits ${spanText(ts)}.`}
      report={report}
      loading={loading}
      error={error}
      renderRow={row => (
        <CardRow
          key={row.channel}
          chipLabel={
            <Typography component="span">
              {formatter.format(row.visits || 0)}
              <Change current={row.visits} previous={row.previous} />
            </Typography>
          }
        >
          <Typography component="span">
            {channelLabels[row.channel!] || row.channel}
          </Typography>
        </CardRow>
      )}
    />
  );
}
