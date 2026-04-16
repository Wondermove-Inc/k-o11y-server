import React, { useMemo, useCallback } from 'react';
import { ResponsiveContainer, LineChart, Line, XAxis, YAxis, Tooltip } from 'recharts';
import { WorkloadMetric } from '../types';
import { useTimezone } from '../../../providers/Timezone';
import useTimezoneFormatter from '../../../hooks/useTimezoneFormatter/useTimezoneFormatter';

/**
 * MetricChart Component
 *
 * Performance Metrics 차트를 렌더링하는 최적화된 컴포넌트
 * - O(n) 알고리즘으로 데이터 변환 (Map 자료구조 사용)
 * - React.memo로 불필요한 재렌더링 방지
 * - useMemo로 데이터 변환 메모이제이션
 */

interface MetricChartProps {
  metricData: WorkloadMetric[];
  title: string;
  metricType: 'cpu' | 'memory' | 'networkIO' | 'networkErrors';
  isDarkMode: boolean;
}

interface SeriesInfo {
  queryName: string;
  lineKey: string;
  direction: string | null;
  interface: string | null;
}

interface ChartDataPoint {
  timestamp: string;
  [key: string]: string | number;
}

const convertValue = (value: number, metricType: string): number => {
  switch (metricType) {
    case 'memory':
      return value / (1024 * 1024);
    case 'networkIO':
      if (value >= 1024 * 1024) {
        return value / (1024 * 1024);
      } else if (value >= 1024) {
        return value / 1024;
      } else {
        return value;
      }
    case 'cpu':
      return value * 1000; // cores → millicores 변환
    case 'networkErrors':
    default:
      return value;
  }
};

const getUnit = (value: number, metricType: string): string => {
  switch (metricType) {
    case 'memory':
      return 'MiB';
    case 'networkIO':
      if (value >= 1024 * 1024) {
        return 'MiB/s';
      } else if (value >= 1024) {
        return 'KiB/s';
      } else {
        return 'B/s';
      }
    case 'cpu':
      return 'm'; // millicores (Kubernetes 표준)
    case 'networkErrors':
      return ' errors/min';
    default:
      return '';
  }
};

/**
 * O(n) 최적화된 차트 데이터 변환
 */
const transformChartData = (
  metricData: WorkloadMetric[],
  metricType: string
): { data: ChartDataPoint[]; seriesInfo: SeriesInfo[] } => {
  if (!metricData || !Array.isArray(metricData)) {
    return { data: [], seriesInfo: [] };
  }

  const seriesInfo: SeriesInfo[] = [];

  const metricMaps = metricData
    .map((metric: WorkloadMetric) => {
      if (!metric.values || !Array.isArray(metric.values)) return null;

      const valueMap = new Map<string, number>();
      metric.values.forEach((val) => {
        if (val.timestamp) {
          valueMap.set(val.timestamp, parseFloat(String(val.value)));
        }
      });

      const lineKey = metric.labels?.direction
        ? `line_${metric.queryName}_${metric.labels.direction}`
        : `line_${metric.queryName}`;

      seriesInfo.push({
        queryName: metric.queryName,
        lineKey,
        direction: metric.labels?.direction || null,
        interface: metric.labels?.interface || null,
      });

      return { valueMap, lineKey };
    })
    .filter((item): item is NonNullable<typeof item> => item !== null);

  const allTimestamps = new Set<string>();
  metricMaps.forEach(({ valueMap }) => {
    valueMap.forEach((_, timestamp) => allTimestamps.add(timestamp));
  });

  const sortedTimestamps = Array.from(allTimestamps).sort();

  const data: ChartDataPoint[] = sortedTimestamps.map((timestamp) => {
    const dataPoint: ChartDataPoint = { timestamp };

    metricMaps.forEach(({ valueMap, lineKey }) => {
      const rawValue = valueMap.get(timestamp);
      if (rawValue !== undefined) {
        dataPoint[lineKey] = convertValue(rawValue, metricType);
      }
    });

    return dataPoint;
  });

  return { data, seriesInfo };
};

const getLabelsForMetric = (
  metricType: string,
  seriesInfo: SeriesInfo[]
): Record<string, string> => {
  const labels: Record<string, string> = {};

  switch (metricType) {
    case 'cpu':
    case 'memory':
      seriesInfo.forEach((series) => {
        const queryNameMap: Record<string, string> = {
          A: 'usage',
          B: 'requests',
          C: 'limits',
        };
        labels[series.lineKey] = queryNameMap[series.queryName] || series.queryName;
      });
      break;

    case 'networkIO':
    case 'networkErrors':
      seriesInfo.forEach((series) => {
        if (series.direction) {
          labels[series.lineKey] = `${series.direction} :: eth0`;
        } else {
          labels[series.lineKey] = series.queryName;
        }
      });
      break;

    default:
      seriesInfo.forEach((series) => {
        labels[series.lineKey] = series.queryName;
      });
      break;
  }

  return labels;
};

const getColorsForMetric = (metricType: string, isDarkMode: boolean): string[] => {
  const lightModeColors = {
    cpu: ['#538BFF', '#9463FF', '#666666'],
    network: ['#538BFF', '#9882FF'],
  };

  const darkModeColors = {
    cpu: [
      'rgba(83, 139, 255, 0.7)',
      'rgba(148, 99, 255, 0.8)',
      'rgba(255, 255, 255, 0.7)',
    ],
    network: ['rgba(83, 139, 255, 0.7)', 'rgba(152, 130, 255, 0.8)'],
  };

  const colors = isDarkMode ? darkModeColors : lightModeColors;

  switch (metricType) {
    case 'cpu':
    case 'memory':
      return colors.cpu;
    case 'networkIO':
    case 'networkErrors':
      return colors.network;
    default:
      return [...colors.cpu, ...colors.network.slice(1, 2)];
  }
};

const MetricChart: React.FC<MetricChartProps> = ({ metricData, title, metricType, isDarkMode }) => {
  const { timezone } = useTimezone();
  const { formatTimezoneAdjustedTimestamp } = useTimezoneFormatter({ userTimezone: timezone });

    const { data: chartData, seriesInfo } = useMemo(
      () => transformChartData(metricData, metricType),
      [metricData, metricType]
    );

    const labelMap = useMemo(
      () => getLabelsForMetric(metricType, seriesInfo),
      [metricType, seriesInfo]
    );

    const colors = useMemo(
      () => getColorsForMetric(metricType, isDarkMode),
      [metricType, isDarkMode]
    );

    const yAxisUnit = useMemo(() => {
      const sampleValue =
        chartData.length > 0
          ? (Object.values(chartData[0]).find((v) => typeof v === 'number') as number) || 0
          : 0;
      return getUnit(sampleValue, metricType);
    }, [chartData, metricType]);

    const CustomTooltip = useCallback(
      ({ active, payload, label }: any) => {
        if (active && payload && payload.length) {
          return (
            <div
              style={{
                backgroundColor: isDarkMode ? 'var(--bg-slate-400, #1d212d)' : '#ffffff',
                border: isDarkMode ? '1px solid #2d3748' : '1px solid rgba(0, 0, 0, 0.15)',
                borderRadius: '6px',
                padding: '8px 12px',
                color: isDarkMode ? '#ffffff' : '#1f1f1f',
                fontSize: '12px',
                boxShadow: isDarkMode ? 'none' : '0 4px 12px rgba(0, 0, 0, 0.1)',
              }}
            >
              <p style={{ margin: 0, marginBottom: '4px', color: isDarkMode ? '#a0aec0' : '#6c757d' }}>
                {formatTimezoneAdjustedTimestamp(label, 'YYYY-MM-DD HH:mm:ss')}
              </p>
              {payload.map((entry: any, index: number) => {
                const formattedValue =
                  metricType === 'networkIO' || metricType === 'memory'
                    ? Number(entry.value).toFixed(2)
                    : entry.value;
                const unit = getUnit(entry.value, metricType);

                return (
                  <p key={index} style={{ margin: 0, color: entry.color }}>
                    {entry.name}: {formattedValue} {unit}
                  </p>
                );
              })}
            </div>
          );
        }
        return null;
      },
      [isDarkMode, metricType, formatTimezoneAdjustedTimestamp]
    );

    if (!chartData || chartData.length === 0 || !seriesInfo || seriesInfo.length === 0) {
      return (
        <div style={{ marginBottom: '20px' }}>
          <h4
            style={{
              color: isDarkMode ? '#ffffff' : '#1f1f1f',
              fontSize: '13px',
              marginBottom: '8px',
              fontWeight: 600,
            }}
          >
            {title}
          </h4>
          <div
            style={{
              height: '150px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: isDarkMode ? '#a0aec0' : '#6c757d',
              fontSize: '12px',
            }}
          >
            No data available
          </div>
        </div>
      );
    }

    return (
      <div style={{ marginBottom: '20px' }}>
        <h4
          style={{
            color: isDarkMode ? '#ffffff' : '#1f1f1f',
            fontSize: '13px',
            marginBottom: '8px',
            fontWeight: 600,
          }}
        >
          {title}
        </h4>
        <ResponsiveContainer width="100%" height={150}>
          <LineChart data={chartData}>
            <XAxis
              dataKey="timestamp"
              tick={{ fontSize: 10, fill: isDarkMode ? '#a0aec0' : '#6c757d' }}
              axisLine={{ stroke: isDarkMode ? '#2d3748' : '#d1d5db' }}
              tickLine={{ stroke: isDarkMode ? '#2d3748' : '#d1d5db' }}
              tickFormatter={(timestamp) => formatTimezoneAdjustedTimestamp(timestamp, 'HH:mm:ss')}
            />
            <YAxis
              tick={{ fontSize: 10, fill: isDarkMode ? '#a0aec0' : '#6c757d' }}
              axisLine={{ stroke: isDarkMode ? '#2d3748' : '#d1d5db' }}
              tickLine={{ stroke: isDarkMode ? '#2d3748' : '#d1d5db' }}
              label={{
                value: yAxisUnit,
                angle: -90,
                position: 'insideLeft',
                style: {
                  textAnchor: 'middle',
                  fontSize: '10px',
                  fill: isDarkMode ? '#a0aec0' : '#6c757d',
                },
              }}
              tickFormatter={(value) => {
                return metricType === 'memory' || metricType === 'networkIO'
                  ? Number(value).toFixed(2)
                  : value;
              }}
            />
            <Tooltip content={CustomTooltip} />
            {seriesInfo.map((series, index) => (
              <Line
                key={series.lineKey}
                type="monotone"
                dataKey={series.lineKey}
                stroke={colors[index % colors.length]}
                strokeWidth={1.5}
                dot={false}
                activeDot={{ r: 3, fill: colors[index % colors.length] }}
                name={labelMap[series.lineKey] || series.lineKey}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
        <div style={{ display: 'flex', gap: '16px', marginTop: '4px', flexWrap: 'wrap' }}>
          {seriesInfo.map((series, index) => (
            <div key={series.lineKey} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              <div
                style={{ width: '12px', height: '2px', backgroundColor: colors[index % colors.length] }}
              ></div>
              <span style={{ fontSize: '11px', color: isDarkMode ? '#a0aec0' : '#6c757d' }}>
                {labelMap[series.lineKey] || series.lineKey}
              </span>
            </div>
          ))}
        </div>
      </div>
    );
};

MetricChart.displayName = 'MetricChart';

export default MetricChart;
