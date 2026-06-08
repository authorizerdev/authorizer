import React from 'react';
import { ShieldOff } from 'lucide-react';

// FgaNotEnabled renders an informative empty state shown when the backend
// reports that fine-grained authorization is not enabled (the server is not
// running with --authorization-engine=fga).
const FgaNotEnabled = () => {
	return (
		<div className="flex min-h-[40vh] flex-col items-center justify-center text-center text-gray-400">
			<ShieldOff className="mb-4 h-16 w-16" />
			<p className="text-2xl font-bold text-gray-600">
				Fine-Grained Authorization is not enabled
			</p>
			<p className="mt-2 max-w-md text-sm text-gray-500">
				Start the Authorizer server with{' '}
				<code className="rounded bg-gray-100 px-1.5 py-0.5 text-gray-700">
					--authorization-engine=fga
				</code>{' '}
				to manage authorization models, relationship tuples and run access
				checks from this dashboard.
			</p>
		</div>
	);
};

export default FgaNotEnabled;
