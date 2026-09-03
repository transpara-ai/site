package graph

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/transpara-ai/site/profile"
)

const (
	civilizationMaxResponseBytes = 4 * 1024 * 1024
	civilizationMaxIntakeBytes   = 24 * 1024
)

type CivilizationWorkbench struct {
	Available      bool
	Notice         string
	Items          []CivilizationWork
	Repositories   []string
	IntakeIdentity string
	GeneratedAt    time.Time
}

type CivilizationWork struct {
	WorkID        string                     `json:"work_id"`
	Source        CivilizationSource         `json:"source"`
	IntakeText    string                     `json:"intake_text"`
	Bound         *CivilizationBound         `json:"bound"`
	State         string                     `json:"state"`
	ResumeState   string                     `json:"resume_state"`
	Summary       string                     `json:"summary"`
	Blocker       string                     `json:"blocker"`
	NextAction    string                     `json:"next_action"`
	ProviderRuns  []CivilizationProviderRun  `json:"provider_runs"`
	PullRequest   *CivilizationPullRequest   `json:"pull_request"`
	Interventions []CivilizationIntervention `json:"interventions"`
	MergeDecision *CivilizationMergeDecision `json:"merge_decision"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

type CivilizationSource struct {
	Kind       string `json:"kind"`
	Identity   string `json:"identity"`
	Repository string `json:"repository"`
}

type CivilizationBound struct {
	IdempotencyKey string `json:"idempotency_key"`
	Envelope       struct {
		SchemaVersion string `json:"schema_version"`
		Workflow      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"workflow"`
		Route string `json:"route"`
		Brief struct {
			Outcome     string   `json:"outcome"`
			Scope       []string `json:"scope"`
			NonGoals    []string `json:"non_goals"`
			Assumptions []string `json:"assumptions"`
			Constraints []string `json:"constraints"`
			Tests       []string `json:"tests"`
			NextAction  string   `json:"next_action"`
		} `json:"brief"`
	} `json:"envelope"`
}

type CivilizationProviderRun struct {
	Operation string `json:"operation"`
	AttemptID string `json:"attempt_id"`
	Result    struct {
		Status       string   `json:"status"`
		Summary      string   `json:"summary"`
		ChangedFiles []string `json:"changed_files"`
		Checks       []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
		Review *struct {
			Status   string   `json:"status"`
			Summary  string   `json:"summary"`
			Findings []string `json:"findings"`
		} `json:"review"`
	} `json:"result"`
}

type CivilizationPullRequest struct {
	Repository       string   `json:"repository"`
	Number           int      `json:"number"`
	URL              string   `json:"url"`
	HeadSHA          string   `json:"head_sha"`
	ReviewedHeadSHA  string   `json:"reviewed_head_sha"`
	ValidatedHeadSHA string   `json:"validated_head_sha"`
	Open             bool     `json:"open"`
	Merged           bool     `json:"merged"`
	Draft            bool     `json:"draft"`
	ChecksPassing    bool     `json:"checks_passing"`
	ChecksState      string   `json:"checks_state"`
	ChangedFiles     []string `json:"changed_files"`
}

type CivilizationIntervention struct {
	ID         string `json:"id"`
	Prompt     string `json:"prompt"`
	Status     string `json:"status"`
	Resolution string `json:"resolution"`
}

type CivilizationMergeDecision struct {
	Eligible bool     `json:"eligible"`
	Reasons  []string `json:"reasons"`
}

type civilizationClient struct {
	base   string
	key    string
	client *http.Client
}

func newCivilizationClient(timeout time.Duration) (*civilizationClient, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("CIVILIZATION_API_BASE_URL")), "/")
	if base == "" {
		return nil, errors.New("Civilization API is not configured")
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return nil, errors.New("Civilization API URL is invalid")
	}
	keyFile := strings.TrimSpace(os.Getenv("CIVILIZATION_API_KEY_FILE"))
	if keyFile == "" {
		return nil, errors.New("Civilization API credential file is not configured")
	}
	raw, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, errors.New("Civilization API credential is unavailable")
	}
	key := strings.TrimSpace(string(raw))
	if len(key) < 32 {
		return nil, errors.New("Civilization API credential is invalid")
	}
	return &civilizationClient{base: base, key: key, client: &http.Client{Timeout: timeout}}, nil
}

func (c *civilizationClient) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.key)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(request)
	if err != nil {
		return errors.New("Civilization service is unavailable")
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, civilizationMaxResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil || len(raw) > civilizationMaxResponseBytes {
		return errors.New("Civilization response is invalid")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &failure) == nil && failure.Error != "" {
			return errors.New(failure.Error)
		}
		return fmt.Errorf("Civilization request failed with status %d", response.StatusCode)
	}
	if output != nil {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		if err := decoder.Decode(output); err != nil {
			return errors.New("Civilization response schema is incompatible")
		}
	}
	return nil
}

func loadCivilizationWorkbench(ctx context.Context) CivilizationWorkbench {
	result := CivilizationWorkbench{GeneratedAt: time.Now().UTC(), Repositories: civilizationRepositories(), IntakeIdentity: newCivilizationIntakeIdentity()}
	if len(result.Repositories) == 0 {
		result.Notice = "No repositories are configured for Civilization."
		return result
	}
	client, err := newCivilizationClient(8 * time.Second)
	if err != nil {
		result.Notice = err.Error()
		return result
	}
	var response struct {
		Items []CivilizationWork `json:"items"`
	}
	if err := client.request(ctx, http.MethodGet, "/api/civilization/v1/work", nil, &response); err != nil {
		result.Notice = err.Error()
		return result
	}
	result.Available = true
	result.Items = response.Items
	return result
}

func civilizationRepositories() []string {
	seen := map[string]bool{}
	var result []string
	for _, item := range strings.Split(os.Getenv("CIVILIZATION_REPOSITORIES"), ",") {
		item = strings.TrimSpace(item)
		if strings.HasPrefix(item, "transpara-ai/") && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func newCivilizationIntakeIdentity() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("intake-%d", time.Now().UTC().UnixNano())
	}
	return "intake-" + hex.EncodeToString(raw)
}

func (h *Handlers) handleCivilizationWorkbench(response http.ResponseWriter, request *http.Request) {
	data := loadCivilizationWorkbench(request.Context())
	ConsolePage(ConsolePageData{Title: "Workbench", Active: "workbench", Workbench: &data}, h.viewUser(request), profile.FromContext(request.Context())).Render(request.Context(), response)
}

func (h *Handlers) handleCivilizationWorkbenchFragment(response http.ResponseWriter, request *http.Request) {
	CivilizationWorkbenchFragment(loadCivilizationWorkbench(request.Context())).Render(request.Context(), response)
}

func (h *Handlers) handleCivilizationIntake(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, civilizationMaxIntakeBytes)
	if err := request.ParseForm(); err != nil {
		h.renderCivilizationMutationError(response, request, "The request was too large or malformed.")
		return
	}
	repository := strings.TrimSpace(request.FormValue("repository"))
	if !containsString(civilizationRepositories(), repository) {
		h.renderCivilizationMutationError(response, request, "Choose a configured Transpara repository.")
		return
	}
	text := strings.TrimSpace(request.FormValue("text"))
	if text == "" {
		h.renderCivilizationMutationError(response, request, "Describe the outcome you want.")
		return
	}
	identity := strings.TrimSpace(request.FormValue("source_identity"))
	if identity == "" {
		identity = newCivilizationIntakeIdentity()
	}
	viewer := h.viewUser(request)
	identity = "human:" + viewer.ID + ":" + identity
	client, err := newCivilizationClient(35 * time.Minute)
	if err == nil {
		var ignored CivilizationWork
		err = client.request(request.Context(), http.MethodPost, "/api/civilization/v1/intake", map[string]string{
			"source_kind": "human", "source_identity": identity, "repository": repository, "text": text,
		}, &ignored)
	}
	if err != nil {
		h.renderCivilizationMutationError(response, request, err.Error())
		return
	}
	h.renderCivilizationAfterMutation(response, request)
}

func (h *Handlers) handleCivilizationRun(response http.ResponseWriter, request *http.Request) {
	client, err := newCivilizationClient(35 * time.Minute)
	if err == nil {
		var ignored CivilizationWork
		err = client.request(request.Context(), http.MethodPost, "/api/civilization/v1/work/"+url.PathEscape(request.PathValue("workID"))+"/run", nil, &ignored)
	}
	if err != nil {
		h.renderCivilizationMutationError(response, request, err.Error())
		return
	}
	h.renderCivilizationAfterMutation(response, request)
}

func (h *Handlers) handleCivilizationResolve(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, civilizationMaxIntakeBytes)
	if err := request.ParseForm(); err != nil || strings.TrimSpace(request.FormValue("resolution")) == "" {
		h.renderCivilizationMutationError(response, request, "Write an answer before continuing.")
		return
	}
	client, err := newCivilizationClient(35 * time.Minute)
	if err == nil {
		var ignored CivilizationWork
		path := "/api/civilization/v1/work/" + url.PathEscape(request.PathValue("workID")) + "/interventions/" + url.PathEscape(request.PathValue("interventionID")) + "/resolve"
		err = client.request(request.Context(), http.MethodPost, path, map[string]string{"resolution": strings.TrimSpace(request.FormValue("resolution"))}, &ignored)
	}
	if err != nil {
		h.renderCivilizationMutationError(response, request, err.Error())
		return
	}
	h.renderCivilizationAfterMutation(response, request)
}

func (h *Handlers) renderCivilizationAfterMutation(response http.ResponseWriter, request *http.Request) {
	if request.Header.Get("HX-Request") == "true" {
		h.handleCivilizationWorkbenchFragment(response, request)
		return
	}
	http.Redirect(response, request, "/console/workbench", http.StatusSeeOther)
}

func (h *Handlers) renderCivilizationMutationError(response http.ResponseWriter, request *http.Request, message string) {
	response.WriteHeader(http.StatusUnprocessableEntity)
	data := loadCivilizationWorkbench(request.Context())
	data.Notice = message
	CivilizationWorkbenchFragment(data).Render(request.Context(), response)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func civilizationWorkOutcome(work CivilizationWork) string {
	if work.Bound != nil && work.Bound.Envelope.Brief.Outcome != "" {
		return work.Bound.Envelope.Brief.Outcome
	}
	return work.IntakeText
}

func civilizationWorkRoute(work CivilizationWork) string {
	if work.Bound == nil {
		return "Routing"
	}
	return work.Bound.Envelope.Route
}

func civilizationWorkRunnable(work CivilizationWork) bool {
	for _, intervention := range work.Interventions {
		if intervention.Status == "open" {
			return false
		}
	}
	switch work.State {
	case "queued", "implementing", "validating", "reviewing", "publishing":
		return true
	default:
		return false
	}
}

func civilizationLatestProviderRun(work CivilizationWork, operation string) *CivilizationProviderRun {
	for index := len(work.ProviderRuns) - 1; index >= 0; index-- {
		if work.ProviderRuns[index].Operation == operation {
			return &work.ProviderRuns[index]
		}
	}
	return nil
}

func civilizationChangedFiles(work CivilizationWork) []string {
	if work.PullRequest != nil && len(work.PullRequest.ChangedFiles) > 0 {
		return work.PullRequest.ChangedFiles
	}
	if implementation := civilizationLatestProviderRun(work, "implement"); implementation != nil {
		return implementation.Result.ChangedFiles
	}
	return nil
}

func civilizationStateClass(state string) string {
	switch state {
	case "ready", "completed", "merge_queued":
		return "border-emerald-400/30 bg-emerald-400/10 text-emerald-300"
	case "blocked", "human_required":
		return "border-amber-400/30 bg-amber-400/10 text-amber-300"
	default:
		return "border-brand/30 bg-brand/10 text-brand"
	}
}
