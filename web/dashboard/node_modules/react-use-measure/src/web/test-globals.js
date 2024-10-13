/*  Global stubs for running tests with Webpack 5 */
const process = {
  version: '',
  env: {
    NODE_ENV: 'development',
  },
  cwd: function () {
    return '/'
  },
}

window.process = process
