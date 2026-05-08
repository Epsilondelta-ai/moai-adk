import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { StringEnum } from "@earendil-works/pi-ai";
import { Type } from "typebox";
import { callBridge, bridgeResponseText } from "./bridge.js";
import { registerInteractionTool } from "./interaction.js";

const EmptyParams = Type.Object({});

const GenericToolParams = Type.Object({
	action: StringEnum(["get", "set", "list", "create", "update", "run", "status"] as const, {
		description: "Operation to perform through the MoAI bridge",
	}),
	payload: Type.Optional(Type.Record(Type.String(), Type.Unknown(), { description: "Tool-specific payload" })),
});

const AgentInvokeParams = Type.Object({
	action: StringEnum(["list", "run", "parallel", "chain"] as const, { description: "Agent invocation mode" }),
	agent: Type.Optional(Type.String({ description: "Agent name for run mode" })),
	task: Type.Optional(Type.String({ description: "Task for run mode" })),
	tasks: Type.Optional(Type.Array(Type.Object({ agent: Type.String(), task: Type.String() }))),
	chain: Type.Optional(Type.Array(Type.Object({ agent: Type.String(), task: Type.String() }))),
	model: Type.Optional(Type.String()),
	timeoutSeconds: Type.Optional(Type.Number()),
	worktree: Type.Optional(Type.Object({
		enabled: Type.Boolean(),
		keep: Type.Optional(Type.Boolean()),
		branch: Type.Optional(Type.String()),
		baseDir: Type.Optional(Type.String()),
	})),
});

function registerBridgeTool(pi: ExtensionAPI, name: string, label: string, description: string): void {
	pi.registerTool({
		name,
		label,
		description,
		promptSnippet: description,
		promptGuidelines: [`Use ${name} only for MoAI project state and workflow operations.`],
		parameters: GenericToolParams,
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			const response = await callBridge(
				ctx,
				{ kind: "tool", payload: { tool: name, action: params.action, payload: params.payload ?? {} } },
				{ signal },
			);
			return {
				content: [{ type: "text", text: bridgeResponseText(response) }],
				details: response,
			};
		},
	});
}

export function registerTools(pi: ExtensionAPI): void {
	registerInteractionTool(pi);

	pi.registerTool({
		name: "moai_bridge_capabilities",
		label: "MoAI Capabilities",
		description: "Show MoAI Pi bridge capabilities and supported operations.",
		promptSnippet: "Show MoAI Pi bridge capabilities",
		parameters: EmptyParams,
		async execute(_toolCallId, _params, signal, _onUpdate, ctx) {
			const response = await callBridge(ctx, { kind: "capabilities" }, { signal });
			return { content: [{ type: "text", text: bridgeResponseText(response) }], details: response };
		},
	});

	pi.registerTool({
		name: "moai_doctor",
		label: "MoAI Doctor",
		description: "Check MoAI Pi extension bridge readiness for the current project.",
		promptSnippet: "Check MoAI Pi extension bridge readiness",
		parameters: EmptyParams,
		async execute(_toolCallId, _params, signal, _onUpdate, ctx) {
			const response = await callBridge(ctx, { kind: "doctor" }, { signal });
			return { content: [{ type: "text", text: bridgeResponseText(response) }], details: response };
		},
	});

	registerBridgeTool(pi, "moai_config_get", "MoAI Config Get", "Read MoAI configuration through the bridge.");
	registerBridgeTool(pi, "moai_config_set", "MoAI Config Set", "Update MoAI configuration through the bridge.");
	registerBridgeTool(pi, "moai_spec_status", "MoAI SPEC Status", "Read MoAI SPEC status through the bridge.");
	registerBridgeTool(pi, "moai_spec_create", "MoAI SPEC Create", "Create a MoAI SPEC through the bridge.");
	registerBridgeTool(pi, "moai_spec_update", "MoAI SPEC Update", "Update a MoAI SPEC through the bridge.");
	registerBridgeTool(pi, "moai_spec_list", "MoAI SPEC List", "List MoAI SPECs through the bridge.");
	registerBridgeTool(pi, "moai_quality_gate", "MoAI Quality Gate", "Run MoAI quality gate operations through the bridge.");
	registerBridgeTool(pi, "moai_lsp_check", "MoAI LSP Check", "Run MoAI LSP diagnostics through the bridge.");
	registerBridgeTool(pi, "moai_mx_scan", "MoAI MX Scan", "Run MoAI MX tag scanning through the bridge.");
	registerBridgeTool(pi, "moai_mx_update", "MoAI MX Update", "Update MoAI MX annotations through the bridge.");
	registerBridgeTool(pi, "moai_memory_read", "MoAI Memory Read", "Read MoAI project memory through the bridge.");
	registerBridgeTool(pi, "moai_memory_write", "MoAI Memory Write", "Write MoAI project memory through the bridge.");
	registerBridgeTool(pi, "moai_context_search", "MoAI Context Search", "Search MoAI project/session context through the bridge.");
	registerBridgeTool(pi, "moai_worktree_create", "MoAI Worktree Create", "Create an isolated MoAI worktree through the bridge.");
	registerBridgeTool(pi, "moai_worktree_merge", "MoAI Worktree Merge", "Prepare or perform MoAI worktree merge handoff through the bridge.");
	registerBridgeTool(pi, "moai_worktree_cleanup", "MoAI Worktree Cleanup", "Clean up MoAI worktrees through the bridge.");
	registerBridgeTool(pi, "moai_task_create", "MoAI Task Create", "Create MoAI task state through the bridge.");
	registerBridgeTool(pi, "moai_task_update", "MoAI Task Update", "Update MoAI task state through the bridge.");
	registerBridgeTool(pi, "moai_task_list", "MoAI Task List", "List MoAI tasks through the bridge.");
	registerBridgeTool(pi, "moai_task_get", "MoAI Task Get", "Read a MoAI task through the bridge.");

	pi.registerTool({
		name: "moai_agent_invoke",
		label: "MoAI Agent Invoke",
		description: "List, run, parallel-run, or chain-run MoAI agents through Pi subprocess workers.",
		promptSnippet: "Invoke MoAI specialized agents",
		promptGuidelines: ["Use moai_agent_invoke when MoAI workflow requires specialized agent delegation."],
		parameters: AgentInvokeParams,
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			const response = await callBridge(ctx, { kind: "tool", payload: { tool: "moai_agent_invoke", ...params } }, { signal });
			return { content: [{ type: "text", text: bridgeResponseText(response) }], details: response };
		},
	});

	registerBridgeTool(pi, "moai_team_run", "MoAI Team Run", "Run a MoAI agent team through the bridge.");
}
