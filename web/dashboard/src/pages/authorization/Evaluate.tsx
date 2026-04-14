import React from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { Play, ShieldCheck, ShieldX, Info } from 'lucide-react';
import {
	CheckPermissionQuery,
	ResourcesQuery,
	ScopesQuery,
} from '../../graphql/queries';
import { getGraphQLErrorMessage } from '../../utils';
import { Button } from '../../components/ui/button';
import { Label } from '../../components/ui/label';
import { Badge } from '../../components/ui/badge';
import { Skeleton } from '../../components/ui/skeleton';
import { Select } from '../../components/ui/select';
import type {
	AuthzResource,
	AuthzResourcesResponse,
	AuthzScope,
	AuthzScopesResponse,
	CheckPermissionResponse,
} from '../../types';

export default function Evaluate() {
	const client = useClient();
	const [resources, setResources] = React.useState<AuthzResource[]>([]);
	const [scopes, setScopes] = React.useState<AuthzScope[]>([]);
	const [selectedResource, setSelectedResource] = React.useState('');
	const [selectedScope, setSelectedScope] = React.useState('');
	const [loading, setLoading] = React.useState(false);
	const [evaluating, setEvaluating] = React.useState(false);
	const [result, setResult] = React.useState<{
		allowed: boolean;
		matched_policy?: string;
	} | null>(null);

	React.useEffect(() => {
		const fetchData = async () => {
			setLoading(true);
			const [resourcesRes, scopesRes] = await Promise.all([
				client
					.query<AuthzResourcesResponse>(ResourcesQuery, {
						params: { pagination: { limit: 100, page: 1 } },
					})
					.toPromise(),
				client
					.query<AuthzScopesResponse>(ScopesQuery, {
						params: { pagination: { limit: 100, page: 1 } },
					})
					.toPromise(),
			]);
			if (resourcesRes.data?._resources) {
				setResources(resourcesRes.data._resources.resources);
			}
			if (scopesRes.data?._scopes) {
				setScopes(scopesRes.data._scopes.scopes);
			}
			setLoading(false);
		};
		fetchData();
	}, []);

	const handleEvaluate = async () => {
		if (!selectedResource) {
			toast.error('Select a resource');
			return;
		}
		if (!selectedScope) {
			toast.error('Select a scope');
			return;
		}
		setEvaluating(true);
		setResult(null);
		const { data, error } = await client
			.query<CheckPermissionResponse>(CheckPermissionQuery, {
				params: {
					resource: selectedResource,
					scope: selectedScope,
				},
			})
			.toPromise();
		if (error) {
			toast.error(
				getGraphQLErrorMessage(error, 'Failed to evaluate permission'),
			);
		} else if (data?.check_permission) {
			setResult(data.check_permission);
		}
		setEvaluating(false);
	};

	if (loading) {
		return (
			<div className="space-y-3">
				{[1, 2, 3].map((i) => (
					<Skeleton key={i} className="h-10 w-full" />
				))}
			</div>
		);
	}

	return (
		<div>
			<div className="mb-4">
				<h2 className="text-lg font-semibold text-gray-900">
					Evaluate Permission
				</h2>
				<p className="text-sm text-gray-500">
					Test whether a permission check would be allowed or denied.
				</p>
			</div>

			<div className="rounded-md border border-blue-200 bg-blue-50 p-3 mb-6 flex items-start gap-2">
				<Info className="h-4 w-4 text-blue-500 mt-0.5 shrink-0" />
				<p className="text-sm text-blue-700">
					This evaluates permissions for the currently logged-in admin session.
					The <code className="bg-blue-100 px-1 rounded">check_permission</code>{' '}
					query uses the authenticated user context to determine access.
				</p>
			</div>

			<div className="max-w-md space-y-4">
				{/* Resource */}
				<div>
					<Label htmlFor="eval-resource">Resource</Label>
					<Select
						id="eval-resource"
						value={selectedResource}
						onChange={(e) => setSelectedResource(e.target.value)}
						className="mt-1"
					>
						<option value="">Select a resource...</option>
						{resources.map((r) => (
							<option key={r.id} value={r.name}>
								{r.name}
							</option>
						))}
					</Select>
				</div>

				{/* Scope */}
				<div>
					<Label htmlFor="eval-scope">Scope</Label>
					<Select
						id="eval-scope"
						value={selectedScope}
						onChange={(e) => setSelectedScope(e.target.value)}
						className="mt-1"
					>
						<option value="">Select a scope...</option>
						{scopes.map((s) => (
							<option key={s.id} value={s.name}>
								{s.name}
							</option>
						))}
					</Select>
				</div>

				{/* Evaluate button */}
				<Button
					onClick={handleEvaluate}
					disabled={evaluating || !selectedResource || !selectedScope}
					className="w-full"
				>
					<Play className="h-4 w-4 mr-2" />
					{evaluating ? 'Evaluating...' : 'Evaluate'}
				</Button>
			</div>

			{/* Result panel */}
			{result !== null && (
				<div
					className={`mt-6 rounded-md border p-4 ${
						result.allowed
							? 'border-green-200 bg-green-50'
							: 'border-red-200 bg-red-50'
					}`}
				>
					<div className="flex items-center gap-3 mb-2">
						{result.allowed ? (
							<ShieldCheck className="h-8 w-8 text-green-600" />
						) : (
							<ShieldX className="h-8 w-8 text-red-600" />
						)}
						<div>
							<h3
								className={`text-lg font-semibold ${result.allowed ? 'text-green-800' : 'text-red-800'}`}
							>
								{result.allowed ? 'Allowed' : 'Denied'}
							</h3>
							<p
								className={`text-sm ${result.allowed ? 'text-green-600' : 'text-red-600'}`}
							>
								{result.allowed
									? 'The permission check passed.'
									: 'The permission check failed.'}
							</p>
						</div>
					</div>
					<div className="mt-3 space-y-2 text-sm">
						<div className="flex items-center gap-2">
							<span className="font-medium text-gray-700">Resource:</span>
							<Badge>{selectedResource}</Badge>
						</div>
						<div className="flex items-center gap-2">
							<span className="font-medium text-gray-700">Scope:</span>
							<Badge variant="secondary">{selectedScope}</Badge>
						</div>
						{result.matched_policy && (
							<div className="flex items-center gap-2">
								<span className="font-medium text-gray-700">
									Matched Policy:
								</span>
								<Badge variant="outline">{result.matched_policy}</Badge>
							</div>
						)}
					</div>
				</div>
			)}
		</div>
	);
}
