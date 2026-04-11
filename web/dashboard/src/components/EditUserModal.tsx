import React, { useState, useEffect } from 'react';
import { useClient } from 'urql';
import { Save, Plus } from 'lucide-react';
import { toast } from 'sonner';
import InputField from './InputField';
import {
	DateInputType,
	MultiSelectInputType,
	SelectInputType,
	TextInputType,
} from '../constants';
import { getObjectDiff, getGraphQLErrorMessage } from '../utils';
import { UpdateUser } from '../graphql/mutation';
import { Button } from './ui/button';
import { Input } from './ui/input';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';

const GenderTypes: Record<string, string | null> = {
	Undisclosed: null,
	Male: 'Male',
	Female: 'Female',
};

interface UserData {
	id: string;
	email: string;
	given_name: string;
	family_name: string;
	middle_name: string;
	nickname: string;
	gender: string;
	birthdate: string;
	phone_number: string;
	picture: string;
	roles: string[];
}

interface EditUserModalProps {
	user: UserData;
	updateUserList: () => void;
}

const EditUserModal = ({ user, updateUserList }: EditUserModalProps) => {
	const client = useClient();
	const [newRole, setNewRole] = useState('');
	const [open, setOpen] = useState(false);
	const [userData, setUserData] = useState<UserData>({
		id: '',
		email: '',
		given_name: '',
		family_name: '',
		middle_name: '',
		nickname: '',
		gender: '',
		birthdate: '',
		phone_number: '',
		picture: '',
		roles: [],
	});

	const availableRoles = Array.from(
		new Set([...(userData.roles || []), ...(user.roles || [])]),
	);

	useEffect(() => {
		setUserData(user);
	}, [user]);

	const saveHandler = async () => {
		const diff = getObjectDiff(
			user as unknown as Record<string, unknown>,
			userData as unknown as Record<string, unknown>,
		);
		const updatedUserData = diff.reduce(
			(acc: Record<string, unknown>, property: string) => ({
				...acc,
				[property]: userData[property as keyof UserData],
			}),
			{},
		);
		const res = await client
			.mutation(UpdateUser, {
				params: { ...updatedUserData, id: userData.id },
			})
			.toPromise();
		if (res.error) {
			toast.error(getGraphQLErrorMessage(res.error, 'User data update failed'));
		} else if (res.data?._update_user?.id) {
			toast.success('User data update successful');
		}
		setOpen(false);
		updateUserList();
	};

	const setUserDataTyped = (
		vars: Record<string, string | boolean | string[]>,
	) => {
		setUserData(vars as unknown as UserData);
	};

	return (
		<>
			<button
				className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm"
				onClick={() => setOpen(true)}
			>
				Edit User Details
			</button>
			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent className="overflow-y-auto">
					<SheetHeader>
						<SheetTitle>Edit User Details</SheetTitle>
						<SheetDescription>
							Update the user profile information below.
						</SheetDescription>
					</SheetHeader>
					<div className="mt-6 space-y-4">
						{[
							{ label: 'Given Name', type: TextInputType.GIVEN_NAME },
							{ label: 'Middle Name', type: TextInputType.MIDDLE_NAME },
							{ label: 'Family Name', type: TextInputType.FAMILY_NAME },
						].map(({ label, type }) => (
							<div key={type} className="flex items-center gap-4">
								<label className="w-28 text-sm text-gray-600 shrink-0">
									{label}:
								</label>
								<div className="flex-1">
									<InputField
										variables={
											userData as unknown as Record<
												string,
												string | boolean | string[]
											>
										}
										setVariables={setUserDataTyped}
										inputType={type}
									/>
								</div>
							</div>
						))}

						<div className="flex items-center gap-4">
							<label className="w-28 text-sm text-gray-600 shrink-0">
								Birth Date:
							</label>
							<div className="flex-1">
								<InputField
									variables={
										userData as unknown as Record<
											string,
											string | boolean | string[]
										>
									}
									setVariables={setUserDataTyped}
									inputType={DateInputType.BIRTHDATE}
								/>
							</div>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-28 text-sm text-gray-600 shrink-0">
								Nickname:
							</label>
							<div className="flex-1">
								<InputField
									variables={
										userData as unknown as Record<
											string,
											string | boolean | string[]
										>
									}
									setVariables={setUserDataTyped}
									inputType={TextInputType.NICKNAME}
								/>
							</div>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-28 text-sm text-gray-600 shrink-0">
								Gender:
							</label>
							<div className="flex-1">
								<InputField
									variables={
										userData as unknown as Record<
											string,
											string | boolean | string[]
										>
									}
									setVariables={setUserDataTyped}
									inputType={SelectInputType.GENDER}
									options={GenderTypes}
								/>
							</div>
						</div>

						{[
							{ label: 'Phone', type: TextInputType.PHONE_NUMBER },
							{ label: 'Picture', type: TextInputType.PICTURE },
						].map(({ label, type }) => (
							<div key={type} className="flex items-center gap-4">
								<label className="w-28 text-sm text-gray-600 shrink-0">
									{label}:
								</label>
								<div className="flex-1">
									<InputField
										variables={
											userData as unknown as Record<
												string,
												string | boolean | string[]
											>
										}
										setVariables={setUserDataTyped}
										inputType={type}
									/>
								</div>
							</div>
						))}

						<div className="flex items-start gap-4">
							<label className="w-28 text-sm text-gray-600 shrink-0 pt-2">
								Roles:
							</label>
							<div className="flex-1 space-y-2">
								<InputField
									variables={
										userData as unknown as Record<
											string,
											string | boolean | string[]
										>
									}
									setVariables={setUserDataTyped}
									availableRoles={availableRoles}
									inputType={MultiSelectInputType.USER_ROLES}
								/>
								<div className="flex gap-2">
									<Input
										className="h-8 text-sm"
										placeholder="Add role"
										value={newRole}
										onChange={(e) => setNewRole(e.target.value)}
										onKeyDown={(e) => {
											if (e.key === 'Enter' && newRole.trim()) {
												setUserData({
													...userData,
													roles: [...(userData.roles || []), newRole.trim()],
												});
												setNewRole('');
											}
										}}
									/>
									<Button
										size="sm"
										variant="outline"
										onClick={() => {
											if (newRole.trim()) {
												setUserData({
													...userData,
													roles: [...(userData.roles || []), newRole.trim()],
												});
												setNewRole('');
											}
										}}
									>
										<Plus className="mr-1 h-3 w-3" />
										Add
									</Button>
								</div>
							</div>
						</div>
					</div>
					<SheetFooter className="mt-6">
						<Button onClick={saveHandler}>
							<Save className="mr-2 h-4 w-4" />
							Save
						</Button>
					</SheetFooter>
				</SheetContent>
			</Sheet>
		</>
	);
};

export default EditUserModal;
