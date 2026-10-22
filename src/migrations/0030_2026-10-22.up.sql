-- Team-level Auth Wall logins.
--
-- A login used to belong to a single environment. It now belongs to a team and
-- opens the apps it is granted (auth_wall_apps), or every app of the team when
-- all_apps is set. The wall itself is still switched on per environment
-- (apps_build_conf.auth_wall_conf).
--
-- Logins sharing an email within a team are merged into the most recently used
-- one, which keeps its password and gains access to every app the merged logins
-- opened. The original rows are kept in auth_wall_backup.
CREATE TABLE IF NOT EXISTS auth_wall_backup AS TABLE auth_wall;

ALTER TABLE auth_wall ADD COLUMN IF NOT EXISTS team_id bigint;
ALTER TABLE auth_wall ADD COLUMN IF NOT EXISTS login_metadata text DEFAULT '' NOT NULL;
ALTER TABLE auth_wall ADD COLUMN IF NOT EXISTS all_apps boolean DEFAULT false NOT NULL;

-- Each login takes the team of its environment's app. Apps created before
-- teams may have no team_id; only those fall back to the owner's personal team.
UPDATE auth_wall aw
SET team_id = COALESCE(a.team_id, t.team_id)
FROM apps_build_conf e
JOIN apps a ON a.app_id = e.app_id
LEFT JOIN teams t ON t.user_id = a.user_id AND t.is_default IS TRUE
WHERE e.env_id = aw.env_id;

DELETE FROM auth_wall WHERE team_id IS NULL;

CREATE TABLE IF NOT EXISTS auth_wall_apps (
    login_id bigint NOT NULL REFERENCES auth_wall(login_id) ON DELETE CASCADE,
    app_id bigint NOT NULL REFERENCES apps(app_id) ON DELETE CASCADE,
    PRIMARY KEY (login_id, app_id)
);

CREATE INDEX IF NOT EXISTS auth_wall_apps_app_id ON auth_wall_apps (app_id);

CREATE TEMPORARY TABLE auth_wall_merge AS
SELECT
    login_id,
    env_id,
    FIRST_VALUE(login_id) OVER (
        PARTITION BY team_id, LOWER(login_email)
        ORDER BY last_login_at DESC NULLS LAST, created_at DESC, login_id DESC
    ) AS keep_id
FROM auth_wall;

INSERT INTO auth_wall_apps (login_id, app_id)
SELECT DISTINCT m.keep_id, e.app_id
FROM auth_wall_merge m
JOIN apps_build_conf e ON e.env_id = m.env_id
ON CONFLICT DO NOTHING;

DELETE FROM auth_wall aw
USING auth_wall_merge m
WHERE aw.login_id = m.login_id AND m.login_id <> m.keep_id;

DROP TABLE auth_wall_merge;

ALTER TABLE auth_wall DROP COLUMN IF EXISTS env_id;
ALTER TABLE auth_wall ALTER COLUMN team_id SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'auth_wall_team_id_fkey') THEN
        ALTER TABLE auth_wall
            ADD CONSTRAINT auth_wall_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(team_id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS auth_wall_team_id_login_email ON auth_wall (team_id, LOWER(login_email));
