export interface StatusResponse {
  staged: string[];
  modified: string[];
  untracked: string[];
  currentBranch: string;
  headCommit: string;
  merging: boolean;
  conflicts: string[];
}

export interface Commit {
  id: string;
  message: string;
  timestamp: string;
  parents: string[];
}

export interface Overview {
  name: string;
  status: StatusResponse;
  branchList: { branches: string[]; current: string };
  log: Commit[];
}

export type ResetMode = 'soft' | 'mixed' | 'hard';

async function request(path: string, body?: object): Promise<Response> {
  const response = await fetch(`/api/${path}`, body && {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!response.ok) throw new Error((await response.text()).trim());
  return response;
}

export const api = {
  overview: async (): Promise<Overview> => (await request('overview')).json(),
  stage: (path: string) => request('add', { paths: [path] }),
  commit: (message: string) => request('commit', { message }),
  branch: (name: string) => request('branch', { name }),
  checkout: (target: string) => request('checkout', { target }),
  merge: (targetBranch: string) => request('merge', { targetBranch }),
  reset: (mode: ResetMode, target: string) => request('reset', { mode, target }),
};

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'The operation failed. Try again.';
}
