import React, { ReactNode } from 'react';
import {
	IconButton,
	Avatar,
	Box,
	CloseButton,
	Flex,
	Image,
	HStack,
	VStack,
	Icon,
	useColorModeValue,
	Link,
	Text,
	BoxProps,
	FlexProps,
	Menu,
	MenuButton,
	MenuItem,
	MenuList,
} from '@chakra-ui/react';
import {
	FiHome,
	FiTrendingUp,
	FiCompass,
	FiStar,
	FiSettings,
	FiMenu,
	FiUser,
	FiUsers,
	FiChevronDown,
} from 'react-icons/fi';
import { IconType } from 'react-icons';
import { ReactText } from 'react';
import { useMutation } from 'urql';
import { NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useAuthContext } from '../contexts/AuthContext';
import { AdminLogout } from '../graphql/mutation';

interface LinkItemProps {
	name: string;
	icon: IconType;
	route: string;
}
const LinkItems: Array<LinkItemProps> = [
	{ name: 'Home', icon: FiHome, route: '/' },
	{ name: 'Users', icon: FiUsers, route: '/users' },
	{ name: 'Environment Variables', icon: FiSettings, route: '/environment' },
];

interface SidebarProps extends BoxProps {
	onClose: () => void;
}

export const Sidebar = ({ onClose, ...rest }: SidebarProps) => {
	const { pathname } = useLocation();
	return (
		<Box
			transition="3s ease"
			bg={useColorModeValue('white', 'gray.900')}
			borderRight="1px"
			borderRightColor={useColorModeValue('gray.200', 'gray.700')}
			w={{ base: 'full', md: 60 }}
			pos="fixed"
			h="full"
			{...rest}
		>
			<Flex h="20" alignItems="center" mx="8" justifyContent="space-between">
				<NavLink to="/">
					<Flex alignItems="center">
						<Image
							src="https://authorizer.dev/images/logo.png"
							alt="logo"
							height="36px"
						/>
						<Text fontSize="large" ml="2" letterSpacing="3">
							AUTHORIZER
						</Text>
					</Flex>
				</NavLink>
				<CloseButton display={{ base: 'flex', md: 'none' }} onClick={onClose} />
			</Flex>
			{LinkItems.map((link) => (
				<NavLink key={link.name} to={link.route}>
					<NavItem
						icon={link.icon}
						color={pathname === link.route ? 'blue.500' : ''}
					>
						{link.name}
					</NavItem>
				</NavLink>
			))}
		</Box>
	);
};

interface NavItemProps extends FlexProps {
	icon: IconType;
	children: ReactText;
}
export const NavItem = ({ icon, children, ...rest }: NavItemProps) => {
	return (
		<Link
			href="#"
			style={{ textDecoration: 'none' }}
			_focus={{ boxShadow: 'none' }}
		>
			<Flex
				align="center"
				p="3"
				mx="3"
				borderRadius="md"
				role="group"
				cursor="pointer"
				_hover={{
					bg: 'blue.500',
					color: 'white',
				}}
				{...rest}
			>
				{icon && (
					<Icon
						mr="4"
						fontSize="16"
						_groupHover={{
							color: 'white',
						}}
						as={icon}
					/>
				)}
				{children}
			</Flex>
		</Link>
	);
};

interface MobileProps extends FlexProps {
	onOpen: () => void;
}
export const MobileNav = ({ onOpen, ...rest }: MobileProps) => {
	const [_, logout] = useMutation(AdminLogout);
	const { setIsLoggedIn } = useAuthContext();
	const navigate = useNavigate();

	const handleLogout = async () => {
		await logout();
		setIsLoggedIn(false);
		navigate('/', { replace: true });
	};

	return (
		<Flex
			ml={{ base: 0, md: 60 }}
			px={{ base: 4, md: 4 }}
			height="20"
			position="fixed"
			right="0"
			left="0"
			alignItems="center"
			bg={useColorModeValue('white', 'gray.900')}
			borderBottomWidth="1px"
			borderBottomColor={useColorModeValue('gray.200', 'gray.700')}
			justifyContent={{ base: 'space-between', md: 'flex-end' }}
			zIndex={99}
			{...rest}
		>
			<IconButton
				display={{ base: 'flex', md: 'none' }}
				onClick={onOpen}
				variant="outline"
				aria-label="open menu"
				icon={<FiMenu />}
			/>

			<Image
				src="https://authorizer.dev/images/logo.png"
				alt="logo"
				height="36px"
				display={{ base: 'flex', md: 'none' }}
			/>

			<HStack spacing={{ base: '0', md: '6' }}>
				<Flex alignItems={'center'}>
					<Menu>
						<MenuButton
							py={2}
							transition="all 0.3s"
							_focus={{ boxShadow: 'none' }}
						>
							<HStack>
								<FiUser />
								<VStack
									display={{ base: 'none', md: 'flex' }}
									alignItems="flex-start"
									spacing="1px"
									ml="2"
								>
									<Text fontSize="sm">Admin</Text>
								</VStack>
								<Box display={{ base: 'none', md: 'flex' }}>
									<FiChevronDown />
								</Box>
							</HStack>
						</MenuButton>
						<MenuList
							bg={useColorModeValue('white', 'gray.900')}
							borderColor={useColorModeValue('gray.200', 'gray.700')}
						>
							<MenuItem onClick={handleLogout}>Sign out</MenuItem>
						</MenuList>
					</Menu>
				</Flex>
			</HStack>
		</Flex>
	);
};
