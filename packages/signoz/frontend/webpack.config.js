/* eslint-disable @typescript-eslint/no-var-requires */
// shared config (dev and prod)
const { resolve } = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const { sentryWebpackPlugin } = require('@sentry/webpack-plugin');
const portFinderSync = require('portfinder-sync');
const dotenv = require('dotenv');
const webpack = require('webpack');
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin');
const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer');

// 로컬 개발 시에만 .env를 로드하고, Docker 이미지 빌드 시(DOCKER_IMG_BUILD=true)에는
// 외부에서 주입된 환경변수만 사용하도록 분기
if (process.env.DOCKER_IMG_BUILD !== 'true') {
	dotenv.config();
}

console.log(resolve(__dirname, './src/'));

const cssLoader = 'css-loader';
const sassLoader = 'sass-loader';
const styleLoader = 'style-loader';

const plugins = [
	new HtmlWebpackPlugin({
		template: 'src/index.html.ejs',
		PYLON_APP_ID: process.env.PYLON_APP_ID,
		APPCUES_APP_ID: process.env.APPCUES_APP_ID,
		POSTHOG_KEY: process.env.POSTHOG_KEY,
		SENTRY_AUTH_TOKEN: process.env.SENTRY_AUTH_TOKEN,
		SENTRY_ORG: process.env.SENTRY_ORG,
		SENTRY_PROJECT_ID: process.env.SENTRY_PROJECT_ID,
		SENTRY_DSN: process.env.SENTRY_DSN,
		TUNNEL_URL: process.env.TUNNEL_URL,
		TUNNEL_DOMAIN: process.env.TUNNEL_DOMAIN,
	}),
	new webpack.ProvidePlugin({
		process: 'process/browser',
	}),
	new webpack.DefinePlugin({
		'process.env': JSON.stringify({
			NODE_ENV: process.env.NODE_ENV,
			FRONTEND_API_ENDPOINT: process.env.FRONTEND_API_ENDPOINT,
			WEBSOCKET_API_ENDPOINT: process.env.WEBSOCKET_API_ENDPOINT,
			K_O11Y_ENDPOINT: process.env.K_O11Y_ENDPOINT,
			PYLON_APP_ID: process.env.PYLON_APP_ID,
			PYLON_IDENTITY_SECRET: process.env.PYLON_IDENTITY_SECRET,
			APPCUES_APP_ID: process.env.APPCUES_APP_ID,
			POSTHOG_KEY: process.env.POSTHOG_KEY,
			SENTRY_AUTH_TOKEN: process.env.SENTRY_AUTH_TOKEN,
			SENTRY_ORG: process.env.SENTRY_ORG,
			SENTRY_PROJECT_ID: process.env.SENTRY_PROJECT_ID,
			SENTRY_DSN: process.env.SENTRY_DSN,
			TUNNEL_URL: process.env.TUNNEL_URL,
			TUNNEL_DOMAIN: process.env.TUNNEL_DOMAIN,
		}),
	}),
	sentryWebpackPlugin({
		authToken: process.env.SENTRY_AUTH_TOKEN,
		org: process.env.SENTRY_ORG,
		project: process.env.SENTRY_PROJECT_ID,
	}),
];

if (process.env.BUNDLE_ANALYSER === 'true') {
	plugins.push(new BundleAnalyzerPlugin({ analyzerMode: 'server' }));
}

/**
 * @type {import('webpack').Configuration}
 */
const config = {
	mode: 'development',
	devtool: 'source-map',
	entry: resolve(__dirname, './src/index.tsx'),
	devServer: {
		historyApiFallback: {
			disableDotRule: true,
		},
		open: true,
		hot: true,
		liveReload: true,
		port: portFinderSync.getPort(3301),
		// // 🎯 HTTPS 활성화 (Mixed Content 에러 방지)
		// server: {
		// 	type: 'https',
		// },
		static: {
			directory: resolve(__dirname, 'public'),
			publicPath: '/',
			watch: true,
		},
		allowedHosts: 'all',
		headers: {
			'Access-Control-Allow-Origin': '*',
			'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
			'Access-Control-Allow-Headers': 'X-Requested-With, content-type, Authorization',
		},
	},
	target: 'web',
	output: {
		path: resolve(__dirname, './build'),
		publicPath: '/',
	},
	resolve: {
		extensions: ['.ts', '.tsx', '.js', '.jsx'],
		plugins: [new TsconfigPathsPlugin({})],
		fallback: { 'process/browser': require.resolve('process/browser') },
	},
	module: {
		rules: [
			// Fix for @grafana/data ESM imports
			{
				test: /\.m?js$/,
				resolve: {
					fullySpecified: false,
				},
			},
			{
				test: [/\.jsx?$/, /\.tsx?$/],
				use: ['babel-loader'],
				exclude: /node_modules/,
			},
			// Add a rule for Markdown files using raw-loader
			{
				test: /\.md$/,
				use: 'raw-loader',
			},
			// CSS Modules - only for *.module.css files
			{
				test: /\.module\.css$/,
				use: [
					styleLoader,
					{
						loader: cssLoader,
						options: {
							modules: true,
						},
					},
				],
			},
			// Regular CSS - for all other .css files (like ServiceMap CSS)
			{
				test: /\.css$/,
				exclude: /\.module\.css$/,
				use: [styleLoader, cssLoader],
			},
			{
				test: /\.(jpe?g|png|gif|svg)$/i,
				type: 'asset',
			},
			{
				test: /\.(ttf|eot|woff|woff2)$/,
				use: ['file-loader'],
			},
			{
				test: /\.less$/i,
				use: [
					{
						loader: styleLoader,
					},
					{
						loader: cssLoader,
						options: {
							modules: true,
						},
					},
					{
						loader: 'less-loader',
						options: {
							lessOptions: {
								javascriptEnabled: true,
							},
						},
					},
				],
			},
			{
				test: /\.s[ac]ss$/i,
				use: [
					// Creates `style` nodes from JS strings
					styleLoader,
					// Translates CSS into CommonJS
					cssLoader,
					// Compiles Sass to CSS
					sassLoader,
				],
			},
		],
	},
	plugins,
	performance: {
		hints: false,
	},
	optimization: {
		minimize: false,
	},
};

module.exports = config;
