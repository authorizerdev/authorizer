import { Box, Image, Link, Text, Button } from '@chakra-ui/react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import React from 'react';
import { LOGO_URL } from '../constants';
import { useMutation } from 'urql';
import { AdminLogout } from '../graphql/mutation';
import { useAuthContext } from '../contexts/AuthContext';

const routes = [
	{
		route: '/users',
		name: 'Users',
	},
	{
		route: '/',
		name: 'Environment Variable',
	},
];

export const Sidebar = () => {
	const { pathname } = useLocation();
	const [_, logout] = useMutation(AdminLogout);
	const { setIsLoggedIn } = useAuthContext();
	const navigate = useNavigate();

	const handleLogout = async () => {
		await logout();
		setIsLoggedIn(false);
		navigate('/', { replace: true });
	};

	return (
		<Box as="nav" h="100%">
			<NavLink to="/">
				<Box d="flex" alignItems="center" p="4" mt="4" mb="10">
					<Image w="8" src={LOGO_URL} alt="" />
					<Text
						color="white"
						casing="uppercase"
						fontSize="1xl"
						ml="3"
						letterSpacing="1.5px"
					>
						Authorizer
					</Text>
				</Box>
			</NavLink>
			{routes.map(({ route, name }: any) => (
				<Link
					color={pathname === route ? 'blue.500' : 'white'}
					transition="all ease-in 0.2s"
					_hover={{ color: pathname === route ? 'blue.200' : 'whiteAlpha.700' }}
					px="4"
					py="2"
					bg={pathname === route ? 'white' : ''}
					fontWeight="bold"
					display="block"
					as={NavLink}
					key={name}
					to={route}
				>
					{name}
				</Link>
			))}

			<Box
				as="div"
				w="100%"
				position="absolute"
				bottom="5"
				display="flex"
				justifyContent="center"
			>
				<Button onClick={handleLogout}>Logout</Button>
			</Box>
		</Box>
	);
};
