import React, { useEffect, useState } from 'react';
import {
	Plus,
	MinusCircle,
	Copy,
	ChevronDown,
	ChevronUp,
	CheckCircle,
	AlertCircle,
	AlertTriangle,
} from 'lucide-react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import {
	webhookEventNames,
	ArrayInputOperations,
	WebhookInputDataFields,
	WebhookInputHeaderFields,
	UpdateModalViews,
	webhookVerifiedStatus,
	webhookPayloadExample,
} from '../constants';
import {
	capitalizeFirstLetter,
	copyTextToClipboard,
	getGraphQLErrorMessage,
	validateURI,
} from '../utils';
import { AddWebhook, EditWebhook, TestEndpoint } from '../graphql/mutation';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select } from './ui/select';
import { Switch } from './ui/switch';
import { Separator } from './ui/separator';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';

interface HeadersData {
	[WebhookInputHeaderFields.KEY]: string;
	[WebhookInputHeaderFields.VALUE]: string;
}

interface HeadersValidatorData {
	[WebhookInputHeaderFields.KEY]: boolean;
	[WebhookInputHeaderFields.VALUE]: boolean;
}

interface SelectedWebhookData {
	[WebhookInputDataFields.ID]: string;
	[WebhookInputDataFields.EVENT_NAME]: string;
	[WebhookInputDataFields.EVENT_DESCRIPTION]?: string;
	[WebhookInputDataFields.ENDPOINT]: string;
	[WebhookInputDataFields.ENABLED]: boolean;
	[WebhookInputDataFields.HEADERS]?: Record<string, string>;
}

interface UpdateWebhookModalProps {
	view: UpdateModalViews;
	selectedWebhook?: SelectedWebhookData;
	fetchWebookData: () => void;
}

interface WebhookData {
	[WebhookInputDataFields.EVENT_NAME]: string;
	[WebhookInputDataFields.EVENT_DESCRIPTION]?: string;
	[WebhookInputDataFields.ENDPOINT]: string;
	[WebhookInputDataFields.ENABLED]: boolean;
	[WebhookInputDataFields.HEADERS]: HeadersData[];
}

interface ValidatorData {
	[WebhookInputDataFields.ENDPOINT]: boolean;
	[WebhookInputDataFields.HEADERS]: HeadersValidatorData[];
}

const initHeadersData: HeadersData = {
	[WebhookInputHeaderFields.KEY]: '',
	[WebhookInputHeaderFields.VALUE]: '',
};

const initHeadersValidatorData: HeadersValidatorData = {
	[WebhookInputHeaderFields.KEY]: true,
	[WebhookInputHeaderFields.VALUE]: true,
};

const initWebhookData: WebhookData = {
	[WebhookInputDataFields.EVENT_NAME]: webhookEventNames['User login'],
	[WebhookInputDataFields.EVENT_DESCRIPTION]: '',
	[WebhookInputDataFields.ENDPOINT]: '',
	[WebhookInputDataFields.ENABLED]: true,
	[WebhookInputDataFields.HEADERS]: [{ ...initHeadersData }],
};

const initWebhookValidatorData: ValidatorData = {
	[WebhookInputDataFields.ENDPOINT]: true,
	[WebhookInputDataFields.HEADERS]: [{ ...initHeadersValidatorData }],
};

const UpdateWebhookModal = ({
	view,
	selectedWebhook,
	fetchWebookData,
}: UpdateWebhookModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState<boolean>(false);
	const [verifyingEndpoint, setVerifyingEndpoint] = useState<boolean>(false);
	const [isShowingPayload, setIsShowingPayload] = useState<boolean>(false);
	const [webhook, setWebhook] = useState<WebhookData>({
		...initWebhookData,
	});
	const [validator, setValidator] = useState<ValidatorData>({
		...initWebhookValidatorData,
	});
	const [verifiedStatus, setVerifiedStatus] = useState<webhookVerifiedStatus>(
		webhookVerifiedStatus.PENDING,
	);

	const inputChangehandler = (
		inputType: string,
		value: string | boolean,
		headerInputType: string = WebhookInputHeaderFields.KEY,
		headerIndex: number = 0,
	) => {
		if (
			verifiedStatus !== webhookVerifiedStatus.PENDING &&
			inputType !== WebhookInputDataFields.ENABLED
		) {
			setVerifiedStatus(webhookVerifiedStatus.PENDING);
		}
		switch (inputType) {
			case WebhookInputDataFields.EVENT_NAME:
			case WebhookInputDataFields.EVENT_DESCRIPTION:
				setWebhook({ ...webhook, [inputType]: value as string });
				break;
			case WebhookInputDataFields.ENDPOINT:
				setWebhook({ ...webhook, [inputType]: value as string });
				setValidator({
					...validator,
					[WebhookInputDataFields.ENDPOINT]: validateURI(value as string),
				});
				break;
			case WebhookInputDataFields.ENABLED:
				setWebhook({ ...webhook, [inputType]: value as boolean });
				break;
			case WebhookInputDataFields.HEADERS: {
				const updatedHeaders = [
					...webhook[WebhookInputDataFields.HEADERS],
				];
				const updatedHeadersValidatorData = [
					...validator[WebhookInputDataFields.HEADERS],
				];
				const otherHeaderInputType =
					headerInputType === WebhookInputHeaderFields.KEY
						? WebhookInputHeaderFields.VALUE
						: WebhookInputHeaderFields.KEY;
				updatedHeaders[headerIndex][
					headerInputType as keyof HeadersData
				] = value as string;
				const strValue = value as string;
				updatedHeadersValidatorData[headerIndex][
					headerInputType as keyof HeadersValidatorData
				] =
					strValue.length > 0
						? updatedHeaders[headerIndex][otherHeaderInputType]
								.length > 0
						: updatedHeaders[headerIndex][otherHeaderInputType]
								.length === 0;
				updatedHeadersValidatorData[headerIndex][
					otherHeaderInputType as keyof HeadersValidatorData
				] =
					strValue.length > 0
						? updatedHeaders[headerIndex][otherHeaderInputType]
								.length > 0
						: updatedHeaders[headerIndex][otherHeaderInputType]
								.length === 0;
				setWebhook({ ...webhook, [inputType]: updatedHeaders });
				setValidator({
					...validator,
					[inputType]: updatedHeadersValidatorData,
				});
				break;
			}
			default:
				break;
		}
	};

	const updateHeaders = (operation: string, index: number = 0) => {
		if (verifiedStatus !== webhookVerifiedStatus.PENDING) {
			setVerifiedStatus(webhookVerifiedStatus.PENDING);
		}
		switch (operation) {
			case ArrayInputOperations.APPEND:
				setWebhook({
					...webhook,
					[WebhookInputDataFields.HEADERS]: [
						...(webhook[WebhookInputDataFields.HEADERS] || []),
						{ ...initHeadersData },
					],
				});
				setValidator({
					...validator,
					[WebhookInputDataFields.HEADERS]: [
						...(validator[WebhookInputDataFields.HEADERS] || []),
						{ ...initHeadersValidatorData },
					],
				});
				break;
			case ArrayInputOperations.REMOVE:
				if (webhook[WebhookInputDataFields.HEADERS]?.length) {
					const updatedHeaders = [
						...webhook[WebhookInputDataFields.HEADERS],
					];
					updatedHeaders.splice(index, 1);
					setWebhook({
						...webhook,
						[WebhookInputDataFields.HEADERS]: updatedHeaders,
					});
				}
				if (validator[WebhookInputDataFields.HEADERS]?.length) {
					const updatedHeadersData = [
						...validator[WebhookInputDataFields.HEADERS],
					];
					updatedHeadersData.splice(index, 1);
					setValidator({
						...validator,
						[WebhookInputDataFields.HEADERS]: updatedHeadersData,
					});
				}
				break;
			default:
				break;
		}
	};

	const validateData = () => {
		return (
			!loading &&
			!verifyingEndpoint &&
			webhook[WebhookInputDataFields.EVENT_NAME].length > 0 &&
			webhook[WebhookInputDataFields.ENDPOINT].length > 0 &&
			validator[WebhookInputDataFields.ENDPOINT] &&
			!validator[WebhookInputDataFields.HEADERS].some(
				(headerData: HeadersValidatorData) =>
					!headerData.key || !headerData.value,
			)
		);
	};

	const getParams = () => {
		const params: Record<string, unknown> = {
			[WebhookInputDataFields.EVENT_NAME]:
				webhook[WebhookInputDataFields.EVENT_NAME],
			[WebhookInputDataFields.EVENT_DESCRIPTION]:
				webhook[WebhookInputDataFields.EVENT_DESCRIPTION],
			[WebhookInputDataFields.ENDPOINT]:
				webhook[WebhookInputDataFields.ENDPOINT],
			[WebhookInputDataFields.ENABLED]:
				webhook[WebhookInputDataFields.ENABLED],
			[WebhookInputDataFields.HEADERS]: {},
		};
		if (webhook[WebhookInputDataFields.HEADERS].length) {
			const headers = webhook[WebhookInputDataFields.HEADERS].reduce(
				(acc: Record<string, string>, data) => {
					return data.key ? { ...acc, [data.key]: data.value } : acc;
				},
				{},
			);
			if (Object.keys(headers).length) {
				params[WebhookInputDataFields.HEADERS] = headers;
			}
		}
		return params;
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		const params = getParams();
		let res: { error?: unknown; data?: Record<string, { message?: string }> };
		if (
			view === UpdateModalViews.Edit &&
			selectedWebhook?.[WebhookInputDataFields.ID]
		) {
			res = await client
				.mutation(EditWebhook, {
					params: {
						...params,
						id: selectedWebhook[WebhookInputDataFields.ID],
					},
				})
				.toPromise();
		} else {
			res = await client.mutation(AddWebhook, { params }).toPromise();
		}
		setLoading(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(
						res.error,
						'Failed to update webhook',
					),
				),
			);
		} else if (res.data?._add_webhook || res.data?._update_webhook) {
			toast.success(
				capitalizeFirstLetter(
					res.data?._add_webhook?.message ||
						res.data?._update_webhook?.message ||
						'Webhook saved',
				),
			);
			setWebhook({
				...initWebhookData,
				[WebhookInputDataFields.HEADERS]: [{ ...initHeadersData }],
			});
			setValidator({ ...initWebhookValidatorData });
			fetchWebookData();
		}
		if (view === UpdateModalViews.ADD) setOpen(false);
	};

	useEffect(() => {
		if (
			open &&
			view === UpdateModalViews.Edit &&
			selectedWebhook &&
			Object.keys(selectedWebhook).length
		) {
			const { headers, ...rest } = selectedWebhook;
			const headerItems = Object.entries(headers || {});
			if (headerItems.length) {
				const formattedHeadersData = headerItems.map(
					(headerData) => ({
						[WebhookInputHeaderFields.KEY]: headerData[0],
						[WebhookInputHeaderFields.VALUE]: headerData[1],
					}),
				);
				setWebhook({
					...rest,
					[WebhookInputDataFields.HEADERS]: formattedHeadersData,
				});
				setValidator({
					...validator,
					[WebhookInputDataFields.HEADERS]: new Array(
						formattedHeadersData.length,
					)
						.fill({})
						.map(() => ({ ...initHeadersValidatorData })),
				});
			} else {
				setWebhook({
					...rest,
					[WebhookInputDataFields.HEADERS]: [
						{ ...initHeadersData },
					],
				});
			}
		}
	}, [open]);

	const verifyEndpoint = async () => {
		if (!validateData()) return;
		setVerifyingEndpoint(true);
		const { [WebhookInputDataFields.ENABLED]: _, ...params } = getParams();
		const res = await client
			.mutation(TestEndpoint, { params })
			.toPromise();
		if (
			res.data?._test_endpoint?.http_status >= 200 &&
			res.data?._test_endpoint?.http_status < 400
		) {
			setVerifiedStatus(webhookVerifiedStatus.VERIFIED);
		} else {
			setVerifiedStatus(webhookVerifiedStatus.NOT_VERIFIED);
		}
		setVerifyingEndpoint(false);
	};

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button size="sm" onClick={() => setOpen(true)}>
					<Plus className="mr-2 h-4 w-4" />
					Add Webhook
				</Button>
			) : (
				<button
					className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm"
					onClick={() => setOpen(true)}
				>
					Edit
				</button>
			)}
			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>
							{view === UpdateModalViews.ADD
								? 'Add New Webhook'
								: 'Edit Webhook'}
						</SheetTitle>
						<SheetDescription>
							Configure webhook endpoint and event settings.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						{/* Event Name */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Event Name
							</label>
							<Select
								value={
									webhook[
										WebhookInputDataFields.EVENT_NAME
									].split('-')[0]
								}
								onChange={(e) =>
									inputChangehandler(
										WebhookInputDataFields.EVENT_NAME,
										e.currentTarget.value,
									)
								}
							>
								{Object.entries(webhookEventNames).map(
									([key, value]) => (
										<option value={value} key={key}>
											{key}
										</option>
									),
								)}
							</Select>
						</div>

						{/* Event Description */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Description
							</label>
							<Input
								placeholder="User event"
								value={
									webhook[
										WebhookInputDataFields.EVENT_DESCRIPTION
									] || ''
								}
								onChange={(e) =>
									inputChangehandler(
										WebhookInputDataFields.EVENT_DESCRIPTION,
										e.currentTarget.value,
									)
								}
							/>
						</div>

						{/* Endpoint */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Endpoint
							</label>
							<Input
								placeholder="https://domain.com/webhook"
								value={
									webhook[WebhookInputDataFields.ENDPOINT]
								}
								isInvalid={
									!validator[WebhookInputDataFields.ENDPOINT]
								}
								onChange={(e) =>
									inputChangehandler(
										WebhookInputDataFields.ENDPOINT,
										e.currentTarget.value,
									)
								}
							/>
						</div>

						{/* Enabled */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Enabled
							</label>
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium">
									Off
								</span>
								<Switch
									checked={
										webhook[
											WebhookInputDataFields.ENABLED
										]
									}
									onCheckedChange={(checked: boolean) =>
										inputChangehandler(
											WebhookInputDataFields.ENABLED,
											checked,
										)
									}
								/>
								<span className="text-sm font-medium">On</span>
							</div>
						</div>

						{/* Headers */}
						<div className="flex items-center justify-between">
							<span className="text-sm font-medium">
								Headers
							</span>
							<Button
								variant="ghost"
								size="sm"
								onClick={() =>
									updateHeaders(ArrayInputOperations.APPEND)
								}
							>
								<Plus className="mr-1 h-3 w-3" />
								Add more Headers
							</Button>
						</div>
						<div className="max-h-48 space-y-2 overflow-y-auto">
							{webhook[WebhookInputDataFields.HEADERS]?.map(
								(headerData, index) => (
									<div
										key={`header-data-${index}`}
										className="flex items-center gap-2"
									>
										<Input
											placeholder="key"
											value={
												headerData[
													WebhookInputHeaderFields.KEY
												]
											}
											isInvalid={
												!validator[
													WebhookInputDataFields
														.HEADERS
												][index]?.[
													WebhookInputHeaderFields.KEY
												]
											}
											onChange={(e) =>
												inputChangehandler(
													WebhookInputDataFields.HEADERS,
													e.target.value,
													WebhookInputHeaderFields.KEY,
													index,
												)
											}
											className="w-1/3"
										/>
										<span className="font-bold">:</span>
										<Input
											placeholder="value"
											value={
												headerData[
													WebhookInputHeaderFields
														.VALUE
												]
											}
											isInvalid={
												!validator[
													WebhookInputDataFields
														.HEADERS
												][index]?.[
													WebhookInputHeaderFields
														.VALUE
												]
											}
											onChange={(e) =>
												inputChangehandler(
													WebhookInputDataFields.HEADERS,
													e.target.value,
													WebhookInputHeaderFields.VALUE,
													index,
												)
											}
											className="flex-1"
										/>
										<Button
											variant="ghost"
											size="icon"
											onClick={() =>
												updateHeaders(
													ArrayInputOperations.REMOVE,
													index,
												)
											}
										>
											<MinusCircle className="h-4 w-4" />
										</Button>
									</div>
								),
							)}
						</div>

						<Separator />

						{/* Example Payload */}
						<button
							type="button"
							className="flex w-full items-center justify-between rounded-md bg-blue-50 px-4 py-2 text-sm text-blue-800"
							onClick={() =>
								setIsShowingPayload(!isShowingPayload)
							}
						>
							<span>Checkout the example payload</span>
							{isShowingPayload ? (
								<ChevronUp className="h-4 w-4" />
							) : (
								<ChevronDown className="h-4 w-4" />
							)}
						</button>
						{isShowingPayload && (
							<div className="relative rounded-md bg-gray-100 p-3">
								<pre className="overflow-auto text-xs">
									{webhookPayloadExample}
								</pre>
								<button
									type="button"
									className="absolute right-3 top-3 text-gray-400 hover:text-gray-600"
									onClick={() =>
										copyTextToClipboard(
											webhookPayloadExample,
										)
									}
								>
									<Copy className="h-4 w-4" />
								</button>
							</div>
						)}
					</div>

					<SheetFooter className="mt-6">
						<Button
							variant="outline"
							onClick={verifyEndpoint}
							isLoading={verifyingEndpoint}
							disabled={!validateData()}
							className={
								verifiedStatus ===
								webhookVerifiedStatus.VERIFIED
									? 'border-green-500 text-green-700'
									: verifiedStatus ===
									  webhookVerifiedStatus.NOT_VERIFIED
									? 'border-red-500 text-red-700'
									: 'border-yellow-500 text-yellow-700'
							}
						>
							{verifiedStatus ===
							webhookVerifiedStatus.VERIFIED ? (
								<CheckCircle className="mr-2 h-4 w-4" />
							) : verifiedStatus ===
							  webhookVerifiedStatus.PENDING ? (
								<AlertCircle className="mr-2 h-4 w-4" />
							) : (
								<AlertTriangle className="mr-2 h-4 w-4" />
							)}
							{verifiedStatus ===
							webhookVerifiedStatus.VERIFIED
								? 'Endpoint Verified'
								: verifiedStatus ===
								  webhookVerifiedStatus.PENDING
								? 'Test Endpoint'
								: 'Endpoint Not Verified'}
						</Button>
						<Button
							onClick={saveData}
							disabled={!validateData()}
						>
							Save
						</Button>
					</SheetFooter>
				</SheetContent>
			</Sheet>
		</>
	);
};

export default UpdateWebhookModal;
