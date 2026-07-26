import { useEffect, useState } from 'react';
import { api, type StatusResponse, type Commit, type BranchList } from './api';
import { GitBranch, GitMerge, Check, RefreshCw, GitCommit, FileText, Plus } from 'lucide-react';
import './index.css';

function App() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [branches, setBranches] = useState<BranchList | null>(null);
  const [logs, setLogs] = useState<Commit[]>([]);
  const [commitMsg, setCommitMsg] = useState('');
  const [newBranchName, setNewBranchName] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refreshAll = async () => {
    setLoading(true);
    setError('');
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
      setError(err.message || 'Failed to refresh data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshAll();
  }, []);

  const handleStageAll = async () => {
    if (!status) return;
    const allFiles = [...(status.modified || []), ...(status.untracked || [])];
    if (allFiles.length === 0) return;
    try {
      await api.addFiles(['.']);
      await refreshAll();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleCommit = async () => {
    if (!commitMsg.trim()) return;
    try {
      await api.commit(commitMsg);
      setCommitMsg('');
      await refreshAll();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleCreateBranch = async () => {
    if (!newBranchName.trim()) return;
    try {
      await api.createBranch(newBranchName);
      setNewBranchName('');
      await refreshAll();
    } catch (err: any) {
      setError(err.message);
    }
  };

  return (
    <div className="app-container">
      {/* Header */}
      <header className="app-header">
        <div className="app-title">
          <GitBranch size={16} />
          mgit Source Control
        </div>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          {error && <span style={{ color: 'var(--color-deleted)' }}>{error}</span>}
          <button className="icon-btn" onClick={refreshAll} title="Refresh">
            <RefreshCw size={16} className={loading ? 'spin' : ''} />
          </button>
        </div>
      </header>

      {/* Main Workspace */}
      <div className="workspace">
        
        {/* Left Panel: Source Control */}
        <div className="panel-left">
          
          <div className="section-header">
            <span>Source Control</span>
            <button className="icon-btn" onClick={handleStageAll} title="Stage All">
              <Plus size={14} />
            </button>
          </div>

          <div className="scrollable-content" style={{ paddingBottom: '16px' }}>
            {/* Staged */}
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
            
            {/* Modified */}
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

            {/* Untracked */}
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
              rows={3}
            />
            <button 
              className="btn" 
              style={{ marginTop: '8px' }}
              onClick={handleCommit}
              disabled={!commitMsg.trim() || (status?.staged?.length || 0) === 0}
            >
              <Check size={14} /> Commit
            </button>
          </div>

        </div>

        {/* Right Panel: History & Branches */}
        <div className="panel-right">
          
          <div className="branches-bar">
            {branches?.branches.map(b => (
              <div key={b} className={`branch-tag ${branches.current === b ? 'active' : ''}`}>
                <GitBranch size={12} /> {b}
                {branches.current !== b && (
                  <span style={{ marginLeft: '6px', display: 'inline-flex', gap: '4px' }}>
                    <button className="icon-btn" title="Checkout" onClick={() => api.checkout(b).then(refreshAll)}>
                       <Check size={12} />
                    </button>
                    <button className="icon-btn" title="Merge into current" onClick={() => api.merge(b).then(refreshAll)}>
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
                    <td className="history-cell date-cell">{new Date(log.timestamp).toLocaleString()}</td>
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

        </div>

      </div>
    </div>
  );
}

export default App;
