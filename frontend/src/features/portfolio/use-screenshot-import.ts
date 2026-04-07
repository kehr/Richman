import { useMutation } from "@tanstack/react-query";
import { importPortfolioScreenshot } from "./api";
import type { RecognizeResponse } from "./screenshot-types";

// useScreenshotImport wraps the multipart upload in a TanStack mutation so
// the modal can render initial / recognizing / recognized / failed states
// off mutation.status. The hook does not invalidate any caches because the
// recognize endpoint is read-only; the bulk-confirm step uses
// useCreateHolding individually for each accepted row.
export function useScreenshotImport() {
	return useMutation<RecognizeResponse, Error, File>({
		mutationFn: async (file: File) => {
			const res = await importPortfolioScreenshot(file);
			return res.data;
		},
	});
}
