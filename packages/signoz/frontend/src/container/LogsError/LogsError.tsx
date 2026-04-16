/* eslint-disable jsx-a11y/no-static-element-interactions */
/* eslint-disable jsx-a11y/click-events-have-key-events */
import './LogsError.styles.scss';

import { Typography } from 'antd';


export default function LogsError(): JSX.Element {



	return (
		<div className="logs-error-container">
			<div className="logs-error-content">
				<Typography.Text>
					Something went wrong. Please
					try again.
				</Typography.Text>

			</div>
		</div>
	);
}
