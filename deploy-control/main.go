package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var immutableTag = regexp.MustCompile(`^sha-[0-9a-f]{40}$`)

type environment struct {
	Name            string
	StackName       string
	EndpointID      int
	ImageVariable   string
	ImageRepository string
}

type config struct {
	ListenAddr       string
	Token            string
	PortainerURL     string
	PortainerAPIKey  string
	SkipTLSVerify    bool
	StatePath        string
	Environments     map[string]environment
}

type envVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type stack struct {
	ID         int      `json:"Id"`
	Name       string   `json:"Name"`
	EndpointID int      `json:"EndpointId"`
	Env        []envVar `json:"Env"`
}

type environmentState struct {
	Current  string   `json:"current,omitempty"`
	Previous []string `json:"previous,omitempty"`
}

type persistedState struct {
	Environments map[string]*environmentState `json:"environments"`
}

type controller struct {
	cfg        config
	httpClient *http.Client
	mu         sync.Mutex
	state      persistedState
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	c, err := newController(cfg)
	if err != nil {
		slog.Error("initialize controller", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", c.health)
	mux.Handle("GET /v1/environments", c.authenticate(http.HandlerFunc(c.environments)))
	mux.Handle("POST /v1/environments/{environment}/deploy", c.authenticate(http.HandlerFunc(c.deploy)))
	mux.Handle("POST /v1/environments/{environment}/rollback", c.authenticate(http.HandlerFunc(c.rollback)))

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      6 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
	slog.Info("deployment controller listening", "address", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("deployment controller stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig() (config, error) {
	c := config{
		ListenAddr:      valueOr("CONTROL_LISTEN_ADDR", ":8090"),
		Token:           os.Getenv("CONTROL_API_TOKEN"),
		PortainerURL:    strings.TrimRight(os.Getenv("PORTAINER_URL"), "/"),
		PortainerAPIKey: os.Getenv("PORTAINER_API_KEY"),
		StatePath:       valueOr("CONTROL_STATE_PATH", "/data/state.json"),
		Environments:    make(map[string]environment),
	}
	var err error
	c.SkipTLSVerify, err = strconv.ParseBool(valueOr("PORTAINER_INSECURE_SKIP_VERIFY", "false"))
	if err != nil {
		return c, fmt.Errorf("PORTAINER_INSECURE_SKIP_VERIFY: %w", err)
	}
	if len(c.Token) < 32 {
		return c, errors.New("CONTROL_API_TOKEN must contain at least 32 characters")
	}
	if c.PortainerAPIKey == "" {
		return c, errors.New("PORTAINER_API_KEY is required")
	}
	u, err := url.Parse(c.PortainerURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c, errors.New("PORTAINER_URL must be an HTTP or HTTPS origin")
	}

	for _, name := range []string{"dev", "prod"} {
		prefix := strings.ToUpper(name)
		endpointID, parseErr := strconv.Atoi(os.Getenv(prefix + "_ENDPOINT_ID"))
		if parseErr != nil || endpointID < 1 {
			return c, fmt.Errorf("%s_ENDPOINT_ID must be a positive integer", prefix)
		}
		env := environment{
			Name:            name,
			StackName:       os.Getenv(prefix + "_STACK_NAME"),
			EndpointID:      endpointID,
			ImageVariable:   os.Getenv(prefix + "_IMAGE_VARIABLE"),
			ImageRepository: strings.TrimSuffix(os.Getenv(prefix+"_IMAGE_REPOSITORY"), ":"),
		}
		if env.StackName == "" || env.ImageVariable == "" || env.ImageRepository == "" {
			return c, fmt.Errorf("%s stack name, image variable, and image repository are required", prefix)
		}
		c.Environments[name] = env
	}
	return c, nil
}

func newController(cfg config) (*controller, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	transport.TLSClientConfig.InsecureSkipVerify = cfg.SkipTLSVerify // #nosec G402 -- explicit NAS setting
	c := &controller{
		cfg:        cfg,
		httpClient: &http.Client{Transport: transport, Timeout: 5 * time.Minute},
		state:      persistedState{Environments: make(map[string]*environmentState)},
	}
	if err := c.loadState(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *controller) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(provided) != len(c.cfg.Token) || subtle.ConstantTimeCompare([]byte(provided), []byte(c.cfg.Token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (c *controller) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (c *controller) environments(w http.ResponseWriter, r *http.Request) {
	type result struct {
		Name          string   `json:"name"`
		Stack         string   `json:"stack"`
		CurrentImage  string   `json:"currentImage"`
		RollbackImages []string `json:"rollbackImages"`
	}
	results := make([]result, 0, len(c.cfg.Environments))
	for _, name := range []string{"dev", "prod"} {
		env := c.cfg.Environments[name]
		st, err := c.findStack(r.Context(), env)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		current, _ := imageValue(st.Env, env.ImageVariable)
		c.mu.Lock()
		previous := append([]string(nil), c.environmentState(name).Previous...)
		c.mu.Unlock()
		results = append(results, result{Name: name, Stack: env.StackName, CurrentImage: current, RollbackImages: previous})
	}
	writeJSON(w, http.StatusOK, map[string]any{"environments": results})
}

func (c *controller) deploy(w http.ResponseWriter, r *http.Request) {
	env, ok := c.cfg.Environments[r.PathValue("environment")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown environment"})
		return
	}
	var request struct {
		Tag string `json:"tag"`
	}
	if err := decodeJSON(r.Body, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !immutableTag.MatchString(request.Tag) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tag must be sha- followed by a full 40-character lowercase commit SHA"})
		return
	}
	target := env.ImageRepository + ":" + request.Tag
	result, err := c.changeImage(r.Context(), env, target, false)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (c *controller) rollback(w http.ResponseWriter, r *http.Request) {
	env, ok := c.cfg.Environments[r.PathValue("environment")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown environment"})
		return
	}
	result, err := c.changeImage(r.Context(), env, "", true)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errNoRollback) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

var errNoRollback = errors.New("no previous image is available for rollback")

func (c *controller) changeImage(ctx context.Context, env environment, target string, rollback bool) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	st, err := c.findStack(ctx, env)
	if err != nil {
		return nil, err
	}
	current, found := imageValue(st.Env, env.ImageVariable)
	if !found || current == "" {
		return nil, fmt.Errorf("stack %q does not define %s", env.StackName, env.ImageVariable)
	}
	state := c.environmentState(env.Name)
	if rollback {
		if len(state.Previous) == 0 {
			return nil, errNoRollback
		}
		target = state.Previous[len(state.Previous)-1]
	}
	if current == target {
		return map[string]any{"environment": env.Name, "image": target, "changed": false}, nil
	}

	updated := replaceImageValue(st.Env, env.ImageVariable, target)
	if err := c.redeploy(ctx, st.ID, env.EndpointID, updated); err != nil {
		return nil, err
	}
	if rollback {
		state.Previous = state.Previous[:len(state.Previous)-1]
	} else {
		state.Previous = append(state.Previous, current)
		if len(state.Previous) > 20 {
			state.Previous = state.Previous[len(state.Previous)-20:]
		}
	}
	state.Current = target
	if err := c.saveState(); err != nil {
		return nil, fmt.Errorf("deployment succeeded but saving rollback state failed: %w", err)
	}
	slog.Info("stack image changed", "environment", env.Name, "from", current, "to", target, "rollback", rollback)
	return map[string]any{"environment": env.Name, "previousImage": current, "image": target, "changed": true, "rollback": rollback}, nil
}

func (c *controller) findStack(ctx context.Context, env environment) (stack, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.PortainerURL+"/api/stacks", nil)
	if err != nil {
		return stack{}, err
	}
	request.Header.Set("X-API-Key", c.cfg.PortainerAPIKey)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return stack{}, fmt.Errorf("list Portainer stacks: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return stack{}, portainerError("list stacks", response)
	}
	var stacks []stack
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&stacks); err != nil {
		return stack{}, fmt.Errorf("decode Portainer stacks: %w", err)
	}
	for _, candidate := range stacks {
		if candidate.Name == env.StackName && candidate.EndpointID == env.EndpointID {
			return candidate, nil
		}
	}
	return stack{}, fmt.Errorf("Portainer stack %q was not found in endpoint %d", env.StackName, env.EndpointID)
}

func (c *controller) redeploy(ctx context.Context, stackID, endpointID int, env []envVar) error {
	body, err := json.Marshal(map[string]any{"env": env, "prune": false, "pullImage": true})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/stacks/%d/git/redeploy?endpointId=%d", c.cfg.PortainerURL, stackID, endpointID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", c.cfg.PortainerAPIKey)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("redeploy Portainer stack: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return portainerError("redeploy stack", response)
	}
	return nil
}

func portainerError(operation string, response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return fmt.Errorf("Portainer %s returned %d: %s", operation, response.StatusCode, message)
}

func imageValue(values []envVar, name string) (string, bool) {
	for _, value := range values {
		if value.Name == name {
			return value.Value, true
		}
	}
	return "", false
}

func replaceImageValue(values []envVar, name, target string) []envVar {
	result := append([]envVar(nil), values...)
	for index := range result {
		if result[index].Name == name {
			result[index].Value = target
			return result
		}
	}
	return append(result, envVar{Name: name, Value: target})
}

func (c *controller) environmentState(name string) *environmentState {
	state := c.state.Environments[name]
	if state == nil {
		state = &environmentState{}
		c.state.Environments[name] = state
	}
	return state
}

func (c *controller) loadState() error {
	data, err := os.ReadFile(c.cfg.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, &c.state); err != nil {
		return fmt.Errorf("decode state: %w", err)
	}
	if c.state.Environments == nil {
		c.state.Environments = make(map[string]*environmentState)
	}
	return nil
}

func (c *controller) saveState() error {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(c.cfg.StatePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".state-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, c.cfg.StatePath)
}

func decodeJSON(body io.ReadCloser, target any) error {
	defer body.Close()
	decoder := json.NewDecoder(io.LimitReader(body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	slog.Error("request failed", "status", status, "error", err)
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func valueOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
