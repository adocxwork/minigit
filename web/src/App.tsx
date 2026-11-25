import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import {
  ArrowRight, ArrowRightLeft, Check, ChevronRight, CircleAlert, FileText, FolderGit2,
  GitBranch, GitCommitHorizontal, GitMerge, History, Layers3, Plus, RefreshCw,
  RotateCcw, Search, Terminal, Workflow,
} from 'lucide-react';
import { api, errorMessage, type Commit, type Overview, type ResetMode, type StatusResponse } from './api';
import { Modal } from './components/Modal';
import { useToast } from './components/toast-context';

type RunAction = (action: () => Promise<unknown>, success: string, onError?: (message: string) => void) => Promise<boolean>;

function RefreshButton({ working, onClick }: { working: boolean; onClick: () => void }) {
  return (
    <button type="button" className="button button-secondary refresh-button" onClick={onClick} disabled={working}>
      <RefreshCw size={15} className={working ? 'spinning' : ''} /> Refresh
    </button>
  );
}

function Stat({ label, value, detail, icon, color }: {
  label: string;
  value: number;
  detail: string;
  icon: ReactNode;
  color: string;
}) {
  return (
    <div className="stat">
      <div className={`stat-icon tone-${color}`}>{icon}</div>
      <div className="stat-label">{label}</div>
      <strong>{value}</strong>
      <span className="stat-detail">{detail}</span>
    </div>
  );
}

function FileGroup({ title, files, color, onStage, busy }: {
  title: string;
  files: string[];
  color: string;
  onStage?: (path: string) => void;
  busy: boolean;
}) {
  if (files.length === 0) return null;
  return (
    <section className="file-group" aria-label={title}>
      <div className="file-group-title">
        <span className={`small-dot tone-${color}`} />
        <h3>{title}</h3>
        <span className="count">{files.length}</span>
      </div>
      <ul>
        {files.map(file => (
          <li className="file-row" key={file}>
            <FileText size={15} aria-hidden="true" />
            <span title={file}>{file}</span>
            {onStage && (
              <button className="icon-button" type="button" aria-label={`Stage ${file}`} title="Stage file" onClick={() => onStage(file)} disabled={busy}>
                <Plus size={16} />
              </button>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}

function WorkingTree({ status, busy, run }: { status: StatusResponse; busy: boolean; run: RunAction }) {
  const [message, setMessage] = useState('');
  const conflicts = new Set(status.conflicts);
  const modified = status.modified.filter(path => !conflicts.has(path));
  const untracked = status.untracked.filter(path => !conflicts.has(path));
  const staged = status.staged.filter(path => !conflicts.has(path));
  const hasChanges = modified.length + untracked.length + staged.length + conflicts.size > 0;
  const canStage = modified.length + untracked.length + conflicts.size > 0;
  const canCommit = !busy && conflicts.size === 0 && (staged.length > 0 || status.merging) && message.trim() !== '';

  const stage = (path: string) => run(() => api.stage(path), path === '.' ? 'All changes staged' : `Staged ${path}`);
  const commit = async () => {
    if (canCommit && await run(() => api.commit(message.trim()), 'Commit created')) setMessage('');
  };

  return (
    <section className="panel working-tree" aria-labelledby="working-heading">
      <header className="panel-header">
        <div>
          <h2 id="working-heading">Working tree</h2>
          <p>Review and stage your changes.</p>
        </div>
        <button className="button button-secondary button-small" type="button" onClick={() => stage('.')} disabled={!canStage || busy}>
          <Plus size={15} /> Stage all
        </button>
      </header>

      <div className="files">
        {status.merging && (
          <p className={`notice ${conflicts.size ? 'notice-alert' : ''}`} role="status">
            <GitMerge size={16} />
            {conflicts.size ? 'Resolve conflicts in your editor, then stage each file.' : 'Merge resolutions staged. Create the merge commit.'}
          </p>
        )}
        <FileGroup title="Merge conflicts" files={status.conflicts} color="amber" onStage={stage} busy={busy} />
        <FileGroup title="Staged changes" files={staged} color="lime" busy={busy} />
        <FileGroup title="Modified files" files={modified} color="amber" onStage={stage} busy={busy} />
        <FileGroup title="Untracked files" files={untracked} color="slate" onStage={stage} busy={busy} />
        {!hasChanges && !status.merging && (
          <div className="empty-state">
            <Check size={22} />
            <strong>Working tree is clean</strong>
            <span>Edit a file to see changes here.</span>
          </div>
        )}
      </div>

      <form className="commit-form" onSubmit={event => {
        event.preventDefault();
        void commit();
      }}>
        <label htmlFor="commit-message">Commit message</label>
        <textarea
          id="commit-message"
          placeholder="Describe what changed"
          rows={3}
          value={message}
          onChange={event => setMessage(event.target.value)}
          onKeyDown={event => {
            if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
              event.preventDefault();
              void commit();
            }
          }}
        />
        <button className="button button-primary" type="submit" disabled={!canCommit}>
          <GitCommitHorizontal size={18} /> {status.merging ? 'Commit merge' : 'Commit changes'}
          <ArrowRight size={16} className="commit-arrow" />
        </button>
      </form>
    </section>
  );
}

function Branches({ overview, busy, run }: { overview: Overview; busy: boolean; run: RunAction }) {
  const [name, setName] = useState('');
  const { branchList, status } = overview;
  const branches = branchList.current
    ? [branchList.current, ...branchList.branches.filter(branch => branch !== branchList.current)]
    : branchList.branches;
  return (
    <section className="panel" aria-labelledby="branches-heading">
      <header className="panel-header">
        <div>
          <h2 id="branches-heading">Branches <span className="count">{branchList.branches.length}</span></h2>
          <p>Switch branches or merge into HEAD.</p>
        </div>
        <form className="branch-form" onSubmit={async event => {
          event.preventDefault();
          const branch = name.trim();
          if (branch && await run(() => api.branch(branch), `Created branch ${branch}`)) setName('');
        }}>
          <input aria-label="New branch name" placeholder="Branch name" value={name} onChange={event => setName(event.target.value)} disabled={!status.headCommit || busy} />
          <button type="submit" className="button button-secondary button-small" disabled={!status.headCommit || !name.trim() || busy}>
            <Plus size={15} /> Create
          </button>
        </form>
      </header>
      <ul className="branch-list">
        {branches.map(branch => {
          const current = branch === branchList.current;
          return (
            <li className={`branch-row ${current ? 'current' : ''}`} key={branch}>
              <GitBranch size={16} aria-hidden="true" />
              <span className="branch-name" title={branch}>{branch}</span>
              {current ? <span className="branch-current">Current</span> : (
                <div className="branch-actions">
                  <button className="button button-quiet button-small" type="button" onClick={() => run(() => api.checkout(branch), `Switched to ${branch}`)} aria-label={`Switch to ${branch}`} disabled={busy || status.merging}>
                    <ArrowRightLeft size={14} /> Switch
                  </button>
                  <button className="button button-quiet button-small" type="button" onClick={() => run(() => api.merge(branch), `Finished merging ${branch}`)} aria-label={`Merge ${branch}`} disabled={busy || status.merging}>
                    <GitMerge size={15} /> Merge
                  </button>
                </div>
              )}
            </li>
          );
        })}
      </ul>
      {!status.headCommit && <p className="branch-hint">Create the first commit to start another branch.</p>}
    </section>
  );
}

const dateFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
const resetModes: { mode: ResetMode; label: string; description: string }[] = [
  { mode: 'soft', label: 'Soft', description: 'Move HEAD and keep staged and working files.' },
  { mode: 'mixed', label: 'Mixed', description: 'Move HEAD and reset the staging area. Keep working files.' },
  { mode: 'hard', label: 'Hard', description: 'Restore the selected snapshot and discard changes to tracked and staged files.' },
];

function CommitHistory({ commits, busy, merging, run }: { commits: Commit[]; busy: boolean; merging: boolean; run: RunAction }) {
  const [query, setQuery] = useState('');
  const [target, setTarget] = useState<Commit | null>(null);
  const [mode, setMode] = useState<ResetMode>('mixed');
  const [resetError, setResetError] = useState('');
  const term = query.trim().toLowerCase();
  const results = commits.filter(commit => commit.id.startsWith(term) || commit.message.toLowerCase().includes(term));

  return (
    <section className="panel" aria-labelledby="history-heading">
      <header className="panel-header">
        <div>
          <h2 id="history-heading">Commit history</h2>
          <p>First-parent timeline for the current revision.</p>
        </div>
        <label className="search">
          <Search size={15} aria-hidden="true" />
          <input type="search" aria-label="Search commits" placeholder="Search commits" value={query} onChange={event => setQuery(event.target.value)} />
        </label>
      </header>
      {results.length ? (
        <ol className="timeline">
          {results.map(commit => {
            const isHead = commit.id === commits[0]?.id;
            const [subject, ...body] = commit.message.split('\n');
            return (
              <li className="commit-row" key={commit.id}>
                <span className={`timeline-node ${isHead ? 'head-node' : ''}`}>
                  {commit.parents.length > 1 ? <GitMerge size={14} /> : <GitCommitHorizontal size={16} />}
                </span>
                <div className="commit-copy">
                  <div className="commit-subject">
                    <span title={subject}>{subject}</span>
                    {isHead && <span className="head-label">HEAD</span>}
                    {commit.parents.length > 1 && <span className="merge-label">Merge</span>}
                  </div>
                  <div className="commit-meta">
                    <code title={commit.id}>{commit.id.slice(0, 7)}</code>
                    <time dateTime={commit.timestamp}>{dateFormat.format(new Date(commit.timestamp))}</time>
                  </div>
                  {body.some(line => line.trim()) && (
                    <details className="commit-details">
                      <summary>Commit details</summary>
                      <p>{body.join('\n').trim()}</p>
                    </details>
                  )}
                </div>
                <button
                  className="icon-button reset-button"
                  type="button"
                  title="Reset to this commit"
                  aria-label={`Reset to commit ${commit.id.slice(0, 7)}`}
                  disabled={busy}
                  onClick={() => {
                    setTarget(commit);
                    setMode(merging ? 'hard' : 'mixed');
                    setResetError('');
                  }}
                >
                  <RotateCcw size={16} />
                </button>
              </li>
            );
          })}
        </ol>
      ) : (
        <div className="empty-state">
          {term ? <Search size={22} /> : <History size={22} />}
          <strong>{term ? 'No matching commits' : 'No commits yet'}</strong>
          <span>{term ? 'Search by message or commit ID.' : 'Your first commit will appear here.'}</span>
        </div>
      )}

      <Modal isOpen={target !== null} onClose={() => setTarget(null)} title="Reset repository" busy={busy}>
        {target && (
          <form className="reset-form" onSubmit={async event => {
            event.preventDefault();
            setResetError('');
            if (await run(() => api.reset(mode, target.id), `Reset to ${target.id.slice(0, 7)}`, setResetError)) setTarget(null);
          }}>
            <p>Move HEAD to <code>{target.id.slice(0, 7)}</code>.</p>
            {merging && <p className="notice notice-alert"><CircleAlert size={16} /> A hard reset abandons the merge.</p>}
            <fieldset>
              <legend className="visually-hidden">Reset mode</legend>
              {resetModes.map(option => (
                <label className={`reset-option ${mode === option.mode ? 'selected' : ''}`} key={option.mode}>
                  <input type="radio" name="reset-mode" checked={mode === option.mode} onChange={() => setMode(option.mode)} disabled={busy || (merging && option.mode !== 'hard')} />
                  <span><strong>{option.label}</strong><small>{option.description}</small></span>
                </label>
              ))}
            </fieldset>
            {resetError && <p className="notice notice-alert" role="alert"><CircleAlert size={16} /> {resetError}</p>}
            <div className="modal-actions">
              <button type="button" className="button button-secondary" onClick={() => setTarget(null)} disabled={busy}>Cancel</button>
              <button type="submit" className={`button ${mode === 'hard' ? 'button-danger' : 'button-primary'}`} disabled={busy}>
                <RotateCcw size={15} /> {mode === 'hard' ? 'Hard reset' : 'Confirm reset'}
              </button>
            </div>
          </form>
        )}
      </Modal>
    </section>
  );
}

export default function App() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const inAction = useRef(false);
  const refreshState = useRef({ id: 0, active: false });
  const { showToast } = useToast();

  const refresh = useCallback(async (silent = false) => {
    const state = refreshState.current;
    if (silent && (state.active || inAction.current)) return;
    const id = ++state.id;
    state.active = true;
    if (!silent) setLoading(true);
    try {
      const data = await api.overview();
      if (id !== state.id) return;
      setOverview(data);
      setError('');
    } catch (failure) {
      if (id === state.id) setError(errorMessage(failure));
    } finally {
      if (id === state.id) {
        state.active = false;
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const state = refreshState.current;
    void refresh();
    const timer = window.setInterval(() => void refresh(true), 3000);
    return () => {
      window.clearInterval(timer);
      state.id++;
      state.active = false;
    };
  }, [refresh]);

  const run: RunAction = async (action, success, onError) => {
    if (inAction.current) return false;
    inAction.current = true;
    setBusy(true);
    try {
      await action();
      showToast(success, 'success');
      return true;
    } catch (failure) {
      const message = errorMessage(failure);
      if (onError) onError(message);
      else showToast(message, 'error');
      return false;
    } finally {
      await refresh();
      inAction.current = false;
      setBusy(false);
    }
  };

  const status = overview?.status;
  const unstaged = status ? new Set([...status.modified, ...status.untracked, ...status.conflicts]).size : 0;
  return (
    <>
      <header className="topbar">
        <div className="topbar-inner">
          <div className="identity">
            <span className="logo"><Workflow size={22} strokeWidth={2.3} /></span>
            <span className="wordmark">mgit<span>.</span></span>
            <ChevronRight size={16} className="breadcrumb-divider" />
            <span className="breadcrumb">Repository</span>
          </div>
          <div className="topbar-actions">
            <span className={`connection ${error ? 'disconnected' : ''}`} role="status">
              <span /> {error ? 'Disconnected' : overview ? 'Local workspace' : 'Connecting'}
            </span>
            <RefreshButton working={loading || busy} onClick={() => void refresh()} />
          </div>
        </div>
      </header>
      <main className="dashboard">
        {error && <p className="notice notice-alert connection-alert" role="alert"><CircleAlert size={16} /> Could not refresh the repository. {error}</p>}
        {overview && status ? (
          <>
            <section className="repository-heading" aria-labelledby="repository-name">
              <div className="repository-label"><FolderGit2 size={17} /> Local repository</div>
              <h1 id="repository-name">{overview.name}</h1>
              <div className="repository-meta">
                <span className="head-branch"><GitBranch size={15} /> {status.currentBranch || 'Detached HEAD'}</span>
                {status.headCommit ? <code title={status.headCommit}>{status.headCommit.slice(0, 7)}</code> : <span>No commits yet</span>}
              </div>
            </section>
            <section className="stats" aria-label="Repository activity">
              <Stat label="Staged files" value={status.staged.length} detail={status.staged.length ? 'Ready for a commit' : 'Nothing staged'} icon={<Layers3 size={19} />} color="lime" />
              <Stat label="Unstaged files" value={unstaged} detail={status.merging ? 'Merge in progress' : unstaged ? 'Working tree changes' : 'Working tree clean'} icon={<FileText size={19} />} color="amber" />
              <Stat label="Commits" value={overview.log.length} detail="First-parent history" icon={<GitCommitHorizontal size={21} />} color="slate" />
            </section>
            <div className="workspace-grid">
              <WorkingTree status={status} busy={busy || Boolean(error)} run={run} />
              <div className="right-column">
                <Branches overview={overview} busy={busy || Boolean(error)} run={run} />
                <CommitHistory commits={overview.log} busy={busy || Boolean(error)} merging={status.merging} run={run} />
              </div>
            </div>
          </>
        ) : (
          <div className="panel empty-state initial-state">
            <Terminal size={25} />
            <strong>{error ? 'Repository unavailable' : 'Opening your repository'}</strong>
          </div>
        )}
      </main>
    </>
  );
}
