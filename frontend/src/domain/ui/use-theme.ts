import { StorageKeys } from "@/domain/storage/local-storage";
import { useLocalStorage } from "@/domain/storage/use-local-storage";
import { useCallback } from "react";

type ThemeMode = "light" | "dark";

export function useThemeMode(): { mode: ThemeMode; toggle: () => void } {
	const [mode, setMode] = useLocalStorage<ThemeMode>(StorageKeys.themeMode, "light");

	const toggle = useCallback(() => {
		setMode((prev) => (prev === "light" ? "dark" : "light"));
	}, [setMode]);

	return { mode, toggle };
}
