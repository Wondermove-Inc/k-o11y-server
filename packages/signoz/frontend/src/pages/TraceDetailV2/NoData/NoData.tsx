import './NoData.styles.scss';

import { Button, Typography } from 'antd';
import {  RefreshCw } from 'lucide-react';

function NoData(): JSX.Element {

	return (
		<div className="not-found-trace">
			<section className="description">
				<Typography.Text className="not-found-text-1">
					We cannot show the selected trace.
					<span className="not-found-text-2">
						This can happen in either of the two scenraios -
					</span>
				</Typography.Text>
			</section>
			<section className="reasons">
				<div className="reason-1">
					<Typography.Text className="text">
						The trace data has not been rendered yet. You can
						wait for a bit and refresh this page if this is the case.
					</Typography.Text>
				</div>
				<div className="reason-2">
					<Typography.Text className="text">
						The trace has been deleted as the data has crossed it’s retention period.
					</Typography.Text>
				</div>
			</section>
			<section className="none-of-above">
				<div className="action-btns">
					<Button
						className="action-btn"
						icon={<RefreshCw size={14} />}
						onClick={(): void => window.location.reload()}
					>
						Refresh this page
					</Button>
				</div>
			</section>
		</div>
	);
}

export default NoData;
