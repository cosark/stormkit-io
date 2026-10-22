package authwall

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// envApp selects the app of the environment $1 and the app's team. Apps
// created before teams may have no team_id; only those fall back to the
// owner's personal team.
const envApp = `
	SELECT
		a.app_id, COALESCE(a.team_id, t.team_id) AS team_id
	FROM
		apps_build_conf e
	JOIN
		apps a ON a.app_id = e.app_id
	LEFT JOIN
		teams t ON t.user_id = a.user_id AND t.is_default IS TRUE
	WHERE
		e.env_id = $1
`

// opensApp is true when the login aw opens the wall of the app ea.
const opensApp = `
	aw.team_id = ea.team_id AND (
		aw.all_apps OR EXISTS (
			SELECT 1 FROM auth_wall_apps awa
			WHERE awa.login_id = aw.login_id AND awa.app_id = ea.app_id
		)
	)
`

var stmt = struct {
	createLogin       string
	removeAppAccess   string
	selectLogin       string
	selectLogins      string
	updatePassword    string
	updateLastLogin   string
	setAuthWallConfig string
}{
	createLogin: `
		WITH login AS (
			INSERT INTO auth_wall
				(team_id, login_email, login_password, login_metadata, all_apps, created_at)
			VALUES
				($1, $2, $3, $4, $5, NOW())
			RETURNING
				login_id
		), access AS (
			INSERT INTO auth_wall_apps (login_id, app_id)
			SELECT login.login_id, UNNEST($6::bigint[]) FROM login
		)
		SELECT login_id FROM login;
	`,

	// The main statement sees the access rows as they were before the delete,
	// so the app being removed is excluded when looking for remaining access.
	removeAppAccess: `
		WITH removed AS (
			DELETE FROM auth_wall_apps
			WHERE app_id = $1 AND login_id = ANY($2)
			RETURNING login_id
		)
		DELETE FROM auth_wall aw
		WHERE
			aw.login_id IN (SELECT login_id FROM removed) AND
			aw.all_apps IS FALSE AND
			NOT EXISTS (
				SELECT 1 FROM auth_wall_apps awa
				WHERE awa.login_id = aw.login_id AND awa.app_id <> $1
			);
	`,

	selectLogin: `
		WITH ea AS (` + envApp + `)
		SELECT
			aw.login_id, aw.login_email, aw.login_password, aw.login_metadata
		FROM
			auth_wall aw, ea
		WHERE
			LOWER(aw.login_email) = LOWER($2) AND ` + opensApp + `;
	`,

	selectLogins: `
		WITH ea AS (` + envApp + `)
		SELECT
			aw.login_id, aw.team_id, aw.login_email, aw.login_metadata,
			aw.all_apps, aw.last_login_at, aw.created_at
		FROM
			auth_wall aw, ea
		WHERE ` + opensApp + `
		ORDER BY
			aw.login_id DESC
		LIMIT
			50;
	`,

	updatePassword: `
		UPDATE auth_wall SET login_password = $1 WHERE login_id = $2;
	`,

	updateLastLogin: `
		UPDATE auth_wall SET last_login_at = NOW() AT TIME ZONE 'UTC' WHERE login_id = $1;
	`,

	setAuthWallConfig: `
		UPDATE apps_build_conf SET auth_wall_conf = $1 WHERE env_id = $2;
	`,
}

// dummyHash is compared against when no login matches the email, so a failed
// login takes as long whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("auth-wall-dummy-password"), bcrypt.DefaultCost)

type store struct {
	*database.Store
}

// Store returns a store instance.
func Store() *store {
	return &store{database.NewStore()}
}

// CreateLogin creates a login in aw.TeamID that opens the apps in aw.AppIDs,
// or every app of the team when aw.AllApps is set. The password is stored as a
// bcrypt hash.
func (s *store) CreateLogin(ctx context.Context, aw *AuthWall) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(aw.LoginPassword), bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	appIDs := aw.AppIDs

	if appIDs == nil {
		appIDs = []types.ID{}
	}

	params := []any{
		aw.TeamID,
		aw.LoginEmail,
		string(hash),
		aw.Metadata,
		aw.AllApps,
		pq.Array(appIDs),
	}

	row, err := s.QueryRow(ctx, stmt.createLogin, params...)

	if err != nil {
		return err
	}

	if row == nil {
		return errors.New("no row returned")
	}

	return row.Scan(&aw.LoginID)
}

// RemoveAppAccessParams are the parameters of RemoveAppAccess.
type RemoveAppAccessParams struct {
	AppID    types.ID
	LoginIDs []types.ID
}

// RemoveAppAccess takes away the access of the logins to the app. Logins left
// without access to any app are deleted.
func (s *store) RemoveAppAccess(ctx context.Context, p RemoveAppAccessParams) error {
	_, err := s.Exec(ctx, stmt.removeAppAccess, p.AppID, pq.Array(p.LoginIDs))
	return err
}

// LoginParams are the credentials submitted to the wall of an environment.
type LoginParams struct {
	EnvID    types.ID
	Email    string
	Password string
}

// Login returns the login matching the credentials when it opens the wall of
// the environment, and nil otherwise.
func (s *store) Login(ctx context.Context, p LoginParams) (*AuthWall, error) {
	row, err := s.QueryRow(ctx, stmt.selectLogin, p.EnvID, p.Email)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	aw := &AuthWall{}
	err = row.Scan(&aw.LoginID, &aw.LoginEmail, &aw.LoginPassword, &aw.Metadata)

	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(p.Password))
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if !s.verifyPassword(ctx, aw, p.Password) {
		return nil, nil
	}

	return aw, nil
}

// verifyPassword compares the password with the stored one. Logins created
// before passwords were hashed store them encrypted; those are hashed on the
// first successful login.
func (s *store) verifyPassword(ctx context.Context, aw *AuthWall, password string) bool {
	if strings.HasPrefix(aw.LoginPassword, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(aw.LoginPassword), []byte(password)) == nil
	}

	if subtle.ConstantTimeCompare([]byte(utils.DecryptToString(aw.LoginPassword)), []byte(password)) != 1 {
		return false
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err == nil {
		_, err = s.Exec(ctx, stmt.updatePassword, string(hash), aw.LoginID)
	}

	if err != nil {
		slog.Errorf("error while hashing auth wall password: %s", err.Error())
	}

	return true
}

// EnvTeamID returns the team whose logins open the wall of an environment, or
// 0 when the environment's app has no team.
func (s *store) EnvTeamID(ctx context.Context, envID types.ID) (types.ID, error) {
	row, err := s.QueryRow(ctx, envApp, envID)

	if err != nil || row == nil {
		return 0, err
	}

	var appID types.ID
	var teamID sql.NullInt64

	if err := row.Scan(&appID, &teamID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return types.ID(teamID.Int64), nil
}

// UpdateLastLogin updates the last login time.
func (s *store) UpdateLastLogin(ctx context.Context, loginID types.ID) error {
	_, err := s.Exec(ctx, stmt.updateLastLogin, loginID)
	return err
}

// Logins returns the logins that open the wall of an environment.
func (s *store) Logins(ctx context.Context, envID types.ID) ([]AuthWall, error) {
	rows, err := s.Query(ctx, stmt.selectLogins, envID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var logins []AuthWall

	for rows.Next() {
		aw := AuthWall{}
		err := rows.Scan(
			&aw.LoginID, &aw.TeamID, &aw.LoginEmail, &aw.Metadata,
			&aw.AllApps, &aw.LastLogin, &aw.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		logins = append(logins, aw)
	}

	return logins, nil
}

func (s *store) SetAuthWallConfig(ctx context.Context, envID types.ID, cfg *Config) error {
	_, err := s.Exec(ctx, stmt.setAuthWallConfig, cfg, envID)
	return err
}

// AuthWallConfig returns the configuration associated with the environment.
func (s *store) AuthWallConfig(ctx context.Context, envID types.ID) (*Config, error) {
	row, err := s.QueryRow(ctx, "SELECT auth_wall_conf FROM apps_build_conf WHERE env_id = $1", envID)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	cfg := &Config{}

	if err := row.Scan(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
