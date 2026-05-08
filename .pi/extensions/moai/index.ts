import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { registerCommands } from "./commands.js";
import { registerEvents } from "./events.js";
import { registerRenderers } from "./renderers.js";
import { registerTools } from "./tools.js";
import { registerMoaiUI } from "./ui.js";

export default function moaiPiExtension(pi: ExtensionAPI): void {
	registerRenderers(pi);
	registerCommands(pi);
	registerTools(pi);
	registerMoaiUI(pi);
	registerEvents(pi);
}
