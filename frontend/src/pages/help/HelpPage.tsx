import { getHelpContent } from "@/i18n/help";
import { PageContainer, Typography } from "@/ui-kit/eat";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router";
import { HelpSection } from "./components/HelpSection";
import { HelpSidebar } from "./components/HelpSidebar";

const { Title, Paragraph } = Typography;

// HelpPage is the single-page reference document described in PRD §7. Layout
// is a 240px sticky sidebar on the left with anchor navigation, plus the
// scrolling main column on the right. The 9 sections come from the typed
// JSON content loaded via getHelpContent. The page is responsible for:
//
//   1. Reading the route hash on mount and scrolling the matching section
//      into view so `/help#badge` deep links work.
//   2. Tracking which section is currently visible via IntersectionObserver
//      and forwarding the active id to the sidebar so it can highlight.
//   3. Handling sidebar clicks by updating window.location.hash and calling
//      scrollIntoView on the target section.
//
// IntersectionObserver is created in a ref so it can be disconnected cleanly
// when the locale switches (which replaces all section DOM nodes).
export default function HelpPage() {
	const { i18n } = useTranslation();
	const locale = i18n.language as "en" | "zh";
	const content = useMemo(() => getHelpContent(locale), [locale]);
	const location = useLocation();

	const [activeId, setActiveId] = useState<string | null>(content.sections[0]?.id ?? null);
	const mainRef = useRef<HTMLDivElement | null>(null);

	// Scroll to the hash on mount and whenever the hash changes. We query the
	// DOM directly rather than refs because sections live in a child component
	// and refs would require plumbing one callback per section. Use instant
	// scroll on hash jumps so the IntersectionObserver cannot race with a
	// smooth-scroll animation and briefly highlight the top-of-page section.
	useEffect(() => {
		const hash = location.hash.replace(/^#/, "");
		if (!hash) return;
		const target = document.getElementById(hash);
		if (target) {
			target.scrollIntoView({ behavior: "instant", block: "start" });
			setActiveId(hash);
		}
	}, [location.hash]);

	// IntersectionObserver highlights whichever section heading is nearest the
	// top of the viewport. A persistent Set tracks all currently-visible
	// sections so the callback is not limited to the transient batch of changed
	// entries. On each update the topmost section in document order wins.
	// A single threshold (0) avoids repeated firing as a section partially
	// enters/leaves, which was the cause of the previous anchor flickering.
	useEffect(() => {
		if (typeof IntersectionObserver === "undefined") return;
		const visibleSet = new Set<string>();
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					const id = entry.target.getAttribute("id");
					if (!id) continue;
					if (entry.isIntersecting) visibleSet.add(id);
					else visibleSet.delete(id);
				}
				for (const section of content.sections) {
					if (visibleSet.has(section.id)) {
						setActiveId(section.id);
						return;
					}
				}
			},
			{
				rootMargin: "-10% 0px -60% 0px",
				threshold: 0,
			},
		);

		for (const section of content.sections) {
			const el = document.getElementById(section.id);
			if (el) observer.observe(el);
		}
		return () => observer.disconnect();
	}, [content]);

	const handleNavigate = (id: string) => {
		const target = document.getElementById(id);
		if (target) {
			target.scrollIntoView({ behavior: "smooth", block: "start" });
		}
		// Update the URL hash without a full navigation so deep links work.
		if (typeof window !== "undefined") {
			window.history.replaceState(null, "", `#${id}`);
		}
		setActiveId(id);
	};

	const sidebarSections = content.sections.map((section) => ({
		id: section.id,
		title: section.title,
	}));

	return (
		<PageContainer title={content.title} data-testid="help-page">
			<div
				style={{
					display: "flex",
					gap: 32,
					alignItems: "flex-start",
				}}
			>
				<HelpSidebar sections={sidebarSections} activeId={activeId} onNavigate={handleNavigate} />
				<div ref={mainRef} data-testid="help-main" style={{ flex: 1, minWidth: 0, maxWidth: 880 }}>
					<Title level={1} style={{ marginTop: 0 }}>
						{content.title}
					</Title>
					<Paragraph style={{ color: "#595959", marginBottom: 32 }}>{content.subtitle}</Paragraph>
					{content.sections.map((section) => (
						<HelpSection key={section.id} section={section} />
					))}
				</div>
			</div>
		</PageContainer>
	);
}
