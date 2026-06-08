import React, { useCallback, useEffect, useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { AlertCircle, Save } from 'lucide-react';
import { FgaGetModelQuery } from '../../graphql/queries';
import { FgaWriteModel } from '../../graphql/mutation';
import { Button } from '../../components/ui/button';
import { Textarea } from '../../components/ui/textarea';
import { Skeleton } from '../../components/ui/skeleton';
import { Badge } from '../../components/ui/badge';
import FgaNotEnabled from '../../components/FgaNotEnabled';
import { isFgaNotEnabledError } from '../../lib/utils';
import type {
	FgaGetModelResponse,
	FgaWriteModelResponse,
} from '../../types';

const Model = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(true);
	const [saving, setSaving] = useState<boolean>(false);
	const [fgaDisabled, setFgaDisabled] = useState<boolean>(false);
	const [dsl, setDsl] = useState<string>('');
	const [modelId, setModelId] = useState<string>('');
	const [validationError, setValidationError] = useState<string>('');

	const fetchModel = useCallback(async () => {
		setLoading(true);
		try {
			const res = await client
				.query<FgaGetModelResponse>(
					FgaGetModelQuery,
					{},
					{ requestPolicy: 'network-only' },
				)
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) {
					setFgaDisabled(true);
				} else {
					toast.error('Failed to load authorization model');
				}
				return;
			}

			if (res.data?._fga_get_model) {
				setDsl(res.data._fga_get_model.dsl || '');
				setModelId(res.data._fga_get_model.id || '');
			}
		} catch {
			toast.error('Failed to load authorization model');
		} finally {
			setLoading(false);
		}
	}, [client]);

	useEffect(() => {
		fetchModel();
	}, [fetchModel]);

	const handleSave = async () => {
		if (!dsl.trim()) {
			setValidationError('Model DSL cannot be empty.');
			return;
		}
		setValidationError('');
		setSaving(true);
		try {
			const res = await client
				.mutation<FgaWriteModelResponse>(FgaWriteModel, {
					params: { dsl },
				})
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) {
					setFgaDisabled(true);
				} else {
					setValidationError(res.error.message.replace('[GraphQL] ', ''));
					toast.error('Failed to save authorization model');
				}
				return;
			}

			if (res.data?._fga_write_model) {
				setModelId(res.data._fga_write_model.id);
				setDsl(res.data._fga_write_model.dsl || dsl);
				toast.success('Authorization model saved');
			}
		} catch {
			toast.error('Failed to save authorization model');
		} finally {
			setSaving(false);
		}
	};

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="my-4 flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">
						Authorization Model
					</h1>
					<p className="mt-1 text-sm text-gray-500">
						Define the OpenFGA authorization model in DSL form.
					</p>
				</div>
				{!fgaDisabled && (
					<Button onClick={handleSave} disabled={saving || loading}>
						<Save className="mr-2 h-4 w-4" />
						{saving ? 'Saving...' : 'Save Model'}
					</Button>
				)}
			</div>

			{loading ? (
				<div className="space-y-3">
					<Skeleton className="h-10 w-1/3" />
					<Skeleton className="h-64 w-full" />
				</div>
			) : fgaDisabled ? (
				<FgaNotEnabled />
			) : (
				<div className="space-y-4">
					{modelId && (
						<div className="flex items-center gap-2 text-sm text-gray-600">
							<span>Current model id:</span>
							<Badge variant="secondary">{modelId}</Badge>
						</div>
					)}

					<Textarea
						value={dsl}
						onChange={(e) => setDsl(e.target.value)}
						spellCheck={false}
						className="min-h-[420px] font-mono text-xs leading-relaxed"
						placeholder={
							'model\n  schema 1.1\n\ntype user\n\ntype document\n  relations\n    define viewer: [user]\n    define editor: [user]'
						}
					/>

					{validationError && (
						<div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
							<AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
							<span className="whitespace-pre-wrap break-words">
								{validationError}
							</span>
						</div>
					)}
				</div>
			)}
		</div>
	);
};

export default Model;
