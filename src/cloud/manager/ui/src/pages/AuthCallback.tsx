import { useContext, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Stack from "@mui/material/Stack";
import { ApiError, request } from "../api";
import type { Account } from "../api";
import { AccountContext } from "../Account.context";

const failed = "We could not sign you in. Try again.";

export const errorMessages: Record<string, string> = {
  denied: "Sign-in was cancelled.",
  state:
    "This sign-in link expired or was started in another browser. Try again.",
  provider: "The sign-in provider did not confirm who you are. Try again.",
  email:
    "Your account at this provider has no verified email address. Verify one there, or use another provider.",
  ambiguous:
    "Your verified email addresses belong to more than one Stormkit Cloud account. Sign in with the provider you used before, or contact support.",
  server: "Something went wrong on our side. Try again in a minute.",
};

/**
 * AuthCallback completes a sign-in: the API sends the browser here with a
 * one-time code, which this page trades for a session.
 */
export default function AuthCallback() {
  const navigate = useNavigate();
  const { pathname, search } = useLocation();
  const { signIn } = useContext(AccountContext);
  const [params] = useState(() => new URLSearchParams(search));
  const [error, setError] = useState<string>(() => {
    const reason = params.get("error");

    if (reason) {
      return errorMessages[reason] || failed;
    }

    return params.get("code") ? "" : failed;
  });

  // The code is single use; keep it out of the address bar and history. The
  // page sends no Referer, so the code cannot leak before this runs.
  useEffect(() => {
    if (search) {
      navigate(pathname, { replace: true });
    }
  }, []);

  const started = useRef(false);

  useEffect(() => {
    const code = params.get("code");

    if (started.current || !code || params.get("error")) {
      return;
    }

    started.current = true;

    request<{ token: string; account: Account }>(
      "POST",
      "/cloud/auth/session",
      {
        code,
      },
    )
      .then(({ token, account }) => {
        signIn(token, account);
        navigate("/", { replace: true });
      })
      .catch((err: ApiError) => {
        setError(
          err.status === 401
            ? "This sign-in link expired or was used already. Try again."
            : failed,
        );
      });
  }, []);

  return (
    <Stack spacing={3} sx={{ maxWidth: 420, width: "100%", m: "auto", p: 4 }}>
      {error ? (
        <>
          <Alert severity="error">{error}</Alert>
          <Button
            variant="contained"
            onClick={() => navigate("/login", { replace: true })}
          >
            Back to sign in
          </Button>
        </>
      ) : (
        <CircularProgress
          sx={{ alignSelf: "center" }}
          aria-label="Signing you in"
        />
      )}
    </Stack>
  );
}
