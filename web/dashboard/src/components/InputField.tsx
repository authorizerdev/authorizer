import React, { useState } from 'react';
import { Copy, Eye, EyeOff, Plus, X, ChevronDown } from 'lucide-react';
import { Input } from './ui/input';
import { Select } from './ui/select';
import { Textarea } from './ui/textarea';
import { Switch } from './ui/switch';
import {
	ArrayInputOperations,
	ArrayInputType,
	SelectInputType,
	HiddenInputType,
	TextInputType,
	TextAreaInputType,
	SwitchInputType,
	DateInputType,
	MultiSelectInputType,
} from '../constants';
import { copyTextToClipboard } from '../utils';

interface InputFieldProps {
	inputType: string;
	variables: Record<string, string | boolean | string[]>;
	setVariables: (vars: Record<string, string | boolean | string[]>) => void;
	fieldVisibility?: Record<string, boolean>;
	setFieldVisibility?: (vis: Record<string, boolean>) => void;
	availableRoles?: string[];
	hasReversedValue?: boolean;
	options?: Record<string, string | null>;
	value?: string;
}

const InputField = ({
	inputType,
	variables,
	setVariables,
	fieldVisibility,
	setFieldVisibility,
	availableRoles,
	hasReversedValue,
	options,
}: InputFieldProps) => {
	const [availableUserRoles] = useState<string[]>(availableRoles || []);
	const [inputFieldVisibility, setInputFieldVisibility] = useState<
		Record<string, boolean>
	>({
		ROLES: false,
		DEFAULT_ROLES: false,
		PROTECTED_ROLES: false,
		ALLOWED_ORIGINS: false,
		roles: false,
	});
	const [inputData, setInputData] = useState<Record<string, string>>({
		ROLES: '',
		DEFAULT_ROLES: '',
		PROTECTED_ROLES: '',
		ALLOWED_ORIGINS: '',
		roles: '',
	});
	const [multiSelectOpen, setMultiSelectOpen] = useState(false);

	const updateInputHandler = (
		type: string,
		operation: string,
		role: string = '',
	) => {
		if (operation === ArrayInputOperations.APPEND) {
			if (inputData[type] !== '') {
				setVariables({
					...variables,
					[type]: [...(variables[type] as string[]), inputData[type]],
				});
				setInputData({ ...inputData, [type]: '' });
			}
			setInputFieldVisibility({ ...inputFieldVisibility, [type]: false });
		}
		if (operation === ArrayInputOperations.REMOVE) {
			const updatedEnvVars = (variables[type] as string[]).filter(
				(item: string) => item !== role,
			);
			setVariables({
				...variables,
				[type]: updatedEnvVars,
			});
		}
	};

	if (Object.values(TextInputType).includes(inputType)) {
		return (
			<div className="relative w-full">
				<Input
					value={(variables[inputType] as string) || ''}
					onChange={(e) =>
						setVariables({
							...variables,
							[inputType]: e.target.value,
						})
					}
					className="pr-8 h-8 text-sm"
				/>
				<button
					type="button"
					className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
					onClick={() =>
						copyTextToClipboard((variables[inputType] as string) || '')
					}
				>
					<Copy className="h-3.5 w-3.5" />
				</button>
			</div>
		);
	}

	if (Object.values(HiddenInputType).includes(inputType)) {
		return (
			<div className="relative w-full">
				<Input
					value={(variables[inputType] as string) || ''}
					onChange={(e) =>
						setVariables({
							...variables,
							[inputType]: e.target.value,
						})
					}
					type={
						fieldVisibility && !fieldVisibility[inputType] ? 'password' : 'text'
					}
					className="pr-16 h-8 text-sm"
				/>
				<div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
					<button
						type="button"
						className="text-gray-400 hover:text-gray-600"
						onClick={() =>
							setFieldVisibility?.({
								...fieldVisibility!,
								[inputType]: !fieldVisibility![inputType],
							})
						}
					>
						{fieldVisibility?.[inputType] ? (
							<EyeOff className="h-3.5 w-3.5" />
						) : (
							<Eye className="h-3.5 w-3.5" />
						)}
					</button>
					<button
						type="button"
						className="text-gray-400 hover:text-gray-600"
						onClick={() =>
							copyTextToClipboard((variables[inputType] as string) || '')
						}
					>
						<Copy className="h-3.5 w-3.5" />
					</button>
				</div>
			</div>
		);
	}

	if (Object.values(ArrayInputType).includes(inputType)) {
		const items = variables[inputType] as string[];
		return (
			<div className="flex w-full items-center gap-1 rounded-md border border-gray-300 px-2 py-1 overflow-x-auto">
				{items.map((role: string, index: number) => (
					<span
						key={index}
						className="group inline-flex items-center gap-1 rounded border border-gray-300 px-2 py-0.5 text-xs whitespace-nowrap"
					>
						{role}
						<button
							type="button"
							className="hidden group-hover:inline-flex text-gray-400 hover:text-gray-600"
							onClick={() =>
								updateInputHandler(inputType, ArrayInputOperations.REMOVE, role)
							}
						>
							<X className="h-3 w-3" />
						</button>
					</span>
				))}
				{inputFieldVisibility[inputType] ? (
					<Input
						type="text"
						className="h-6 min-w-[150px] border-0 p-0 text-xs focus-visible:ring-0"
						placeholder="add a new value"
						value={inputData[inputType] || ''}
						onChange={(e) => {
							setInputData({ ...inputData, [inputType]: e.target.value });
						}}
						onBlur={() =>
							updateInputHandler(inputType, ArrayInputOperations.APPEND)
						}
						onKeyDown={(event) => {
							if (event.key === 'Enter') {
								updateInputHandler(inputType, ArrayInputOperations.APPEND);
							}
						}}
					/>
				) : (
					<button
						type="button"
						className="inline-flex items-center rounded border border-gray-300 px-1.5 py-0.5"
						onClick={() =>
							setInputFieldVisibility({
								...inputFieldVisibility,
								[inputType]: true,
							})
						}
					>
						<Plus className="h-3 w-3" />
					</button>
				)}
			</div>
		);
	}

	if (Object.values(SelectInputType).includes(inputType)) {
		return (
			<Select
				value={(variables[inputType] as string) || ''}
				onChange={(e) =>
					setVariables({ ...variables, [inputType]: e.target.value })
				}
				className="h-8 text-sm"
			>
				{Object.entries(options || {}).map(([key, value]) => (
					<option value={value ?? ''} key={key}>
						{key}
					</option>
				))}
			</Select>
		);
	}

	if (Object.values(MultiSelectInputType).includes(inputType)) {
		const selectedRoles = variables[inputType] as string[];
		return (
			<div className="relative w-full">
				<div className="flex w-full items-center justify-between rounded-md border border-gray-300 px-2 py-1 min-h-[32px]">
					<div className="flex flex-wrap gap-1">
						{selectedRoles.map((role: string, index: number) => (
							<span
								key={index}
								className="group inline-flex items-center gap-1 rounded border border-gray-300 px-2 py-0.5 text-xs"
							>
								{role}
								<button
									type="button"
									className="hidden group-hover:inline-flex text-gray-400 hover:text-gray-600"
									onClick={() =>
										updateInputHandler(
											inputType,
											ArrayInputOperations.REMOVE,
											role,
										)
									}
								>
									<X className="h-3 w-3" />
								</button>
							</span>
						))}
					</div>
					<button
						type="button"
						className="ml-2 text-gray-400"
						onClick={() => setMultiSelectOpen(!multiSelectOpen)}
					>
						<ChevronDown className="h-4 w-4" />
					</button>
				</div>
				{multiSelectOpen && (
					<div className="absolute right-0 top-full z-10 mt-1 max-h-36 w-full overflow-y-auto rounded-md border border-gray-200 bg-white shadow-md">
						{availableUserRoles.map((role) => {
							const isChecked = selectedRoles.includes(role);
							return (
								<label
									key={`multiselect-menu-${role}`}
									className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm hover:bg-gray-100"
								>
									<input
										type="checkbox"
										checked={isChecked}
										onChange={() => {
											if (isChecked) {
												setVariables({
													...variables,
													[inputType]: selectedRoles.filter((r) => r !== role),
												});
											} else {
												setVariables({
													...variables,
													[inputType]: [...selectedRoles, role],
												});
											}
										}}
									/>
									{role}
								</label>
							);
						})}
					</div>
				)}
			</div>
		);
	}

	if (Object.values(TextAreaInputType).includes(inputType)) {
		return (
			<Textarea
				value={(variables[inputType] as string) || ''}
				onChange={(e) =>
					setVariables({
						...variables,
						[inputType]: e.target.value,
					})
				}
				className="text-sm"
			/>
		);
	}

	if (Object.values(SwitchInputType).includes(inputType)) {
		const checked = hasReversedValue
			? !(variables[inputType] as boolean)
			: (variables[inputType] as boolean);
		return (
			<div className="flex items-center gap-2">
				<span className="text-sm font-medium">Off</span>
				<Switch
					checked={checked}
					onCheckedChange={() => {
						setVariables({
							...variables,
							[inputType]: !variables[inputType],
						});
					}}
				/>
				<span className="text-sm font-medium">On</span>
			</div>
		);
	}

	if (Object.values(DateInputType).includes(inputType)) {
		return (
			<Input
				type="date"
				value={(variables[inputType] as string) || ''}
				onChange={(e) =>
					setVariables({ ...variables, [inputType]: e.target.value })
				}
				className="h-8 text-sm"
			/>
		);
	}

	return null;
};

export default InputField;
