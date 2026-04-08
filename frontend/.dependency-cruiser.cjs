/** @type {import('dependency-cruiser').IConfiguration} */
module.exports = {
	forbidden: [
		{
			name: "no-cross-feature-imports",
			comment: "Features must not import from other features.",
			severity: "error",
			from: { path: "^src/features/([^/]+)" },
			to: {
				path: "^src/features/([^/]+)",
				pathNot: "^src/features/$1",
			},
		},
		{
			name: "no-domain-to-feature",
			comment:
				"Domain layer must not import from features or pages. The only " +
				"allowed exception is that domain/money and the onboarding guard " +
				"consume the user-settings feature as their read-model for capital " +
				"and onboarding state.",
			severity: "error",
			from: {
				path: "^src/domain/",
				pathNot:
					"^src/domain/money/|^src/domain/auth/onboarding-guard\\.tsx$",
			},
			to: { path: "^src/(features|pages)/" },
		},
		{
			name: "domain-money-only-user-settings",
			comment:
				"domain/money may only cross into features via the user-settings " +
				"barrel; any other feature import is a layering violation.",
			severity: "error",
			from: { path: "^src/domain/money/" },
			to: {
				path: "^src/features/",
				pathNot: "^src/features/user-settings(/|$)",
			},
		},
		{
			name: "onboarding-guard-only-user-settings",
			comment:
				"domain/auth/onboarding-guard.tsx may only cross into features via " +
				"the user-settings barrel.",
			severity: "error",
			from: { path: "^src/domain/auth/onboarding-guard\\.tsx$" },
			to: {
				path: "^src/features/",
				pathNot: "^src/features/user-settings(/|$)",
			},
		},
		{
			name: "no-ui-kit-to-app-layers",
			comment: "UI kit must not import from features, pages, or domain.",
			severity: "error",
			from: { path: "^src/ui-kit/" },
			to: { path: "^src/(features|pages|domain)/" },
		},
	],
	options: {
		doNotFollow: {
			path: "node_modules",
		},
		tsPreCompilationDeps: true,
		tsConfig: {
			fileName: "tsconfig.json",
		},
	},
};
