import React, { lazy, Suspense } from 'react';
import { Outlet, Route, Routes } from 'react-router-dom';

import { useAuthContext } from '../contexts/AuthContext';
import { DashboardLayout } from '../layouts/DashboardLayout';

const Auth = lazy(() => import('../pages/Auth'));
const Overview = lazy(() => import('../pages/Overview'));
const Users = lazy(() => import('../pages/Users'));
const Webhooks = lazy(() => import('../pages/Webhooks'));
const EmailTemplates = lazy(() => import('../pages/EmailTemplates'));
const AuditLogs = lazy(() => import('../pages/AuditLogs'));
const AuthorizationModel = lazy(() => import('../pages/authorization/Model'));
const AuthorizationTuples = lazy(() => import('../pages/authorization/Tuples'));
const Clients = lazy(() => import('../pages/Clients'));
const TrustedIssuers = lazy(() => import('../pages/TrustedIssuers'));

export const AppRoutes = () => {
	const { isLoggedIn } = useAuthContext();

	if (isLoggedIn) {
		return (
			<div>
				<Suspense fallback={<></>}>
					<Routes>
						<Route
							element={
								<DashboardLayout>
									<Outlet />
								</DashboardLayout>
							}
						>
							<Route path="/" element={<Overview />} />
							<Route path="users" element={<Users />} />
							<Route path="webhooks" element={<Webhooks />} />
							<Route path="email-templates" element={<EmailTemplates />} />
							<Route path="audit-logs" element={<AuditLogs />} />
							<Route
								path="authorization/model"
								element={<AuthorizationModel />}
							/>
							<Route
								path="authorization/tuples"
								element={<AuthorizationTuples />}
							/>
							<Route path="identity/clients" element={<Clients />} />
							<Route
								path="identity/trusted-issuers"
								element={<TrustedIssuers />}
							/>
							<Route path="*" element={<Overview />} />
						</Route>
					</Routes>
				</Suspense>
			</div>
		);
	}
	return (
		<Suspense fallback={<></>}>
			<Routes>
				<Route path="/" element={<Auth />} />
				<Route path="*" element={<Auth />} />
			</Routes>
		</Suspense>
	);
};
