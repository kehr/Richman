import { requestV1 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { LLMConsentRequest, LLMSettingsDTO, ProbeResultDTO, UpsertLLMRequest } from "./types";

// getLLMSettings loads the current user's LLM provider configuration. When
// the user has no active row the response's `configured` flag is false and
// the provider-specific fields are absent.
export function getLLMSettings() {
	return requestV1<ApiResponse<LLMSettingsDTO>>("/settings/llm");
}

// putLLMSettings upserts the LLM provider configuration. The handler
// encrypts the api key at rest; the plaintext `apiKey` in the body is
// discarded immediately on the server after encryption. An empty `apiKey`
// on edit means "leave the stored key unchanged".
export function putLLMSettings(body: UpsertLLMRequest) {
	return requestV1<ApiResponse<LLMSettingsDTO>>("/settings/llm", {
		method: "PUT",
		body: JSON.stringify(body),
	});
}

// deleteLLMSettings clears the user's active llm_configs row. The backend
// respects soft delete (is_deleted=true) so historical audit trails stay
// intact.
export function deleteLLMSettings() {
	return requestV1<ApiResponse<null>>("/settings/llm", {
		method: "DELETE",
	});
}

// postProbeLLMSettings runs a liveness probe against the stored provider
// (no payload — the backend reads the current encrypted config). Used by
// the "测试连通性" CTA.
export function postProbeLLMSettings() {
	return requestV1<ApiResponse<ProbeResultDTO>>("/settings/llm/probe", {
		method: "POST",
	});
}

// postLLMConsent writes the user's onboarding consent decision for the
// system-default provider. Called from the Onboarding consent step.
export function postLLMConsent(body: LLMConsentRequest) {
	return requestV1<ApiResponse<null>>("/onboarding/llm-consent", {
		method: "POST",
		body: JSON.stringify(body),
	});
}
