export interface StatusResponse {
  staged: string[];
  modified: string[];
  untracked: string[];
  currentBranch: string;
  headCommit: string;
}

export interface Commit {
  id: string;
  message: string;
  timestamp: string;
  parents: string[];
}

export interface BranchList {
  branches: string[];
  current: string;
}

const API_BASE = '/api';

export const api = {
  getStatus: async (): Promise<StatusResponse> => {
    const res = await fetch(`${API_BASE}/status`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  addFiles: async (paths: string[]): Promise<void> => {
    const res = await fetch(`${API_BASE}/add`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ paths })
    });
    if (!res.ok) throw new Error(await res.text());
  },

  commit: async (message: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/commit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message })
    });
    if (!res.ok) throw new Error(await res.text());
  },

  getLog: async (): Promise<Commit[]> => {
    const res = await fetch(`${API_BASE}/log`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getBranches: async (): Promise<BranchList> => {
    const res = await fetch(`${API_BASE}/branch`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  createBranch: async (name: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/branch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name })
    });
    if (!res.ok) throw new Error(await res.text());
  },

  checkout: async (target: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/checkout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target })
    });
    if (!res.ok) throw new Error(await res.text());
  },
  
  merge: async (targetBranch: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/merge`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetBranch })
    });
    if (!res.ok) throw new Error(await res.text());
  },

  reset: async (mode: string, target: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/reset`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode, target })
    });
    if (!res.ok) throw new Error(await res.text());
  }
};
