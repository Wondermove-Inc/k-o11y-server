export type DurationUnit = 's' | 'm' | 'h';

export interface DurationInput {
	value: number;
	unit: DurationUnit;
}

export const parseDuration = (value: string): DurationInput => {
	const matches = value.matchAll(/(\d+)([hms])/g);
	let seconds = 0;
	for (const match of matches) {
		const amount = Number(match[1]);
		if (Number.isNaN(amount)) {
			continue;
		}
		if (match[2] === 'h') {
			seconds += amount * 3600;
		} else if (match[2] === 'm') {
			seconds += amount * 60;
		} else {
			seconds += amount;
		}
	}

	if (seconds % 3600 === 0 && seconds >= 3600) {
		return { value: seconds / 3600, unit: 'h' };
	}
	if (seconds % 60 === 0 && seconds >= 60) {
		return { value: seconds / 60, unit: 'm' };
	}
	return { value: seconds || 0, unit: 's' };
};

export const formatDuration = (input: DurationInput): string =>
	`${input.value}${input.unit}`;
