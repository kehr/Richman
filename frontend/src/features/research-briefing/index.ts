// Public barrel for the research-briefing feature.
// Pages must consume this feature exclusively through this entry point.

export { fetchBriefing } from "./api";
export { BRIEFING_QUERY_KEY, useBriefing } from "./use-briefing";
export type { BriefingDto, BriefingCardDto } from "./types";
