// Webpack Config - Production
const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const TerserPlugin = require('terser-webpack-plugin');

const htmlChunks = [ 'app', 'login' ];
const getExcludeHtmlChunks = (value) => htmlChunks.filter((htmlChunk) => value !== htmlChunk);
const nodeModulesRE = /[\\/]node_modules[\\/]/;
const antdRE = /[\\/]node_modules[\\/](antd|@ant-design|rc-[^\\/]+)[\\/]/;
const graphRE = /[\\/]node_modules[\\/](@antv|d3|d3-[^\\/]+|dagre|graphlib)[\\/]/;
const editorRE = /[\\/]node_modules[\\/](codemirror|codemirror-rego|react-codemirror2)[\\/]/;

module.exports = require('./webpack.base.babel')({
  mode: 'production',
  entry: {
    app: path.join(process.cwd(), 'app/app.js'),
    login: path.join(process.cwd(), 'login/login.js')
  },
  output: {
    filename: 'js/[name].[chunkhash].js',
    chunkFilename: 'js/[name].[chunkhash].chunk.js'
  },
  optimization: {
    minimize: true,
    minimizer: [
      new TerserPlugin({
        terserOptions: {
          compress: {
            comparisons: false
          },
          mangle: true,
          format: {
            comments: false,
            ascii_only: true
          }
        },
        parallel: true,
        extractComments: false
      })
    ],
    moduleIds: 'deterministic',
    chunkIds: 'deterministic',
    nodeEnv: 'production',
    sideEffects: true,
    concatenateModules: true,
    runtimeChunk: 'single',
    splitChunks: {
      chunks: 'all',
      maxInitialRequests: 20,
      maxAsyncRequests: 20,
      minSize: 30000,
      cacheGroups: {
        antdVendor: {
          test: antdRE,
          name: 'vendor-antd',
          chunks: 'all',
          priority: 30,
          reuseExistingChunk: true
        },
        graphVendor: {
          test: graphRE,
          name: 'vendor-graph',
          chunks: 'async',
          priority: 25,
          reuseExistingChunk: true
        },
        editorVendor: {
          test: editorRE,
          name: 'vendor-editor',
          chunks: 'async',
          priority: 25,
          reuseExistingChunk: true
        },
        asyncVendor: {
          test: nodeModulesRE,
          name: 'vendor-async',
          chunks: 'async',
          priority: 10,
          reuseExistingChunk: true
        },
        commonVendor: {
          test: nodeModulesRE,
          name: 'vendor-common',
          chunks: 'initial',
          priority: 5,
          reuseExistingChunk: true
        },
        routeCommon: {
          name: 'route-common',
          minChunks: 2,
          chunks: 'async',
          priority: 0,
          reuseExistingChunk: true
        }
      }
    }
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: 'app/index.html',
      minify: {
        removeComments: true,
        collapseWhitespace: true,
        removeRedundantAttributes: true,
        useShortDoctype: true,
        removeEmptyAttributes: true,
        removeStyleLinkTypeAttributes: true,
        keepClosingSlash: true,
        minifyJS: true,
        minifyCSS: true,
        minifyURLs: true
      },
      inject: true,
      excludeChunks: getExcludeHtmlChunks('app')
    }),
    new HtmlWebpackPlugin({
      template: 'login/login.html',
      filename: 'login.html',
      minify: {
        removeComments: true,
        collapseWhitespace: true,
        removeRedundantAttributes: true,
        useShortDoctype: true,
        removeEmptyAttributes: true,
        removeStyleLinkTypeAttributes: true,
        keepClosingSlash: true,
        minifyJS: true,
        minifyCSS: true,
        minifyURLs: true
      },
      excludeChunks: getExcludeHtmlChunks('login'),
      inject: true
    })
  ],
  performance: {
    assetFilter: assetFilename =>
      !/(\.map$)|(^(main\.|favicon\.))/.test(assetFilename)
  }
});
