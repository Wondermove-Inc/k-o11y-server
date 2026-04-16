import { GetUsageDataAction } from './usage';

export enum ActionTypes {
	updateTimeInterval = 'UPDATE_TIME_INTERVAL',
	getServices = 'GET_SERVICES',
	getUsageData = 'GET_USAGE_DATE',
	fetchTraces = 'FETCH_TRACES',
	fetchTraceItem = 'FETCH_TRACE_ITEM',
}

export type Action = GetUsageDataAction;
