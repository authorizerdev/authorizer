// tupleCondition.ts — helpers for conditional (time-limited) tuples on the
// Tuples page. The server evaluates conditions against its own clock
// (current_time); these helpers only build the stored parameters and describe
// them for display.

import type { FgaCondition } from '../../types';

// The condition name used by the time-limited templates in modelDsl.ts.
export const DEFAULT_EXPIRY_CONDITION = 'non_expired_grant';

// Expiry presets offered by the add form, as Go duration strings (the CEL
// `duration` type accepts the same syntax).
export const EXPIRY_PRESETS: { label: string; duration: string }[] = [
	{ label: '10 minutes', duration: '10m' },
	{ label: '1 hour', duration: '1h' },
	{ label: '1 day', duration: '24h' },
	{ label: '7 days', duration: '168h' },
	{ label: '30 days', duration: '720h' },
];

const UNIT_MS: Record<string, number> = {
	h: 3_600_000,
	m: 60_000,
	s: 1_000,
	ms: 1,
};

// parseDurationMs parses a Go-style duration ("10m", "1h30m", "720h",
// "1.5h"). Returns null for anything it does not understand.
export function parseDurationMs(d: string): number | null {
	const s = d.trim();
	if (!s) return null;
	const re = /(\d+(?:\.\d+)?)(ms|h|m|s)/g;
	let total = 0;
	let consumed = 0;
	for (const m of s.matchAll(re)) {
		total += parseFloat(m[1]) * UNIT_MS[m[2]];
		consumed += m[0].length;
	}
	return consumed === s.length ? total : null;
}

// expiryCondition builds the stored condition for a grant that starts now and
// lasts `duration`.
export function expiryCondition(
	name: string,
	duration: string,
	now: Date = new Date(),
): FgaCondition {
	return {
		name: name.trim() || DEFAULT_EXPIRY_CONDITION,
		context: {
			grant_time: now.toISOString().replace(/\.\d{3}Z$/, 'Z'),
			grant_duration: duration,
		},
	};
}

export type ConditionSummary =
	| { kind: 'none' }
	| { kind: 'expiry'; expiresAt: Date; expired: boolean }
	| { kind: 'other'; text: string };

// describeCondition turns a stored condition into something an admin can read
// at a glance. A grant_time + grant_duration pair is shown as an expiry; any
// other condition shows its name and parameters verbatim.
export function describeCondition(
	c: FgaCondition | null | undefined,
	now: Date = new Date(),
): ConditionSummary {
	if (!c) return { kind: 'none' };
	const ctx = c.context ?? {};
	const start =
		typeof ctx.grant_time === 'string' ? Date.parse(ctx.grant_time) : NaN;
	const dur =
		typeof ctx.grant_duration === 'string'
			? parseDurationMs(ctx.grant_duration)
			: null;
	if (!Number.isNaN(start) && dur !== null) {
		const expiresAt = new Date(start + dur);
		return { kind: 'expiry', expiresAt, expired: expiresAt <= now };
	}
	const params = Object.keys(ctx).length ? ` ${JSON.stringify(ctx)}` : '';
	return { kind: 'other', text: `${c.name}${params}` };
}
