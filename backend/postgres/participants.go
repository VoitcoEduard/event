package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/user/go-expo-sample/backend/models"
)

const participantColumns = `id,event_id::text,COALESCE(join_key,''),display_name,normalized_name,COALESCE(team_id::text,''),profile,status,created_at,updated_at`

func scanParticipant(row scanner) (*models.Participant, error) {
	var p models.Participant
	var profile []byte
	if err := row.Scan(&p.ID, &p.EventID, &p.JoinKey, &p.DisplayName, &p.NormalizedName, &p.TeamID, &profile, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, translate(err)
	}
	if err := unmarshal(profile, &p.Profile); err != nil {
		return nil, err
	}
	return &p, nil
}

// EnsureParticipant returns the same participant for repeated event/device or
// event/join-key registration. Join keys do not merge unrelated event records.
func (s *Store) EnsureParticipant(ctx context.Context, p *models.Participant) (*models.Participant, error) {
	var profile []byte
	var err error
	if p.Profile != nil {
		profile, err = marshal(p.Profile)
		if err != nil {
			return nil, err
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if p.JoinKey != "" {
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, p.EventID+"/"+p.JoinKey); err != nil {
			return nil, translate(err)
		}
		existing, findErr := scanParticipant(tx.QueryRow(ctx, `SELECT `+participantColumns+` FROM participants WHERE event_id=$1 AND join_key=$2 FOR UPDATE`, p.EventID, p.JoinKey))
		if findErr == nil {
			if p.ID != "" && p.ID != existing.ID {
				return nil, ErrConflict
			}
			p.ID = existing.ID
		} else if !errors.Is(findErr, ErrNotFound) {
			return nil, findErr
		}
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	row := tx.QueryRow(ctx, `INSERT INTO participants(event_id,id,join_key,display_name,normalized_name,team_id,profile,status,created_at,updated_at)
        VALUES ($1,$2,NULLIF($3,''),$4,$5,NULLIF($6,'')::uuid,COALESCE($7::jsonb,'{}'::jsonb),$8,$9,$10)
        ON CONFLICT(event_id,id) DO UPDATE SET
        display_name=CASE WHEN participants.display_name='' THEN EXCLUDED.display_name ELSE participants.display_name END,
        normalized_name=CASE WHEN participants.display_name='' THEN EXCLUDED.normalized_name ELSE participants.normalized_name END,
        join_key=COALESCE(participants.join_key,EXCLUDED.join_key),
        profile=COALESCE($7::jsonb,participants.profile),updated_at=EXCLUDED.updated_at
        RETURNING `+participantColumns, p.EventID, p.ID, p.JoinKey, p.DisplayName, p.NormalizedName, p.TeamID, profile, p.Status, p.CreatedAt, p.UpdatedAt)
	out, err := scanParticipant(row)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, translate(err)
	}
	return out, nil
}
func (s *Store) Participant(ctx context.Context, eventID, id string) (*models.Participant, error) {
	return scanParticipant(s.Pool.QueryRow(ctx, `SELECT `+participantColumns+` FROM participants WHERE event_id=$1 AND id=$2`, eventID, id))
}
func (s *Store) Participants(ctx context.Context, eventID string) ([]models.Participant, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+participantColumns+` FROM participants WHERE event_id=$1 ORDER BY created_at,id`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.Participant{}
	for rows.Next() {
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}
func (s *Store) UpdateParticipant(ctx context.Context, p *models.Participant) error {
	profile, err := marshal(p.Profile)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE participants SET join_key=NULLIF($3,''),display_name=$4,normalized_name=$5,team_id=NULLIF($6,'')::uuid,profile=$7,status=$8,updated_at=$9 WHERE event_id=$1 AND id=$2`, p.EventID, p.ID, p.JoinKey, p.DisplayName, p.NormalizedName, p.TeamID, profile, p.Status, p.UpdatedAt)
	return affected(tag, err)
}
func (s *Store) SetParticipantName(ctx context.Context, eventID, id, name, normalized string) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE participants SET display_name=$3,normalized_name=$4,updated_at=now() WHERE event_id=$1 AND id=$2`, eventID, id, name, normalized)
	return affected(tag, err)
}
func (s *Store) AssignParticipantTeam(ctx context.Context, eventID, id, teamID string) (*models.Participant, error) {
	return scanParticipant(s.Pool.QueryRow(ctx, `UPDATE participants SET team_id=NULLIF($3,'')::uuid,updated_at=now() WHERE event_id=$1 AND id=$2 RETURNING `+participantColumns, eventID, id, teamID))
}
func (s *Store) ClearTeamAssignments(ctx context.Context, eventID, teamID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE participants SET team_id=NULL,updated_at=now() WHERE event_id=$1 AND team_id=$2`, eventID, teamID)
	return translate(err)
}

func scanTeam(row scanner) (*models.Team, error) {
	var team models.Team
	err := row.Scan(&team.ID, &team.EventID, &team.Name, &team.Color, &team.CreatedAt)
	if err != nil {
		return nil, translate(err)
	}
	return &team, nil
}

const teamColumns = `id::text,event_id::text,name,color,created_at`

func (s *Store) CreateTeam(ctx context.Context, t *models.Team) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO teams(id,event_id,name,color,created_at) VALUES ($1,$2,$3,$4,$5)`, t.ID, t.EventID, t.Name, t.Color, t.CreatedAt)
	return translate(err)
}
func (s *Store) Team(ctx context.Context, id string) (*models.Team, error) {
	return scanTeam(s.Pool.QueryRow(ctx, `SELECT `+teamColumns+` FROM teams WHERE id=$1`, id))
}
func (s *Store) Teams(ctx context.Context, eventID string) ([]models.Team, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+teamColumns+` FROM teams WHERE event_id=$1 ORDER BY created_at,id`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	result := []models.Team{}
	for rows.Next() {
		team, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *team)
	}
	return result, rows.Err()
}
func (s *Store) UpdateTeam(ctx context.Context, t *models.Team) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE teams SET name=$3,color=$4 WHERE id=$1 AND event_id=$2`, t.ID, t.EventID, t.Name, t.Color)
	return affected(tag, err)
}
func (s *Store) DeleteTeam(ctx context.Context, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var eventID string
	err = tx.QueryRow(ctx, `SELECT event_id::text FROM teams WHERE id=$1 FOR UPDATE`, id).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE participants SET team_id=NULL,updated_at=now() WHERE event_id=$1 AND team_id=$2`, eventID, id); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM teams WHERE id=$1`, id); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}
func (s *Store) DeleteTeamsForEvent(ctx context.Context, eventID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE participants SET team_id=NULL,updated_at=now() WHERE event_id=$1`, eventID); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM teams WHERE event_id=$1`, eventID); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}

// AutoAssignParticipants assigns only unassigned participants in one transaction.
func (s *Store) AutoAssignParticipants(ctx context.Context, eventID string) ([]models.Participant, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT `+teamColumns+` FROM teams WHERE event_id=$1 ORDER BY created_at,id FOR UPDATE`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	teams := []models.Team{}
	for rows.Next() {
		v, e := scanTeam(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		teams = append(teams, *v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, ErrNotFound
	}
	rows, err = tx.Query(ctx, `SELECT `+participantColumns+` FROM participants WHERE event_id=$1 AND team_id IS NULL ORDER BY created_at,id FOR UPDATE`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	people := []models.Participant{}
	for rows.Next() {
		v, e := scanParticipant(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		people = append(people, *v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range people {
		people[i].TeamID = teams[i%len(teams)].ID
		people[i].UpdatedAt = time.Now().UTC()
		if _, err = tx.Exec(ctx, `UPDATE participants SET team_id=$3,updated_at=$4 WHERE event_id=$1 AND id=$2`, eventID, people[i].ID, people[i].TeamID, people[i].UpdatedAt); err != nil {
			return nil, translate(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, translate(err)
	}
	return people, nil
}
