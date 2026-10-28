import { useContext } from "react";
import type { ReactNode } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import CircularProgress from "@mui/material/CircularProgress";
import { AccountContext } from "./Account.context";
import Login from "./pages/Login";
import AuthCallback from "./pages/AuthCallback";
import Instances from "./pages/Instances";

function RequireAccount({ children }: { children: ReactNode }) {
  const { account, loading } = useContext(AccountContext);

  if (loading) {
    return <CircularProgress sx={{ display: "block", m: "auto", mt: 8 }} />;
  }

  return account ? children : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route
        path="/"
        element={
          <RequireAccount>
            <Instances />
          </RequireAccount>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
