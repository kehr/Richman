"use client";

import { useCallback, useEffect, useState } from "react";

const THEME_KEY = "theme_mode";
type ThemeMode = "light" | "dark";

function getInitialMode(): ThemeMode {
	if (typeof window === "undefined") return "light";
	const stored = localStorage.getItem(THEME_KEY);
	if (stored === "light" || stored === "dark") return stored;
	return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function useThemeMode(): { mode: ThemeMode; toggle: () => void } {
	const [mode, setMode] = useState<ThemeMode>("light");

	useEffect(() => {
		setMode(getInitialMode());
	}, []);

	const toggle = useCallback(() => {
		setMode((prev) => {
			const next = prev === "light" ? "dark" : "light";
			localStorage.setItem(THEME_KEY, next);
			return next;
		});
	}, []);

	return { mode, toggle };
}
