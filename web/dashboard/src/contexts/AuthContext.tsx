import React, { createContext, useState, useContext, useEffect } from 'react';
import { useQuery } from 'urql';
import { useLocation, useNavigate } from 'react-router-dom';
import { Skeleton } from '../components/ui/skeleton';

import { AdminSessionQuery } from '../graphql/queries';
import type { AdminSessionResponse } from '../types';

interface AuthContextValue {
	isLoggedIn: boolean;
	setIsLoggedIn: (data: boolean) => void;
}

const AuthContext = createContext<AuthContextValue>({
	isLoggedIn: false,
	setIsLoggedIn: () => {},
});

export const AuthContextProvider = ({
	children,
}: {
	children: React.ReactNode;
}) => {
	const [isLoggedIn, setIsLoggedIn] = useState(false);

	const { pathname } = useLocation();
	const navigate = useNavigate();

	const [{ fetching, data, error }] = useQuery<AdminSessionResponse>({
		query: AdminSessionQuery,
	});

	useEffect(() => {
		if (!fetching && !error) {
			setIsLoggedIn(true);
			if (pathname === '/login' || pathname === 'signup') {
				navigate('/', { replace: true });
			}
		}
	}, [fetching, error]);

	if (fetching) {
		return (
			<div className="flex h-full items-center justify-center">
				<div className="space-y-4 w-64">
					<Skeleton className="h-8 w-full" />
					<Skeleton className="h-4 w-3/4" />
					<Skeleton className="h-4 w-1/2" />
				</div>
			</div>
		);
	}

	return (
		<AuthContext.Provider value={{ isLoggedIn, setIsLoggedIn }}>
			{children}
		</AuthContext.Provider>
	);
};

export const useAuthContext = () => useContext(AuthContext);
