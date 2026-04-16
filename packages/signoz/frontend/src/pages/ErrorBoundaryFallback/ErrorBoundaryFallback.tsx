import './ErrorBoundaryFallback.styles.scss';

import { Button } from 'antd';
import ROUTES from 'constants/routes';
import { Home } from 'lucide-react';

function ErrorBoundaryFallback(): JSX.Element {
	const handleReload = (): void => {
		// Go to home page
		window.location.href = ROUTES.HOME;
	};

	return (
		<div className="error-boundary-fallback-container">
			<div className="error-boundary-fallback-content">
				<div className="error-icon">
					<img src="/Images/cloud.svg" alt="error-cloud-icon" />
				</div>
				<div className="title">Something went wrong :/</div>

				<div className="description">
					Our team is getting on top to resolve this.
				</div>

				<div className="actions">
					<Button
						type="primary"
						onClick={handleReload}
						icon={<Home size={16} />}
						className="periscope-btn primary"
					>
						Go to Home
					</Button>
				</div>
			</div>
		</div>
	);
}

export default ErrorBoundaryFallback;
