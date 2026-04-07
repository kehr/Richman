import { Image } from "@/ui-kit/eat";

interface ImagePreviewProps {
	src: string;
}

// ImagePreview renders the uploaded screenshot inside a bordered, scrollable
// container. The wrapped antd Image component handles click-to-zoom via its
// built-in preview overlay (PRD §4.3 — left pane).
export function ImagePreview({ src }: ImagePreviewProps) {
	return (
		<div
			data-testid="screenshot-image-preview"
			style={{
				border: "1px solid #f0f0f0",
				borderRadius: 8,
				padding: 8,
				background: "#fafafa",
				maxHeight: 540,
				overflow: "auto",
				display: "flex",
				justifyContent: "center",
				alignItems: "flex-start",
			}}
		>
			<Image
				src={src}
				alt="portfolio screenshot"
				preview={{ mask: "点击查看大图" }}
				style={{ maxWidth: "100%", height: "auto" }}
			/>
		</div>
	);
}
