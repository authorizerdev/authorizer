const esbuild = require('esbuild');

const __is_prod__ = process.env.NODE_ENV === 'production';

const commonConfig = {
	entryPoints: ['src/index.tsx'],
	chunkNames: '[name]-[hash]',
	bundle: true,
	minify: __is_prod__,
	outdir: 'build',
	splitting: true,
	format: 'esm',
	logLevel: 'info',
};

if (__is_prod__) {
	esbuild.build(commonConfig).catch(() => process.exit(1));
} else {
	esbuild
		.context(commonConfig)
		.then((ctx) => {
			ctx.watch();
			console.log('Watching for changes...');
		})
		.catch(() => process.exit(1));
}
