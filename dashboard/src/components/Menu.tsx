import React, { Fragment, ReactNode } from 'react';
import {
	IconButton,
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
	Accordion,
	AccordionButton,
	AccordionPanel,
	AccordionItem,
	AccordionIcon,
	useMediaQuery,
} from '@chakra-ui/react';
import { FiUser, FiCode, FiSettings, FiMenu, FiUsers, FiChevronDown } from 'react-icons/fi';
import { BiCustomize } from 'react-icons/bi';
import { AiOutlineKey } from 'react-icons/ai';
import { SiOpenaccess, SiJsonwebtokens } from 'react-icons/si';
import { MdSecurity } from 'react-icons/md';
import { RiDatabase2Line } from 'react-icons/ri';
import { BsCheck2Circle } from 'react-icons/bs';
import { HiOutlineMail, HiOutlineOfficeBuilding } from 'react-icons/hi';
import { IconType } from 'react-icons';
import { ReactText } from 'react';
import { useMutation, useQuery } from 'urql';
import { NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useAuthContext } from '../contexts/AuthContext';
import { AdminLogout } from '../graphql/mutation';
import { MetaQuery } from '../graphql/queries';

interface SubRoutes {
	name: string;
	icon: IconType;
	route: string;
}

interface LinkItemProps {
	name: string;
	icon: IconType;
	route: string;
	subRoutes?: SubRoutes[];
}
const LinkItems: Array<LinkItemProps> = [
	{
		name: 'Environment ',
		icon: FiSettings,
		route: '/',
		subRoutes: [
			{
				name: 'OAuth Config',
				icon: AiOutlineKey,
				route: '/oauth-setting',
			},

			{ name: 'Roles', icon: FiUser, route: '/roles' },
			{
				name: 'JWT Secrets',
				icon: SiJsonwebtokens,
				route: '/jwt-config',
			},
			{
				name: 'Session Storage',
				icon: RiDatabase2Line,
				route: '/session-storage',
			},
			{
				name: 'Email Configurations',
				icon: HiOutlineMail,
				route: '/email-config',
			},
			{
				name: 'Domain White Listing',
				icon: BsCheck2Circle,
				route: '/whitelist-variables',
			},
			{
				name: 'Organization Info',
				icon: HiOutlineOfficeBuilding,
				route: '/organization-info',
			},
			{ name: 'Access Token', icon: SiOpenaccess, route: '/access-token' },
			{
				name: 'UI Customization',
				icon: BiCustomize,
				route: '/ui-customization',
			},
			{ name: 'Database', icon: RiDatabase2Line, route: '/db-cred' },
			{
				name: ' Security',
				icon: MdSecurity,
				route: '/admin-secret',
			},
		],
	},
	{ name: 'Users', icon: FiUsers, route: '/users' },
];

interface SidebarProps extends BoxProps {
	onClose: () => void;
}

export const Sidebar = ({ onClose, ...rest }: SidebarProps) => {
	const { pathname } = useLocation();
	const [{ fetching, data }] = useQuery({ query: MetaQuery });
	const [isNotSmallerScreen] = useMediaQuery('(min-width:600px)');
	return (
		<Box
			transition='3s ease'
			bg={useColorModeValue('white', 'gray.900')}
			borderRight='1px'
			borderRightColor={useColorModeValue('gray.200', 'gray.700')}
			w={{ base: 'full', md: 60 }}
			pos='fixed'
			h='full'
			{...rest}
		>
			<Flex h='20' alignItems='center' mx='18' justifyContent='space-between' flexDirection='column'>
				<NavLink to='/'>
					<Flex alignItems='center' mt='6'>
						<Image src='https://authorizer.dev/images/logo.png' alt='logo' height='36px' />
						<Text fontSize='large' ml='2' letterSpacing='3'>
							AUTHORIZER
						</Text>
					</Flex>
				</NavLink>
				<CloseButton display={{ base: 'flex', md: 'none' }} onClick={onClose} />
			</Flex>

			<Accordion defaultIndex={[0]} allowMultiple>
				<AccordionItem textAlign='center' border='none' w='100%'>
					{LinkItems.map((link) =>
						link?.subRoutes ? (
							<div key={link.name}>
								<AccordionButton>
									<Text as='div' fontSize='md'>
										<NavItem
											icon={link.icon}
											color={pathname === link.route ? 'blue.500' : ''}
											style={{ outline: 'unset' }}
											height={12}
											ml={-1}
											mb={isNotSmallerScreen ? -1 : -4}
											w={isNotSmallerScreen ? '100%' : '310%'}
										>
											<Fragment>
												{link.name}
												<Box display={{ base: 'none', md: 'flex' }} ml={12}>
													<FiChevronDown />
												</Box>
											</Fragment>
										</NavItem>
									</Text>
								</AccordionButton>
								<AccordionPanel>
									{link.subRoutes?.map((sublink) => (
										<NavLink key={sublink.name} to={sublink.route} onClick={onClose}>
											{' '}
											<Text as='div' fontSize='xs' ml={2}>
												<NavItem icon={sublink.icon} color={pathname === sublink.route ? 'blue.500' : ''} height={8}>
													{sublink.name}
												</NavItem>{' '}
											</Text>
										</NavLink>
									))}
								</AccordionPanel>
							</div>
						) : (
							<NavLink key={link.name} to={link.route}>
								{' '}
								<Text as='div' fontSize='md' w='100%' mt={-2}>
									<NavItem
										icon={link.icon}
										color={pathname === link.route ? 'blue.500' : ''}
										height={12}
										onClick={onClose}
									>
										{link.name}
									</NavItem>{' '}
								</Text>
							</NavLink>
						)
					)}
					<Link
						href='/playground'
						target='_blank'
						style={{
							textDecoration: 'none',
						}}
						_focus={{ _boxShadow: 'none' }}
					>
						<NavItem icon={FiCode}>API Playground</NavItem>
					</Link>
				</AccordionItem>
			</Accordion>

			{data?.meta?.version && (
				<Flex alignContent='center'>
					{' '}
					<Text color='gray.400' fontSize='sm' textAlign='center' position='absolute' bottom='5' left='7'>
						Current Version: {data.meta.version}
					</Text>
				</Flex>
			)}
		</Box>
	);
};

interface NavItemProps extends FlexProps {
	icon: IconType;
	children: ReactText | JSX.Element | JSX.Element[];
}
export const NavItem = ({ icon, children, ...rest }: NavItemProps) => {
	return (
		<Flex
			align='center'
			p='3'
			mx='3'
			borderRadius='md'
			role='group'
			cursor='pointer'
			_hover={{
				bg: 'blue.500',
				color: 'white',
			}}
			{...rest}
		>
			{icon && (
				<Icon
					mr='4'
					fontSize='16'
					_groupHover={{
						color: 'white',
					}}
					as={icon}
				/>
			)}
			{children}
		</Flex>
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
			height='20'
			position='fixed'
			right='0'
			left='0'
			alignItems='center'
			bg={useColorModeValue('white', 'gray.900')}
			borderBottomWidth='1px'
			borderBottomColor={useColorModeValue('gray.200', 'gray.700')}
			justifyContent={{ base: 'space-between', md: 'flex-end' }}
			zIndex={99}
			{...rest}
		>
			<IconButton
				display={{ base: 'flex', md: 'none' }}
				onClick={onOpen}
				variant='outline'
				aria-label='open menu'
				icon={<FiMenu />}
			/>

			<Image
				src='https://authorizer.dev/images/logo.png'
				alt='logo'
				height='36px'
				display={{ base: 'flex', md: 'none' }}
			/>

			<HStack spacing={{ base: '0', md: '6' }}>
				<Flex alignItems={'center'}>
					<Menu>
						<MenuButton py={2} transition='all 0.3s' _focus={{ boxShadow: 'none' }}>
							<HStack mr={5}>
								<FiUser />
								<VStack display={{ base: 'none', md: 'flex' }} alignItems='flex-start' spacing='1px' ml='2'>
									<Text fontSize='sm'>Admin</Text>
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
