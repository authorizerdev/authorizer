import React from 'react';
import { NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useMutation, useQuery } from 'urql';
import {
	LayoutDashboard,
	Users,
	Webhook,
	Mail,
	ScrollText,
	SquareTerminal,
	LogOut,
	Menu,
	ExternalLink,
	Shield,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '../lib/utils';
import { useAuthContext } from '../contexts/AuthContext';
import { AdminLogout } from '../graphql/mutation';
import { MetaQuery } from '../graphql/queries';
import type { MetaResponse } from '../types';

interface NavItemConfig {
	name: string;
	icon: LucideIcon;
	route: string;
	external?: boolean;
}

const navItems: NavItemConfig[] = [
	{ name: 'Overview', icon: LayoutDashboard, route: '/' },
	{ name: 'Users', icon: Users, route: '/users' },
	{ name: 'Webhooks', icon: Webhook, route: '/webhooks' },
	{ name: 'Email Templates', icon: Mail, route: '/email-templates' },
	{ name: 'Authorization', icon: Shield, route: '/authorization' },
	{ name: 'Audit Logs', icon: ScrollText, route: '/audit-logs' },
	{
		name: 'API Playground',
		icon: SquareTerminal,
		route: '/playground',
		external: true,
	},
];

interface SidebarProps {
	onClose: () => void;
}

export const Sidebar = ({ onClose }: SidebarProps) => {
	const { pathname } = useLocation();
	const [{ data }] = useQuery<MetaResponse>({ query: MetaQuery });
	const [, logout] = useMutation(AdminLogout);
	const { setIsLoggedIn } = useAuthContext();
	const navigate = useNavigate();

	const handleLogout = async () => {
		await logout({});
		setIsLoggedIn(false);
		navigate('/', { replace: true });
	};

	return (
		<div className="fixed inset-y-0 left-0 z-40 flex h-full w-64 flex-col border-r border-gray-200 bg-white">
			{/* Logo */}
			<div className="flex h-16 items-center px-4">
				<NavLink to="/" className="flex items-center" onClick={onClose}>
					<img
						src="https://authorizer.dev/images/logo.png"
						alt="Authorizer logo"
						className="h-9"
					/>
					<span className="ml-2 text-lg tracking-widest font-semibold text-gray-800">
						AUTHORIZER
					</span>
				</NavLink>
			</div>

			{/* Navigation */}
			<nav className="flex-1 space-y-1 px-3 py-4">
				{navItems.map((item) => {
					if (item.external) {
						return (
							<a
								key={item.name}
								href={item.route}
								target="_blank"
								rel="noopener noreferrer"
								className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 hover:text-gray-900 transition-colors"
							>
								<item.icon className="h-4 w-4" />
								{item.name}
								<ExternalLink className="ml-auto h-3 w-3 text-gray-400" />
							</a>
						);
					}

					const isActive =
						item.route === '/'
							? pathname === '/'
							: pathname.startsWith(item.route);

					return (
						<NavLink
							key={item.name}
							to={item.route}
							onClick={onClose}
							className={cn(
								'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
								isActive
									? 'bg-blue-50 text-blue-600'
									: 'text-gray-700 hover:bg-gray-100 hover:text-gray-900',
							)}
						>
							<item.icon className="h-4 w-4" />
							{item.name}
						</NavLink>
					);
				})}
			</nav>

			{/* Footer */}
			<div className="border-t border-gray-200 px-3 py-3 space-y-2">
				<button
					onClick={handleLogout}
					className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 hover:text-red-600 transition-colors"
				>
					<LogOut className="h-4 w-4" />
					Logout
				</button>
				{data?.meta?.version && (
					<p className="px-3 text-xs text-gray-400">
						Version {data.meta.version}
					</p>
				)}
			</div>
		</div>
	);
};

interface MobileNavProps {
	onOpen: () => void;
}

export const MobileNav = ({ onOpen }: MobileNavProps) => {
	return (
		<div className="fixed top-0 right-0 left-0 z-30 flex h-16 items-center justify-between border-b border-gray-200 bg-white px-4 md:left-64">
			<button
				onClick={onOpen}
				className="rounded-md p-2 text-gray-600 hover:bg-gray-100 md:hidden"
				aria-label="Open menu"
			>
				<Menu className="h-5 w-5" />
			</button>

			<img
				src="https://authorizer.dev/images/logo.png"
				alt="Authorizer logo"
				className="h-9 md:hidden"
			/>

			<div className="w-10" />
		</div>
	);
};
