import React from 'react';
import { Outlet, Route, Routes } from 'react-router-dom';

import { useAuthContext } from '../contexts/AuthContext';
import { DashboardLayout } from '../layouts/DashboardLayout';
import { Auth } from '../pages/Auth';
import { Home } from '../pages/Home';
import { Users } from '../pages/Users';

export const AppRoutes = () => {
	const { isLoggedIn } = useAuthContext();

	if (isLoggedIn) {
		return (
			<Routes>
				<Route
					element={
						<DashboardLayout>
							<Outlet />
						</DashboardLayout>
					}
				>
					<Route path="/" element={<Home />} />
					<Route path="users" element={<Users />} />
				</Route>
			</Routes>
		);
	}
	return (
		<Routes>
			<Route path="/" element={<Auth />} />
		</Routes>
	);
};
