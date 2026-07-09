import React, { useState } from 'react';
import { Copy, Check, AlertTriangle } from 'lucide-react';
import { toast } from 'sonner';
import { copyTextToClipboard } from '../utils';
import { Button } from './ui/button';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from './ui/dialog';

interface ClientSecretDialogProps {
	// Plaintext secret returned once by the server (client secret, SCIM token).
	secret: string | null;
	onClose: () => void;
	// Label used in the dialog title, copy toast and aria-label.
	label?: string;
}

// One-time display of a secret. The server never returns it again.
const ClientSecretDialog = ({
	secret,
	onClose,
	label = 'Client Secret',
}: ClientSecretDialogProps) => {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		if (!secret) return;
		await copyTextToClipboard(secret);
		setCopied(true);
		toast.success(`${label} copied`);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<Dialog
			open={!!secret}
			onOpenChange={(isOpen) => {
				if (!isOpen) onClose();
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{label}</DialogTitle>
					<DialogDescription>
						Copy this secret and store it securely.
					</DialogDescription>
				</DialogHeader>
				<div className="rounded-md border border-yellow-300 bg-yellow-50 p-4">
					<p className="flex items-start gap-2 text-sm text-yellow-800">
						<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
						This secret is shown only once. You won&apos;t be able to see it
						again after closing this dialog.
					</p>
				</div>
				<div className="flex items-center gap-2 rounded-md bg-gray-100 p-3">
					<code className="flex-1 break-all font-mono text-sm">{secret}</code>
					<button
						type="button"
						onClick={handleCopy}
						className="text-gray-400 hover:text-gray-600"
						aria-label={`Copy ${label.toLowerCase()}`}
					>
						{copied ? (
							<Check className="h-4 w-4 text-green-500" />
						) : (
							<Copy className="h-4 w-4" />
						)}
					</button>
				</div>
				<DialogFooter>
					<Button onClick={onClose}>I have stored the secret</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
};

export default ClientSecretDialog;
