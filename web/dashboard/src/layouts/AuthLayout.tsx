import React from 'react';
import { useQuery } from 'urql';
import { MetaQuery } from '../graphql/queries';
import { Skeleton } from '../components/ui/skeleton';
import type { MetaResponse } from '../types';

export function AuthLayout({ children }: { children: React.ReactNode }) {
	const [{ fetching, data }] = useQuery<MetaResponse>({ query: MetaQuery });
	return (
		<div className="flex h-screen flex-col items-center justify-center bg-gray-50 p-4">
			<div className="flex items-center mb-6">
				<img
					src="https://authorizer.dev/images/logo.png"
					alt="Authorizer logo"
					className="h-12"
				/>
				<span className="ml-3 text-xl tracking-widest font-semibold text-gray-800">
					AUTHORIZER
				</span>
			</div>

			{fetching ? (
				<div className="w-full max-w-md space-y-4">
					<Skeleton className="h-48 w-full rounded-lg" />
				</div>
			) : (
				<>
					<div className="w-full max-w-md rounded-lg bg-white p-6 shadow-sm border border-gray-200">
						{children}
					</div>
					<p className="mt-4 text-sm text-gray-500">
						Current Version: {data?.meta?.version}
					</p>
				</>
			)}
		</div>
	);
}
