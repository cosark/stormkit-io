import { useCallback, useContext, useEffect, useRef, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";
import { AuthContext } from "./Auth.context";

const rejected =
  "We could not sign you in with this link. It may have expired or been used already, or your account may not have access to this instance. Open the instance again, and if it keeps failing, ask the instance owner for access.";

const unavailable =
  "The identity provider could not be reached. Try again in a minute.";

const errorMessages: Record<number, string> = {
  400: rejected,
  401: rejected,
  404: "Signing in through an identity provider is not enabled on this instance.",
  429: "Too many sign-in attempts. Try again in a minute.",
  500: "Signing in through an identity provider is misconfigured on this instance. Contact the instance owner.",
  502: unavailable,
  503: unavailable,
  504: unavailable,
};

const defaultError = "Something went wrong while signing you in.";

// These answers come before the server spends the code, so retrying it can
// still succeed. 0 is a request that never got an answer.
const retryable = new Set([0, 429, 502, 503, 504]);

/**
 * TrustedLogin completes a login started by an identity provider, such as
 * Stormkit Cloud opening a customer's instance. The provider redirects here
 * with a one-time code, which the API redeems for a session.
 */
export default function TrustedLogin() {
  const { trustedLoginCode: code } = useContext(AuthContext);
  const [error, setError] = useState<string>();
  const [canRetry, setCanRetry] = useState(false);

  const redeem = useCallback(() => {
    setError(undefined);
    setCanRetry(false);

    api
      .post<{ sessionToken?: string }>("/auth/trusted", { code })
      .then(({ sessionToken }) => {
        // The server spent the code, so this cannot be retried.
        if (!sessionToken) {
          setError(defaultError);
          return;
        }

        api.setAuthToken(sessionToken);
        window.location.replace("/");
      })
      .catch((res: { status?: number }) => {
        const status = res?.status || 0;

        setError(errorMessages[status] || defaultError);
        setCanRetry(retryable.has(status));
      });
  }, [code]);

  // The code is single use: redeeming it twice, as React does with effects
  // in development, would make the second attempt fail.
  const started = useRef(false);

  useEffect(() => {
    if (started.current) {
      return;
    }

    started.current = true;

    if (!code) {
      setError(rejected);
      return;
    }

    redeem();
  }, []);

  return (
    <Box sx={{ maxWidth: "600px", width: "100%", m: "auto", p: 4 }}>
      <Card error={error} sx={{ width: "100%" }}>
        <CardHeader
          title={error ? "Could not sign you in" : "Signing you in"}
          subtitle={error ? undefined : "This only takes a moment."}
          sx={{ textAlign: "center" }}
        />
        {!error && (
          <Box sx={{ display: "flex", justifyContent: "center" }}>
            <CircularProgress />
          </Box>
        )}
        {canRetry && (
          <CardFooter>
            <Button variant="contained" color="secondary" onClick={redeem}>
              Try again
            </Button>
          </CardFooter>
        )}
      </Card>
    </Box>
  );
}
