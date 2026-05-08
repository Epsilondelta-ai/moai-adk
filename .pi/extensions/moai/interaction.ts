import { appendFileSync, mkdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { StringEnum } from "@earendil-works/pi-ai";
import { Editor, type EditorTheme, Key, matchesKey, Text, truncateToWidth } from "@earendil-works/pi-tui";
import { Type } from "typebox";

const InteractionKind = ["select", "confirm", "input", "editor", "notify"] as const;

type InteractionKindValue = (typeof InteractionKind)[number];

const InteractionOption = Type.Union([
	Type.String({ description: "Display label for the option." }),
	Type.Object({
		label: Type.String({ description: "Display label for the option." }),
		value: Type.Optional(Type.String({ description: "Stable value returned when selected." })),
		description: Type.Optional(Type.String({ description: "Option detail shown in the picker." })),
	}),
]);

const InteractionQuestion = Type.Object({
	id: Type.Optional(Type.String({ description: "Stable question identifier returned with the answer." })),
	type: Type.Optional(StringEnum(InteractionKind, { description: "UI interaction type." })),
	title: Type.String({ description: "Question title shown to the user." }),
	message: Type.Optional(Type.String({ description: "Additional question body or explanation." })),
	options: Type.Optional(Type.Array(InteractionOption, { maxItems: 4, description: "Options for select questions; max 4." })),
	defaultValue: Type.Optional(Type.String({ description: "Default text for input/editor." })),
	placeholder: Type.Optional(Type.String({ description: "Placeholder text for input." })),
});

export const AskUserParams = Type.Object({
	type: Type.Optional(StringEnum(InteractionKind, { description: "Single-question UI interaction type." })),
	title: Type.Optional(Type.String({ description: "Single-question title." })),
	message: Type.Optional(Type.String({ description: "Single-question message." })),
	options: Type.Optional(Type.Array(InteractionOption, { maxItems: 4, description: "Single-question options; max 4." })),
	defaultValue: Type.Optional(Type.String({ description: "Single-question default text." })),
	placeholder: Type.Optional(Type.String({ description: "Single-question placeholder text." })),
	questions: Type.Optional(Type.Array(InteractionQuestion, { maxItems: 4, description: "Batch of up to 4 questions." })),
});

type AskUserInput = {
	type?: InteractionKindValue;
	title?: string;
	message?: string;
	options?: InteractionInputOption[];
	defaultValue?: string;
	placeholder?: string;
	questions?: InteractionInputQuestion[];
};

type InteractionInputOption = string | {
	label: string;
	value?: string;
	description?: string;
};

type InteractionInputQuestion = {
	id?: string;
	type?: InteractionKindValue;
	title: string;
	message?: string;
	options?: InteractionInputOption[];
	defaultValue?: string;
	placeholder?: string;
};

type InteractionAnswer = {
	id: string;
	type: InteractionKindValue;
	answer?: string | boolean;
	value?: string | boolean;
	selectedIndex?: number;
	wasCustom?: boolean;
	cancelled?: boolean;
};

type DisplayOption = {
	label: string;
	value: string;
	description?: string;
	kind: "option" | "other" | "cancel";
	originalIndex?: number;
};

type ChoiceResult = {
	option?: DisplayOption;
	custom?: string;
	cancelled: boolean;
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
			"Select/confirm questions automatically provide a direct text input option and a visible cancel option.",
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
				if (answers[answers.length - 1]?.cancelled) break;
			}
			const result = { answers, answeredAt: new Date().toISOString() };
			persistInteraction(ctx.cwd, result);
			pi.appendEntry("moai-interaction", result);
			return {
				content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
				details: result,
			};
		},
		renderCall(args, theme) {
			const title = typeof args.title === "string" ? args.title : "MoAI question";
			const batchCount = Array.isArray(args.questions) ? args.questions.length : 1;
			const suffix = batchCount > 1 ? ` (${batchCount} questions)` : "";
			return new Text(theme.fg("toolTitle", theme.bold("moai_ask_user ")) + theme.fg("muted", `${title}${suffix}`), 0, 0);
		},
		renderResult(result, _options, theme) {
			const details = result.details as { answers?: InteractionAnswer[] } | undefined;
			const answers = details?.answers ?? [];
			if (answers.some((answer) => answer.cancelled)) return new Text(theme.fg("warning", "Cancelled"), 0, 0);
			if (answers.length === 0) return new Text(theme.fg("muted", "No answer"), 0, 0);
			const lines = answers.map((answer) => {
				const prefix = answer.wasCustom ? "(wrote) " : "";
				return `${theme.fg("success", "✓ ")}${theme.fg("accent", answer.id)}: ${prefix}${String(answer.answer ?? answer.value ?? "")}`;
			});
			return new Text(lines.join("\n"), 0, 0);
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
			const result = await askChoice(ctx, title, withRecommendedMarker(normalizeOptions(question.options ?? ["Continue"]), ctx.cwd));
			return answerFromChoice(id, type, result);
		}
		case "confirm": {
			const labels = localizedLabels(ctx.cwd);
			const result = await askChoice(ctx, title, withRecommendedMarker([
				{ label: labels.yes, value: "true" },
				{ label: labels.no, value: "false" },
			], ctx.cwd));
			if (result.cancelled) return { id, type, cancelled: true };
			if (result.custom !== undefined) return { id, type, answer: result.custom, value: result.custom, wasCustom: true, cancelled: false };
			const value = result.option?.value === "true";
			return { id, type, answer: value, value, selectedIndex: result.option?.originalIndex, cancelled: false };
		}
		case "input": {
			const answer = await ctx.ui.input(title, question.placeholder ?? question.defaultValue ?? "");
			return { id, type, answer, value: answer, wasCustom: true, cancelled: answer === undefined };
		}
		case "editor": {
			const answer = await ctx.ui.editor(title, question.defaultValue ?? "");
			return { id, type, answer, value: answer, wasCustom: true, cancelled: answer === undefined };
		}
		case "notify": {
			ctx.ui.notify(question.message ?? question.title, "info");
			return { id, type, answer: true, value: true, cancelled: false };
		}
	}
}

function answerFromChoice(id: string, type: InteractionKindValue, result: ChoiceResult): InteractionAnswer {
	if (result.cancelled) return { id, type, cancelled: true };
	if (result.custom !== undefined) {
		return { id, type, answer: result.custom, value: result.custom, wasCustom: true, cancelled: false };
	}
	const option = result.option;
	return {
		id,
		type,
		answer: option?.label,
		value: option?.value,
		selectedIndex: option?.originalIndex,
		wasCustom: false,
		cancelled: false,
	};
}

async function askChoice(ctx: ExtensionContext, title: string, options: DisplayOption[]): Promise<ChoiceResult> {
	const labels = localizedLabels(ctx.cwd);
	const allOptions: DisplayOption[] = [
		...options,
		{ label: labels.other, value: "__other__", description: labels.otherDescription, kind: "other" },
		{ label: labels.cancel, value: "__cancel__", description: labels.cancelDescription, kind: "cancel" },
	];

	return ctx.ui.custom<ChoiceResult>((tui, theme, _keybindings, done) => {
		let optionIndex = 0;
		let editMode = false;
		let cachedLines: string[] | undefined;

		const editorTheme: EditorTheme = {
			borderColor: (text) => theme.fg("accent", text),
			selectList: {
				selectedPrefix: (text) => theme.fg("accent", text),
				selectedText: (text) => theme.fg("accent", text),
				description: (text) => theme.fg("muted", text),
				scrollInfo: (text) => theme.fg("dim", text),
				noMatch: (text) => theme.fg("warning", text),
			},
		};
		const editor = new Editor(tui, editorTheme);

		editor.onSubmit = (value) => {
			const trimmed = value.trim();
			if (!trimmed) {
				editMode = false;
				editor.setText("");
				refresh();
				return;
			}
			done({ custom: trimmed, cancelled: false });
		};

		function refresh() {
			cachedLines = undefined;
			tui.requestRender();
		}

		function handleInput(data: string) {
			if (editMode) {
				if (matchesKey(data, Key.escape)) {
					editMode = false;
					editor.setText("");
					refresh();
					return;
				}
				editor.handleInput(data);
				refresh();
				return;
			}

			if (matchesKey(data, Key.up)) {
				optionIndex = Math.max(0, optionIndex - 1);
				refresh();
				return;
			}
			if (matchesKey(data, Key.down)) {
				optionIndex = Math.min(allOptions.length - 1, optionIndex + 1);
				refresh();
				return;
			}
			if (matchesKey(data, Key.enter)) {
				const selected = allOptions[optionIndex];
				if (selected.kind === "cancel") {
					done({ cancelled: true });
					return;
				}
				if (selected.kind === "other") {
					editMode = true;
					refresh();
					return;
				}
				done({ option: selected, cancelled: false });
				return;
			}
			if (matchesKey(data, Key.escape)) {
				done({ cancelled: true });
			}
		}

		function render(width: number): string[] {
			if (cachedLines) return cachedLines;
			const lines: string[] = [];
			const add = (text: string) => lines.push(truncateToWidth(text, width));

			add(theme.fg("accent", "─".repeat(width)));
			for (const line of title.split(/\r?\n/)) add(theme.fg("text", ` ${line}`));
			lines.push("");

			for (let i = 0; i < allOptions.length; i++) {
				const option = allOptions[i];
				const selected = i === optionIndex;
				const prefix = selected ? theme.fg("accent", "> ") : "  ";
				const label = `${i + 1}. ${option.label}${option.kind === "other" && editMode ? " ✎" : ""}`;
				add(prefix + theme.fg(selected ? "accent" : "text", label));
				if (option.description) add(`     ${theme.fg("muted", option.description)}`);
			}

			if (editMode) {
				lines.push("");
				add(theme.fg("muted", ` ${labels.answerPrompt}`));
				for (const line of editor.render(Math.max(1, width - 2))) add(` ${line}`);
			}

			lines.push("");
			add(theme.fg("dim", editMode ? labels.editHelp : labels.choiceHelp));
			add(theme.fg("accent", "─".repeat(width)));

			cachedLines = lines;
			return lines;
		}

		return {
			render,
			invalidate: () => {
				cachedLines = undefined;
			},
			handleInput,
		};
	}, { overlay: true });
}

function inferType(question: InteractionInputQuestion): InteractionKindValue {
	if (question.type) return question.type;
	if (question.options?.length) return "select";
	return "input";
}

function normalizeOptions(options: InteractionInputOption[]): DisplayOption[] {
	return options.map((option, index) => {
		const normalized = typeof option === "string" ? { label: option } : option;
		return {
			label: normalized.label,
			value: normalized.value ?? normalized.label,
			description: normalized.description,
			kind: "option",
			originalIndex: index,
		};
	});
}

function withRecommendedMarker(options: DisplayOption[], cwd: string): DisplayOption[] {
	if (options.length === 0 || options.length > 4) {
		throw new Error("Select questions require 1 to 4 options");
	}
	const marker = conversationLanguage(cwd) === "ko" ? "(권장)" : "(Recommended)";
	return options.map((option, index) => {
		if (index !== 0 || option.label.includes("(권장)") || option.label.includes("(Recommended)")) return option;
		return { ...option, label: `${option.label} ${marker}` };
	});
}

function localizedLabels(cwd: string) {
	const ko = conversationLanguage(cwd) === "ko";
	return ko
		? {
			yes: "예",
			no: "아니오",
			other: "직접 입력",
			otherDescription: "제공된 선택지 대신 직접 답변을 입력합니다.",
			cancel: "취소",
			cancelDescription: "질문을 취소하고 작업을 중단합니다.",
			answerPrompt: "직접 답변:",
			choiceHelp: " ↑↓ 선택 • Enter 확정 • Esc 취소",
			editHelp: " Enter 제출 • Esc 선택지로 돌아가기",
		}
		: {
			yes: "Yes",
			no: "No",
			other: "Type something",
			otherDescription: "Enter a free-form answer instead of the listed options.",
			cancel: "Cancel",
			cancelDescription: "Cancel this question and stop the current flow.",
			answerPrompt: "Your answer:",
			choiceHelp: " ↑↓ select • Enter confirm • Esc cancel",
			editHelp: " Enter submit • Esc back to options",
		};
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
