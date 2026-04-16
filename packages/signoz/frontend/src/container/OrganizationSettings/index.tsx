import './OrganizationSettings.styles.scss';

import { Space } from 'antd';
import { useAppContext } from 'providers/App/App';

import DisplayName from './DisplayName';
import Members from './Members';
import PendingInvitesContainer from './PendingInvitesContainer';

function OrganizationSettings(): JSX.Element {
	const { org } = useAppContext();

	if (!org) {
		return <div />;
	}

	return (
		<div className="organization-settings-container">
			<Space direction="vertical">
				{org.map((e, index) => (
					<DisplayName key={e.id} id={e.id} index={index} />
				))}
			</Space>

			<PendingInvitesContainer />

			<Members />
		</div>
	);
}

export default OrganizationSettings;
