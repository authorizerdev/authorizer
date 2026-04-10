import React, { useEffect } from 'react';
import { useMutation } from 'urql';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

import { AuthLayout } from '../layouts/AuthLayout';
import { AdminLogin, AdminSignup } from '../graphql/mutation';
import { useAuthContext } from '../contexts/AuthContext';
import {
	capitalizeFirstLetter,
	getGraphQLErrorMessage,
	hasAdminSecret,
} from '../utils';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';

export default function Auth() {
	const [loginResult, login] = useMutation(AdminLogin);
	const [signUpResult, signup] = useMutation(AdminSignup);
	const { setIsLoggedIn } = useAuthContext();

	const navigate = useNavigate();
	const isLogin = hasAdminSecret();

	const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		const formValues = [...(e.target as HTMLFormElement).elements].reduce(
			(agg: Record<string, string>, elem) => {
				const el = elem as HTMLInputElement;
				if (el.id) {
					return { ...agg, [el.id]: el.value };
				}
				return agg;
			},
			{},
		);

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
		if (errors) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(errors, 'Authentication failed'),
				),
			);
		}
	}, [errors]);

	return (
		<AuthLayout>
			<p className="mb-2 text-center text-lg font-bold text-gray-600">
				Hello Admin
			</p>
			<p className="mb-8 text-center text-lg text-gray-500">
				Welcome to Admin Dashboard
			</p>
			<form onSubmit={handleSubmit} className="space-y-5">
				<div className="space-y-2">
					<Label htmlFor="admin-username">Username</Label>
					<Input
						id="admin-username"
						placeholder="Username"
						disabled
						value="admin"
						className="h-12"
					/>
				</div>
				<div className="space-y-2">
					<Label htmlFor="admin-secret">Password</Label>
					<Input
						id="admin-secret"
						placeholder="Admin secret"
						type="password"
						minLength={!isLogin ? 6 : 1}
						required
						className="h-12"
					/>
				</div>
				<Button
					isLoading={signUpResult.fetching || loginResult.fetching}
					className="w-full h-12 text-base"
					type="submit"
				>
					{isLogin ? 'Login' : 'Sign up'}
				</Button>
			</form>
		</AuthLayout>
	);
}
