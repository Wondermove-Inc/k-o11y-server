import GridValueComponent from 'container/GridValueComponent';
import { getUPlotChartData } from 'lib/uPlotLib/utils/getUplotChartData';

import { PanelWrapperProps } from './panelWrapper.types';

function ValuePanelWrapper({
	widget,
	queryResponse,
	enableDrillDown = false,
}: PanelWrapperProps): JSX.Element {
	const { yAxisUnit, thresholds } = widget;
	const data = getUPlotChartData(queryResponse?.data?.payload);
	const formulaCount = widget?.query?.builder?.queryFormulas?.length || 0;
	const builderCount = widget?.query?.builder?.queryData?.length || 0;
	const formulaSeriesIndex =
		formulaCount > 0 ? 1 + builderCount : 1;
	const dataForValue =
		formulaCount > 0 && data?.length > formulaSeriesIndex
			? [data[0], data[formulaSeriesIndex]]
			: data;
	const formulaKey = widget?.query?.builder?.queryFormulas?.[0]?.queryName;
	const resultsTable =
		queryResponse?.data?.payload?.data?.newResult?.data?.result || [];
	const formulaValue =
		formulaKey &&
		resultsTable.reduce<number | undefined>((value, result) => {
			if (value !== undefined) return value;
			const rowData = result?.table?.rows?.[0]?.data;
			if (rowData && Object.prototype.hasOwnProperty.call(rowData, formulaKey)) {
				return rowData[formulaKey] as number;
			}
			return undefined;
		}, undefined);
	const dataNew = Object.values(
		queryResponse?.data?.payload?.data?.newResult?.data?.result[0]?.table
			?.rows?.[0]?.data || {},
	);

	// this is for handling both query_range v3 and v5 responses
	const gridValueData =
		formulaValue !== undefined
			? [[0], [formulaValue]]
			: dataForValue?.[0]?.length > 0
				? dataForValue
				: [[0], dataNew];

	return (
		<GridValueComponent
			data={gridValueData}
			yAxisUnit={yAxisUnit}
			thresholds={thresholds}
			widget={widget}
			queryResponse={queryResponse}
			contextLinks={widget.contextLinks}
			enableDrillDown={enableDrillDown}
		/>
	);
}

export default ValuePanelWrapper;
