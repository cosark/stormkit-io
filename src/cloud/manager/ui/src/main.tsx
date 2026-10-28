import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import CssBaseline from "@mui/material/CssBaseline";
import { AccountProvider } from "./Account.context";
import App from "./App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <CssBaseline />
    <BrowserRouter>
      <AccountProvider>
        <App />
      </AccountProvider>
    </BrowserRouter>
  </StrictMode>,
);
