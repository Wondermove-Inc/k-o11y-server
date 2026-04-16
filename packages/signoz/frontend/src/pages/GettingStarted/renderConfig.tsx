import {
	AlertFilled,
	AlignLeftOutlined,
	ApiFilled,
	BarChartOutlined,
	DashboardFilled,
	SoundFilled,
} from '@ant-design/icons';
import store from 'store';

import { TGetStartedContentSection } from './types';

export const GetStartedContent = (): TGetStartedContentSection[] => {
	const {
		app: { currentVersion },
	} = store.getState();
	return [
		{
			heading: 'Send data from your applications to SigNoz',
			items: [
				{
					title: 'Instrument your Java Application',
					icon: (
						<img src={`/Logos/java.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/java/',
				},
				{
					title: 'Instrument your Python Application',
					icon: (
						<img src={`/Logos/python.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/python/',
				},
				{
					title: 'Instrument your JS Application',
					icon: (
						<img
							src={`/Logos/javascript.png?currentVersion=${currentVersion}`}
							alt=""
						/>
					),
					url: 'https://signoz.io/docs/instrumentation/javascript/',
				},
				{
					title: 'Instrument your Go Application',
					icon: (
						<img src={`/Logos/go.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/golang/',
				},
				{
					title: 'Instrument your .NET Application',
					icon: (
						<img
							src={`/Logos/ms-net-framework.png?currentVersion=${currentVersion}`}
							alt=""
						/>
					),
					url: 'https://signoz.io/docs/instrumentation/dotnet/',
				},
				{
					title: 'Instrument your PHP Application',
					icon: (
						<img src={`/Logos/php.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/php/',
				},
				{
					title: 'Instrument your Rails Application',
					icon: (
						<img src={`/Logos/rails.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/ruby-on-rails/',
				},
				{
					title: 'Instrument your Rust Application',
					icon: (
						<img src={`/Logos/rust.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/rust/',
				},
				{
					title: 'Instrument your Elixir Application',
					icon: (
						<img src={`/Logos/elixir.png?currentVersion=${currentVersion}`} alt="" />
					),
					url: 'https://signoz.io/docs/instrumentation/elixir/',
				},
			],
		},
		{
			heading: 'Send Metrics from your Infrastructure & create Dashboards',
			items: [
				{
					title: 'Send metrics to SigNoz',
					icon: <BarChartOutlined style={{ fontSize: '3.5rem' }} />,
					url: 'https://signoz.io/docs/userguide/send-metrics/',
				},
				{
					title: 'Create and Manage Dashboards',
					icon: <DashboardFilled style={{ fontSize: '3.5rem' }} />,
					url: 'https://signoz.io/docs/userguide/manage-dashboards-and-panels/',
				},
			],
		},
		{
			heading: 'Send your logs to SigNoz',
			items: [
				{
					title: 'Send your logs to SigNoz',
					icon: <AlignLeftOutlined style={{ fontSize: '3.5rem' }} />,
					url: 'https://signoz.io/docs/userguide/logs/',
				},
				{
					title: 'Existing log collectors to SigNoz',
					icon: <ApiFilled style={{ fontSize: '3.5rem' }} />,
					url: 'https://signoz.io/docs/userguide/fluentbit_to_signoz/',
				},
			],
		},
		{
			heading: 'Create alerts on Metrics',
			items: [
				{
					title: 'Create alert rules on metrics',
					icon: <AlertFilled style={{ fontSize: '3.5rem' }} />,
					url: 'https://signoz.io/docs/userguide/alerts-management/',
				},
				{
					title: 'Configure alert notification channels',
					icon: <SoundFilled style={{ fontSize: '3.5rem' }} />,
					url:
						'https://signoz.io/docs/userguide/alerts-management/#setting-up-a-notification-channel',
				},
			],
		},
	];
};
