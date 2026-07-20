import React from 'react';
import { BookOpen, ExternalLink } from 'lucide-react';

// Curated links to the OpenFGA / ReBAC documentation that back Authorizer's
// fine-grained authorization. Shown on the FGA pages.
const LINKS = [
	{ label: 'What is FGA / ReBAC', href: 'https://openfga.dev/docs/fga' },
	{ label: 'Concepts', href: 'https://openfga.dev/docs/concepts' },
	{
		label: 'Modeling guide',
		href: 'https://openfga.dev/docs/modeling/getting-started',
	},
	{
		label: 'Model DSL',
		href: 'https://openfga.dev/docs/configuration-language',
	},
	{
		label: 'Relationship tuples',
		href: 'https://openfga.dev/docs/concepts#what-is-a-relationship-tuple',
	},
];

const DocsLinks = () => (
	<div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 text-xs text-gray-500">
		<span className="inline-flex items-center gap-1.5 font-medium text-gray-600">
			<BookOpen className="h-3.5 w-3.5" aria-hidden="true" />
			Learn more
		</span>
		{LINKS.map((l) => (
			<a
				key={l.href}
				href={l.href}
				target="_blank"
				rel="noopener noreferrer"
				className="inline-flex items-center gap-0.5 text-blue-600 hover:underline"
			>
				{l.label}
				<ExternalLink className="h-3 w-3" aria-hidden="true" />
			</a>
		))}
	</div>
);

export default DocsLinks;
