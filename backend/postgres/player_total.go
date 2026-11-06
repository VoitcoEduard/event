package postgres

import "context"

// PlayerTotal reads only one participant's contributions. The four terms mirror
// LoadEventScores: latest quiz run per module, current published Simple Quiz,
// manual entries, and recipient awards from team activities.
func (s *Store) PlayerTotal(ctx context.Context, eventID, playerID string) (float64, error) {
	const query = `
WITH latest AS (
    SELECT DISTINCT ON (r.module_id) r.id
    FROM quiz_runs r JOIN modules m ON m.event_id=r.event_id AND m.id=r.module_id
    WHERE r.event_id=$1 AND m.type IN ('quiz','wrong_pictures')
    ORDER BY r.module_id,r.finished_at DESC,r.sequence DESC
)
SELECT
    COALESCE((SELECT SUM(p.score)::double precision FROM latest l JOIN quiz_run_players p ON p.run_id=l.id WHERE p.participant_id=$2),0)
  + COALESCE((SELECT SUM((player.value->>'totalScore')::double precision)
      FROM simple_quiz_sessions q
      JOIN modules m ON m.event_id=q.event_id AND m.id=q.module_id AND m.type='simple_quiz'
      CROSS JOIN LATERAL jsonb_array_elements(COALESCE(q.data #> '{published,players}','[]'::jsonb)) AS player(value)
      LEFT JOIN simple_quiz_players sp ON sp.module_id=q.module_id AND sp.id=(player.value->>'playerId')
      WHERE q.event_id=$1 AND COALESCE(NULLIF(sp.data->>'participantId',''),player.value->>'playerId')=$2),0)
  + COALESCE((SELECT SUM(e.points)::double precision FROM score_entries e
      JOIN modules m ON m.event_id=e.event_id AND m.id=e.module_id
      WHERE e.event_id=$1 AND e.participant_id=$2),0)
  + COALESCE((SELECT SUM((award.value->>'points')::double precision)
      FROM teams_activities a
      JOIN modules m ON m.event_id=a.event_id AND m.id=a.module_id
      CROSS JOIN LATERAL jsonb_array_elements(COALESCE(a.data->'awards','[]'::jsonb)) AS award(value)
      WHERE a.event_id=$1 AND NOT a.deleted AND (award.value->'recipients') ? $2),0)`
	var total float64
	err := s.Pool.QueryRow(ctx, query, eventID, playerID).Scan(&total)
	return total, translate(err)
}
