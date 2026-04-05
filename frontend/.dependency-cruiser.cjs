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
			comment: "Domain layer must not import from features or pages.",
			severity: "error",
			from: { path: "^src/domain/" },
			to: { path: "^src/(features|pages)/" },
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
