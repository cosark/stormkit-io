export const apiUrl = (
  import.meta.env.VITE_API_URL || "https://api.stormkit.io"
).replace(/\/+$/, "");

const tokenKey = "skc_session";

export const session = {
  token(): string | null {
    try {
      return localStorage.getItem(tokenKey);
    } catch {
      return null;
    }
  },

  set(token: string) {
    try {
      localStorage.setItem(tokenKey, token);
    } catch {
      // Without storage the session lasts until the page reloads.
    }
  },

  clear() {
    try {
      localStorage.removeItem(tokenKey);
    } catch {
      // Nothing to clear.
    }
  },
};

export class ApiError extends Error {
  status: number;

  constructor(status: number) {
    super(`request failed with ${status}`);
    this.status = status;
  }
}

interface RequestOptions {
  // anonymous requests carry no session token, so their 401s say nothing
  // about the stored session.
  anonymous?: boolean;
}

/**
 * request calls the instance manager API with the session token, if any.
 * A 401 clears the stored session, unless a newer sign-in replaced the
 * token while the request was in flight.
 */
export async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  options: RequestOptions = {},
): Promise<T> {
  const headers: Record<string, string> = {};
  const token = options.anonymous ? null : session.token();

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  let res: Response;

  try {
    res = await fetch(`${apiUrl}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0);
  }

  if (res.status === 401 && token && session.token() === token) {
    session.clear();
  }

  if (!res.ok) {
    throw new ApiError(res.status);
  }

  const text = await res.text();

  return (text ? JSON.parse(text) : {}) as T;
}

export interface Account {
  id: string;
  displayName: string;
  avatarUrl: string;
  emails: string[];
  createdAt: number;
}
