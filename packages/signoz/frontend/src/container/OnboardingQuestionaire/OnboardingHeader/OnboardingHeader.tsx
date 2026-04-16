import './OnboardingHeader.styles.scss';

export function OnboardingHeader(): JSX.Element {
	return (
		<div className="header-container">
			<div className="logo-container">
				<img src="/Logos/ko11y_logo_large.svg" alt="K-O11y" />
				<span className="logo-text">K-O11y +</span>
			</div>
		</div>
	);
}
