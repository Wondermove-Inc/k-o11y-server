/* eslint-disable jsx-a11y/no-static-element-interactions */
/* eslint-disable jsx-a11y/click-events-have-key-events */
import './FullScreenHeader.styles.scss';

import history from 'lib/history';

export default function FullScreenHeader({
	overrideRoute,
}: {
	overrideRoute?: string;
}): React.ReactElement {
	const handleLogoClick = (): void => {
		history.push(overrideRoute || '/');
	};
	return (
		<div className="full-screen-header-container">
			<div className="brand-logo" onClick={handleLogoClick}>
				<img src="/Logos/ko11y_logo_large.svg" alt="K-O11y" />

				<div className="brand-logo-name">K-O11y +</div>
			</div>
		</div>
	);
}

FullScreenHeader.defaultProps = {
	overrideRoute: '/',
};
