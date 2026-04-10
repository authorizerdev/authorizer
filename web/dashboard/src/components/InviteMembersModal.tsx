import React, { useState, useCallback, useEffect } from 'react';
import { useClient } from 'urql';
import { UserPlus, MinusCircle, Plus, Upload } from 'lucide-react';
import { useDropzone } from 'react-dropzone';
import { toast } from 'sonner';
import { validateEmail, validateURI, getGraphQLErrorMessage } from '../utils';
import { InviteMembers } from '../graphql/mutation';
import { ArrayInputOperations } from '../constants';
import parseCSV from '../utils/parseCSV';
import { Button } from './ui/button';
import { Input } from './ui/input';
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from './ui/dialog';

interface StateData {
	value: string;
	isInvalid: boolean;
}

interface RequestParams {
	emails: string[];
	redirect_uri?: string;
}

const initData: StateData = {
	value: '',
	isInvalid: false,
};

interface InviteMembersModalProps {
	updateUserList: () => void;
}

const InviteMembersModal = ({ updateUserList }: InviteMembersModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [tabIndex, setTabIndex] = useState<number>(0);
	const [redirectURI, setRedirectURI] = useState<StateData>({
		...initData,
	});
	const [emails, setEmails] = useState<StateData[]>([{ ...initData }]);
	const [disableSendButton, setDisableSendButton] = useState<boolean>(false);
	const [loading, setLoading] = useState<boolean>(false);

	useEffect(() => {
		if (redirectURI.isInvalid) {
			setDisableSendButton(true);
		} else if (emails.some((emailData) => emailData.isInvalid)) {
			setDisableSendButton(true);
		} else {
			setDisableSendButton(false);
		}
	}, [redirectURI, emails]);

	useEffect(() => {
		return () => {
			setRedirectURI({ ...initData });
			setEmails([{ ...initData }]);
		};
	}, []);

	const sendInviteHandler = async () => {
		setLoading(true);
		try {
			const emailList = emails
				.filter((emailData) => !emailData.isInvalid)
				.map((emailData) => emailData.value);
			const params: RequestParams = {
				emails: emailList,
			};
			if (redirectURI.value !== '' && !redirectURI.isInvalid) {
				params.redirect_uri = redirectURI.value;
			}
			if (emailList.length > 0) {
				const res = await client
					.mutation(InviteMembers, { params })
					.toPromise();
				if (res.error) {
					throw new Error(
						getGraphQLErrorMessage(res.error, 'Failed to send invites'),
					);
				}
				toast.success('Invites sent successfully!');
				setLoading(false);
				updateUserList();
			} else {
				throw new Error('Please add emails');
			}
		} catch (error: unknown) {
			const message =
				error instanceof Error
					? error.message
					: 'Error occurred, try again!';
			toast.error(message);
			setLoading(false);
		}
		closeModalHandler();
	};

	const updateEmailListHandler = (operation: string, index: number = 0) => {
		switch (operation) {
			case ArrayInputOperations.APPEND:
				setEmails([...emails, { ...initData }]);
				break;
			case ArrayInputOperations.REMOVE: {
				const updatedEmailList = [...emails];
				updatedEmailList.splice(index, 1);
				setEmails(updatedEmailList);
				break;
			}
			default:
				break;
		}
	};

	const inputChangeHandler = (value: string, index: number) => {
		const updatedEmailList = [...emails];
		updatedEmailList[index].value = value;
		updatedEmailList[index].isInvalid = !validateEmail(value);
		setEmails(updatedEmailList);
	};

	const onDrop = useCallback(async (acceptedFiles: File[]) => {
		const result = await parseCSV(acceptedFiles[0], ',');
		setEmails(result);
		setTabIndex(0);
	}, []);

	const setRedirectURIHandler = (value: string) => {
		setRedirectURI({
			value,
			isInvalid: !validateURI(value),
		});
	};

	const { getRootProps, getInputProps, isDragActive } = useDropzone({
		onDrop,
		accept: { 'text/csv': ['.csv'] },
	});

	const closeModalHandler = () => {
		setRedirectURI({ value: '', isInvalid: false });
		setEmails([{ value: '', isInvalid: false }]);
		setOpen(false);
	};

	return (
		<Dialog open={open} onOpenChange={setOpen}>
			<DialogTrigger asChild>
				<Button size="sm">
					<UserPlus className="mr-2 h-4 w-4" />
					Invite Members
				</Button>
			</DialogTrigger>
			<DialogContent className="max-w-xl">
				<DialogHeader>
					<DialogTitle>Invite Members</DialogTitle>
				</DialogHeader>

				{/* Tabs */}
				<div className="flex border-b border-gray-200">
					<button
						className={`flex-1 py-2 text-sm font-medium border-b-2 transition-colors ${
							tabIndex === 0
								? 'border-blue-500 text-blue-600'
								: 'border-transparent text-gray-500 hover:text-gray-700'
						}`}
						onClick={() => setTabIndex(0)}
					>
						Enter emails
					</button>
					<button
						className={`flex-1 py-2 text-sm font-medium border-b-2 transition-colors ${
							tabIndex === 1
								? 'border-blue-500 text-blue-600'
								: 'border-transparent text-gray-500 hover:text-gray-700'
						}`}
						onClick={() => setTabIndex(1)}
					>
						Upload CSV
					</button>
				</div>

				<div className="border border-t-0 border-gray-200 rounded-b-md p-4">
					{tabIndex === 0 ? (
						<div className="space-y-4">
							<div>
								<label className="block text-sm font-medium text-gray-700 mb-1">
									Redirect URI
								</label>
								<Input
									type="text"
									placeholder="https://domain.com/sign-up"
									value={redirectURI.value}
									isInvalid={redirectURI.isInvalid}
									onChange={(e) =>
										setRedirectURIHandler(e.currentTarget.value)
									}
								/>
							</div>

							<div className="flex items-center justify-between">
								<label className="text-sm font-medium text-gray-700">
									Emails
								</label>
								<Button
									variant="ghost"
									size="sm"
									onClick={() =>
										updateEmailListHandler(
											ArrayInputOperations.APPEND,
										)
									}
								>
									<Plus className="mr-1 h-3 w-3" />
									Add more emails
								</Button>
							</div>

							<div className="max-h-60 space-y-2 overflow-y-auto">
								{emails.map((emailData, index) => (
									<div
										key={`email-data-${index}`}
										className="flex items-center gap-2"
									>
										<Input
											type="text"
											placeholder="name@domain.com"
											value={emailData.value}
											isInvalid={emailData.isInvalid}
											onChange={(e) =>
												inputChangeHandler(
													e.currentTarget.value,
													index,
												)
											}
										/>
										<Button
											variant="ghost"
											size="icon"
											onClick={() =>
												updateEmailListHandler(
													ArrayInputOperations.REMOVE,
													index,
												)
											}
										>
											<MinusCircle className="h-4 w-4" />
										</Button>
									</div>
								))}
							</div>
						</div>
					) : (
						<div
							className="flex flex-col items-center justify-center rounded-md bg-gray-50 p-12 text-center cursor-pointer"
							{...getRootProps()}
						>
							<input {...getInputProps()} />
							{isDragActive ? (
								<p className="text-sm text-gray-600">
									Drop the files here...
								</p>
							) : (
								<>
									<Upload className="h-10 w-10 text-gray-400 mb-3" />
									<p className="text-sm text-gray-600">
										Drag and drop the csv file here, or
										click to select.
									</p>
									<p className="text-xs text-gray-500 mt-1">
										Download{' '}
										<a
											href="/dashboard/public/sample.csv"
											download="sample.csv"
											className="text-blue-600 hover:underline"
											onClick={(e) => e.stopPropagation()}
										>
											sample.csv
										</a>{' '}
										and modify it.
									</p>
								</>
							)}
						</div>
					)}
				</div>

				<DialogFooter>
					<Button
						onClick={sendInviteHandler}
						disabled={disableSendButton || loading}
						isLoading={loading}
					>
						Send
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
};

export default InviteMembersModal;
