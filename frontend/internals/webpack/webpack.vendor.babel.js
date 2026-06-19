// Webpack Config - Vendor
const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');

module.exports = {
  mode: 'production',
  entry: path.resolve(process.cwd(), 'app/vendor.js'),
  output: {
    filename: 'vendors.js',
    path: path.resolve(process.cwd(), 'vendor/react/'),
    library: {
      name: '[name]',
      type: 'window'
    }
  },
  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        use: [
          {
            loader: 'babel-loader'
          }
        ]
      }
    ]
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
    ]
  },
  resolve: {
    modules: [ 'app', 'node_modules' ],
    aliasFields: ['main'],
    descriptionFiles: ['package.json'],
    mainFields: [ 'browser', 'module', 'main' ],
    extensions: [ '.js', '.jsx' ]
  }
};
