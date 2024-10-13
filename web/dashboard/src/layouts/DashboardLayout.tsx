import {
	Box,
	Drawer,
	DrawerContent,
	useDisclosure,
	useColorModeValue,
} from '@chakra-ui/react';
import React, { ReactNode } from 'react';
import { Sidebar, MobileNav } from '../components/Menu';

export function DashboardLayout({ children }: { children: ReactNode }) {
	const { isOpen, onOpen, onClose } = useDisclosure();
	return (
		<Box minH="100vh" bg={useColorModeValue('gray.100', 'gray.900')}>
			<Sidebar
				onClose={() => onClose}
				display={{ base: 'none', md: 'block' }}
			/>
			<Drawer
				autoFocus={false}
				isOpen={isOpen}
				placement="left"
				onClose={onClose}
				returnFocusOnClose={false}
				onOverlayClick={onClose}
				size="full"
			>
				<DrawerContent>
					<Sidebar onClose={onClose} />
				</DrawerContent>
			</Drawer>
			{/* mobilenav */}
			<MobileNav onOpen={onOpen} />
			<Box ml={{ base: 0, md: '64' }} p="4" pt="24">
				{children}
			</Box>
		</Box>
	);
}
