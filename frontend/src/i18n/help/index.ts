// Help content loader. The help page consumes a pre-built, typed JSON
// document rather than runtime-translated strings, so the provider stays
// simple (zh / en are the only locales the app ships) and avoids duplicating
// structured content into the flat key-value files under domain/i18n/.
//
// Content is statically imported so Vite bundles both locales together and
// the user can switch languages without a network round-trip. At ~12 KB per
// locale this is well below any meaningful bundle budget.

import en from "./en.json";
import type { HelpContent } from "./types";
import zh from "./zh.json";

export type { HelpBlock, HelpContent, HelpSection } from "./types";

// Locale codes align with the existing useLocale hook (domain/i18n/provider).
// Keep this union in sync if new locales are added.
export type HelpLocale = "zh" | "en";

const CONTENT: Record<HelpLocale, HelpContent> = {
	zh: zh as HelpContent,
	en: en as HelpContent,
};

// getHelpContent returns the typed help document for the requested locale.
// Unknown locales fall back to zh so downstream rendering never crashes.
export function getHelpContent(locale: HelpLocale): HelpContent {
	return CONTENT[locale] ?? CONTENT.zh;
}

// getSectionIds returns the ordered list of section ids. Used by the page to
// set up the IntersectionObserver and by tests to assert the 9-section
// contract without hand-rolling the ids list.
export function getSectionIds(locale: HelpLocale): string[] {
	return getHelpContent(locale).sections.map((section) => section.id);
}
