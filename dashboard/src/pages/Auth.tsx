import {
	Button,
	FormControl,
	FormLabel,
	Input,
	useToast,
	VStack,
	Text,
} from '@chakra-ui/react';
import React, { useEffect } from 'react';
import { useMutation } from 'urql';

import { AuthLayout } from '../layouts/AuthLayout';
import { AdminLogin, AdminSignup } from '../graphql/mutation';
import { useNavigate } from 'react-router-dom';
import { useAuthContext } from '../contexts/AuthContext';
import { capitalizeFirstLetter, hasAdminSecret } from '../utils';

export default function Auth() {
	const [loginResult, login] = useMutation(AdminLogin);
	const [signUpResult, signup] = useMutation(AdminSignup);
	const { setIsLoggedIn } = useAuthContext();

	const toast = useToast();
	const navigate = useNavigate();
	const isLogin = hasAdminSecret();

	const handleSubmit = (e: any) => {
		e.preventDefault();
		const formValues = [...e.target.elements].reduce((agg: any, elem: any) => {
			if (elem.id) {
				return {
					...agg,
					[elem.id]: elem.value,
				};
			}

			return agg;
		}, {});

		(isLogin ? login : signup)({
			secret: formValues['admin-secret'],
		}).then((res) => {
			if (res.data) {
				setIsLoggedIn(true);
				navigate('/', { replace: true });
			}
		});
	};

	const errors = isLogin ? loginResult.error : signUpResult.error;

	useEffect(() => {
		if (errors?.graphQLErrors) {
			(errors?.graphQLErrors || []).map((error: any) => {
				toast({
					title: capitalizeFirstLetter(error.message),
					isClosable: true,
					status: 'error',
					position: 'bottom-right',
				});
			});
		}
	}, [errors]);

	return (
		<AuthLayout>
			<Text
				fontSize="large"
				textAlign="center"
				color="gray.600"
				fontWeight="bold"
				mb="2"
			>
				Hello Admin 👋 <br />
			</Text>
			<Text fontSize="large" textAlign="center" color="gray.500" mb="8">
				Welcome to Admin Dashboard
			</Text>
			<form onSubmit={handleSubmit}>
				<VStack spacing="5" justify="space-between">
					<FormControl isRequired>
						<FormLabel htmlFor="admin-username">Username</FormLabel>
						<Input
							size="lg"
							id="admin-username"
							placeholder="Username"
							disabled
							value="admin"
						/>
					</FormControl>
					<FormControl isRequired>
						<FormLabel htmlFor="admin-secret">Password</FormLabel>
						<Input
							size="lg"
							id="admin-secret"
							placeholder="Admin secret"
							type="password"
							minLength={!isLogin ? 6 : 1}
						/>
					</FormControl>
					<Button
						isLoading={signUpResult.fetching || loginResult.fetching}
						loadingText="Submitting"
						colorScheme="blue"
						size="lg"
						w="100%"
						type="submit"
					>
						{isLogin ? 'Login' : 'Sign up'}
					</Button>
					{isLogin ? (
						<Text color="gray.600" fontSize="sm">
							<b>Note:</b> In case if you have forgot your admin secret, you can
							reset it by updating <code>ADMIN_SECRET</code> environment
							variable. For more information, please refer to the{' '}
							<a href="https://docs.authorizer.dev/core/env/">documentation</a>.
						</Text>
					) : (
						<Text color="gray.600" fontSize="sm">
							<b>Note:</b> Configure the password to start using your dashboard.
						</Text>
					)}
				</VStack>
			</form>
		</AuthLayout>
	);
}
