import js from '@eslint/js';
import importPlugin from 'eslint-plugin-import';
import jestPlugin from 'eslint-plugin-jest';
import promisePlugin from 'eslint-plugin-promise';
import unusedImportsPlugin from 'eslint-plugin-unused-imports';

export default [
	js.configs.recommended,
	{
		ignores: ['coverage/**', 'node_modules/**'],
	},
	{
		files: ['**/*.js'],
		languageOptions: {
			ecmaVersion: 'latest',
			sourceType: 'module',
			globals: {
				console: 'readonly',
				process: 'readonly',
				Buffer: 'readonly',
				__dirname: 'readonly',
				__filename: 'readonly',
				exports: 'writable',
				global: 'readonly',
				module: 'writable',
				require: 'readonly',
			},
		},
		plugins: {
			import: importPlugin,
			jest: jestPlugin,
			promise: promisePlugin,
			'unused-imports': unusedImportsPlugin,
		},
		rules: {
			camelcase: 0,
			curly: 2,
			eqeqeq: 2,
			'func-call-spacing': 0,
			'guard-for-in': 2,
			indent: ['error', 'tab', { SwitchCase: 1 }],
			'key-spacing': 0,
			'max-depth': ['error', { max: 5 }],
			'no-irregular-whitespace': 2,
			'no-multi-spaces': 0,
			'padded-blocks': 0,
			quotes: ['error', 'single', { allowTemplateLiterals: true }],
			semi: 0,
			'no-path-concat': 1,
			'no-undef': 2,
			'unused-imports/no-unused-imports': 'error',
			'no-unused-vars': 2,
			'no-var': 1,
		},
	},
	{
		files: ['**/*.test.js', '**/*.spec.js'],
		languageOptions: {
			globals: {
				afterAll: 'readonly',
				afterEach: 'readonly',
				beforeAll: 'readonly',
				beforeEach: 'readonly',
				describe: 'readonly',
				expect: 'readonly',
				it: 'readonly',
				jest: 'readonly',
				test: 'readonly',
			},
		},
	},
];
