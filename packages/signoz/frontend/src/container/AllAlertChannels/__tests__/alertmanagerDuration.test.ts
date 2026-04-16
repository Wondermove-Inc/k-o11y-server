import { formatDuration, parseDuration } from '../alertmanagerDuration';

describe('alertmanagerDuration', () => {
	it('parses seconds when no unit aggregation is possible', () => {
		expect(parseDuration('45s')).toEqual({ value: 45, unit: 's' });
	});

	it('parses minutes when divisible by 60', () => {
		expect(parseDuration('120s')).toEqual({ value: 2, unit: 'm' });
	});

	it('parses hours when divisible by 3600', () => {
		expect(parseDuration('7200s')).toEqual({ value: 2, unit: 'h' });
	});

	it('parses compound durations', () => {
		expect(parseDuration('1h30m')).toEqual({ value: 90, unit: 'm' });
	});

	it('formats duration input', () => {
		expect(formatDuration({ value: 5, unit: 'm' })).toBe('5m');
	});
});
