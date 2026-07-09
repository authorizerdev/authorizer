import React, { useState } from 'react';
import { useClient } from 'urql';
import { Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { DeleteOrganization } from '../graphql/mutation';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
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

interface DeleteOrganizationModalProps {
	organizationId: string;
	organizationName: string;
	fetchOrganizations: () => void;
}

const DeleteOrganizationModal = ({
	organizationId,
	organizationName,
	fetchOrganizations,
}: DeleteOrganizationModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);

	const deleteHandler = async () => {
		const res = await client
			.mutation(DeleteOrganization, { params: { id: organizationId } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to delete organization'),
				),
			);
			return;
		} else if (res.data?._delete_organization) {
			toast.success(
				capitalizeFirstLetter(res.data._delete_organization.message),
			);
		}
		setOpen(false);
		fetchOrganizations();
	};

	return (
		<Dialog open={open} onOpenChange={setOpen}>
			<DialogTrigger asChild>
				<button className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm">
					Delete
				</button>
			</DialogTrigger>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Delete Organization</DialogTitle>
					<DialogDescription>Are you sure?</DialogDescription>
				</DialogHeader>
				<div className="rounded-md border border-red-300 bg-red-50 p-4">
					<p className="text-sm">
						Organization <strong>{organizationName}</strong> will be deleted
						permanently! Its memberships, SSO connections and SCIM endpoint will
						stop working.
					</p>
				</div>
				<DialogFooter>
					<Button variant="destructive" onClick={deleteHandler}>
						<Trash2 className="mr-2 h-4 w-4" />
						Delete
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
};

export default DeleteOrganizationModal;
