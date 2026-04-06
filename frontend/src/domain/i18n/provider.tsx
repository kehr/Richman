import { type ReactNode, createContext, useCallback, useContext, useEffect, useState } from "react";
import en from "./en.json";
import zh from "./zh.json";

type Locale = "zh" | "en";
type Messages = Record<string, Record<string, string> | string>;

const messages: Record<Locale, Messages> = { zh, en };

interface I18nContextType {
	locale: Locale;
	setLocale: (locale: Locale) => void;
	t: (namespace: string, key: string) => string;
}

const I18nContext = createContext<I18nContextType | null>(null);

const LOCALE_KEY = "richman_locale";

export function I18nProvider({ children }: { children: ReactNode }) {
	const [locale, setLocaleState] = useState<Locale>(() => {
		if (typeof window === "undefined") return "zh";
		return (localStorage.getItem(LOCALE_KEY) as Locale) || "zh";
	});

	const setLocale = useCallback((newLocale: Locale) => {
		setLocaleState(newLocale);
		localStorage.setItem(LOCALE_KEY, newLocale);
		document.documentElement.lang = newLocale;
	}, []);

	useEffect(() => {
		document.documentElement.lang = locale;
	}, [locale]);

	const t = useCallback(
		(namespace: string, key: string): string => {
			const ns = messages[locale]?.[namespace];
			if (!ns) return `${namespace}.${key}`;
			if (typeof ns === "string") return ns;
			return ns[key] || `${namespace}.${key}`;
		},
		[locale],
	);

	return <I18nContext.Provider value={{ locale, setLocale, t }}>{children}</I18nContext.Provider>;
}

export function useLocale() {
	const ctx = useContext(I18nContext);
	if (!ctx) throw new Error("useLocale must be used within I18nProvider");
	return ctx;
}
