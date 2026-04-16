import NewDashboard from 'container/NewDashboard';
import { DashboardProvider } from 'providers/Dashboard/Dashboard';
import { INSIGHT_DASHBOARD } from 'lib/dashboard/staticDashboards';

function InsightPage(): JSX.Element {
	return (
		<DashboardProvider staticDashboard={INSIGHT_DASHBOARD as any}>
			<NewDashboard />
		</DashboardProvider>
	);
}

export default InsightPage;
