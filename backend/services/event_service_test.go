package services

import (
	"context"
	"strings"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

type mockEventStore struct {
	events       map[string]*models.Event
	setSlidesErr error
	setSlidesLog [][]models.Slide
}

func (m *mockEventStore) Create(_ context.Context, e *models.Event) error {
	cp := *e
	m.events[e.ID] = &cp
	return nil
}
func (m *mockEventStore) PINExists(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockEventStore) FindByPIN(_ context.Context, _ string) (*models.Event, error) {
	return nil, nil
}
func (m *mockEventStore) Get(_ context.Context, id string) (*models.Event, error) {
	return m.events[id], nil
}
func (m *mockEventStore) GetByModule(_ context.Context, _ string) (*models.Event, error) {
	return nil, nil
}
func (m *mockEventStore) List(_ context.Context) ([]models.Event, error) { return nil, nil }
func (m *mockEventStore) Update(_ context.Context, _ string, _ *models.Event) error {
	return nil
}
func (m *mockEventStore) Delete(_ context.Context, _ string) error                   { return nil }
func (m *mockEventStore) SetStatus(_ context.Context, _ string, _ bool) error        { return nil }
func (m *mockEventStore) SetScreensaver(_ context.Context, _ string, _ models.ScreensaverConfig) error {
	return nil
}
func (m *mockEventStore) SetSlides(_ context.Context, id string, v []models.Slide) error {
	if m.setSlidesErr != nil {
		return m.setSlidesErr
	}
	m.setSlidesLog = append(m.setSlidesLog, v)
	if ev, ok := m.events[id]; ok {
		ev.Slides = v
	}
	return nil
}
func (m *mockEventStore) ParticipantStatus(_ context.Context, _, _ string) (*models.Participant, float64, error) {
	return nil, 0, nil
}
func (m *mockEventStore) ActiveCards(_ context.Context, _ string) ([]models.DynamicCard, error) {
	return nil, nil
}

func TestGetSlidesIsReadOnlyAndPreservesExplicitEmptySlides(t *testing.T) {
	ctx := context.Background()
	store := &mockEventStore{
		events: map[string]*models.Event{
			"legacy": {ID: "legacy", Pin: "123456", Slides: nil},
			"empty":  {ID: "empty", Pin: "654321", Slides: []models.Slide{}},
			"custom": {ID: "custom", Pin: "111222", Slides: []models.Slide{{ID: "s1", Name: "Admin Slide"}}},
		},
	}
	svc := &EventService{store: store}

	slides, err := svc.GetSlides(ctx, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if len(slides) != 2 || !strings.Contains(slides[0].Elements[0].Content, "?pin=123456") {
		t.Fatalf("unexpected default slides: %+v", slides)
	}
	if len(store.setSlidesLog) != 0 {
		t.Fatalf("GetSlides mutated DB via SetSlides (%d calls)", len(store.setSlidesLog))
	}

	emptySlides, err := svc.GetSlides(ctx, "empty")
	if err != nil {
		t.Fatal(err)
	}
	if emptySlides == nil || len(emptySlides) != 0 {
		t.Fatalf("expected non-nil empty slice, got %+v", emptySlides)
	}
	if len(store.setSlidesLog) != 0 {
		t.Fatalf("GetSlides mutated DB on empty slides")
	}

	customSlides, err := svc.GetSlides(ctx, "custom")
	if err != nil || len(customSlides) != 1 || customSlides[0].Name != "Admin Slide" {
		t.Fatalf("unexpected custom slides: %+v %v", customSlides, err)
	}

	created := &models.Event{Name: "Fresh Event"}
	if err := svc.Create(ctx, created); err != nil {
		t.Fatal(err)
	}
	if len(created.Slides) != 2 || !strings.Contains(created.Slides[0].Elements[0].Content, "?pin="+created.Pin) {
		t.Fatalf("Create did not initialize default slides: %+v", created.Slides)
	}
}

func TestGetSlidesNormalizesDefaultWelcomeOriginToPublicBaseURL(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8081")
	ctx := context.Background()
	store := &mockEventStore{
		events: map[string]*models.Event{
			"evt-1": {
				ID:  "evt-1",
				Pin: "777613",
				Slides: []models.Slide{
					{
						ID:   "default-welcome-evt-1",
						Type: "welcome",
						Name: "Welcome",
						Elements: []models.CanvasElement{
							{ID: "default-welcome-qr-evt-1", Type: "qr", Content: "http://localhost:8080/?pin=777613"},
							{ID: "default-welcome-url-evt-1", Type: "text", Content: "http://localhost:8080/?pin=777613"},
							{ID: "default-welcome-pin-evt-1", Type: "text", Content: "PIN: 777613"},
						},
					},
				},
			},
		},
	}
	svc := &EventService{store: store}

	slides, err := svc.GetSlides(ctx, "evt-1")
	if err != nil {
		t.Fatal(err)
	}
	want := "http://localhost:8081/?pin=777613"
	if got := slides[0].Elements[0].Content; got != want {
		t.Fatalf("QR content = %q, want %q", got, want)
	}
	if got := slides[0].Elements[1].Content; got != want {
		t.Fatalf("URL text content = %q, want %q", got, want)
	}
	if got := slides[0].Elements[2].Content; got != "PIN: 777613" {
		t.Fatalf("PIN text unexpectedly changed to %q", got)
	}
	if len(store.setSlidesLog) != 0 {
		t.Fatalf("GetSlides mutated DB via SetSlides (%d calls)", len(store.setSlidesLog))
	}
}
