import React, { lazy, Suspense } from 'react';
import { NavLink, Routes, Route, Navigate } from 'react-router-dom';
import { cn } from '../lib/utils';

const Resources = lazy(() => import('./authorization/Resources'));
const Scopes = lazy(() => import('./authorization/Scopes'));
const Policies = lazy(() => import('./authorization/Policies'));
const Permissions = lazy(() => import('./authorization/Permissions'));
const Evaluate = lazy(() => import('./authorization/Evaluate'));

const tabs = [
	{ name: 'Resources', path: 'resources' },
	{ name: 'Scopes', path: 'scopes' },
	{ name: 'Policies', path: 'policies' },
	{ name: 'Permissions', path: 'permissions' },
	{ name: 'Evaluate', path: 'evaluate' },
];

export default function Authorization() {
	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="my-4">
				<h1 className="text-2xl font-semibold text-gray-900">
					Authorization
				</h1>
				<p className="mt-1 text-sm text-gray-500">
					Fine-grained authorization management: resources, scopes, policies,
					and permissions.
				</p>
			</div>

			{/* Tab navigation */}
			<div className="border-b border-gray-200 mb-6">
				<nav className="-mb-px flex gap-4">
					{tabs.map((tab) => (
						<NavLink
							key={tab.path}
							to={tab.path}
							className={({ isActive }) =>
								cn(
									'whitespace-nowrap border-b-2 pb-3 pt-2 px-1 text-sm font-medium transition-colors',
									isActive
										? 'border-blue-500 text-blue-600'
										: 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
								)
							}
						>
							{tab.name}
						</NavLink>
					))}
				</nav>
			</div>

			{/* Tab content */}
			<Suspense fallback={<div className="py-8 text-center text-gray-400">Loading...</div>}>
				<Routes>
					<Route path="resources" element={<Resources />} />
					<Route path="scopes" element={<Scopes />} />
					<Route path="policies" element={<Policies />} />
					<Route path="permissions" element={<Permissions />} />
					<Route path="evaluate" element={<Evaluate />} />
					<Route path="" element={<Navigate to="resources" replace />} />
				</Routes>
			</Suspense>
		</div>
	);
}
