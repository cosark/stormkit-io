import type { ReactNode } from "react";
import type { AnalyticsReport, AnalyticsRow } from "./index.d";
import Box from "@mui/material/Box";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import LinearProgress from "@mui/material/LinearProgress";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";

interface Props {
  title: string;
  subtitle: ReactNode;
  footer?: ReactNode;
  empty: ReactNode;
  report?: AnalyticsReport;
  loading: boolean;
  error: string;
  fullWidth?: boolean;
  renderRow: (row: AnalyticsRow, index: number) => ReactNode;
}

// ReportCard lists the rows of an analytics report, with the loading, error
// and empty states the report cards share.
export default function ReportCard({
  title,
  subtitle,
  footer,
  empty,
  report,
  loading,
  error,
  fullWidth,
  renderRow,
}: Props) {
  const rows = report?.rows || [];

  return (
    <Card
      error={error}
      loading={loading && !report}
      sx={{
        width: { xs: "100%", md: fullWidth ? "100%" : "50%" },
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <CardHeader title={title} subtitle={subtitle} />
      {loading && report && <LinearProgress color="secondary" />}
      <Box sx={{ maxHeight: "300px", overflow: "auto", flex: 1 }}>
        {!report ? null : rows.length ? (
          rows.map(renderRow)
        ) : (
          <Alert
            color="info"
            sx={{ mx: 2, mb: 2, bgcolor: "rgba(255,255,255,0.025)" }}
          >
            {empty}
          </Alert>
        )}
      </Box>
      {footer && (
        <CardFooter sx={{ textAlign: "left" }}>
          <Typography sx={{ fontSize: 12 }}>{footer}</Typography>
        </CardFooter>
      )}
    </Card>
  );
}
