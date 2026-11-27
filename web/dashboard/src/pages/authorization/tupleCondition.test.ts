import { describe, expect, it } from 'vitest';
import {
	describeCondition,
	expiryCondition,
	parseDurationMs,
} from './tupleCondition';

describe('parseDurationMs', () => {
	it('parses Go durations', () => {
		expect(parseDurationMs('10m')).toBe(600_000);
		expect(parseDurationMs('1h30m')).toBe(5_400_000);
		expect(parseDurationMs('720h')).toBe(720 * 3_600_000);
		expect(parseDurationMs('1.5h')).toBe(5_400_000);
		expect(parseDurationMs('250ms')).toBe(250);
	});
	it('rejects anything else', () => {
		expect(parseDurationMs('')).toBeNull();
		expect(parseDurationMs('10 minutes')).toBeNull();
		expect(parseDurationMs('10d')).toBeNull();
	});
});

describe('expiryCondition', () => {
	it('stores grant_time (second precision, UTC) and grant_duration', () => {
		const c = expiryCondition('', '10m', new Date('2026-11-27T10:00:00.123Z'));
		expect(c).toEqual({
			name: 'non_expired_grant',
			context: { grant_time: '2026-11-27T10:00:00Z', grant_duration: '10m' },
		});
	});
});

describe('describeCondition', () => {
	const now = new Date('2026-11-27T10:05:00Z');
	it('a permanent grant has no condition', () => {
		expect(describeCondition(null, now)).toEqual({ kind: 'none' });
	});
	it('shows an unexpired grant', () => {
		const s = describeCondition(
			{
				name: 'non_expired_grant',
				context: { grant_time: '2026-11-27T10:00:00Z', grant_duration: '10m' },
			},
			now,
		);
		expect(s).toEqual({
			kind: 'expiry',
			expiresAt: new Date('2026-11-27T10:10:00Z'),
			expired: false,
		});
	});
	it('shows an expired grant', () => {
		const s = describeCondition(
			{
				name: 'non_expired_grant',
				context: { grant_time: '2026-11-27T09:00:00Z', grant_duration: '10m' },
			},
			now,
		);
		expect(s.kind === 'expiry' && s.expired).toBe(true);
	});
	it('shows any other condition verbatim', () => {
		expect(
			describeCondition(
				{ name: 'ip_ok', context: { cidr: '10.0.0.0/8' } },
				now,
			),
		).toEqual({
			kind: 'other',
			text: 'ip_ok {"cidr":"10.0.0.0/8"}',
		});
	});
});
