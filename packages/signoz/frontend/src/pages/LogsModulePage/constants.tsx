import { TabRoutes } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import { Compass, TowerControl } from 'lucide-react';
import LogsExplorer from 'pages/LogsExplorer';
import SaveView from 'pages/SaveView';

export const logsExplorer: TabRoutes = {
	Component: (): JSX.Element => <LogsExplorer />,
	name: (
		<div className="tab-item">
			<Compass size={16} /> Explorer
		</div>
	),
	route: ROUTES.LOGS,
	key: ROUTES.LOGS,
};

export const logSaveView: TabRoutes = {
	Component: SaveView,
	name: (
		<div className="tab-item">
			<TowerControl size={16} /> Views
		</div>
	),
	route: ROUTES.LOGS_SAVE_VIEWS,
	key: ROUTES.LOGS_SAVE_VIEWS,
};
