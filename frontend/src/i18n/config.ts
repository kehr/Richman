import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

import enApp from "./locales/en/app.json";
import enAuth from "./locales/en/auth.json";
import enCommon from "./locales/en/common.json";
import enSettings from "./locales/en/settings.json";
import zhApp from "./locales/zh/app.json";
import zhAuth from "./locales/zh/auth.json";
import zhCommon from "./locales/zh/common.json";
import zhSettings from "./locales/zh/settings.json";

export const resources = {
	en: { common: enCommon, auth: enAuth, app: enApp, settings: enSettings },
	zh: { common: zhCommon, auth: zhAuth, app: zhApp, settings: zhSettings },
} as const;

i18n
	.use(LanguageDetector)
	.use(initReactI18next)
	.init({
		resources,
		fallbackLng: "en",
		supportedLngs: ["en", "zh"],
		load: "languageOnly",
		defaultNS: "common",
		ns: ["common", "auth", "app", "settings"],
		interpolation: {
			escapeValue: false,
		},
		react: {
			useSuspense: false,
		},
		detection: {
			order: ["localStorage", "navigator"],
			lookupLocalStorage: "richman_locale",
			caches: ["localStorage"],
		},
	});

i18n.on("languageChanged", (lng) => {
	document.documentElement.lang = lng;
});

export default i18n;
