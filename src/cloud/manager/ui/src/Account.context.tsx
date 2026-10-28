import { createContext, useCallback, useEffect, useState } from "react";
import type { ReactNode } from "react";
import { request, session } from "./api";
import type { Account } from "./api";

interface AccountContextProps {
  account?: Account;
  loading: boolean;
  signIn: (token: string, account: Account) => void;
  signOut: () => Promise<void>;
}

export const AccountContext = createContext<AccountContextProps>({
  loading: false,
  signIn: () => {},
  signOut: async () => {},
});

export function AccountProvider({ children }: { children: ReactNode }) {
  const [account, setAccount] = useState<Account>();
  const [loading, setLoading] = useState(() => session.token() !== null);

  useEffect(() => {
    if (!session.token()) {
      return;
    }

    request<{ account: Account }>("GET", "/cloud/account")
      .then(({ account }) => setAccount(account))
      .catch(() => setAccount(undefined))
      .finally(() => setLoading(false));
  }, []);

  const signIn = useCallback((token: string, account: Account) => {
    session.set(token);
    setAccount(account);
  }, []);

  const signOut = useCallback(async () => {
    try {
      await request("DELETE", "/cloud/auth/session");
    } catch {
      // The session is forgotten locally either way.
    }

    session.clear();
    setAccount(undefined);
  }, []);

  return (
    <AccountContext.Provider value={{ account, loading, signIn, signOut }}>
      {children}
    </AccountContext.Provider>
  );
}
