import { Typography } from 'antd';

export default function HostsEmptyOrIncorrectMetrics({
	noData,
	incorrectData,
}: {
	noData: boolean;
	incorrectData: boolean;
}): JSX.Element {
	return (
		<div className="hosts-empty-state-container">
			<div className="hosts-empty-state-container-content">
				{noData && (
					<div className="no-hosts-message">
						<Typography.Title level={5} className="no-hosts-message-title">
							No data received yet.
						</Typography.Title>
					</div>
				)}
			</div>
		</div>
	);
}
