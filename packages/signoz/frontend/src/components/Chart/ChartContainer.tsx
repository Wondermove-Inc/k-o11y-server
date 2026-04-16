import React, { createContext, useContext, useId } from 'react';
import { ResponsiveContainer } from 'recharts';
import { Color } from '@signozhq/design-tokens';
import { useIsDarkMode } from 'hooks/useDarkMode';

export type ChartConfig = {
	[k in string]: {
		label?: React.ReactNode;
		color?: string;
		icon?: React.ComponentType;
	};
};

type ChartContextProps = {
	config: ChartConfig;
};

const ChartContext = createContext<ChartContextProps | null>(null);

export function useChart(): ChartContextProps {
	const context = useContext(ChartContext);

	if (!context) {
		throw new Error('useChart must be used within a <ChartContainer />');
	}

	return context;
}

interface ChartContainerProps extends React.ComponentProps<'div'> {
	config: ChartConfig;
	children: React.ComponentProps<typeof ResponsiveContainer>['children'];
}

export function ChartContainer({
	id,
	className = '',
	children,
	config,
	...props
}: ChartContainerProps): JSX.Element {
	const uniqueId = useId();
	const chartId = `chart-${id || uniqueId.replace(/:/g, '')}`;
	const isDarkMode = useIsDarkMode();

	return (
		<ChartContext.Provider value={{ config }}>
			<div
				data-chart={chartId}
				className={`relative flex h-full w-full ${className}`}
				{...props}
			>
				<ChartStyle id={chartId} config={config} isDarkMode={isDarkMode} />
				<ResponsiveContainer width="100%" height="100%">
					{children}
				</ResponsiveContainer>
			</div>
		</ChartContext.Provider>
	);
}

interface ChartStyleProps {
	id: string;
	config: ChartConfig;
	isDarkMode: boolean;
}

function ChartStyle({ id, config, isDarkMode }: ChartStyleProps): JSX.Element {
	const colorConfig = Object.entries(config).filter(([, itemConfig]) => itemConfig.color);

	const cssVars = colorConfig
		.map(([key, itemConfig]) => {
			if (!itemConfig.color) return null;
			return `  --color-${key}: ${itemConfig.color};`;
		})
		.filter(Boolean)
		.join('\n');

	// shadcn-style chart styling
	const axisColor = isDarkMode ? 'rgba(255, 255, 255, 0.5)' : 'rgba(0, 0, 0, 0.5)';
	const gridColor = isDarkMode ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';
	const cursorColor = isDarkMode ? 'rgba(255, 255, 255, 0.2)' : 'rgba(0, 0, 0, 0.1)';

	return (
		<style
			dangerouslySetInnerHTML={{
				__html: `
[data-chart="${id}"] {
${cssVars}
}

/* shadcn-style chart enhancements */
[data-chart="${id}"] .recharts-cartesian-axis-tick text {
  fill: ${axisColor};
  font-size: 12px;
}

[data-chart="${id}"] .recharts-cartesian-grid line {
  stroke: ${gridColor};
  stroke-dasharray: 3 3;
}

[data-chart="${id}"] .recharts-curve.recharts-tooltip-cursor {
  stroke: ${cursorColor};
}

[data-chart="${id}"] .recharts-rectangle.recharts-tooltip-cursor {
  fill: ${cursorColor};
}

[data-chart="${id}"] .recharts-dot {
  stroke: transparent;
}

[data-chart="${id}"] .recharts-layer {
  outline: none;
}

[data-chart="${id}"] .recharts-sector {
  outline: none;
  stroke: transparent;
  transition: opacity 0.2s ease;
}

[data-chart="${id}"] .recharts-sector:hover {
  opacity: 0.8;
}

[data-chart="${id}"] .recharts-surface {
  outline: none;
}

/* Smooth animations */
[data-chart="${id}"] .recharts-area,
[data-chart="${id}"] .recharts-line {
  animation: chartFadeIn 0.5s ease-in-out;
}

@keyframes chartFadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
`,
			}}
		/>
	);
}
