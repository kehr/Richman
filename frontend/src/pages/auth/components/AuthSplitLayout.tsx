import { useTypewriter } from "@/domain/ui/use-typewriter";
import { Fragment, type ReactNode, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SampleDecisionCard } from "./SampleDecisionCard";

interface AuthSplitLayoutProps {
	form: ReactNode;
}

// Hold the fully-typed slogan on screen for 3 seconds before beginning the
// deletion animation. The default hook value (1.5s) is too short for users to
// comfortably read a slogan on page load, so we extend it at the call site
// without changing the hook's general-purpose default.
const AUTH_SLOGAN_HOLD_MS = 3000;

export function AuthSplitLayout({ form }: AuthSplitLayoutProps) {
	const { t } = useTranslation("auth");

	// AUTH_SLOGANS is the rotating copy shown in the hero section. Each entry is
	// a slogan, each slogan is an array of lines that render on their own visual
	// row. Keep every slogan at exactly two lines so the container can reserve a
	// stable min-height and prevent layout shift between rotations.
	const authSlogans = useMemo(
		() => t("hero.slogans", { returnObjects: true }) as readonly (readonly string[])[],
		[t],
	);

	// ARIA-visible fallback text. Screen readers get the brand-anchor slogan in
	// full immediately; the animated copy is aria-hidden so readers are not
	// flooded with partial strings as characters type in.
	const sloganAriaLabel = `${authSlogans[0]?.[0] ?? ""} ${authSlogans[0]?.[1] ?? ""}`;

	const typewriter = useTypewriter(authSlogans, { holdMs: AUTH_SLOGAN_HOLD_MS });
	const { displayed, cursorLine, isReducedMotion } = typewriter;

	return (
		<div className="auth-split-layout" data-testid="auth-split-layout">
			<style>{AUTH_SPLIT_LAYOUT_CSS}</style>
			<section className="auth-split-layout__left" data-testid="auth-split-layout-left">
				<div className="auth-split-layout__content">
					<header className="auth-split-layout__brand">
						<img
							src="/logo.svg"
							alt="Richman logo"
							className="auth-split-layout__brand-mark"
							width={36}
							height={36}
						/>
						<span className="auth-split-layout__brand-name">Richman</span>
					</header>

					<div className="auth-split-layout__hero">
						<h1
							className="auth-split-layout__slogan"
							aria-label={sloganAriaLabel}
							data-testid="auth-split-layout-slogan"
						>
							{displayed.map((line, lineIdx) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: every slogan has the same fixed number of lines in the same positional order, so line index is the stable identity here
								<Fragment key={lineIdx}>
									<span className="auth-split-layout__slogan-line" aria-hidden="true">
										{line}
										{lineIdx === cursorLine && !isReducedMotion && (
											<span className="auth-split-layout__slogan-cursor" aria-hidden="true" />
										)}
									</span>
									{lineIdx < displayed.length - 1 && <br aria-hidden="true" />}
								</Fragment>
							))}
						</h1>
						<p className="auth-split-layout__subtitle">{t("hero.subtitle")}</p>
					</div>

					<div className="auth-split-layout__sample">
						<SampleDecisionCard />
					</div>
				</div>

				<div className="auth-split-layout__footer-rail">
					<div className="auth-split-layout__footer">
						<span className="auth-split-layout__footer-mark">RICHMAN</span>
						<span className="auth-split-layout__footer-sep">·</span>
						<span className="auth-split-layout__footer-tag">DECISION, NOT NEWS</span>
					</div>
				</div>
			</section>

			<section className="auth-split-layout__right" data-testid="auth-split-layout-right">
				<div className="auth-split-layout__form-wrapper">{form}</div>
			</section>
		</div>
	);
}

// AUTH_SPLIT_LAYOUT_CSS keeps the responsive behavior fully in CSS so we can
// guarantee the desktop split is server-side-stable and not subject to JS
// resize listeners. Class names are scoped via the `auth-split-layout`
// prefix to avoid leaking into the rest of the app.
const AUTH_SPLIT_LAYOUT_CSS = `
.auth-split-layout {
	display: grid;
	grid-template-columns: 1fr;
	min-height: 100vh;
	width: 100%;
	color: #0b0b0d;
	font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "Segoe UI",
		"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
	font-feature-settings: "ss01", "cv11";
}
.auth-split-layout__left {
	position: relative;
	display: flex;
	flex-direction: column;
	padding: 56px 0 32px;
	background:
		radial-gradient(120% 80% at 100% 0%, rgba(11, 11, 13, 0.04) 0%, transparent 55%),
		linear-gradient(180deg, #fbfbfc 0%, #f3f3f5 100%);
	overflow: hidden;
}
.auth-split-layout__left::before {
	content: "";
	position: absolute;
	inset: 0;
	background-image:
		linear-gradient(to right, rgba(11, 11, 13, 0.035) 1px, transparent 1px),
		linear-gradient(to bottom, rgba(11, 11, 13, 0.035) 1px, transparent 1px);
	background-size: 48px 48px;
	mask-image: radial-gradient(120% 100% at 70% 50%, #000 30%, transparent 75%);
	-webkit-mask-image: radial-gradient(120% 100% at 70% 50%, #000 30%, transparent 75%);
	pointer-events: none;
}
.auth-split-layout__content {
	position: relative;
	flex: 1;
	display: flex;
	flex-direction: column;
	justify-content: center;
	gap: 44px;
	margin-inline: auto;
	max-width: 640px;
	width: calc(100% - 80px);
	box-sizing: border-box;
}
.auth-split-layout__brand {
	display: flex;
	align-items: center;
	gap: 12px;
}
.auth-split-layout__brand-mark {
	display: block;
}
.auth-split-layout__brand-name {
	font-size: 22px;
	font-weight: 500;
	letter-spacing: -0.01em;
	line-height: 1;
	color: #0b0b0d;
}
.auth-split-layout__hero {
	display: flex;
	flex-direction: column;
	gap: 20px;
}
.auth-split-layout__slogan {
	margin: 0;
	font-size: clamp(32px, 3.6vw, 52px);
	font-weight: 500;
	line-height: 1.12;
	letter-spacing: -0.02em;
	color: #0b0b0d;
	/* Reserve vertical space for two lines at the largest font-size so the
	   subtitle and sample card do not shift as slogans rotate. 2 lines x 1.12
	   line-height = 2.24em. */
	min-height: 2.24em;
}
.auth-split-layout__slogan-line {
	display: inline;
}
.auth-split-layout__slogan-cursor {
	display: inline-block;
	width: 0.08em;
	height: 0.9em;
	margin-left: 0.08em;
	vertical-align: -0.08em;
	background-color: currentColor;
	animation: auth-split-layout__slogan-blink 1.05s steps(2, start) infinite;
	will-change: opacity;
}
@keyframes auth-split-layout__slogan-blink {
	0% {
		opacity: 1;
	}
	100% {
		opacity: 0;
	}
}
@media (prefers-reduced-motion: reduce) {
	.auth-split-layout__slogan-cursor {
		animation: none;
		opacity: 0;
	}
}
.auth-split-layout__subtitle {
	margin: 0;
	font-size: 16px;
	line-height: 1.7;
	color: #4a4a52;
	max-width: 440px;
}
.auth-split-layout__sample {
	max-width: 560px;
}
.auth-split-layout__footer-rail {
	position: relative;
	margin-top: 40px;
}
.auth-split-layout__footer {
	position: relative;
	display: flex;
	align-items: center;
	gap: 10px;
	margin-inline: auto;
	max-width: 640px;
	width: calc(100% - 80px);
	padding-top: 24px;
	border-top: 1px solid rgba(11, 11, 13, 0.08);
	font-size: 11px;
	letter-spacing: 0.14em;
	color: #8e8e93;
	text-transform: uppercase;
}
.auth-split-layout__footer-mark {
	font-weight: 600;
	color: #4a4a52;
}
.auth-split-layout__footer-sep {
	opacity: 0.5;
}
.auth-split-layout__right {
	position: relative;
	display: flex;
	flex-direction: column;
	justify-content: center;
	padding: 56px 0;
	background: #ffffff;
}
.auth-split-layout__right::before {
	content: "";
	position: absolute;
	top: 0;
	bottom: 0;
	left: 0;
	width: 1px;
	background: linear-gradient(
		to bottom,
		transparent 0%,
		rgba(11, 11, 13, 0.08) 20%,
		rgba(11, 11, 13, 0.08) 80%,
		transparent 100%
	);
}
.auth-split-layout__form-wrapper {
	margin-inline: auto;
	max-width: 400px;
	width: calc(100% - 80px);
}
@media (min-width: 1024px) and (max-width: 1199px) {
	.auth-split-layout {
		grid-template-columns: 1fr 1fr;
	}
	.auth-split-layout__content,
	.auth-split-layout__footer {
		max-width: 540px;
	}
}
@media (min-width: 1200px) {
	.auth-split-layout {
		grid-template-columns: 6fr 4fr;
	}
}
@media (max-width: 1023px) {
	.auth-split-layout__left {
		padding: 48px 0 24px;
	}
	.auth-split-layout__content {
		width: calc(100% - 48px);
		gap: 32px;
	}
	.auth-split-layout__footer {
		width: calc(100% - 48px);
	}
	.auth-split-layout__slogan {
		font-size: clamp(28px, 6.5vw, 40px);
	}
	.auth-split-layout__right {
		padding: 48px 0;
	}
	.auth-split-layout__right::before {
		top: 0;
		left: 0;
		right: 0;
		bottom: auto;
		width: 100%;
		height: 1px;
		background: linear-gradient(
			to right,
			transparent 0%,
			rgba(11, 11, 13, 0.08) 20%,
			rgba(11, 11, 13, 0.08) 80%,
			transparent 100%
		);
	}
	.auth-split-layout__form-wrapper {
		width: calc(100% - 48px);
	}
}
`;
