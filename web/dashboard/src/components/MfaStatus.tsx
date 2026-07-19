import React from 'react';
import { Badge } from './ui/badge';

// Human-readable labels for the raw enrolled_mfa_methods identifiers the API
// returns. Anything unmapped (a future method) falls back to its raw id rather
// than being dropped.
const METHOD_LABELS: Record<string, string> = {
	totp: 'TOTP',
	webauthn: 'Passkey',
	email_otp: 'Email OTP',
	sms_otp: 'SMS OTP',
};

interface MfaStatusProps {
	// mfaServiceEnabled is the org-wide signal (_admin_meta
	// .is_multi_factor_auth_service_enabled): whether ANY MFA method is usable
	// on this server at all.
	mfaServiceEnabled: boolean;
	// enrolledMethods are the factors this user has actually verified.
	enrolledMethods?: string[];
	// isMfaEnabled is the per-user required-at-login flag
	// (is_multi_factor_auth_enabled).
	isMfaEnabled?: boolean;
}

// MfaStatus renders the three mutually-exclusive MFA states the dashboard shows
// per user. Shared by the Users table and the user-detail modal so both stay in
// lockstep.
//
//  1. Server has no MFA method available    → "Disabled" (org-wide, muted).
//  2. Available, user enrolled nothing        → "Not enrolled" (neutral).
//  3. Available, user has >=1 verified factor  → "Enrolled" + which factors,
//     plus whether MFA is actually required at login.
//
// State 2 also surfaces the required-at-login flag, since the reachable combo
// "is_multi_factor_auth_enabled=true but nothing enrolled yet" (e.g. mid
// offer/enroll flow) means the user will be forced to enroll at next login —
// worth showing rather than hiding behind a bare "Not enrolled".
const MfaStatus = ({
	mfaServiceEnabled,
	enrolledMethods,
	isMfaEnabled,
}: MfaStatusProps) => {
	if (!mfaServiceEnabled) {
		return <Badge variant="secondary">Disabled</Badge>;
	}

	const methods = enrolledMethods ?? [];
	const requiredLabel = isMfaEnabled ? 'Required at login' : 'Not required';

	if (methods.length === 0) {
		return (
			<div className="flex flex-col gap-1">
				<Badge variant="outline">Not enrolled</Badge>
				<span className="text-xs text-gray-500">{requiredLabel}</span>
			</div>
		);
	}

	return (
		<div className="flex flex-col gap-1">
			<Badge variant="success">Enrolled</Badge>
			<div className="flex flex-wrap gap-1">
				{methods.map((m) => (
					<Badge key={m} variant="secondary">
						{METHOD_LABELS[m] ?? m}
					</Badge>
				))}
			</div>
			<span className="text-xs text-gray-500">{requiredLabel}</span>
		</div>
	);
};

export default MfaStatus;
