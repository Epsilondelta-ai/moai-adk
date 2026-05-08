import { appendFileSync, mkdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { StringEnum } from "@earendil-works/pi-ai";
import { Type } from "typebox";

const InteractionKind = ["select", "confirm", "input", "editor", "notify"] as const;

type InteractionKindValue = (typeof InteractionKind)[number];

const InteractionQuestion = Type.Object({
	id: Type.Optional(Type.String({ description: "Stable question identifier returned with the answer." })),
	type: Type.Optional(StringEnum(InteractionKind, { description: "UI interaction type." })),
	title: Type.String({ description: "Question title shown to the user." }),
	message: Type.Optional(Type.String({ description: "Additional question body or explanation." })),
	options: Type.Optional(Type.Array(Type.String(), { maxItems: 4, description: "Options for select questions; max 4." })),
	defaultValue: Type.Optional(Type.String({ description: "Default text for input/editor." })),
	placeholder: Type.Optional(Type.String({ description: "Placeholder text for input." })),
});

export const AskUserParams = Type.Object({
	type: Type.Optional(StringEnum(InteractionKind, { description: "Single-question UI interaction type." })),
	title: Type.Optional(Type.String({ description: "Single-question title." })),
	message: Type.Optional(Type.String({ description: "Single-question message." })),
	options: Type.Optional(Type.Array(Type.String(), { maxItems: 4, description: "Single-question options; max 4." })),
	defaultValue: Type.Optional(Type.String({ description: "Single-question default text." })),
	placeholder: Type.Optional(Type.String({ description: "Single-question placeholder text." })),
	questions: Type.Optional(Type.Array(InteractionQuestion, { maxItems: 4, description: "Batch of up to 4 questions." })),
});

type AskUserInput = {
	type?: InteractionKindValue;
	title?: string;
	message?: string;
	options?: string[];
	defaultValue?: string;
	placeholder?: string;
	questions?: InteractionInputQuestion[];
};

type InteractionInputQuestion = {
	id?: string;
	type?: InteractionKindValue;
	title: string;
	message?: string;
	options?: string[];
	defaultValue?: string;
	placeholder?: string;
};

type InteractionAnswer = {
	id: string;
	type: InteractionKindValue;
	answer?: string | boolean;
	selectedIndex?: number;
	cancelled?: boolean;
};

export function registerInteractionTool(pi: ExtensionAPI): void {
	pi.registerTool({
		name: "moai_ask_user",
		label: "MoAI Ask User",
		description: "Ask the user structured MoAI questions through Pi UI controls. Use for all MoAI user-facing questions.",
		promptSnippet: "Ask the user structured MoAI questions through Pi UI controls",
		promptGuidelines: [
			"Use moai_ask_user for every question directed at the user in MoAI workflows.",
			"Ask at most 4 questions per call and at most 4 options per question.",
			"The first option is treated as the recommended option and displayed with a recommendation marker when missing.",
		],
		parameters: AskUserParams,
		async execute(_toolCallId, params: AskUserInput, _signal, _onUpdate, ctx) {
			if (!ctx.hasUI) {
				throw new Error("moai_ask_user requires interactive Pi UI mode");
			}
			const questions = normalizeQuestions(params);
			const answers: InteractionAnswer[] = [];
			for (const [index, question] of questions.entries()) {
				answers.push(await askOne(ctx, question, index));
			}
			const result = { answers, answeredAt: new Date().toISOString() };
			persistInteraction(ctx.cwd, result);
			pi.appendEntry("moai-interaction", result);
			return {
				content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
				details: result,
			};
		},
	});
}

function normalizeQuestions(params: AskUserInput): InteractionInputQuestion[] {
	const questions = params.questions?.length
		? params.questions
		: [{
			id: "q1",
			type: params.type,
			title: params.title ?? params.message ?? "MoAI question",
			message: params.message,
			options: params.options,
			defaultValue: params.defaultValue,
			placeholder: params.placeholder,
		}];
	if (questions.length === 0 || questions.length > 4) {
		throw new Error("moai_ask_user supports 1 to 4 questions per call");
	}
	for (const question of questions) {
		if (!question.title?.trim()) {
			throw new Error("Each moai_ask_user question requires a title");
		}
		if (question.options && question.options.length > 4) {
			throw new Error("Each moai_ask_user question supports at most 4 options");
		}
	}
	return questions;
}

async function askOne(ctx: ExtensionContext, question: InteractionInputQuestion, index: number): Promise<InteractionAnswer> {
	const type = inferType(question);
	const id = question.id || `q${index + 1}`;
	const title = question.message ? `${question.title}\n\n${question.message}` : question.title;
	switch (type) {
		case "select": {
			const options = withRecommendedMarker(question.options ?? ["Continue"], ctx.cwd);
			const selected = await ctx.ui.select(title, options);
			const selectedIndex = selected ? options.indexOf(selected) : -1;
			return { id, type, answer: selected, selectedIndex: selectedIndex >= 0 ? selectedIndex : undefined, cancelled: selected === undefined };
		}
		case "confirm": {
			const answer = await ctx.ui.confirm(question.title, question.message ?? question.title);
			return { id, type, answer, cancelled: false };
		}
		case "input": {
			const answer = await ctx.ui.input(title, question.placeholder ?? question.defaultValue ?? "");
			return { id, type, answer, cancelled: answer === undefined };
		}
		case "editor": {
			const answer = await ctx.ui.editor(title, question.defaultValue ?? "");
			return { id, type, answer, cancelled: answer === undefined };
		}
		case "notify": {
			ctx.ui.notify(question.message ?? question.title, "info");
			return { id, type, answer: true, cancelled: false };
		}
	}
}

function inferType(question: InteractionInputQuestion): InteractionKindValue {
	if (question.type) return question.type;
	if (question.options?.length) return "select";
	return "input";
}

function withRecommendedMarker(options: string[], cwd: string): string[] {
	if (options.length === 0 || options.length > 4) {
		throw new Error("Select questions require 1 to 4 options");
	}
	const marker = conversationLanguage(cwd) === "ko" ? "(권장)" : "(Recommended)";
	const first = options[0];
	if (first.includes("(권장)") || first.includes("(Recommended)")) {
		return options;
	}
	return [`${first} ${marker}`, ...options.slice(1)];
}

function conversationLanguage(cwd: string): string {
	try {
		const content = readFileSync(join(cwd, ".moai", "config", "sections", "language.yaml"), "utf8");
		const match = content.match(/conversation_language(?:_name)?:\s*['\"]?([^'\"\s#]+)/);
		if (match?.[1]) return match[1].toLowerCase();
	} catch {
		// default below
	}
	return "en";
}

function persistInteraction(cwd: string, result: unknown): void {
	try {
		const dir = join(cwd, ".moai", "runtime");
		mkdirSync(dir, { recursive: true });
		appendFileSync(join(dir, "pi-interactions.jsonl"), `${JSON.stringify(result)}\n`, "utf8");
	} catch {
		// Session entry persistence is the primary path; file persistence is best effort.
	}
}
