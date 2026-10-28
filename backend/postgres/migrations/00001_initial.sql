-- +goose Up
CREATE TABLE events (
    id uuid PRIMARY KEY,
    pin text NOT NULL UNIQUE,
    name text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT '',
    is_active boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL,
    screensaver_config jsonb,
    screen_state jsonb,
    slides jsonb NOT NULL DEFAULT '[]',
    participant_policy jsonb,
    scoring_policy jsonb
);
CREATE INDEX events_created_at_idx ON events (created_at DESC);

CREATE TABLE modules (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    position bigint NOT NULL,
    type text NOT NULL,
    name text NOT NULL DEFAULT '',
    is_active boolean NOT NULL DEFAULT false,
    data jsonb NOT NULL DEFAULT '{}',
    inputs jsonb,
    outputs jsonb,
    activation_rules jsonb,
    UNIQUE (event_id, id),
    UNIQUE (event_id, position)
);

CREATE TABLE teams (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name text NOT NULL,
    color text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    UNIQUE (event_id, id)
);

CREATE TABLE participants (
    event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    id text NOT NULL,
    join_key text,
    display_name text NOT NULL DEFAULT '',
    normalized_name text NOT NULL DEFAULT '',
    team_id uuid,
    profile jsonb NOT NULL DEFAULT '{}',
    status text NOT NULL DEFAULT 'joined',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (event_id, id),
    FOREIGN KEY (event_id, team_id) REFERENCES teams(event_id, id)
);
CREATE UNIQUE INDEX participants_join_key_idx ON participants (event_id, join_key) WHERE join_key IS NOT NULL AND join_key <> '';
CREATE INDEX participants_normalized_name_idx ON participants (event_id, normalized_name);
CREATE INDEX participants_team_idx ON participants (event_id, team_id);

CREATE TABLE templates (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    module_type text NOT NULL,
    data jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL
);
CREATE TABLE dynamic_cards (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    is_active boolean NOT NULL DEFAULT false,
    data jsonb NOT NULL
);
CREATE INDEX dynamic_cards_active_idx ON dynamic_cards (event_id, is_active);

CREATE TABLE event_sessions (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    status text NOT NULL,
    active_module_id uuid,
    module_order jsonb NOT NULL DEFAULT '[]',
    module_states jsonb,
    started_at timestamptz,
    ended_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (event_id, id),
    FOREIGN KEY (event_id, active_module_id) REFERENCES modules(event_id, id) ON DELETE SET NULL (active_module_id)
);
CREATE INDEX event_sessions_event_idx ON event_sessions (event_id);

CREATE TABLE module_attempts (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    participant_id text NOT NULL,
    module_id uuid NOT NULL,
    status text NOT NULL,
    score_delta double precision NOT NULL DEFAULT 0,
    result_data jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE,
    FOREIGN KEY (event_id, participant_id) REFERENCES participants(event_id, id) ON DELETE CASCADE
);
CREATE INDEX module_attempts_event_module_idx ON module_attempts (event_id, module_id);
CREATE INDEX module_attempts_participant_idx ON module_attempts (participant_id);

CREATE TABLE shared_state (
    id uuid PRIMARY KEY,
    event_session_id uuid NOT NULL REFERENCES event_sessions(id) ON DELETE CASCADE,
    key text NOT NULL,
    scope text NOT NULL,
    producer_module_id text NOT NULL DEFAULT '',
    consumer_module_ids jsonb,
    value_type text NOT NULL,
    value jsonb,
    updated_at timestamptz NOT NULL,
    UNIQUE (event_session_id, key)
);

CREATE TABLE responses (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    module_id uuid NOT NULL,
    user_id text NOT NULL,
    data jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL,
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE
);
CREATE INDEX responses_event_module_idx ON responses (event_id, module_id);

CREATE TABLE quiz_runs (
    id uuid PRIMARY KEY,
    sequence bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
    event_id uuid NOT NULL,
    module_id uuid NOT NULL,
    finished_at timestamptz NOT NULL,
    player_signals jsonb NOT NULL DEFAULT '[]',
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE
);
CREATE INDEX quiz_runs_latest_idx ON quiz_runs (event_id, module_id, finished_at DESC, sequence DESC);
CREATE TABLE quiz_run_players (
    run_id uuid NOT NULL REFERENCES quiz_runs(id) ON DELETE CASCADE,
    participant_id text NOT NULL,
    nickname text NOT NULL,
    score integer NOT NULL,
    PRIMARY KEY (run_id, participant_id)
);
CREATE TABLE team_results (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    module_id uuid NOT NULL,
    module_name text NOT NULL,
    finished_at timestamptz NOT NULL,
    teams jsonb NOT NULL DEFAULT '[]',
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE
);
CREATE INDEX team_results_event_module_idx ON team_results (event_id, module_id, finished_at DESC);
CREATE TABLE score_entries (
    id text PRIMARY KEY,
    event_id uuid NOT NULL,
    module_id uuid NOT NULL,
    participant_id text,
    team_id text,
    category text NOT NULL,
    points double precision NOT NULL CHECK (points NOT IN ('NaN'::float8, 'Infinity'::float8, '-Infinity'::float8)),
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    CHECK ((participant_id IS NULL) <> (team_id IS NULL)),
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE
);
CREATE INDEX score_entries_event_module_idx ON score_entries (event_id, module_id);
CREATE INDEX score_entries_participant_idx ON score_entries (event_id, participant_id);

CREATE TABLE teams_activities (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    module_id uuid NOT NULL,
    status text NOT NULL,
    revision bigint NOT NULL,
    deleted boolean NOT NULL DEFAULT false,
    data jsonb NOT NULL,
    UNIQUE (event_id, module_id),
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE
);
CREATE INDEX teams_activities_event_idx ON teams_activities (event_id) WHERE NOT deleted;

CREATE TABLE simple_quiz_sessions (
    module_id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    status text NOT NULL,
    current_index integer NOT NULL DEFAULT 0,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL,
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE CASCADE
);
CREATE TABLE simple_quiz_players (
    module_id uuid NOT NULL REFERENCES simple_quiz_sessions(module_id) ON DELETE CASCADE,
    id text NOT NULL,
    nickname text NOT NULL,
    secret text NOT NULL,
    data jsonb NOT NULL,
    PRIMARY KEY (module_id, id)
);
CREATE UNIQUE INDEX simple_quiz_nickname_idx ON simple_quiz_players (module_id, lower(nickname));
CREATE TABLE simple_quiz_answers (
    module_id uuid NOT NULL REFERENCES simple_quiz_sessions(module_id) ON DELETE CASCADE,
    id uuid NOT NULL,
    player_id text NOT NULL,
    question_index integer NOT NULL,
    data jsonb NOT NULL,
    PRIMARY KEY (module_id, id),
    UNIQUE (module_id, player_id, question_index),
    FOREIGN KEY (module_id, player_id) REFERENCES simple_quiz_players(module_id, id) ON DELETE CASCADE
);
CREATE TABLE simple_quiz_grades (
    module_id uuid NOT NULL,
    answer_id uuid NOT NULL,
    data jsonb NOT NULL,
    PRIMARY KEY (module_id, answer_id),
    FOREIGN KEY (module_id, answer_id) REFERENCES simple_quiz_answers(module_id, id) ON DELETE CASCADE
);

CREATE TABLE media (
    id uuid PRIMARY KEY,
    storage_key text NOT NULL UNIQUE,
    filename text NOT NULL,
    original_name text NOT NULL,
    mime_type text NOT NULL,
    size bigint NOT NULL,
    width integer NOT NULL DEFAULT 0,
    height integer NOT NULL DEFAULT 0,
    event_id uuid REFERENCES events(id) ON DELETE SET NULL,
    module_id uuid,
    created_at timestamptz NOT NULL,
    deleted_at timestamptz,
    CHECK (module_id IS NULL OR event_id IS NOT NULL),
    FOREIGN KEY (event_id, module_id) REFERENCES modules(event_id, id) ON DELETE SET NULL
);
CREATE INDEX media_event_idx ON media (event_id) WHERE deleted_at IS NULL;
CREATE INDEX media_module_idx ON media (module_id) WHERE deleted_at IS NULL;
CREATE TABLE screensavers (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    media_id uuid NOT NULL REFERENCES media(id),
    thumbnail_url text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE TABLE tracks (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    media_id uuid NOT NULL REFERENCES media(id),
    url text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE auth_users (
    id text PRIMARY KEY,
    issuer text NOT NULL,
    subject text NOT NULL,
    email text NOT NULL DEFAULT '',
    name text NOT NULL DEFAULT '',
    role text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (issuer, subject)
);
CREATE TABLE auth_sessions (
    hash text PRIMARY KEY,
    user_id text NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    csrf text NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX auth_sessions_expiry_idx ON auth_sessions (expires_at);
CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id);
CREATE TABLE auth_logins (
    hash text PRIMARY KEY,
    browser text NOT NULL,
    nonce text NOT NULL,
    verifier text NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX auth_logins_expiry_idx ON auth_logins (expires_at);
CREATE TABLE auth_audit (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor text NOT NULL,
    target text NOT NULL,
    action text NOT NULL,
    at timestamptz NOT NULL
);

-- +goose Down
-- Forward-only migrations: restore a matching backup instead of downgrading.
SELECT 1 / 0;
