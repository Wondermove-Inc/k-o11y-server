import { Button, Typography } from 'antd';
import { RotateCw } from 'lucide-react';

function ErrorState({ refetch }: { refetch: () => void }): JSX.Element {
	return (
		<div className="error-state-container">
			<div className="error-state-content-wrapper">
				<div className="error-state-content">
					<div className="icon">
					</div>
					<div className="error-state-text">
						<Typography.Text>We ran into an error.</Typography.Text>
						<Typography.Text type="secondary">
							Please refresh this panel.
						</Typography.Text>
					</div>
				</div>
				<Button
					className="refresh-cta"
					onClick={(): void => refetch()}
					icon={<RotateCw size={16} />}
				>
					Refresh this panel
				</Button>
			</div>
		</div>
	);
}

export default ErrorState;
