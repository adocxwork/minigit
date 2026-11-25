package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"mgit/pkg/core"
)

var UIAssets fs.FS

type API struct {
	RepoRoot string
	mu       sync.Mutex
}

func StartServer(repoRoot, port string) error {
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("invalid port: %q", port)
	}
	if UIAssets == nil {
		return fmt.Errorf("web UI is unavailable in this build; build with -tags ui after building the frontend")
	}
	assets, err := fs.Sub(UIAssets, "web/dist")
	if err != nil {
		return err
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(number))
	server := &http.Server{
		Addr:              address,
		Handler:           (&API{RepoRoot: repoRoot}).handler(assets),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return err
	}
	fmt.Printf("Starting UI server on http://%s\n", address)
	return server.Serve(listener)
}

func (a *API) handler(assets fs.FS) http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("GET /api/overview", a.handleOverview)
	api.HandleFunc("GET /api/status", a.handleStatus)
	api.HandleFunc("POST /api/add", a.handleAdd)
	api.HandleFunc("POST /api/commit", a.handleCommit)
	api.HandleFunc("GET /api/log", a.handleLog)
	api.HandleFunc("GET /api/branch", a.handleBranches)
	api.HandleFunc("POST /api/branch", a.handleCreateBranch)
	api.HandleFunc("POST /api/checkout", a.handleCheckout)
	api.HandleFunc("POST /api/merge", a.handleMerge)
	api.HandleFunc("POST /api/reset", a.handleReset)

	mux := http.NewServeMux()
	mux.Handle("/api/", a.protectAPI(api))
	if assets != nil {
		static := http.FileServer(http.FS(assets))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			static.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !localHost(r.Host) {
			http.Error(w, "Local host required", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		mux.ServeHTTP(w, r)
	})
}

func localHost(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	return strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "[::1]" || host == "::1"
}

func (a *API) protectAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		origin := r.Header.Get("Origin")
		fetchSite := r.Header.Get("Sec-Fetch-Site")
		if (origin != "" && origin != "http://"+r.Host) ||
			(fetchSite != "" && fetchSite != "same-origin" && fetchSite != "none") {
			http.Error(w, "Same origin required", http.StatusForbidden)
			return
		}

		a.mu.Lock()
		defer a.mu.Unlock()
		unlock, err := core.LockRepository(a.RepoRoot)
		if err != nil {
			if errors.Is(err, core.ErrRepositoryBusy) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer unlock()
		next.ServeHTTP(w, r)
	})
}

func decodeRequest(w http.ResponseWriter, r *http.Request, value any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "Expected application/json", http.StatusUnsupportedMediaType)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(value)
	if err == nil {
		if err = decoder.Decode(new(any)); errors.Is(err, io.EOF) {
			return true
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
	} else {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
	}
	return false
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(value)
}

func operationResult(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) repositoryStatus() (map[string]any, error) {
	staged, modified, untracked, err := core.GetStatus(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	branch, err := core.GetCurrentBranch(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	head, err := core.GetHEADCommit(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	merging, conflicts, err := core.MergeStatus(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"staged": staged, "modified": modified, "untracked": untracked,
		"currentBranch": branch, "headCommit": head, "merging": merging, "conflicts": conflicts,
	}, nil
}

func (a *API) handleStatus(w http.ResponseWriter, _ *http.Request) {
	status, err := a.repositoryStatus()
	respond(w, status, err)
}

func (a *API) handleOverview(w http.ResponseWriter, _ *http.Request) {
	status, err := a.repositoryStatus()
	if err != nil {
		respond(w, nil, err)
		return
	}
	branches, err := a.branches()
	if err != nil {
		respond(w, nil, err)
		return
	}
	log, err := a.history()
	respond(w, map[string]any{
		"name": filepath.Base(a.RepoRoot), "status": status, "branchList": branches, "log": log,
	}, err)
}

func respond(w http.ResponseWriter, result any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (a *API) handleAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
	}
	if decodeRequest(w, r, &req) {
		operationResult(w, core.AddPaths(a.RepoRoot, req.Paths))
	}
}

func (a *API) handleCommit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if decodeRequest(w, r, &req) {
		operationResult(w, core.CreateCommit(a.RepoRoot, req.Message, nil))
	}
}

type logEntry struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Parents   []string  `json:"parents"`
}

func (a *API) history() ([]logEntry, error) {
	commits, err := core.History(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	entries := make([]logEntry, 0, len(commits))
	for _, commit := range commits {
		parents := append([]string{}, commit.Parents...)
		entries = append(entries, logEntry{commit.ID, commit.Message, commit.Timestamp, parents})
	}
	return entries, nil
}

func (a *API) handleLog(w http.ResponseWriter, _ *http.Request) {
	history, err := a.history()
	respond(w, history, err)
}

func (a *API) branches() (map[string]any, error) {
	branches, err := core.ListBranches(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	current, err := core.GetCurrentBranch(a.RepoRoot)
	return map[string]any{"branches": branches, "current": current}, err
}

func (a *API) handleBranches(w http.ResponseWriter, _ *http.Request) {
	branches, err := a.branches()
	respond(w, branches, err)
}

func (a *API) handleCreateBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeRequest(w, r, &req) {
		return
	}
	head, err := core.GetHEADCommit(a.RepoRoot)
	if err == nil {
		err = core.CreateBranch(a.RepoRoot, req.Name, head)
	}
	operationResult(w, err)
}

func (a *API) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
	}
	if decodeRequest(w, r, &req) {
		operationResult(w, core.Checkout(a.RepoRoot, req.Target))
	}
}

func (a *API) handleMerge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetBranch string `json:"targetBranch"`
	}
	if decodeRequest(w, r, &req) {
		operationResult(w, core.Merge(a.RepoRoot, req.TargetBranch))
	}
}

func (a *API) handleReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode   string `json:"mode"`
		Target string `json:"target"`
	}
	if decodeRequest(w, r, &req) {
		operationResult(w, core.Reset(a.RepoRoot, req.Mode, req.Target))
	}
}
