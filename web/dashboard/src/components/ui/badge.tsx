import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

const badgeVariants = cva(
	'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
	{
		variants: {
			variant: {
				default: 'border-transparent bg-blue-100 text-blue-800',
				success: 'border-transparent bg-green-100 text-green-800',
				warning: 'border-transparent bg-yellow-100 text-yellow-800',
				destructive: 'border-transparent bg-red-100 text-red-800',
				outline: 'text-gray-700',
				secondary: 'border-transparent bg-gray-100 text-gray-800',
			},
		},
		defaultVariants: {
			variant: 'default',
		},
	},
);

export interface BadgeProps
	extends
		React.HTMLAttributes<HTMLDivElement>,
		VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
	return (
		<div className={cn(badgeVariants({ variant }), className)} {...props} />
	);
}

export { Badge, badgeVariants };
