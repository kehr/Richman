// Structured content model for the Help page. Rather than shipping a
// markdown parser, help copy is authored as a typed JSON document where each
// section contains an ordered list of blocks. Rendering is then a simple
// switch over the block type, which keeps bundle size minimal and gives us
// deterministic table / code rendering without any third-party dependency.
//
// New block types must be added to both the `HelpBlock` union and the
// renderer in `frontend/src/pages/help/components/HelpSection.tsx`.

// ParagraphBlock renders as a single <p>. Use for narrative copy.
export interface ParagraphBlock {
	type: "paragraph";
	text: string;
}

// ListBlock renders as an unordered or ordered list.
export interface ListBlock {
	type: "list";
	ordered?: boolean;
	items: string[];
}

// TableBlock renders as a standard HTML table. The first row of `rows` is
// NOT the header — `headers` is a separate field to make column count
// validation obvious at author time.
export interface TableBlock {
	type: "table";
	headers: string[];
	rows: string[][];
}

// CodeBlock renders as a <pre><code> fenced block. `language` is informational
// only (no syntax highlighting in MVP) but kept so future upgrades can plug
// in a highlighter without changing JSON shape.
export interface CodeBlock {
	type: "code";
	language?: string;
	code: string;
}

// NoteBlock flags content the author had to infer because the PRD was silent
// on a specific number or threshold. Renders as a callout so reviewers can
// spot it during QA.
export interface NoteBlock {
	type: "note";
	text: string;
}

export type HelpBlock = ParagraphBlock | ListBlock | TableBlock | CodeBlock | NoteBlock;

// HelpSection is one of the nine top-level chapters in the help document.
// `id` is the anchor hash used for direct linking from decision cards,
// Settings, etc (e.g. `/help#badge`). Ids are stable across locales.
export interface HelpSection {
	id: string;
	title: string;
	blocks: HelpBlock[];
}

// HelpContent is the top-level document loaded by HelpPage. `sections` must
// appear in the order defined by PRD §7.2 so the sidebar navigation matches
// the scroll order.
export interface HelpContent {
	title: string;
	subtitle: string;
	sections: HelpSection[];
}
