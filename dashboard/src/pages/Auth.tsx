import {
	Button,
	FormControl,
	FormLabel,
	Input,
	useToast,
	VStack,
} from '@chakra-ui/react';
import React, { useEffect } from 'react';
import { useMutation } from 'urql';

import { AuthLayout } from '../layouts/AuthLayout';
import { AdminLogin, AdminSignup } from '../graphql/mutation';
import { useNavigate } from 'react-router-dom';
import { useAuthContext } from '../contexts/AuthContext';
import { capitalizeFirstLetter, hasAdminSecret } from '../utils';

export const Auth = () => {
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
			setIsLoggedIn(true);
			if (res.data) {
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
			<form onSubmit={handleSubmit}>
				<VStack spacing="2.5" justify="space-between">
					<FormControl isRequired>
						<FormLabel htmlFor="admin-secret">
							{isLogin ? 'Enter' : 'Setup'} Admin Secret
						</FormLabel>
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
						colorScheme="blue"
						size="lg"
						w="100%"
						d="block"
						type="submit"
					>
						{isLogin ? 'Login' : 'Sign up'}
					</Button>
				</VStack>
			</form>
		</AuthLayout>
	);
};
