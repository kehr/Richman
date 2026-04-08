import type { OnboardingStatus, UserSettings } from "@/features/user-settings";
import { act, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode, useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	DEFAULT_ONBOARDING_STATE,
	ONBOARDING_DRAFT_STORAGE_KEY,
	OnboardingStateProvider,
	useOnboardingState,
} from "./state";

// Mock the feature-layer hooks so each test can vary server state without
// spinning up MSW handlers. The returned shapes match the real hook signatures
// (`data` from useQuery) so the provider's guards exercise the same branches.
const mockUseOnboardingStatus = vi.fn<() => { data: OnboardingStatus | undefined }>();
const mockUseUserSettings = vi.fn<() => { data: UserSettings | undefined }>();

vi.mock("@/features/user-settings", async () => {
	const actual = await vi.importActual<typeof import("@/features/user-settings")>(
		"@/features/user-settings",
	);
	return {
		...actual,
		useOnboardingStatus: () => mockUseOnboardingStatus(),
		useUserSettings: () => mockUseUserSettings(),
	};
});

// StateProbe renders the provider's current state as a testid-addressable JSON
// blob so assertions can read the entire shape without mounting a real page.
// A render-counting ref would be more ergonomic but the JSON blob keeps the
// assertion surface small and deterministic.
function StateProbe() {
	const { state } = useOnboardingState();
	return <pre data-testid="probe">{JSON.stringify(state)}</pre>;
}

function Wrapper({ children }: { children: ReactNode }) {
	return <OnboardingStateProvider>{children}</OnboardingStateProvider>;
}

function readProbe() {
	const raw = screen.getByTestId("probe").textContent ?? "{}";
	return JSON.parse(raw);
}

// Default mocks used by tests that do not care about server-side state. The
// "new user" status means both flags false, which is the only combination
// that lets the provider load a sessionStorage draft.
function newUserStatus(): { data: OnboardingStatus } {
	return { data: { completed: false, skipped: false } };
}

function emptyUserSettings(): { data: UserSettings } {
	return {
		data: {
			userId: 1,
			riskPreference: "neutral",
			categories: [],
			onboardingCompleted: false,
		},
	};
}

describe("OnboardingStateProvider", () => {
	beforeEach(() => {
		sessionStorage.clear();
		localStorage.clear();
		mockUseOnboardingStatus.mockReset();
		mockUseUserSettings.mockReset();
		mockUseOnboardingStatus.mockReturnValue(newUserStatus());
		mockUseUserSettings.mockReturnValue(emptyUserSettings());
	});

	afterEach(() => {
		sessionStorage.clear();
		localStorage.clear();
	});

	it("returns DEFAULT_ONBOARDING_STATE when no sessionStorage draft exists", () => {
		render(
			<Wrapper>
				<StateProbe />
			</Wrapper>,
		);
		expect(readProbe()).toEqual(DEFAULT_ONBOARDING_STATE);
	});

	it("restores the draft from sessionStorage on mount", () => {
		sessionStorage.setItem(
			ONBOARDING_DRAFT_STORAGE_KEY,
			JSON.stringify({
				categories: ["a_share_broad"],
				holdingDraft: { mode: "detail", assetCode: "000001", assetType: "a_share_broad" },
				reachedStep: 2,
				analysisFired: false,
			}),
		);
		render(
			<Wrapper>
				<StateProbe />
			</Wrapper>,
		);
		const state = readProbe();
		expect(state.categories).toEqual(["a_share_broad"]);
		expect(state.holdingDraft).toEqual({
			mode: "detail",
			assetCode: "000001",
			assetType: "a_share_broad",
		});
		expect(state.reachedStep).toBe(2);
	});

	it("wipes sessionStorage and falls back to default when status.completed=true", async () => {
		sessionStorage.setItem(
			ONBOARDING_DRAFT_STORAGE_KEY,
			JSON.stringify({ categories: ["stale"], reachedStep: 3 }),
		);
		mockUseOnboardingStatus.mockReturnValue({ data: { completed: true, skipped: false } });
		render(
			<Wrapper>
				<StateProbe />
			</Wrapper>,
		);
		await waitFor(() => {
			expect(readProbe()).toEqual(DEFAULT_ONBOARDING_STATE);
		});
		expect(sessionStorage.getItem(ONBOARDING_DRAFT_STORAGE_KEY)).toBeNull();
	});

	it("wipes sessionStorage and falls back to default when status.skipped=true", async () => {
		sessionStorage.setItem(
			ONBOARDING_DRAFT_STORAGE_KEY,
			JSON.stringify({ categories: ["stale"], reachedStep: 2 }),
		);
		mockUseOnboardingStatus.mockReturnValue({ data: { completed: false, skipped: true } });
		render(
			<Wrapper>
				<StateProbe />
			</Wrapper>,
		);
		await waitFor(() => {
			expect(readProbe()).toEqual(DEFAULT_ONBOARDING_STATE);
		});
		expect(sessionStorage.getItem(ONBOARDING_DRAFT_STORAGE_KEY)).toBeNull();
	});

	it("cascade-clears holdingDraft asset fields when categories shrink below the asset type", async () => {
		// Seed storage so the draft has both categories and a matching asset type.
		sessionStorage.setItem(
			ONBOARDING_DRAFT_STORAGE_KEY,
			JSON.stringify({
				categories: ["a_share_broad", "gold_etf"],
				holdingDraft: {
					mode: "detail",
					assetCode: "518880",
					assetName: "Gold ETF",
					assetType: "gold_etf",
				},
				reachedStep: 3,
				analysisFired: false,
			}),
		);

		// The Trigger helper lets the test synchronously call update() on the
		// provider's context from inside a child component.
		let shrinkCategories: (() => void) | null = null;
		function Trigger() {
			const { update } = useOnboardingState();
			useEffect(() => {
				shrinkCategories = () => update({ categories: ["a_share_broad"] });
			}, [update]);
			return null;
		}

		render(
			<Wrapper>
				<StateProbe />
				<Trigger />
			</Wrapper>,
		);

		// Sanity: the asset fields are loaded from storage.
		expect(readProbe().holdingDraft.assetType).toBe("gold_etf");

		await waitFor(() => expect(shrinkCategories).not.toBeNull());
		act(() => {
			shrinkCategories?.();
		});

		await waitFor(() => {
			const state = readProbe();
			expect(state.categories).toEqual(["a_share_broad"]);
			expect(state.holdingDraft.assetType).toBeUndefined();
			expect(state.holdingDraft.assetCode).toBeUndefined();
			expect(state.holdingDraft.assetName).toBeUndefined();
			// mode is preserved — only the asset-pointing fields are wiped.
			expect(state.holdingDraft.mode).toBe("detail");
		});
	});

	it("adopts server categories on first load when user-settings returns a non-empty list", async () => {
		mockUseUserSettings.mockReturnValue({
			data: {
				userId: 1,
				riskPreference: "neutral",
				categories: ["us_stock"],
				onboardingCompleted: false,
			},
		});
		render(
			<Wrapper>
				<StateProbe />
			</Wrapper>,
		);
		await waitFor(() => {
			expect(readProbe().categories).toEqual(["us_stock"]);
		});
	});
});
