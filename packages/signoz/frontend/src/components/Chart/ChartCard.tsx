import React from 'react';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { Color } from '@signozhq/design-tokens';

interface ChartCardProps {
	title?: string;
	description?: string;
	children: React.ReactNode;
	className?: string;
}

export function ChartCard({
	title,
	description,
	children,
	className = '',
}: ChartCardProps): JSX.Element {
	const isDarkMode = useIsDarkMode();

	return (
		<div
			className={`rounded-lg border ${className}`}
			style={{
				backgroundColor: isDarkMode ? Color.BG_SLATE_500 : Color.BG_VANILLA_100,
				borderColor: isDarkMode ? Color.BG_SLATE_400 : Color.BG_VANILLA_300,
				boxShadow: isDarkMode
					? '0 1px 3px 0 rgba(0, 0, 0, 0.3)'
					: '0 1px 3px 0 rgba(0, 0, 0, 0.1)',
			}}
		>
			{(title || description) && (
				<div
					className="px-6 py-4 border-b"
					style={{
						borderColor: isDarkMode ? Color.BG_SLATE_400 : Color.BG_VANILLA_300,
					}}
				>
					{title && (
						<h3
							className="text-lg font-semibold"
							style={{
								color: isDarkMode ? Color.TEXT_VANILLA_100 : Color.TEXT_INK_400,
							}}
						>
							{title}
						</h3>
					)}
					{description && (
						<p
							className="mt-1 text-sm"
							style={{
								color: isDarkMode ? Color.TEXT_VANILLA_400 : Color.TEXT_INK_300,
							}}
						>
							{description}
						</p>
					)}
				</div>
			)}
			<div className="p-6">{children}</div>
		</div>
	);
}
