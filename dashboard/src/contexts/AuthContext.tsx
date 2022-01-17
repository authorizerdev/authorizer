import React, { createContext, useState, useContext, useEffect } from 'react';
import { Center, Spinner } from '@chakra-ui/react';
import { useQuery } from 'urql';
import { useLocation, useNavigate } from 'react-router-dom';

import { AdminSessionQuery } from '../graphql/queries';
import { hasAdminSecret } from '../utils';

const AuthContext = createContext({
	isLoggedIn: false,
	setIsLoggedIn: (data: boolean) => {},
});

export const AuthContextProvider = ({ children }: { children: any }) => {
	const [isLoggedIn, setIsLoggedIn] = useState(false);

	const { pathname } = useLocation();
	const navigate = useNavigate();

	const [{ fetching, data, error }] = useQuery({
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
			<Center>
				<Spinner />
			</Center>
		);
	}

	return (
		<AuthContext.Provider value={{ isLoggedIn, setIsLoggedIn }}>
			{children}
		</AuthContext.Provider>
	);
};

export const useAuthContext = () => useContext(AuthContext);
