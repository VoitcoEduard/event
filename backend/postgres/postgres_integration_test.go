//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/user/go-expo-sample/backend/filestore"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

func startDB(t *testing.T) (context.Context, *postgres.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	c, err := pgcontainer.Run(ctx, "postgres:18.6", pgcontainer.WithDatabase("testdb"), pgcontainer.WithUsername("testuser"), pgcontainer.WithPassword("testpassword"), pgcontainer.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(c); err != nil {
			t.Error(err)
		}
	})
	url, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return ctx, store
}

func TestMigrationsAndConstraints(t *testing.T) {
	ctx, store := startDB(t)
	// Independent calls exercise initialization of the version table and migration locking.
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < cap(errs); i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- store.Migrate(ctx) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent migration: %v", err)
		}
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("ready: %v", err)
	}
	v, target, err := store.Versions(ctx)
	if err != nil || v != 1 || target != 1 {
		t.Fatalf("versions %d/%d: %v", v, target, err)
	}

	e1, e2, m1 := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := store.Pool.Exec(ctx, `INSERT INTO events (id,pin,created_at) VALUES ($1,'123456',now()),($2,'654321',now())`, e1, e2); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, `INSERT INTO events (id,pin,created_at) VALUES ($1,'123456',now())`, uuid.NewString()); err == nil {
		t.Fatal("duplicate event PIN accepted")
	}
	if _, err := store.Pool.Exec(ctx, `INSERT INTO modules (id,event_id,position,type) VALUES ($1,$2,0,'form')`, m1, e1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, `INSERT INTO responses (id,event_id,module_id,user_id,created_at) VALUES ($1,$2,$3,'p',now())`, uuid.NewString(), e2, m1); err == nil {
		t.Fatal("cross-event module accepted")
	}
	if _, err := store.Pool.Exec(ctx, `INSERT INTO participants (event_id,id,display_name,created_at,updated_at) VALUES ($1,'same-device','Name',now(),now()),($2,'same-device','Name',now(),now())`, e1, e2); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, `INSERT INTO participants (event_id,id,join_key,created_at,updated_at) VALUES ($1,'p1','key',now(),now())`, e1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, `INSERT INTO participants (event_id,id,join_key,created_at,updated_at) VALUES ($1,'p2','key',now(),now())`, e1); err == nil {
		t.Fatal("duplicate join key accepted")
	}

	newEvent := &models.Event{ID: uuid.NewString(), Pin: "111111", Name: "Event", CreatedAt: time.Now().UTC(), Modules: []models.Module{{ID: uuid.NewString(), Type: models.ModuleTypeForm, Name: "Welcome", Data: map[string]any{"a": "one", "b": "two"}}}}
	if err := store.CreateEvent(ctx, newEvent); err != nil {
		t.Fatalf("create event: %v", err)
	}
	if err := store.PatchModule(ctx, newEvent.ID, newEvent.Modules[0].ID, map[string]any{"a": "changed"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetEvent(ctx, newEvent.ID)
	if err != nil || len(got.Modules) != 1 || got.Modules[0].Data["a"] != "changed" || got.Modules[0].Data["b"] != "two" {
		t.Fatalf("event round trip: %+v, %v", got, err)
	}
	person := &models.Participant{ID: "device-1", EventID: newEvent.ID, DisplayName: "Anna", NormalizedName: "anna", Status: models.ParticipantStatusJoined, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	first, err := store.EnsureParticipant(ctx, person)
	if err != nil {
		t.Fatalf("first participant: %v", err)
	}
	second, err := store.EnsureParticipant(ctx, person)
	if err != nil || second.ID != first.ID {
		t.Fatalf("repeat participant: %+v %v", second, err)
	}
	listed, err := store.Participants(ctx, newEvent.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("participants: %+v %v", listed, err)
	}

	quizID := uuid.NewString()
	if err := store.AddModule(ctx, newEvent.ID, models.Module{ID: quizID, Type: models.ModuleTypeQuiz, Name: "Quiz", Data: map[string]any{}}); err != nil {
		t.Fatalf("add module: %v", err)
	}
	baseTime := time.Now().UTC()
	firstRun, latestRun := uuid.NewString(), uuid.NewString()
	oldResult := &models.QuizResult{EventID: newEvent.ID, ModuleID: quizID, FinishedAt: baseTime, Players: []models.PlayerResult{{ID: "old", Score: 10}, {ID: "device-1", Score: 20}}}
	newResult := &models.QuizResult{EventID: newEvent.ID, ModuleID: quizID, FinishedAt: baseTime, Players: []models.PlayerResult{{ID: "device-1", Score: 30}}}
	if err := store.SaveQuizResult(ctx, firstRun, oldResult); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveQuizResult(ctx, latestRun, newResult); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveQuizResult(ctx, latestRun, newResult); err != nil {
		t.Fatal(err)
	}
	runs, err := store.QuizResults(ctx, newEvent.ID, quizID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("runs: %+v %v", runs, err)
	}
	updated, err := store.UpdateLatestScore(ctx, newEvent.ID, quizID, "old", 100)
	if err != nil || updated {
		t.Fatalf("edited historical player: %v %v", updated, err)
	}
	updated, err = store.UpdateLatestScore(ctx, newEvent.ID, quizID, "device-1", 40)
	if err != nil || !updated {
		t.Fatalf("edit latest: %v %v", updated, err)
	}
	latest, err := store.LatestQuizResults(ctx, newEvent.ID)
	if err != nil || len(latest) != 1 || len(latest[0].Players) != 1 || latest[0].Players[0].Score != 40 {
		t.Fatalf("latest runs: %+v %v", latest, err)
	}
	if err = store.SetManualScore(ctx, newEvent.ID, quizID, "players", "device-1", 2.5); err != nil {
		t.Fatal(err)
	}
	if err = store.SetManualScore(ctx, newEvent.ID, quizID, "players", "device-1", 3.5); err != nil {
		t.Fatal(err)
	}
	entries, err := store.ScoreEntries(ctx, newEvent.ID)
	if err != nil || len(entries) != 1 || entries[0].Points != 3.5 {
		t.Fatalf("manual retries: %+v %v", entries, err)
	}

	user, err := store.UpsertGoogleUser(ctx, "issuer", "subject", "a@example.com", "Anna")
	if err != nil || user.Status != "pending" {
		t.Fatalf("user: %+v %v", user, err)
	}
	login := postgres.AuthLogin{Hash: "state", Browser: "browser", Nonce: "nonce", Verifier: "verifier", ExpiresAt: time.Now().Add(time.Minute)}
	if err = store.CreateAuthLogin(ctx, login); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ConsumeAuthLogin(ctx, "state", "wrong", time.Now()); err == nil {
		t.Fatal("wrong browser accepted")
	}
	if _, err = store.ConsumeAuthLogin(ctx, "state", "browser", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ConsumeAuthLogin(ctx, "state", "browser", time.Now()); err == nil {
		t.Fatal("login replay accepted")
	}
	if err = store.CreateAuthSession(ctx, postgres.AuthSession{Hash: "session", UserID: user.ID, CSRF: "csrf", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err = store.SetAuthUserStatus(ctx, "owner", user.ID, "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.ResolveAuthSession(ctx, "session", time.Now()); err == nil {
		t.Fatal("disabled user's session survived")
	}
	if _, err = store.PromoteOwner(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.SetAuthUserStatus(ctx, "owner", user.ID, "disabled"); err != postgres.ErrForbidden {
		t.Fatalf("owner status change: %v", err)
	}

	simpleID := uuid.NewString()
	if err = store.AddModule(ctx, newEvent.ID, models.Module{ID: simpleID, Type: models.ModuleTypeSimpleQuiz, Name: "Text", Data: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	simple := &models.SimpleQuizSession{EventID: newEvent.ID, ModuleID: simpleID, Status: models.SimpleQuizStateQuestionOpen, CurrentIndex: 0, Questions: []models.SimpleQuizQuestionSnapshot{{ID: "q1", Text: "Question"}}, Players: map[string]*models.SimpleQuizPlayer{}, Answers: []models.SimpleQuizAnswer{}, Grades: map[string]*models.SimpleQuizGrade{}, CreatedAt: now, UpdatedAt: now}
	if err = store.SaveSimpleQuiz(ctx, nil, simple); err != nil {
		t.Fatal(err)
	}
	copySession := func(in *models.SimpleQuizSession) *models.SimpleQuizSession {
		b, e := json.Marshal(in)
		if e != nil {
			t.Fatal(e)
		}
		var out models.SimpleQuizSession
		if e = json.Unmarshal(b, &out); e != nil {
			t.Fatal(e)
		}
		return &out
	}
	next := copySession(simple)
	next.Players["p1"] = &models.SimpleQuizPlayer{ID: "p1", Nickname: "Player", Secret: "secret", JoinedAt: now}
	if err = store.SaveSimpleQuiz(ctx, simple, next); err != nil {
		t.Fatal(err)
	}
	simple = next
	next = copySession(simple)
	firstAnswer := uuid.NewString()
	next.Answers = append(next.Answers, models.SimpleQuizAnswer{ID: firstAnswer, PlayerID: "p1", QuestionIndex: 0, Text: "first", CreatedAt: now})
	if err = store.SaveSimpleQuiz(ctx, simple, next); err != nil {
		t.Fatal(err)
	}
	simple = next
	var before string
	if err = store.Pool.QueryRow(ctx, `SELECT xmin::text FROM simple_quiz_answers WHERE module_id=$1 AND id=$2`, simpleID, firstAnswer).Scan(&before); err != nil {
		t.Fatal(err)
	}
	next = copySession(simple)
	next.Players["p2"] = &models.SimpleQuizPlayer{ID: "p2", Nickname: "Another", Secret: "secret2", JoinedAt: now}
	next.Answers = append(next.Answers, models.SimpleQuizAnswer{ID: uuid.NewString(), PlayerID: "p2", QuestionIndex: 0, Text: "second", CreatedAt: now})
	if err = store.SaveSimpleQuiz(ctx, simple, next); err != nil {
		t.Fatal(err)
	}
	simple = next
	var after string
	if err = store.Pool.QueryRow(ctx, `SELECT xmin::text FROM simple_quiz_answers WHERE module_id=$1 AND id=$2`, simpleID, firstAnswer).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("previous answer row rewritten")
	}
	next = copySession(simple)
	next.Answers = append(next.Answers, models.SimpleQuizAnswer{ID: uuid.NewString(), PlayerID: "p1", QuestionIndex: 0, Text: "duplicate", CreatedAt: now})
	if err = store.SaveSimpleQuiz(ctx, simple, next); err == nil {
		t.Fatal("duplicate answer accepted")
	}
	restored, err := store.ReadSimpleQuiz(ctx, simpleID)
	if err != nil || len(restored.Answers) != 2 {
		t.Fatalf("restore: %+v %v", restored, err)
	}
	paused, err := store.PauseRestoredSimpleQuiz(ctx, restored)
	if err != nil || paused.Status != models.SimpleQuizStatePaused {
		t.Fatalf("pause: %+v %v", paused, err)
	}
	loaded, err := store.ReadSimpleQuiz(ctx, simpleID)
	if err != nil || loaded.Status != models.SimpleQuizStatePaused {
		t.Fatalf("persisted pause: %+v %v", loaded, err)
	}

	root := t.TempDir()
	files, err := filestore.Open(root, store)
	if err != nil {
		t.Fatal(err)
	}
	media, err := files.Save(ctx, filestore.Upload{OriginalName: "../original.wav", Filename: "safe.wav", MimeType: "audio/wav", Data: []byte("example"), Kind: "track", Name: "Test"})
	if err != nil {
		t.Fatalf("save media: %v", err)
	}
	_, payload, err := files.Read(ctx, media.Media.ID)
	if err != nil || string(payload) != "example" {
		t.Fatalf("media read: %q %v", payload, err)
	}
	tracks, err := store.Tracks(ctx)
	if err != nil || len(tracks) != 1 {
		t.Fatalf("tracks: %+v %v", tracks, err)
	}
	deletedID, err := files.DeleteLibrary(ctx, "tracks", media.Track.ID)
	if err != nil || deletedID != media.Media.ID {
		t.Fatalf("delete track: %s %v", deletedID, err)
	}
	if _, _, err = files.Read(ctx, media.Media.ID); err == nil {
		t.Fatal("deleted file still visible")
	}
	if err = files.Reconcile(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, media.Media.StorageKey)); !os.IsNotExist(err) {
		t.Fatalf("file remains after cleanup: %v", err)
	}
	_, err = files.Save(ctx, filestore.Upload{OriginalName: "bad", Filename: "bad", MimeType: "image/png", EventID: uuid.NewString(), Data: []byte("invalid owner")})
	if err == nil {
		t.Fatal("invalid media owner accepted")
	}
	filesLeft, err := os.ReadDir(root)
	if err != nil || len(filesLeft) != 0 {
		t.Fatalf("failed upload left files: %v %v", filesLeft, err)
	}

	// Unsupported future versions must not be accepted as ready or migrated over.
	if _, err := store.Pool.Exec(ctx, `INSERT INTO goose_db_version(version_id,is_applied) VALUES (99,true)`); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("future schema reported ready")
	}
	if err := store.Migrate(ctx); err == nil || !strings.Contains(fmt.Sprint(err), "newer") {
		t.Fatalf("future schema migration: %v", err)
	}
}

func TestDeletionPreservesSharedMedia(t *testing.T) {
	ctx, store := startDB(t)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	files, err := filestore.Open(t.TempDir(), store)
	if err != nil {
		t.Fatal(err)
	}
	for i, kind := range []string{"module", "event"} {
		for j, reference := range []string{"template", "module", "url", "library"} {
			t.Run(kind+"/"+reference, func(t *testing.T) {
				sourceEvent, sourceModule, otherEvent, otherModule := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
				for k, pair := range [][2]string{{sourceEvent, sourceModule}, {otherEvent, otherModule}} {
					event := &models.Event{ID: pair[0], Pin: fmt.Sprintf("9%d%d%d00", i, j, k), CreatedAt: time.Now(), Modules: []models.Module{{ID: pair[1], Type: models.ModuleTypeQuiz}}}
					if err := store.CreateEvent(ctx, event); err != nil {
						t.Fatal(err)
					}
				}
				upload := filestore.Upload{Filename: "image.png", OriginalName: "image.png", MimeType: "image/png", EventID: sourceEvent, ModuleID: sourceModule, Data: []byte("shared image")}
				shared, err := files.Save(ctx, upload)
				if err != nil {
					t.Fatal(err)
				}
				upload.Data = []byte("unused image")
				unused, err := files.Save(ctx, upload)
				if err != nil {
					t.Fatal(err)
				}
				data := map[string]any{"questions": []any{map[string]any{"imageId": shared.Media.ID}}}
				if reference == "url" {
					data = map[string]any{"nested": []any{map[string]any{"url": "/media/" + shared.Media.ID + "/file"}}}
				}
				if reference == "library" {
					_, err = store.Pool.Exec(ctx, `INSERT INTO tracks(id,name,media_id,url,created_at) VALUES ($1,'Shared track',$2,$3,now())`, uuid.NewString(), shared.Media.ID, "/media/"+shared.Media.ID+"/file")
				} else if reference == "template" {
					err = store.CreateTemplate(ctx, &models.Template{ID: uuid.NewString(), Name: "Reusable", ModuleType: models.ModuleTypeQuiz, Data: data, CreatedAt: time.Now()})
				} else {
					err = store.PatchModule(ctx, otherEvent, otherModule, data)
				}
				if err != nil {
					t.Fatal(err)
				}
				if kind == "event" {
					err = store.DeleteEvent(ctx, sourceEvent)
				} else {
					var found bool
					found, err = store.DeleteModule(ctx, sourceEvent, sourceModule)
					if err == nil && !found {
						t.Fatal("module was not deleted")
					}
				}
				if err != nil {
					t.Fatalf("delete %s: %v", kind, err)
				}
				if _, err = store.GetModule(ctx, sourceEvent, sourceModule); err != postgres.ErrNotFound {
					t.Fatalf("source module survived: %v", err)
				}
				if _, err = store.Media(ctx, unused.Media.ID); err != postgres.ErrNotFound {
					t.Fatalf("unused media remains visible: %v", err)
				}
				if err = files.Reconcile(ctx, time.Now()); err != nil {
					t.Fatal(err)
				}
				metadata, payload, err := files.Read(ctx, shared.Media.ID)
				if err != nil || string(payload) != "shared image" {
					t.Fatalf("shared media lost: %q %v", payload, err)
				}
				if metadata.ModuleID != "" || (kind == "event" && metadata.EventID != "") {
					t.Fatalf("stale ownership: %+v", metadata)
				}
				exists, err := store.MediaStorageKeyExists(ctx, unused.Media.StorageKey)
				if err != nil || exists {
					t.Fatalf("unused media not reconciled: %v %v", exists, err)
				}
			})
		}
	}
}
