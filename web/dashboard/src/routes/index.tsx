import React, { lazy, Suspense } from 'react';
import { Outlet, Route, Routes } from 'react-router-dom';

import { useAuthContext } from '../contexts/AuthContext';
import { DashboardLayout } from '../layouts/DashboardLayout';
import EmailTemplates from '../pages/EmailTemplates';

const Auth = lazy(() => import('../pages/Auth'));
const Home = lazy(() => import('../pages/Home'));
const Users = lazy(() => import('../pages/Users'));
const Webhooks = lazy(() => import('../pages/Webhooks'));

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
							<Route path="/" element={<Users />} />
							<Route path="webhooks" element={<Webhooks />} />
							<Route path="email-templates" element={<EmailTemplates />} />
							<Route path="*" element={<Home />} />
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
