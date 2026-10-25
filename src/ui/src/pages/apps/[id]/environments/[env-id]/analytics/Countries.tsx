import type { TimeSpan } from "./index.d";
import { useContext, useMemo } from "react";
import WorldMap from "react-svg-worldmap";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { RootContext } from "~/pages/Root.context";
import LinearProgress from "@mui/material/LinearProgress";
import { grey } from "@mui/material/colors";
import { useFetchAnalyticsReport } from "./actions";
import { spanText } from "./helpers";

interface Props {
  environment: Environment;
  domain?: Domain;
  ts: TimeSpan;
}

export default function Countries({ environment, domain, ts }: Props) {
  const { mode } = useContext(RootContext);
  const { report, error, loading } = useFetchAnalyticsReport({
    envId: environment.id!,
    domainId: domain?.id,
    query: { metric: "countries", ts, limit: 50 },
  });

  const countries = useMemo(
    () =>
      (report?.rows || []).map(row => ({
        country: row.country!,
        value: row.visitors || 0,
      })),
    [report],
  );

  const isDark = mode === "dark";

  return (
    <Card
      error={error}
      loading={loading && !report}
      sx={{
        mt: 2,
        width: "100%",
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <CardHeader
        title="Countries"
        subtitle={`Visitors per country ${spanText(ts)}.`}
      />
      {loading && report && <LinearProgress color="secondary" />}
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          bgcolor: isDark ? "black" : grey[200],
          mb: 4,
        }}
      >
        <WorldMap
          backgroundColor={isDark ? "transparent" : grey[200]}
          borderColor={isDark ? "white" : "black"}
          color="white"
          size="responsive"
          valueSuffix="visitors"
          richInteraction
          data={countries}
        />
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          The top 50 countries. Visitors are counted per day. Double click to
          zoom in and out.
        </Typography>
      </CardFooter>
    </Card>
  );
}
