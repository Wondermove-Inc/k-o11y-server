import { Color } from '@signozhq/design-tokens';
import { Tag, Tooltip } from 'antd';
import { ColumnType } from 'antd/es/table';
import {
	K8sRolloutsData,
	K8sRolloutsListPayload,
} from 'api/infraMonitoring/getK8sRolloutsList';
import { Group } from 'lucide-react';
import { IBuilderQuery } from 'types/api/queryBuilder/queryBuilderData';

import {
	EntityProgressBar,
	formatBytes,
	ValidateColumnValueWrapper,
} from '../commonUtils';
import { IEntityColumn } from '../utils';

export const defaultAddedColumns: IEntityColumn[] = [
	{
		label: 'Rollout Name',
		value: 'rolloutName',
		id: 'rolloutName',
		canRemove: false,
	},
	{
		label: 'Namespace Name',
		value: 'namespaceName',
		id: 'namespaceName',
		canRemove: false,
	},
	{
		label: 'CPU Request Utilization (% of limit)',
		value: 'cpu_request',
		id: 'cpu_request',
		canRemove: false,
	},
	{
		label: 'CPU Limit Utilization (% of request)',
		value: 'cpu_limit',
		id: 'cpu_limit',
		canRemove: false,
	},
	{
		label: 'CPU Utilization (cores)',
		value: 'cpu',
		id: 'cpu',
		canRemove: false,
	},
	{
		label: 'Memory Request Utilization (% of limit)',
		value: 'memory_request',
		id: 'memory_request',
		canRemove: false,
	},
	{
		label: 'Memory Limit Utilization (% of request)',
		value: 'memory_limit',
		id: 'memory_limit',
		canRemove: false,
	},
	{
		label: 'Memory Utilization (bytes)',
		value: 'memory',
		id: 'memory',
		canRemove: false,
	},
];

export interface K8sRolloutsRowData {
	key: string;
	rolloutUID: string;
	rolloutName: React.ReactNode;
	cpu_request: React.ReactNode;
	cpu_limit: React.ReactNode;
	cpu: React.ReactNode;
	memory_request: React.ReactNode;
	memory_limit: React.ReactNode;
	memory: React.ReactNode;
	restarts: React.ReactNode;
	clusterName: string;
	namespaceName: string;
	groupedByMeta?: any;
}

const rolloutGroupColumnConfig = {
	title: (
		<div className="column-header entity-group-header">
			<Group size={14} /> ROLLOUT GROUP
		</div>
	),
	dataIndex: 'rolloutGroup',
	key: 'rolloutGroup',
	ellipsis: true,
	width: 150,
	align: 'left',
	sorter: false,
	className: 'column entity-group-header',
};

export const getK8sRolloutsListQuery = (): K8sRolloutsListPayload => ({
	filters: {
		items: [],
		op: 'and',
	},
	orderBy: { columnName: 'cpu', order: 'desc' },
});

const columnsConfig = [
	{
		title: (
			<div className="column-header-left rollout-name-header">Rollout Name</div>
		),
		dataIndex: 'rolloutName',
		key: 'rolloutName',
		ellipsis: true,
		width: 150,
		sorter: false,
		align: 'left',
	},
	{
		title: (
			<div className="column-header-left namespace-name-header">
				Namespace Name
			</div>
		),
		dataIndex: 'namespaceName',
		key: 'namespaceName',
		ellipsis: true,
		width: 150,
		sorter: false,
		align: 'left',
	},
	{
		title: <div className="column-header-left med-col">CPU Req Usage (%)</div>,
		dataIndex: 'cpu_request',
		key: 'cpu_request',
		width: 80,
		sorter: true,
		align: 'left',
	},
	{
		title: <div className="column-header-left med-col">CPU Limit Usage (%)</div>,
		dataIndex: 'cpu_limit',
		key: 'cpu_limit',
		width: 50,
		sorter: true,
		align: 'left',
	},
	{
		title: <div className="column-header- small-col">CPU Usage (cores)</div>,
		dataIndex: 'cpu',
		key: 'cpu',
		width: 80,
		sorter: true,
		align: 'left',
	},
	{
		title: <div className="column-header-left med-col">Mem Req Usage (%)</div>,
		dataIndex: 'memory_request',
		key: 'memory_request',
		width: 50,
		sorter: true,
		align: 'left',
	},
	{
		title: <div className="column-header-left med-col">Mem Limit Usage (%)</div>,
		dataIndex: 'memory_limit',
		key: 'memory_limit',
		width: 80,
		sorter: true,
		align: 'left',
	},
	{
		title: <div className="column-header-left small-col">Mem Usage (WSS)</div>,
		dataIndex: 'memory',
		key: 'memory',
		width: 120,
		sorter: true,
		align: 'left',
	},
];

export const getK8sRolloutsListColumns = (
	groupBy: IBuilderQuery['groupBy'],
): ColumnType<K8sRolloutsRowData>[] => {
	if (groupBy.length > 0) {
		const filteredColumns = [...columnsConfig].filter(
			(column) => column.key !== 'rolloutName',
		);
		filteredColumns.unshift(rolloutGroupColumnConfig);
		return filteredColumns as ColumnType<K8sRolloutsRowData>[];
	}

	return columnsConfig as ColumnType<K8sRolloutsRowData>[];
};

const dotToUnder: Record<string, keyof K8sRolloutsData['meta']> = {
	'k8s.rollout.name': 'k8s_rollout_name',
	'k8s.namespace.name': 'k8s_namespace_name',
	'k8s.cluster.name': 'k8s_cluster_name',
};

const getGroupByEle = (
	rollout: K8sRolloutsData,
	groupBy: IBuilderQuery['groupBy'],
): React.ReactNode => {
	const groupByValues: string[] = [];

	groupBy.forEach((group) => {
		const rawKey = group.key as string;
		const metaKey = (dotToUnder[rawKey] ?? rawKey) as keyof typeof rollout.meta;
		const value = rollout.meta[metaKey];
		groupByValues.push(value);
	});

	return (
		<div className="pod-group">
			{groupByValues.map((value) => (
				<Tag key={value} color={Color.BG_SLATE_400} className="pod-group-tag-item">
					{value === '' ? '<no-value>' : value}
				</Tag>
			))}
		</div>
	);
};

export const formatDataForTable = (
	data: K8sRolloutsData[],
	groupBy: IBuilderQuery['groupBy'],
): K8sRolloutsRowData[] =>
	data.map((rollout, index) => ({
		key: index.toString(),
		rolloutUID: `${rollout.meta.k8s_rollout_name || ''}||${rollout.meta.k8s_namespace_name || ''}`,
		rolloutName: (
			<Tooltip title={rollout.meta.k8s_rollout_name}>
				{rollout.meta.k8s_rollout_name}
			</Tooltip>
		),
		restarts: (
			<ValidateColumnValueWrapper value={rollout.restarts}>
				{rollout.restarts}
			</ValidateColumnValueWrapper>
		),
		cpu: (
			<ValidateColumnValueWrapper value={rollout.cpuUsage}>
				{rollout.cpuUsage}
			</ValidateColumnValueWrapper>
		),
		cpu_request: (
			<ValidateColumnValueWrapper value={rollout.cpuRequest}>
				<div className="progress-container">
					<EntityProgressBar value={rollout.cpuRequest} type="request" />
				</div>
			</ValidateColumnValueWrapper>
		),
		cpu_limit: (
			<ValidateColumnValueWrapper value={rollout.cpuLimit}>
				<div className="progress-container">
					<EntityProgressBar value={rollout.cpuLimit} type="limit" />
				</div>
			</ValidateColumnValueWrapper>
		),
		memory: (
			<ValidateColumnValueWrapper value={rollout.memoryUsage}>
				{formatBytes(rollout.memoryUsage)}
			</ValidateColumnValueWrapper>
		),
		memory_request: (
			<ValidateColumnValueWrapper value={rollout.memoryRequest}>
				<div className="progress-container">
					<EntityProgressBar value={rollout.memoryRequest} type="request" />
				</div>
			</ValidateColumnValueWrapper>
		),
		memory_limit: (
			<ValidateColumnValueWrapper value={rollout.memoryLimit}>
				<div className="progress-container">
					<EntityProgressBar value={rollout.memoryLimit} type="limit" />
				</div>
			</ValidateColumnValueWrapper>
		),
		clusterName: rollout.meta.k8s_cluster_name,
		namespaceName: rollout.meta.k8s_namespace_name,
		rolloutGroup: getGroupByEle(rollout, groupBy),
		meta: rollout.meta,
		...rollout.meta,
		groupedByMeta: rollout.meta,
	}));
