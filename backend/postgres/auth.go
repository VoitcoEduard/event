package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrForbidden = errors.New("operation forbidden")

type AuthUser struct {
	ID        string    `json:"id"`
	Issuer    string    `json:"-"`
	Subject   string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
type AuthSession struct {
	Hash      string
	UserID    string
	CSRF      string
	ExpiresAt time.Time
}
type AuthLogin struct {
	Hash      string
	Browser   string
	Nonce     string
	Verifier  string
	ExpiresAt time.Time
}

const userColumns = `id,issuer,subject,email,name,role,status,created_at`

func scanUser(row scanner) (AuthUser, error) {
	var u AuthUser
	err := row.Scan(&u.ID, &u.Issuer, &u.Subject, &u.Email, &u.Name, &u.Role, &u.Status, &u.CreatedAt)
	return u, translate(err)
}
func (s *Store) UpsertGoogleUser(ctx context.Context, issuer, subject, email, name string) (AuthUser, error) {
	return scanUser(s.Pool.QueryRow(ctx, `INSERT INTO auth_users(id,issuer,subject,email,name,role,status,created_at) VALUES ($1,$2,$3,$4,$5,'admin','pending',now()) ON CONFLICT(issuer,subject) DO UPDATE SET email=excluded.email,name=excluded.name RETURNING `+userColumns, uuid.NewString(), issuer, subject, email, name))
}
func (s *Store) EnsureDevUser(ctx context.Context) (AuthUser, error) {
	return scanUser(s.Pool.QueryRow(ctx, `INSERT INTO auth_users(id,issuer,subject,email,name,role,status,created_at) VALUES ('development-owner','development','owner','','Development owner','owner','active',now()) ON CONFLICT(issuer,subject) DO UPDATE SET email='',name='Development owner',role='owner',status='active' RETURNING `+userColumns))
}
func (s *Store) CreateAuthSession(ctx context.Context, v AuthSession) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO auth_sessions(hash,user_id,csrf,expires_at) VALUES($1,$2,$3,$4)`, v.Hash, v.UserID, v.CSRF, v.ExpiresAt)
	return translate(err)
}
func (s *Store) ResolveAuthSession(ctx context.Context, hash string, now time.Time) (AuthSession, AuthUser, error) {
	var session AuthSession
	var user AuthUser
	err := s.Pool.QueryRow(ctx, `SELECT s.hash,s.user_id,s.csrf,s.expires_at,u.id,u.issuer,u.subject,u.email,u.name,u.role,u.status,u.created_at FROM auth_sessions s JOIN auth_users u ON u.id=s.user_id WHERE s.hash=$1 AND s.expires_at>$2`, hash, now).Scan(&session.Hash, &session.UserID, &session.CSRF, &session.ExpiresAt, &user.ID, &user.Issuer, &user.Subject, &user.Email, &user.Name, &user.Role, &user.Status, &user.CreatedAt)
	return session, user, translate(err)
}
func (s *Store) DeleteAuthSession(ctx context.Context, hash string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM auth_sessions WHERE hash=$1`, hash)
	return translate(err)
}
func (s *Store) CreateAuthLogin(ctx context.Context, v AuthLogin) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO auth_logins(hash,browser,nonce,verifier,expires_at) VALUES ($1,$2,$3,$4,$5)`, v.Hash, v.Browser, v.Nonce, v.Verifier, v.ExpiresAt)
	return translate(err)
}
func (s *Store) ConsumeAuthLogin(ctx context.Context, hash, browser string, now time.Time) (AuthLogin, error) {
	var v AuthLogin
	err := s.Pool.QueryRow(ctx, `DELETE FROM auth_logins WHERE hash=$1 AND browser=$2 AND expires_at>$3 RETURNING hash,browser,nonce,verifier,expires_at`, hash, browser, now).Scan(&v.Hash, &v.Browser, &v.Nonce, &v.Verifier, &v.ExpiresAt)
	return v, translate(err)
}
func (s *Store) AuthUsers(ctx context.Context, offset, limit int) ([]AuthUser, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+userColumns+` FROM auth_users WHERE issuer<>'development' ORDER BY created_at,id OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []AuthUser{}
	for rows.Next() {
		v, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) AuthUser(ctx context.Context, id string) (AuthUser, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userColumns+` FROM auth_users WHERE id=$1`, id))
}
func (s *Store) SetAuthUserStatus(ctx context.Context, actor, id, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New("invalid user status")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM auth_users WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if u.Role != "admin" || u.Issuer == "development" {
		return ErrForbidden
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_users SET status=$2 WHERE id=$1`, id, status); err != nil {
		return translate(err)
	}
	if status == "disabled" {
		if _, err = tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id=$1`, id); err != nil {
			return translate(err)
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO auth_audit(actor,target,action,at) VALUES ($1,$2,$3,now())`, actor, id, "set-status:"+status); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}
func (s *Store) PromoteOwner(ctx context.Context, id string) (AuthUser, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AuthUser{}, err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM auth_users WHERE id=$1 AND issuer<>'development' FOR UPDATE`, id))
	if err != nil {
		return AuthUser{}, err
	}
	if u.Role == "owner" && u.Status == "active" {
		return u, translate(tx.Commit(ctx))
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_users SET role='owner',status='active' WHERE id=$1`, id); err != nil {
		return AuthUser{}, translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id=$1`, id); err != nil {
		return AuthUser{}, translate(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO auth_audit(actor,target,action,at) VALUES ('server-cli',$1,'promote-owner',now())`, id); err != nil {
		return AuthUser{}, translate(err)
	}
	u.Role = "owner"
	u.Status = "active"
	return u, translate(tx.Commit(ctx))
}
func (s *Store) AuditAuthAction(ctx context.Context, actor, target, action string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO auth_audit(actor,target,action,at) VALUES ($1,$2,$3,now())`, actor, target, action)
	return translate(err)
}
func (s *Store) CleanupAuth(ctx context.Context, now time.Time) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM auth_sessions WHERE expires_at<=$1`, now); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_logins WHERE expires_at<=$1`, now); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}
