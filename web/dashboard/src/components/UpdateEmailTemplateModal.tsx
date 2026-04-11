import React, { useEffect, useRef, useState } from 'react';
import { Plus, ChevronDown, ChevronUp } from 'lucide-react';
import { useClient } from 'urql';
import EmailEditor, { type EditorRef } from 'react-email-editor';
import { toast } from 'sonner';
import {
	UpdateModalViews,
	EmailTemplateInputDataFields,
	emailTemplateEventNames,
	emailTemplateVariables,
	EmailTemplateEditors,
} from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import { AddEmailTemplate, EditEmailTemplate } from '../graphql/mutation';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select } from './ui/select';
import { Textarea } from './ui/textarea';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from './ui/table';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';

interface SelectedEmailTemplateData {
	[EmailTemplateInputDataFields.ID]: string;
	[EmailTemplateInputDataFields.EVENT_NAME]: string;
	[EmailTemplateInputDataFields.SUBJECT]: string;
	[EmailTemplateInputDataFields.CREATED_AT]: number;
	[EmailTemplateInputDataFields.TEMPLATE]: string;
	[EmailTemplateInputDataFields.DESIGN]: string;
}

interface UpdateEmailTemplateProps {
	view: UpdateModalViews;
	selectedTemplate?: SelectedEmailTemplateData;
	fetchEmailTemplatesData: () => void;
}

interface TemplateVariableData {
	text: string;
	value: string;
	description: string;
}

interface EmailTemplateData {
	[EmailTemplateInputDataFields.EVENT_NAME]: string;
	[EmailTemplateInputDataFields.SUBJECT]: string;
	[EmailTemplateInputDataFields.TEMPLATE]: string;
	[EmailTemplateInputDataFields.DESIGN]: string;
}

interface ValidatorData {
	[EmailTemplateInputDataFields.SUBJECT]: boolean;
}

const initTemplateData: EmailTemplateData = {
	[EmailTemplateInputDataFields.EVENT_NAME]: emailTemplateEventNames.Signup,
	[EmailTemplateInputDataFields.SUBJECT]: '',
	[EmailTemplateInputDataFields.TEMPLATE]: '',
	[EmailTemplateInputDataFields.DESIGN]: '',
};

const initTemplateValidatorData: ValidatorData = {
	[EmailTemplateInputDataFields.SUBJECT]: true,
};

const UpdateEmailTemplate = ({
	view,
	selectedTemplate,
	fetchEmailTemplatesData,
}: UpdateEmailTemplateProps) => {
	const client = useClient();
	const emailEditorRef = useRef<EditorRef>(null);
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState<boolean>(false);
	const [editor, setEditor] = useState<string>(
		EmailTemplateEditors.PLAIN_HTML_EDITOR,
	);
	const [templateVariables, setTemplateVariables] = useState<
		TemplateVariableData[]
	>([]);
	const [templateData, setTemplateData] = useState<EmailTemplateData>({
		...initTemplateData,
	});
	const [validator, setValidator] = useState<ValidatorData>({
		...initTemplateValidatorData,
	});
	const [isDynamicVariableInfoOpen, setIsDynamicVariableInfoOpen] =
		useState<boolean>(false);

	const onReady = () => {
		if (selectedTemplate) {
			const { design } = selectedTemplate;
			try {
				if (design) {
					const designData = JSON.parse(design);
					emailEditorRef.current?.editor?.loadDesign(designData);
				}
			} catch (error) {
				console.error(error);
				setOpen(false);
			}
		}
	};

	const inputChangehandler = (inputType: string, value: string) => {
		if (inputType !== EmailTemplateInputDataFields.EVENT_NAME) {
			setValidator({
				...validator,
				[inputType]: value?.trim().length > 0,
			});
		}
		setTemplateData({ ...templateData, [inputType]: value });
	};

	const validateData = () => {
		return (
			!loading &&
			templateData[EmailTemplateInputDataFields.EVENT_NAME].length > 0 &&
			templateData[EmailTemplateInputDataFields.SUBJECT].length > 0 &&
			validator[EmailTemplateInputDataFields.SUBJECT]
		);
	};

	const updateTemplate = async (params: EmailTemplateData) => {
		let res: { error?: unknown; data?: Record<string, { message?: string }> };
		if (
			view === UpdateModalViews.Edit &&
			selectedTemplate?.[EmailTemplateInputDataFields.ID]
		) {
			res = await client
				.mutation(EditEmailTemplate, {
					params: {
						...params,
						id: selectedTemplate[EmailTemplateInputDataFields.ID],
					},
				})
				.toPromise();
		} else {
			res = await client.mutation(AddEmailTemplate, { params }).toPromise();
		}
		setLoading(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to save email template'),
				),
			);
		} else if (
			res.data?._add_email_template ||
			res.data?._update_email_template
		) {
			toast.success(
				capitalizeFirstLetter(
					res.data?._add_email_template?.message ||
						res.data?._update_email_template?.message ||
						'Email template saved',
				),
			);
			setTemplateData({ ...initTemplateData });
			setValidator({ ...initTemplateValidatorData });
			fetchEmailTemplatesData();
		}
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let params: EmailTemplateData = {
			[EmailTemplateInputDataFields.EVENT_NAME]:
				templateData[EmailTemplateInputDataFields.EVENT_NAME],
			[EmailTemplateInputDataFields.SUBJECT]:
				templateData[EmailTemplateInputDataFields.SUBJECT],
			[EmailTemplateInputDataFields.TEMPLATE]:
				templateData[EmailTemplateInputDataFields.TEMPLATE],
			[EmailTemplateInputDataFields.DESIGN]: '',
		};
		if (editor === EmailTemplateEditors.UNLAYER_EDITOR) {
			emailEditorRef.current?.editor?.exportHtml(async (data) => {
				const { design, html } = data;
				if (!html || !design) {
					setLoading(false);
					return;
				}
				params = {
					...params,
					[EmailTemplateInputDataFields.TEMPLATE]: html.trim(),
					[EmailTemplateInputDataFields.DESIGN]: JSON.stringify(design),
				};
				await updateTemplate(params);
			});
		} else {
			await updateTemplate(params);
		}
		if (view === UpdateModalViews.ADD) setOpen(false);
	};

	const resetData = () => {
		if (selectedTemplate) {
			setTemplateData(selectedTemplate);
		} else {
			setTemplateData({ ...initTemplateData });
		}
	};

	useEffect(() => {
		if (
			open &&
			view === UpdateModalViews.Edit &&
			selectedTemplate &&
			Object.keys(selectedTemplate).length
		) {
			const { id, created_at, ...rest } = selectedTemplate;
			setTemplateData(rest);
		}
	}, [open]);

	useEffect(() => {
		const updatedTemplateVariables = Object.entries(
			emailTemplateVariables,
		).reduce((acc: TemplateVariableData[], [key, val]) => {
			if (
				(templateData[EmailTemplateInputDataFields.EVENT_NAME] !==
					emailTemplateEventNames['Verify Otp'] &&
					val === emailTemplateVariables.otp) ||
				(templateData[EmailTemplateInputDataFields.EVENT_NAME] ===
					emailTemplateEventNames['Verify Otp'] &&
					val === emailTemplateVariables.verification_url)
			) {
				return acc;
			}
			return [
				...acc,
				{
					text: key,
					value: val.value,
					description: val.description,
				},
			];
		}, []);
		setTemplateVariables(updatedTemplateVariables);
	}, [templateData[EmailTemplateInputDataFields.EVENT_NAME]]);

	useEffect(() => {
		if (open && selectedTemplate) {
			const { design } = selectedTemplate;
			if (design) {
				setEditor(EmailTemplateEditors.UNLAYER_EDITOR);
			} else {
				setEditor(EmailTemplateEditors.PLAIN_HTML_EDITOR);
			}
		}
	}, [open, selectedTemplate]);

	useEffect(() => {
		if (selectedTemplate?.design) {
			if (editor === EmailTemplateEditors.UNLAYER_EDITOR) {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: selectedTemplate.template,
					[EmailTemplateInputDataFields.DESIGN]: selectedTemplate.design,
				});
			} else {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: '',
					[EmailTemplateInputDataFields.DESIGN]: '',
				});
			}
		} else if (selectedTemplate?.template) {
			if (editor === EmailTemplateEditors.UNLAYER_EDITOR) {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: '',
					[EmailTemplateInputDataFields.DESIGN]: '',
				});
			} else {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: selectedTemplate?.template,
					[EmailTemplateInputDataFields.DESIGN]: '',
				});
			}
		}
	}, [editor]);

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button size="sm" onClick={() => setOpen(true)}>
					<Plus className="mr-2 h-4 w-4" />
					Add Template
				</Button>
			) : (
				<button
					className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm"
					onClick={() => setOpen(true)}
				>
					Edit
				</button>
			)}
			<Sheet
				open={open}
				onOpenChange={(isOpen) => {
					if (!isOpen) resetData();
					setOpen(isOpen);
				}}
			>
				<SheetContent className="overflow-y-auto sm:max-w-4xl">
					<SheetHeader>
						<SheetTitle>
							{view === UpdateModalViews.ADD
								? 'Add New Email Template'
								: 'Edit Email Template'}
						</SheetTitle>
						<SheetDescription>
							Configure email template event and content.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						{/* Dynamic Variables Info */}
						<button
							type="button"
							className="flex w-full items-center justify-between rounded-md bg-blue-50 px-4 py-2 text-sm text-blue-800"
							onClick={() =>
								setIsDynamicVariableInfoOpen(!isDynamicVariableInfoOpen)
							}
						>
							<span>
								<strong>Note:</strong> You can add dynamic variables to subject
								and email body. Click to see the list.
							</span>
							{isDynamicVariableInfoOpen ? (
								<ChevronUp className="h-4 w-4" />
							) : (
								<ChevronDown className="h-4 w-4" />
							)}
						</button>
						{isDynamicVariableInfoOpen && (
							<div className="max-h-48 overflow-y-auto rounded-md bg-gray-50">
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>Variable</TableHead>
											<TableHead>Description</TableHead>
										</TableRow>
									</TableHeader>
									<TableBody>
										{templateVariables.map((i) => (
											<TableRow key={i.text}>
												<TableCell>
													<code className="rounded bg-gray-200 px-1 py-0.5 text-xs">
														{`{{.${i.text}}}`}
													</code>
												</TableCell>
												<TableCell className="text-sm">
													{i.description}
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							</div>
						)}

						{/* Event Name */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Event Name
							</label>
							<Select
								value={templateData[EmailTemplateInputDataFields.EVENT_NAME]}
								onChange={(e) =>
									inputChangehandler(
										EmailTemplateInputDataFields.EVENT_NAME,
										e.currentTarget.value,
									)
								}
							>
								{Object.entries(emailTemplateEventNames).map(([key, value]) => (
									<option value={value} key={key}>
										{key}
									</option>
								))}
							</Select>
						</div>

						{/* Subject */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Subject
							</label>
							<Input
								placeholder="Subject Line"
								value={templateData[EmailTemplateInputDataFields.SUBJECT]}
								isInvalid={!validator[EmailTemplateInputDataFields.SUBJECT]}
								onChange={(e) =>
									inputChangehandler(
										EmailTemplateInputDataFields.SUBJECT,
										e.currentTarget.value,
									)
								}
							/>
						</div>

						{/* Template Body Editor Selection */}
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Template Body
							</label>
							<div className="flex gap-6">
								<label className="flex items-center gap-2 text-sm">
									<input
										type="radio"
										name="editor"
										value={EmailTemplateEditors.PLAIN_HTML_EDITOR}
										checked={editor === EmailTemplateEditors.PLAIN_HTML_EDITOR}
										onChange={(e) => setEditor(e.target.value)}
									/>
									Plain HTML
								</label>
								<label className="flex items-center gap-2 text-sm">
									<input
										type="radio"
										name="editor"
										value={EmailTemplateEditors.UNLAYER_EDITOR}
										checked={editor === EmailTemplateEditors.UNLAYER_EDITOR}
										onChange={(e) => setEditor(e.target.value)}
									/>
									Unlayer Editor
								</label>
							</div>
						</div>

						{/* Editor */}
						<div className="border border-gray-200 rounded-md overflow-hidden">
							{editor === EmailTemplateEditors.UNLAYER_EDITOR ? (
								<EmailEditor ref={emailEditorRef} onReady={onReady} />
							) : (
								<Textarea
									value={templateData.template}
									onChange={(e) => {
										setTemplateData({
											...templateData,
											[EmailTemplateInputDataFields.TEMPLATE]: e.target.value,
										});
									}}
									placeholder="Template HTML"
									className="h-[500px] border-0 rounded-none resize-none"
								/>
							)}
						</div>
					</div>

					<SheetFooter className="mt-6">
						<Button variant="outline" onClick={resetData} disabled={loading}>
							Reset
						</Button>
						<Button
							onClick={saveData}
							isLoading={loading}
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

export default UpdateEmailTemplate;
