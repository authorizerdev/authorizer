// @vitest-environment jsdom
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render } from '@testing-library/react';
import InputField from './InputField';
import { ArrayInputType, MultiSelectInputType } from '../constants';

// Regression coverage for two runtime crashes InputField used to hit despite
// TypeScript considering the code "safe":
//  1. A `fieldVisibility!` non-null assertion threw when a caller passed only
//     setFieldVisibility (the two props are independently optional).
//  2. `variables[inputType] as string[]` .map()'d an undefined array when the
//     field's value was never initialized.
describe('InputField', () => {
	it('does not throw when setFieldVisibility is passed without fieldVisibility', () => {
		const setFieldVisibility = vi.fn();
		const { container } = render(
			<InputField
				inputType="CLIENT_SECRET"
				variables={{ CLIENT_SECRET: 'shhh' }}
				setVariables={() => {}}
				setFieldVisibility={setFieldVisibility}
			/>,
		);
		// The first button is the show/hide (eye) toggle. Clicking it used to
		// throw on `fieldVisibility!`; the guard now degrades gracefully.
		const toggle = container.querySelector('button');
		expect(toggle).not.toBeNull();
		expect(() => fireEvent.click(toggle!)).not.toThrow();
		expect(setFieldVisibility).not.toHaveBeenCalled();
	});

	it('renders an empty ArrayInputType list (no crash) when the array is undefined', () => {
		expect(() =>
			render(
				<InputField
					inputType={ArrayInputType.ROLES}
					variables={{}}
					setVariables={() => {}}
				/>,
			),
		).not.toThrow();
	});

	it('renders an empty MultiSelectInputType list (no crash) when the array is undefined', () => {
		expect(() =>
			render(
				<InputField
					inputType={MultiSelectInputType.USER_ROLES}
					variables={{}}
					setVariables={() => {}}
					availableRoles={['admin']}
				/>,
			),
		).not.toThrow();
	});
});
