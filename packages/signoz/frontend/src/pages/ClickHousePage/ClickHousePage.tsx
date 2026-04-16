import NewDashboard from 'container/NewDashboard';
import { DashboardProvider } from 'providers/Dashboard/Dashboard';
import { CLICKHOUSE_DASHBOARD } from 'lib/dashboard/staticDashboards';

function ClickHousePage(): JSX.Element {
	return (
		<DashboardProvider staticDashboard={CLICKHOUSE_DASHBOARD as any}>
			<NewDashboard />
		</DashboardProvider>
	);
}

export default ClickHousePage;
