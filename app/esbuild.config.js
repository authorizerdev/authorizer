const __is_prod__ = process.env.NODE_ENV === 'production';
require('esbuild').build({
	entryPoints: ['src/index.tsx'],
	chunkNames: '[name]-[hash]',
	bundle: true,
	minify: __is_prod__,
	outdir: 'build',
	splitting: true,
	format: 'esm',
	watch: !__is_prod__,
});
