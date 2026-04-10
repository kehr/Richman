import { StorageKeys, storageGet, storageRemove, storageSet } from "@/domain/storage/local-storage";

export function setToken(token: string): void {
	storageSet(StorageKeys.authToken, token);
}

export function getToken(): string | null {
	return storageGet<string>(StorageKeys.authToken);
}

export function removeToken(): void {
	storageRemove(StorageKeys.authToken);
}

export function setUser(user: unknown): void {
	storageSet(StorageKeys.authUser, user);
}

export function getUser(): unknown | null {
	return storageGet<unknown>(StorageKeys.authUser);
}

export function clearAuth(): void {
	storageRemove(StorageKeys.authToken);
	storageRemove(StorageKeys.authUser);
}
