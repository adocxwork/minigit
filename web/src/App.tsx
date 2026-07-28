import { useEffect, useState, useCallback } from 'react';
import { api, type StatusResponse, type Commit, type BranchList } from './api';
import { GitBranch, GitMerge, Check, RefreshCw, GitCommit, FileText, Plus, RotateCcw } from 'lucide-react';
import { useToast } from './components/Toast';
import { Modal } from './components/Modal';
import './index.css';

// --- Header Component ---
function Header({ loading, onRefresh }: { loading: boolean, onRefresh: () => void }) {
  return (
    <header className="app-header">
      <div className="app-title">
        <GitBranch size={16} />
        mgit Source Control
      </div>
      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
        <button className="icon-btn" onClick={onRefresh} title="Refresh">
          <RefreshCw size={16} className={loading ? 'spin' : ''} />
        </button>
      </div>
    </header>
  );
}

// --- Workspace Panel Component ---
function WorkspacePanel({ status, onRefresh }: { status: StatusResponse | null, onRefresh: () => Promise<void> }) {
  const { showToast } = useToast();
  const [commitMsg, setCommitMsg] = useState('');
  const [isCommitting, setIsCommitting] = useState(false);

  const handleStageAll = async () => {
    if (!status) return;
    const allFiles = [...(status.modified || []), ...(status.untracked || [])];
    if (allFiles.length === 0) return;
    try {
      await api.addFiles(['.']);
      showToast('All files staged', 'success');
      await onRefresh();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  const handleCommit = async () => {
    if (!commitMsg.trim()) return;
    setIsCommitting(true);
    try {
      await api.commit(commitMsg);
      showToast('Commit successful', 'success');
      setCommitMsg('');
      await onRefresh();
    } catch (err: any) {
      showToast(err.message, 'error');
    } finally {
      setIsCommitting(false);
    }
  };

  return (
    <div className="panel-left">
      <div className="section-header">
        <span>Source Control</span>
        <button className="icon-btn" onClick={handleStageAll} title="Stage All">
          <Plus size={14} />
        </button>
      </div>

      <div className="scrollable-content" style={{ paddingBottom: '16px' }}>
        {(status?.staged?.length || 0) > 0 && (
          <div style={{ padding: '4px 0' }}>
            <div style={{ fontSize: '11px', padding: '4px 12px', color: 'var(--text-light)', fontWeight: 600 }}>Staged Changes</div>
            {status?.staged?.map(f => (
              <div key={f} className="file-item">
                <span className="file-name"><FileText size={12} style={{marginRight: '6px', verticalAlign: 'text-bottom'}}/>{f}</span>
                <span className="file-status status-M">M</span>
              </div>
            ))}
          </div>
        )}
        
        {(status?.modified?.length || 0) > 0 && (
          <div style={{ padding: '4px 0' }}>
            <div style={{ fontSize: '11px', padding: '4px 12px', color: 'var(--text-light)', fontWeight: 600 }}>Changes</div>
            {status?.modified?.map(f => (
              <div key={f} className="file-item">
                <span className="file-name"><FileText size={12} style={{marginRight: '6px', verticalAlign: 'text-bottom'}}/>{f}</span>
                <span className="file-status status-M">M</span>
              </div>
            ))}
          </div>
        )}

        {(status?.untracked?.length || 0) > 0 && (
          <div style={{ padding: '4px 0' }}>
            <div style={{ fontSize: '11px', padding: '4px 12px', color: 'var(--text-light)', fontWeight: 600 }}>Untracked Changes</div>
            {status?.untracked?.map(f => (
              <div key={f} className="file-item">
                <span className="file-name"><FileText size={12} style={{marginRight: '6px', verticalAlign: 'text-bottom'}}/>{f}</span>
                <span className="file-status status-U">U</span>
              </div>
            ))}
          </div>
        )}

        {((status?.staged?.length || 0) === 0 && (status?.modified?.length || 0) === 0 && (status?.untracked?.length || 0) === 0) && (
          <div style={{ padding: '24px 12px', textAlign: 'center', color: 'var(--text-muted)' }}>
            No active changes in working directory.
          </div>
        )}
      </div>

      <div className="commit-box">
        <textarea 
          className="input-field" 
          placeholder="Message (Cmd/Ctrl+Enter to commit)"
          value={commitMsg}
          onChange={e => setCommitMsg(e.target.value)}
          onKeyDown={e => {
            if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
              e.preventDefault();
              handleCommit();
            }
          }}
          rows={3}
          disabled={isCommitting}
        />
        <button 
          className="btn" 
          style={{ marginTop: '8px' }}
          onClick={handleCommit}
          disabled={!commitMsg.trim() || (status?.staged?.length || 0) === 0 || isCommitting}
        >
          <Check size={14} /> Commit
        </button>
      </div>
    </div>
  );
}

// --- History & Branches Panel Component ---
function HistoryPanel({ branches, logs, onRefresh }: { branches: BranchList | null, logs: Commit[], onRefresh: () => Promise<void> }) {
  const { showToast } = useToast();
  const [newBranchName, setNewBranchName] = useState('');
  const [isResetModalOpen, setResetModalOpen] = useState(false);
  const [resetTarget, setResetTarget] = useState('');
  const [resetMode, setResetMode] = useState('mixed');

  const handleCreateBranch = async () => {
    if (!newBranchName.trim()) return;
    try {
      await api.createBranch(newBranchName);
      showToast(`Created branch ${newBranchName}`, 'success');
      setNewBranchName('');
      await onRefresh();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  const executeReset = async () => {
    try {
      await api.reset(resetMode, resetTarget);
      showToast(`Reset HEAD to ${resetTarget.substring(0,7)}`, 'success');
      setResetModalOpen(false);
      await onRefresh();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  return (
    <div className="panel-right">
      <div className="branches-bar">
        {branches?.branches.map(b => (
          <div key={b} className={`branch-tag ${branches.current === b ? 'active' : ''}`}>
            <GitBranch size={12} /> {b}
            {branches.current !== b && (
              <span style={{ marginLeft: '6px', display: 'inline-flex', gap: '4px' }}>
                <button className="icon-btn" title="Checkout" onClick={() => api.checkout(b).then(onRefresh).catch(e => showToast(e.message, 'error'))}>
                   <Check size={12} />
                </button>
                <button className="icon-btn" title="Merge into current" onClick={() => api.merge(b).then(() => { showToast('Merged successfully', 'success'); onRefresh(); }).catch(e => showToast(e.message, 'error'))}>
                   <GitMerge size={12} />
                </button>
              </span>
            )}
          </div>
        ))}
        
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '8px' }}>
          <input 
            type="text" 
            className="input-field" 
            style={{ width: '150px' }} 
            placeholder="New branch..." 
            value={newBranchName}
            onChange={e => setNewBranchName(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter') {
                e.preventDefault();
                handleCreateBranch();
              }
            }}
          />
          <button className="btn btn-secondary" onClick={handleCreateBranch}>Create</button>
        </div>
      </div>

      <div className="section-header" style={{ borderBottom: '1px solid var(--border-color)' }}>
        <span>Commit History</span>
      </div>
      
      <div className="scrollable-content">
        <table className="history-table">
          <tbody>
            {logs.map(log => (
              <tr key={log.id} className="history-row">
                <td className="history-cell hash-cell">
                  <GitCommit size={12} style={{ marginRight: '6px', verticalAlign: 'middle', color: 'var(--text-muted)' }}/>
                  {log.id.substring(0, 7)}
                </td>
                <td className="history-cell msg-cell">{log.message}</td>
                <td className="history-cell date-cell" style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '8px' }}>
                  <span>{new Date(log.timestamp).toLocaleString()}</span>
                  <button className="icon-btn" title="Reset to this commit" onClick={() => {
                    setResetTarget(log.id);
                    setResetModalOpen(true);
                  }}>
                    <RotateCcw size={12} />
                  </button>
                </td>
              </tr>
            ))}
            {logs.length === 0 && (
              <tr>
                <td colSpan={3} className="history-cell" style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '32px' }}>
                  No commits to display.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Modal isOpen={isResetModalOpen} onClose={() => setResetModalOpen(false)} title="Reset HEAD">
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <p style={{ margin: 0, color: 'var(--text-main)' }}>Select how you want to reset to commit <strong>{resetTarget.substring(0,7)}</strong>:</p>
          
          <label style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', cursor: 'pointer' }}>
            <input type="radio" name="resetMode" value="soft" checked={resetMode === 'soft'} onChange={() => setResetMode('soft')} style={{ marginTop: '4px' }}/>
            <div>
              <div style={{ fontWeight: 600, color: 'var(--text-light)' }}>Soft</div>
              <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Keeps all your files and staging area untouched. Only moves the branch pointer.</div>
            </div>
          </label>

          <label style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', cursor: 'pointer' }}>
            <input type="radio" name="resetMode" value="mixed" checked={resetMode === 'mixed'} onChange={() => setResetMode('mixed')} style={{ marginTop: '4px' }}/>
            <div>
              <div style={{ fontWeight: 600, color: 'var(--text-light)' }}>Mixed (Default)</div>
              <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Keeps your files untouched but resets the staging area.</div>
            </div>
          </label>

          <label style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', cursor: 'pointer' }}>
            <input type="radio" name="resetMode" value="hard" checked={resetMode === 'hard'} onChange={() => setResetMode('hard')} style={{ marginTop: '4px' }}/>
            <div>
              <div style={{ fontWeight: 600, color: '#dc3545' }}>Hard</div>
              <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Permanently discards all uncommitted local changes and staged files.</div>
            </div>
          </label>

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '8px' }}>
            <button className="btn btn-secondary" onClick={() => setResetModalOpen(false)}>Cancel</button>
            <button className="btn" style={{ backgroundColor: resetMode === 'hard' ? '#dc3545' : 'var(--accent-blue)', width: 'auto' }} onClick={executeReset}>
              Confirm Reset
            </button>
          </div>
        </div>
      </Modal>

    </div>
  );
}

// --- Main App Component ---
function App() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [branches, setBranches] = useState<BranchList | null>(null);
  const [logs, setLogs] = useState<Commit[]>([]);
  const [loading, setLoading] = useState(true);
  const { showToast } = useToast();

  const refreshAll = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const [s, b, l] = await Promise.all([
        api.getStatus().catch(() => null),
        api.getBranches().catch(() => null),
        api.getLog().catch(() => [])
      ]);
      if (s) setStatus(s);
      if (b) setBranches(b);
      setLogs(l || []);
    } catch (err: any) {
      if (!silent) showToast(err.message || 'Failed to refresh data', 'error');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [showToast]);

  // Initial load
  useEffect(() => {
    refreshAll();
  }, [refreshAll]);

  // Background polling every 3 seconds for reactive state updates
  useEffect(() => {
    const interval = setInterval(() => {
      refreshAll(true);
    }, 3000);
    return () => clearInterval(interval);
  }, [refreshAll]);

  return (
    <div className="app-container">
      <Header loading={loading} onRefresh={() => refreshAll(false)} />
      <div className="workspace">
        <WorkspacePanel status={status} onRefresh={async () => { await refreshAll(false); }} />
        <HistoryPanel branches={branches} logs={logs} onRefresh={async () => { await refreshAll(false); }} />
      </div>
    </div>
  );
}

export default App;
