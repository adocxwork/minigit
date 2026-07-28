package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"

	"mgit/pkg/core"
	"mgit/pkg/utils"
)

// UIAssets holds the embedded frontend files
var UIAssets fs.FS

type API struct {
	RepoRoot string
}

func StartServer(repoRoot, port string) error {
	api := &API{RepoRoot: repoRoot}
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", api.handleStatus)
	mux.HandleFunc("/api/add", api.handleAdd)
	mux.HandleFunc("/api/commit", api.handleCommit)
	mux.HandleFunc("/api/log", api.handleLog)
	mux.HandleFunc("/api/branch", api.handleBranch)
	mux.HandleFunc("/api/checkout", api.handleCheckout)
	mux.HandleFunc("/api/merge", api.handleMerge)
	mux.HandleFunc("/api/reset", api.handleReset)

	// Serve the embedded static frontend
	if UIAssets != nil {
		fsys, err := fs.Sub(UIAssets, "web/dist")
		if err != nil {
			return err
		}
		mux.Handle("/", http.FileServer(http.FS(fsys)))
	}

	fmt.Printf("Starting UI server on http://localhost:%s\n", port)
	return http.ListenAndServe(":"+port, mux)
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	staged, modified, untracked, err := core.GetStatus(a.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	branch, _ := core.GetCurrentBranch(a.RepoRoot)
	headCommit, _ := core.GetHEADCommit(a.RepoRoot)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"staged":        staged,
		"modified":      modified,
		"untracked":     untracked,
		"currentBranch": branch,
		"headCommit":    headCommit,
	})
}

func (a *API) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Paths []string `json:"paths"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, p := range req.Paths {
		if err := core.Add(a.RepoRoot, p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Message string `json:"message"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	headHash, _ := core.GetHEADCommit(a.RepoRoot)
	var parents []string
	if headHash != "" {
		parents = []string{headHash}
	}

	if err := core.CreateCommit(a.RepoRoot, req.Message, parents); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commitHash, _ := core.GetHEADCommit(a.RepoRoot)
	commits := []core.Commit{}

	for commitHash != "" {
		commit, err := core.ReadCommit(a.RepoRoot, commitHash)
		if err != nil {
			break
		}
		commits = append(commits, *commit)
		if len(commit.Parents) > 0 {
			commitHash = commit.Parents[0]
		} else {
			commitHash = ""
		}
	}

	json.NewEncoder(w).Encode(commits)
}

func (a *API) handleBranch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		branches := []string{}
		branchDir := filepath.Join(a.RepoRoot, core.MgitDir, "refs", "heads")
		
		utils.Exists(branchDir) // ensure it's loaded
		
		err := filepath.Walk(branchDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(branchDir, path)
			branches = append(branches, rel)
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		curr, _ := core.GetCurrentBranch(a.RepoRoot)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"branches": branches,
			"current":  curr,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct{ Name string `json:"name"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		headCommit, _ := core.GetHEADCommit(a.RepoRoot)
		if err := core.CreateBranch(a.RepoRoot, req.Name, headCommit); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (a *API) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Target string `json:"target"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := core.Checkout(a.RepoRoot, req.Target); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ TargetBranch string `json:"targetBranch"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := core.Merge(a.RepoRoot, req.TargetBranch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Mode   string `json:"mode"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := core.Reset(a.RepoRoot, req.Mode, req.Target); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

