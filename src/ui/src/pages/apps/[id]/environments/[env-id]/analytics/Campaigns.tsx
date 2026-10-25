import type { AnalyticsRow, TimeSpan } from "./index.d";
import Typography from "@mui/material/Typography";
import CardRow from "~/components/CardRow";
import { useFetchAnalyticsReport } from "./actions";
import { spanText } from "./helpers";
import ReportCard from "./ReportCard";

const formatter = new Intl.NumberFormat();

// campaignName names a campaign row by its UTM parameters and ref.
export function campaignName(row: AnalyticsRow): string {
  const utm = [row.utmSource, row.utmMedium, row.utmCampaign]
    .filter(Boolean)
    .join(" / ");

  const ref = row.ref ? `ref=${row.ref}` : "";

  return [utm, ref].filter(Boolean).join(" · ") || "(other)";
}

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

// Campaigns lists visits by the campaign parameters of their landing URL.
export default function Campaigns({ environment, domain, ts }: Props) {
  const { report, error, loading } = useFetchAnalyticsReport({
    envId: environment.id!,
    domainId: domain?.id,
    query: { metric: "campaigns", ts, limit: 50 },
  });

  return (
    <ReportCard
      title="Campaigns"
      subtitle={`Visits by campaign ${spanText(ts)}.`}
      footer="Read from utm_source, utm_medium, utm_campaign and ref on the landing URL."
      empty={
        <>
          No campaign visits {spanText(ts)}. Tag the links you share, for
          example <code>?utm_source=newsletter&amp;utm_campaign=launch</code> or{" "}
          <code>?ref=producthunt</code>.
        </>
      }
      report={report}
      loading={loading}
      error={error}
      fullWidth
      renderRow={(row, i) => (
        <CardRow
          key={`${campaignName(row)}-${i}`}
          chipLabel={
            <Typography component="span">
              {formatter.format(row.visits || 0)}
            </Typography>
          }
        >
          <Typography component="span">{campaignName(row)}</Typography>
        </CardRow>
      )}
    />
  );
}
