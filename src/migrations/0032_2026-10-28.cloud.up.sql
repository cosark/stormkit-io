-- The instance manager (console.stormkit.io, later app.stormkit.io) keeps its
-- accounts apart from the multi-tenant dashboard's users.
CREATE SCHEMA IF NOT EXISTS cloud;

CREATE TABLE IF NOT EXISTS cloud.accounts (
    account_id bigserial PRIMARY KEY NOT NULL,
    display_name text NOT NULL DEFAULT '',
    avatar_url text NOT NULL DEFAULT '',
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    last_login_at timestamp without time zone NULL
);

-- The provider accounts someone signed in with. An identity belongs to one
-- account for good, even if its emails change later.
CREATE TABLE IF NOT EXISTS cloud.account_identities (
    identity_id bigserial PRIMARY KEY NOT NULL,
    account_id bigint NOT NULL REFERENCES cloud.accounts (account_id) ON DELETE CASCADE,
    provider text NOT NULL,
    provider_user_id text NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS account_identities_provider_user_key ON cloud.account_identities (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS account_identities_account_id_idx ON cloud.account_identities (account_id);

-- Verified emails only, lowercased. An email belongs to at most one account,
-- which is how a second provider finds the account to sign in to.
CREATE TABLE IF NOT EXISTS cloud.account_emails (
    email_id bigserial PRIMARY KEY NOT NULL,
    account_id bigint NOT NULL REFERENCES cloud.accounts (account_id) ON DELETE CASCADE,
    email text NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS account_emails_email_key ON cloud.account_emails (email);
CREATE INDEX IF NOT EXISTS account_emails_account_id_idx ON cloud.account_emails (account_id);

-- Sessions hold the SHA-256 of their bearer token, never the token itself.
CREATE TABLE IF NOT EXISTS cloud.sessions (
    session_id bigserial PRIMARY KEY NOT NULL,
    account_id bigint NOT NULL REFERENCES cloud.accounts (account_id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    expires_at timestamp without time zone NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS sessions_token_hash_key ON cloud.sessions (token_hash);
CREATE INDEX IF NOT EXISTS sessions_account_id_idx ON cloud.sessions (account_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON cloud.sessions (expires_at);
