// Webpack Config - Base
const path = require('path');
const webpack = require('webpack');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const packageName = require('../../package.json').name;

const keepRuntimeAssetUrl = url => !url.startsWith('/assets/');

module.exports = options => ({
  mode: options.mode,
  entry: options.entry,
  output: {
    library: {
      name: `${packageName}-[name]`,
      type: 'umd'
    },
    chunkLoadingGlobal: `webpackJsonp_${packageName}`,
    assetModuleFilename: 'assets/[hash][ext][query]',
    publicPath: '/',
    path: path.resolve(process.cwd(), 'build'),
    ...options.output
  },
  optimization: options.optimization,
  module: {
    rules: [
      {
        test: /\.jsx?$/,
        exclude: /node_modules/,
        use: {
          loader: 'babel-loader'
        }
      },
      {
        // Preprocess 3rd party .css files located in node_modules
        test: /\.css$/,
        include: /node_modules/,
        use: [
          'style-loader',
          {
            loader: 'css-loader',
            options: {
              url: {
                filter: keepRuntimeAssetUrl
              }
            }
          }
        ]
      },
      {
        // Keep third-party Less support generic; AntD uses precompiled CSS now.
        test: /\.less$/,
        include: /node_modules/,
        use: [
          'style-loader',
          {
            loader: 'css-loader',
            options: {
              url: {
                filter: keepRuntimeAssetUrl
              }
            }
          },
          {
            loader: 'less-loader',
            options: {}
          }
        ]
      },
      {
        // Preprocess our own .css files
        test: /\.(css|less)$/,
        exclude: /node_modules/,
        use: [
          'style-loader',
          {
            loader: 'css-loader',
            options: {
              esModule: false,
              url: {
                filter: keepRuntimeAssetUrl
              },
              importLoaders: 1,
              modules: {
                namedExport: false,
                localIdentName: '[path][local]-[hash:base64:5]'
              }
            }
          },
          {
            loader: 'less-loader',
            options: {}
          }
        ]
      },
      {
        test: /\.(eot|otf|ttf|woff|woff2)$/,
        type: 'asset/resource'
      },
      {
        test: /\.svg$/,
        type: 'asset',
        parser: {
          dataUrlCondition: {
            maxSize: 10 * 1024
          }
        }
      },
      {
        test: /\.(jpg|png|gif)$/,
        type: 'asset',
        parser: {
          dataUrlCondition: {
            maxSize: 10 * 1024
          }
        }
      },
      {
        test: /\.(mp4|webm)$/,
        type: 'asset',
        parser: {
          dataUrlCondition: {
            maxSize: 10000
          }
        }
      }
    ]
  },
  plugins: options.plugins.concat([
    new webpack.EnvironmentPlugin({
      NODE_ENV: 'development'
    }),
    new webpack.ContextReplacementPlugin(/moment[/\\]locale$/, /zh-cn/),
    new CopyWebpackPlugin({
      patterns: [
        {
          from: path.resolve(process.cwd(), 'vendor'),
          to: 'vendor'
        },
        {
          from: path.resolve(process.cwd(), 'app/assets'),
          to: 'assets'
        }
      ]
    })
  ]),
  resolve: {
    modules: [ 'node_modules', 'app' ],
    aliasFields: [ 'browser' ],
    mainFields: [ 'browser', 'module', 'main' ],
    alias: {
      'intl-relativeformat$': 'intl-relativeformat/lib/main'
    },
    extensions: [ '.js', '.jsx', '.react.js' ]
  },
  externals: {
    'react': 'React',
    'react-dom': 'ReactDOM',
    'react-dom/client': 'ReactDOMClient',
    'react-redux': 'ReactRedux',
    'react-router-dom': 'ReactRouterDOM',
    'connected-react-router': 'ConnectedReactRouter',
    'redux-saga': 'ReduxSaga',
    'redux': 'Redux',
    'moment': 'moment',
    'react-intl': 'ReactIntl',
    'reselect': 'Reselect'
  },
  devtool: options.devtool,
  target: 'web',
  performance: options.performance || {}
});
