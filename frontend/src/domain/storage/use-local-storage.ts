import { useCallback, useState } from "react";
import { storageGet, storageRemove, storageSet } from "./local-storage";

// useLocalStorage mirrors the useState API but syncs to localStorage.
// The third return value (remove) deletes the key and resets to initialValue.
//
// Values are JSON-serialized, so T must be JSON-compatible.
// Storage unavailability (private mode, quota) is handled silently: reads
// fall back to initialValue, writes are best-effort no-ops.
export function useLocalStorage<T>(
	key: string,
	initialValue: T,
): [T, (value: T | ((prev: T) => T)) => void, () => void] {
	const [storedValue, setStoredValue] = useState<T>(() => {
		const item = storageGet<T>(key);
		return item !== null ? item : initialValue;
	});

	const setValue = useCallback(
		(value: T | ((prev: T) => T)) => {
			setStoredValue((prev) => {
				const next = typeof value === "function" ? (value as (p: T) => T)(prev) : value;
				storageSet(key, next);
				return next;
			});
		},
		[key],
	);

	const remove = useCallback(() => {
		storageRemove(key);
		setStoredValue(initialValue);
	}, [key, initialValue]);

	return [storedValue, setValue, remove];
}
