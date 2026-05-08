export interface MoaiUIState {
	activeSpecId?: string;
	phase?: string;
	developmentMode?: string;
	harnessLevel?: string;
	projectName?: string;
	moaiVersion?: string;
	moaiLatestVersion?: string;
	piVersion?: string;
	sessionStartedAt?: number;
	shortWindowPercent?: number;
	weeklyWindowPercent?: number;
	gitBranch?: string;
	gitAdded?: number;
	gitModified?: number;
	gitUntracked?: number;
	worktreePath?: string;
	taskTotal?: number;
	taskCompleted?: number;
	taskInProgress?: number;
	qualityStatus?: string;
	lspErrors?: number;
	mxWarnings?: number;
	clipboardImagePath?: string;
	lastUpdated?: string;
	details?: Record<string, unknown>;
}

export function normalizeState(data: Record<string, unknown> | undefined): MoaiUIState {
	return (data ?? {}) as MoaiUIState;
}
