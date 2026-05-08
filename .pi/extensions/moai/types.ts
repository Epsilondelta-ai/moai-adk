export interface BridgeSessionInfo {
	file?: string;
	leafId?: string;
	branch?: string;
	mode?: string;
}

export interface BridgeRequest {
	version?: string;
	kind: string;
	cwd?: string;
	session?: BridgeSessionInfo;
	payload?: Record<string, unknown>;
}

export interface BridgeError {
	code: string;
	message: string;
}

export interface BridgeResponse {
	ok: boolean;
	kind?: string;
	message?: string;
	data?: Record<string, unknown>;
	error?: BridgeError;
	generatedAt?: string;
}
