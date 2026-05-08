import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";

export function registerRenderers(pi: ExtensionAPI): void {
	pi.registerMessageRenderer("moai-pi", (message, options, theme) => {
		const content = String(message.content ?? "");
		const label = theme.fg("accent", theme.bold("MoAI"));
		let text = content.startsWith("🤖 MoAI") || content.startsWith("★ ") ? content : `${label} ${content}`;
		if (options.expanded && message.details) {
			text += `\n${theme.fg("dim", JSON.stringify(message.details, null, 2))}`;
		}
		return new Text(text, 0, 0);
	});
}
