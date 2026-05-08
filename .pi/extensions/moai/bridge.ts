import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionContext } from "@earendil-works/pi-coding-agent";
import type { BridgeRequest, BridgeResponse, BridgeSessionInfo } from "./types.js";

const DEFAULT_TIMEOUT_MS = 30_000;

function resolveMoaiBridgeInvocation(cwd: string): { command: string; args: string[] } {
	if (process.env.MOAI_BIN) {
		return { command: process.env.MOAI_BIN, args: ["pi", "bridge"] };
	}

	// Project-local development checkout: the globally installed `moai` may be an
	// older binary without `moai pi bridge`. Prefer the checked-out Go command so
	// Pi uses the extension code from this branch without requiring installation.
	if (existsSync(join(cwd, "go.mod")) && existsSync(join(cwd, "cmd", "moai", "main.go"))) {
		return { command: "go", args: ["run", "./cmd/moai", "pi", "bridge"] };
	}

	return { command: "moai", args: ["pi", "bridge"] };
}

export function getSessionInfo(ctx: ExtensionContext): BridgeSessionInfo {
	const sessionManager = ctx.sessionManager;
	let file: string | undefined;
	let leafId: string | undefined;
	try {
		file = sessionManager.getSessionFile() ?? undefined;
	} catch {
		file = undefined;
	}
	try {
		leafId = sessionManager.getLeafId() ?? undefined;
	} catch {
		leafId = undefined;
	}
	return {
		file,
		leafId,
		mode: ctx.hasUI ? "interactive" : "non-interactive",
	};
}

export async function callBridge(
	ctx: ExtensionContext,
	request: Omit<BridgeRequest, "cwd" | "session"> & Partial<Pick<BridgeRequest, "cwd" | "session">>,
	options: { timeoutMs?: number; signal?: AbortSignal } = {},
): Promise<BridgeResponse> {
	const payload: BridgeRequest = {
		version: "moai.pi.bridge.v1",
		cwd: ctx.cwd,
		session: getSessionInfo(ctx),
		...request,
	};
	const input = JSON.stringify(payload);
	const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;

	return new Promise<BridgeResponse>((resolve) => {
		const invocation = resolveMoaiBridgeInvocation(ctx.cwd);
		const child = spawn(invocation.command, invocation.args, {
			cwd: ctx.cwd,
			stdio: ["pipe", "pipe", "pipe"],
		});

		let stdout = "";
		let stderr = "";
		let settled = false;
		const finish = (response: BridgeResponse) => {
			if (settled) return;
			settled = true;
			clearTimeout(timer);
			resolve(response);
		};

		const timer = setTimeout(() => {
			child.kill("SIGTERM");
			finish({
				ok: false,
				kind: request.kind,
				error: { code: "bridge_timeout", message: `MoAI bridge timed out after ${timeoutMs}ms` },
			});
		}, timeoutMs);

		options.signal?.addEventListener(
			"abort",
			() => {
				child.kill("SIGTERM");
				finish({ ok: false, kind: request.kind, error: { code: "aborted", message: "MoAI bridge call aborted" } });
			},
			{ once: true },
		);

		child.stdout.on("data", (data) => {
			stdout += data.toString();
		});
		child.stderr.on("data", (data) => {
			stderr += data.toString();
		});
		child.on("error", (error) => {
			finish({ ok: false, kind: request.kind, error: { code: "spawn_failed", message: error.message } });
		});
		child.on("close", (code) => {
			if (settled) return;
			try {
				const response = JSON.parse(stdout) as BridgeResponse;
				finish(response);
			} catch (error) {
				finish({
					ok: false,
					kind: request.kind,
					error: {
						code: "invalid_bridge_response",
						message: `exit=${code ?? "unknown"} stderr=${stderr.trim()} stdout=${stdout.trim()}`,
					},
				});
			}
		});

		child.stdin.write(input);
		child.stdin.end();
	});
}

export function bridgeResponseText(response: BridgeResponse): string {
	if (response.ok) {
		const details = response.data ? `\n\n${JSON.stringify(response.data, null, 2)}` : "";
		return `${response.message ?? "MoAI bridge request completed."}${details}`;
	}
	return `MoAI bridge error [${response.error?.code ?? "unknown"}]: ${response.error?.message ?? "Unknown error"}`;
}
