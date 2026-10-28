import { useEffect, useState } from "react";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { apiUrl, request } from "../api";

const labels: Record<string, string> = {
  github: "Continue with GitHub",
  gitlab: "Continue with GitLab",
  google: "Continue with Google",
};

export default function Login() {
  const [providers, setProviders] = useState<string[]>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    request<{ providers: string[] }>("GET", "/cloud/auth/providers")
      .then(({ providers }) => setProviders(providers))
      .catch(() =>
        setError("Sign-in is unavailable right now. Try again in a minute."),
      );
  }, []);

  return (
    <Stack spacing={3} sx={{ maxWidth: 360, width: "100%", m: "auto", p: 4 }}>
      <Typography variant="h5" component="h1" textAlign="center">
        Sign in to Stormkit Cloud
      </Typography>
      {error && <Alert severity="error">{error}</Alert>}
      {!providers && !error && (
        <CircularProgress sx={{ alignSelf: "center" }} />
      )}
      {providers?.length === 0 && (
        <Alert severity="info">No sign-in methods are configured.</Alert>
      )}
      {providers?.map((provider) => (
        <Button
          key={provider}
          variant="contained"
          size="large"
          href={`${apiUrl}/cloud/auth/${provider}`}
        >
          {labels[provider] || provider}
        </Button>
      ))}
    </Stack>
  );
}
