import React, { useState } from 'react';
import { useClient } from 'urql';
import { RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import { RotateClientSecret } from '../graphql/mutation';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import type { CreateClientResponse } from '../types';
import ClientSecretDialog from './ClientSecretDialog';
import { Button } from './ui/button';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from './ui/dialog';

interface RotateClientSecretModalProps {
	clientId: string;
	clientName: string;
}

const RotateClientSecretModal = ({
	clientId,
	clientName,
}: RotateClientSecretModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	// Plaintext secret returned once by _rotate_client_secret.
	const [rotatedSecret, setRotatedSecret] = useState<string | null>(null);

	const rotateHandler = async () => {
		setLoading(true);
		const res = await client
			.mutation<{
				_rotate_client_secret: CreateClientResponse;
			}>(RotateClientSecret, { params: { id: clientId } })
			.toPromise();
		setLoading(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to rotate client secret'),
				),
			);
			return;
		}
		toast.success('Client secret rotated');
		setOpen(false);
		setRotatedSecret(res.data?._rotate_client_secret?.client_secret || null);
	};

	return (
		<>
			<Dialog open={open} onOpenChange={setOpen}>
				<DialogTrigger asChild>
					<button className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm">
						Rotate Secret
					</button>
				</DialogTrigger>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Rotate Client Secret</DialogTitle>
						<DialogDescription>Are you sure?</DialogDescription>
					</DialogHeader>
					<div className="rounded-md border border-yellow-300 bg-yellow-50 p-4">
						<p className="text-sm">
							The current secret for <strong>{clientName}</strong> will stop
							working immediately. The new secret is shown only once.
						</p>
					</div>
					<DialogFooter>
						<Button onClick={rotateHandler} isLoading={loading}>
							<RefreshCw className="mr-2 h-4 w-4" />
							Rotate Secret
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
			<ClientSecretDialog
				secret={rotatedSecret}
				onClose={() => setRotatedSecret(null)}
			/>
		</>
	);
};

export default RotateClientSecretModal;
