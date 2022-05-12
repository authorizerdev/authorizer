import { Box, Flex, Image, Text, Spinner, useMediaQuery, Stack } from '@chakra-ui/react';
import React from 'react';
import { useQuery } from 'urql';
import { MetaQuery } from '../graphql/queries';

export function AuthLayout({ children }: { children: React.ReactNode }) {
	const [{ fetching, data }] = useQuery({ query: MetaQuery });
	const [isNotSmallerScreen] = useMediaQuery('(min-width:600px)');
	const [isVerticalRes] = useMediaQuery('(max-height:600px)');
	return (
		<Stack
			h='100vh'
			bg='gray.100'
			alignItems='center'
			justifyContent='center'
			direction={['column', 'column']}
			spacing={6}
		>
			<Flex alignItems='center'>
				<Image src='https://authorizer.dev/images/logo.png' alt='logo' height='50' />
				<Text fontSize='x-large' ml='3' letterSpacing='3'>
					AUTHORIZER
				</Text>
			</Flex>

			{fetching ? (
				<Spinner />
			) : (
				<>
					<Box p='6' m='5' rounded='5' bg='white' w={isNotSmallerScreen ? '500px' : '450px'} shadow='xl'>
						{children}
					</Box>
					<Text color='gray.600' fontSize='sm'>
						Current Version: {data.meta.version}
					</Text>
				</>
			)}
		</Stack>
	);
}
