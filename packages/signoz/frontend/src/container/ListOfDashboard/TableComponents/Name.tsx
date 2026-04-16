import { LockFilled } from '@ant-design/icons';
import ROUTES from 'constants/routes';
import history from 'lib/history';

import { Data } from '../DashboardsList';
import { TableLinkText } from './styles';

function Name(name: Data['name'], data: Data): JSX.Element {
	const { id: DashboardId, isLocked } = data;

	const getLink = (): string => `${ROUTES.ALL_DASHBOARD}/${DashboardId}`;

	const onClickHandler = (): void => {
		history.push(getLink());
	};

	return (
		<TableLinkText onClick={onClickHandler}>
			{isLocked && <LockFilled />} {name}
		</TableLinkText>
	);
}

export default Name;
