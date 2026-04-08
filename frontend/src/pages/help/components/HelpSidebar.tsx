import type { HelpSection } from "@/i18n/help";

// HelpSidebar renders the 9-entry anchor navigation on the left side of the
// help page. It is a pure component — the page owns the IntersectionObserver
// state and passes `activeId` down. Clicking an entry scrolls to the section
// by updating the URL hash and calling scrollIntoView; this keeps deep-link
// behaviour (/help#badge) and in-page navigation consistent.

interface HelpSidebarProps {
	sections: Pick<HelpSection, "id" | "title">[];
	activeId: string | null;
	onNavigate: (id: string) => void;
}

export function HelpSidebar({ sections, activeId, onNavigate }: HelpSidebarProps) {
	return (
		<nav
			aria-label="Help sections"
			data-testid="help-sidebar"
			style={{
				width: 240,
				flexShrink: 0,
				position: "sticky",
				top: 16,
				alignSelf: "flex-start",
				paddingRight: 16,
				borderRight: "1px solid #f0f0f0",
				maxHeight: "calc(100vh - 32px)",
				overflowY: "auto",
			}}
		>
			<ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
				{sections.map((section) => {
					const isActive = section.id === activeId;
					return (
						<li key={section.id} style={{ marginBottom: 4 }}>
							<a
								href={`#${section.id}`}
								data-testid={`help-sidebar-link-${section.id}`}
								aria-current={isActive ? "location" : undefined}
								onClick={(event) => {
									event.preventDefault();
									onNavigate(section.id);
								}}
								style={{
									display: "block",
									padding: "8px 12px",
									borderRadius: 4,
									textDecoration: "none",
									color: isActive ? "#000" : "#595959",
									background: isActive ? "#f5f5f5" : "transparent",
									fontWeight: isActive ? 600 : 400,
									borderLeft: isActive ? "3px solid #000" : "3px solid transparent",
								}}
							>
								{section.title}
							</a>
						</li>
					);
				})}
			</ul>
		</nav>
	);
}
