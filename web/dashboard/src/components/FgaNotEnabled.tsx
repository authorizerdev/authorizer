import React, { useState } from 'react';
import { ShieldCheck, Copy, Check, ExternalLink, Database } from 'lucide-react';

// FgaNotEnabled renders a helpful empty state shown when the backend reports
// that fine-grained authorization is not enabled. FGA reuses the main database
// automatically when it is SQLite/Postgres/MySQL; this screen typically means
// the main database is not OpenFGA-compatible (e.g. MongoDB, DynamoDB), so a
// SQL store must be pointed to explicitly via --fga-store.
const ENABLE_CMD = '--fga-store=postgres --fga-store-url=postgres://user:pass@host:5432/db';

const StoreChip = ({ label }: { label: string }) => (
	<code className="rounded bg-gray-200/70 px-1 py-0.5 text-[11px] font-medium text-gray-700">
		{label}
	</code>
);

const FgaNotEnabled = () => {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		try {
			await navigator.clipboard.writeText(ENABLE_CMD);
			setCopied(true);
			window.setTimeout(() => setCopied(false), 2000);
		} catch {
			/* clipboard unavailable — the command is visible to copy manually */
		}
	};

	return (
		<div className="flex min-h-[55vh] items-center justify-center px-4">
			<div className="w-full max-w-xl rounded-2xl border border-gray-100 bg-white p-8 text-center shadow-sm sm:p-10">
				<div className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-blue-50">
					<ShieldCheck className="h-7 w-7 text-blue-600" aria-hidden="true" />
				</div>

				<h2 className="text-xl font-semibold text-gray-900">
					Fine-Grained Authorization isn&rsquo;t enabled yet
				</h2>
				<p className="mx-auto mt-2 max-w-md text-sm leading-relaxed text-gray-500">
					FGA turns on automatically when Authorizer runs on{' '}
					<span className="font-medium text-gray-700">SQLite, Postgres or MySQL</span>{' '}
					&mdash; it reuses your database. Your current database isn&rsquo;t
					supported by OpenFGA, so point FGA at a SQL store to enable it.
				</p>

				<div className="mt-6 rounded-xl border border-gray-100 bg-gray-50 p-4 text-left">
					<div className="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-gray-500">
						<Database className="h-3.5 w-3.5" aria-hidden="true" />
						Point FGA at a SQL store
					</div>
					<div className="flex items-center justify-between gap-3 rounded-lg bg-gray-900 px-3 py-2.5">
						<code className="overflow-x-auto whitespace-nowrap text-xs text-gray-100">
							{ENABLE_CMD}
						</code>
						<button
							type="button"
							onClick={handleCopy}
							aria-label={copied ? 'Command copied' : 'Copy command'}
							className="flex shrink-0 items-center gap-1 rounded-md bg-white/10 px-2 py-1 text-xs text-gray-200 transition-colors hover:bg-white/20 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
						>
							{copied ? (
								<>
									<Check className="h-3.5 w-3.5 text-green-400" aria-hidden="true" />
									Copied
								</>
							) : (
								<>
									<Copy className="h-3.5 w-3.5" aria-hidden="true" />
									Copy
								</>
							)}
						</button>
					</div>
					<p className="mt-2 text-xs leading-relaxed text-gray-500">
						<StoreChip label="--fga-store" /> accepts{' '}
						<StoreChip label="sqlite" />, <StoreChip label="postgres" /> or{' '}
						<StoreChip label="mysql" /> (or <StoreChip label="memory" /> for
						dev). Then reload this page.
					</p>
				</div>

				<a
					href="https://docs.authorizer.dev"
					target="_blank"
					rel="noopener noreferrer"
					className="mt-6 inline-flex items-center gap-1.5 rounded-lg bg-blue-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
				>
					Read the documentation
					<ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
				</a>
			</div>
		</div>
	);
};

export default FgaNotEnabled;
